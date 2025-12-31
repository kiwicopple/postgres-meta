package pgmeta

import (
	"fmt"
	"strings"
)

// ListTypes returns all custom types (enums and composites) in the specified schemas
func (c *Client) ListTypes(schemas []string) ([]PostgresType, error) {
	if len(schemas) == 0 {
		return []PostgresType{}, nil
	}

	placeholders := make([]string, len(schemas))
	args := make([]interface{}, len(schemas))
	for i, s := range schemas {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}

	sql := fmt.Sprintf(`
		SELECT
			t.oid::int8 AS id,
			t.typname AS name,
			n.nspname AS schema,
			t.typname AS format,
			CASE
				WHEN t.typtype = 'e' THEN
					(
						SELECT array_agg(e.enumlabel ORDER BY e.enumsortorder)
						FROM pg_catalog.pg_enum e
						WHERE e.enumtypid = t.oid
					)
				ELSE ARRAY[]::text[]
			END AS enums,
			CASE
				WHEN t.typtype = 'c' THEN
					(
						SELECT array_agg(
							json_build_object(
								'name', a.attname,
								'type_id', a.atttypid::int8
							)
							ORDER BY a.attnum
						)
						FROM pg_catalog.pg_attribute a
						WHERE a.attrelid = t.typrelid
						  AND a.attnum > 0
						  AND NOT a.attisdropped
					)
				ELSE NULL
			END AS attributes,
			pg_catalog.obj_description(t.oid, 'pg_type') AS comment,
			t.typrelid::int8 AS type_relation_id
		FROM pg_catalog.pg_type t
		JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
		WHERE (t.typtype = 'e' OR t.typtype = 'c')
		  AND n.nspname IN (%s)
		  AND t.typname NOT LIKE '%%_pkey'
		  AND t.typname NOT LIKE '%%_fkey'
		  AND (t.typrelid = 0 OR NOT EXISTS (
			SELECT 1 FROM pg_catalog.pg_class c
			WHERE c.oid = t.typrelid AND c.relkind IN ('r', 'v', 'm', 'f', 'p')
		  ))
		ORDER BY n.nspname, t.typname
	`, strings.Join(placeholders, ", "))

	rows, err := c.query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []PostgresType
	for rows.Next() {
		var t PostgresType
		var enums []string
		var attributesJSON []map[string]interface{}
		var typeRelID *int64

		if err := rows.Scan(
			&t.ID, &t.Name, &t.Schema, &t.Format, &enums,
			&attributesJSON, &t.Comment, &typeRelID,
		); err != nil {
			return nil, err
		}

		if enums == nil {
			t.Enums = []string{}
		} else {
			t.Enums = enums
		}

		t.Attributes = make([]TypeAttribute, 0)
		if attributesJSON != nil {
			for _, attr := range attributesJSON {
				t.Attributes = append(t.Attributes, TypeAttribute{
					Name:   attr["name"].(string),
					TypeID: int64(attr["type_id"].(float64)),
				})
			}
		}

		if typeRelID != nil && *typeRelID != 0 {
			t.TypeRelationID = typeRelID
		}

		types = append(types, t)
	}

	return types, nil
}

// GetTypeByID returns a type by its OID
func (c *Client) GetTypeByID(id int64) (*PostgresType, error) {
	sql := `
		SELECT
			t.oid::int8 AS id,
			t.typname AS name,
			n.nspname AS schema,
			t.typname AS format,
			CASE
				WHEN t.typtype = 'e' THEN
					(
						SELECT array_agg(e.enumlabel ORDER BY e.enumsortorder)
						FROM pg_catalog.pg_enum e
						WHERE e.enumtypid = t.oid
					)
				ELSE ARRAY[]::text[]
			END AS enums,
			pg_catalog.obj_description(t.oid, 'pg_type') AS comment
		FROM pg_catalog.pg_type t
		JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
		WHERE t.oid = $1
	`

	var t PostgresType
	var enums []string
	err := c.queryRow(sql, id).Scan(
		&t.ID, &t.Name, &t.Schema, &t.Format, &enums, &t.Comment,
	)
	if err != nil {
		return nil, err
	}

	if enums == nil {
		t.Enums = []string{}
	} else {
		t.Enums = enums
	}

	return &t, nil
}
