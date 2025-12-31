package pgmeta

import (
	"fmt"
	"strings"
)

// ListFunctions returns all functions in the specified schemas (excluding triggers)
func (c *Client) ListFunctions(schemas []string) ([]PostgresFunction, error) {
	if len(schemas) == 0 {
		return []PostgresFunction{}, nil
	}

	placeholders := make([]string, len(schemas))
	args := make([]interface{}, len(schemas))
	for i, s := range schemas {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}

	sql := fmt.Sprintf(`
		SELECT
			p.oid::int8 AS id,
			n.nspname AS schema,
			p.proname AS name,
			l.lanname AS language,
			CASE
				WHEN l.lanname = 'internal' THEN p.prosrc
				ELSE pg_get_functiondef(p.oid)
			END AS definition,
			pg_get_function_arguments(p.oid) AS argument_types,
			pg_get_function_identity_arguments(p.oid) AS identity_argument_types,
			p.prorettype::int8 AS return_type_id,
			pg_catalog.format_type(p.prorettype, NULL) AS return_type,
			NULLIF(rt.typrelid, 0)::int8 AS return_type_relation_id,
			p.proretset AS is_set_returning_function,
			CASE p.provolatile
				WHEN 'i' THEN 'IMMUTABLE'
				WHEN 's' THEN 'STABLE'
				WHEN 'v' THEN 'VOLATILE'
			END AS behavior,
			p.prosecdef AS security_definer,
			COALESCE(
				(
					SELECT array_agg(
						json_build_object(
							'mode', CASE modes.mode
								WHEN 'i' THEN 'in'
								WHEN 'o' THEN 'out'
								WHEN 'b' THEN 'inout'
								WHEN 'v' THEN 'variadic'
								WHEN 't' THEN 'table'
							END,
							'name', COALESCE(names.name, ''),
							'type_id', types.type_id::int8,
							'has_default', p.pronargdefaults > 0 AND idx.i > (array_length(p.proargtypes, 1) - p.pronargdefaults)
						)
						ORDER BY idx.i
					)
					FROM unnest(
						COALESCE(p.proallargtypes, p.proargtypes::oid[]),
						COALESCE(p.proargmodes, ARRAY[]::char[]),
						COALESCE(p.proargnames, ARRAY[]::text[])
					) WITH ORDINALITY AS args(type_id, mode, name, i)
					CROSS JOIN LATERAL (SELECT COALESCE(mode, 'i') AS mode) modes
					CROSS JOIN LATERAL (SELECT NULLIF(name, '') AS name) names
					CROSS JOIN LATERAL (SELECT i::int AS i) idx
				),
				ARRAY[]::json[]
			) AS args,
			pg_catalog.obj_description(p.oid, 'pg_proc') AS comment,
			p.proconfig AS config_params
		FROM pg_catalog.pg_proc p
		JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		JOIN pg_catalog.pg_language l ON l.oid = p.prolang
		LEFT JOIN pg_catalog.pg_type rt ON rt.oid = p.prorettype
		WHERE n.nspname IN (%s)
		  AND p.prokind = 'f'
		  AND NOT EXISTS (
			SELECT 1 FROM pg_catalog.pg_trigger t
			WHERE t.tgfoid = p.oid
		  )
		ORDER BY n.nspname, p.proname
	`, strings.Join(placeholders, ", "))

	rows, err := c.query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []PostgresFunction
	for rows.Next() {
		var f PostgresFunction
		var argsJSON []map[string]interface{}
		var comment *string
		var configParams []string

		if err := rows.Scan(
			&f.ID, &f.Schema, &f.Name, &f.Language, &f.Definition,
			&f.ArgumentTypes, &f.IdentityArgumentTypes, &f.ReturnTypeID, &f.ReturnType,
			&f.ReturnTypeRelationID, &f.IsSetReturningFunction, &f.Behavior,
			&f.SecurityDefiner, &argsJSON, &comment, &configParams,
		); err != nil {
			return nil, err
		}

		f.Comment = comment
		f.Args = make([]FunctionArg, 0)

		if argsJSON != nil {
			for _, arg := range argsJSON {
				fa := FunctionArg{
					Mode:       getString(arg, "mode"),
					Name:       getString(arg, "name"),
					TypeID:     getInt64(arg, "type_id"),
					HasDefault: getBool(arg, "has_default"),
				}
				f.Args = append(f.Args, fa)
			}
		}

		f.ConfigParams = make(map[string]string)
		if configParams != nil {
			for _, param := range configParams {
				parts := strings.SplitN(param, "=", 2)
				if len(parts) == 2 {
					f.ConfigParams[parts[0]] = parts[1]
				}
			}
		}

		functions = append(functions, f)
	}

	return functions, nil
}

// Helper functions for JSON parsing
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return int64(f)
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
