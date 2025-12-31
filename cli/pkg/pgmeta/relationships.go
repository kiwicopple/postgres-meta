package pgmeta

import (
	"fmt"
	"strings"
)

// ListRelationships returns all foreign key relationships in the specified schemas
func (c *Client) ListRelationships(schemas []string) ([]PostgresRelationship, error) {
	if len(schemas) == 0 {
		return []PostgresRelationship{}, nil
	}

	placeholders := make([]string, len(schemas))
	args := make([]interface{}, len(schemas))
	for i, s := range schemas {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}

	// This query is adapted from PostgREST's schema cache logic
	sql := fmt.Sprintf(`
		WITH
		fk_constraints AS (
			SELECT
				con.oid,
				con.conname,
				con.conrelid,
				con.confrelid,
				con.conkey,
				con.confkey,
				ns1.nspname AS schema,
				rel1.relname AS relation,
				ns2.nspname AS referenced_schema,
				rel2.relname AS referenced_relation
			FROM pg_catalog.pg_constraint con
			JOIN pg_catalog.pg_class rel1 ON rel1.oid = con.conrelid
			JOIN pg_catalog.pg_namespace ns1 ON ns1.oid = rel1.relnamespace
			JOIN pg_catalog.pg_class rel2 ON rel2.oid = con.confrelid
			JOIN pg_catalog.pg_namespace ns2 ON ns2.oid = rel2.relnamespace
			WHERE con.contype = 'f'
			  AND (ns1.nspname IN (%s) OR ns2.nspname IN (%s))
		),
		fk_columns AS (
			SELECT
				fk.oid,
				fk.conname,
				fk.schema,
				fk.relation,
				fk.referenced_schema,
				fk.referenced_relation,
				array_agg(att1.attname ORDER BY ord.n) AS columns,
				array_agg(att2.attname ORDER BY ord.n) AS referenced_columns
			FROM fk_constraints fk
			CROSS JOIN LATERAL unnest(fk.conkey, fk.confkey) WITH ORDINALITY AS ord(fk_col, ref_col, n)
			JOIN pg_catalog.pg_attribute att1 ON att1.attrelid = fk.conrelid AND att1.attnum = ord.fk_col
			JOIN pg_catalog.pg_attribute att2 ON att2.attrelid = fk.confrelid AND att2.attnum = ord.ref_col
			GROUP BY fk.oid, fk.conname, fk.schema, fk.relation, fk.referenced_schema, fk.referenced_relation
		),
		unique_constraints AS (
			SELECT
				con.conrelid,
				con.conkey
			FROM pg_catalog.pg_constraint con
			WHERE con.contype IN ('p', 'u')
		)
		SELECT
			fk.oid::int8 AS id,
			fk.conname AS constraint_name,
			fk.schema,
			fk.relation,
			fk.columns,
			fk.referenced_schema,
			fk.referenced_relation,
			fk.referenced_columns,
			EXISTS (
				SELECT 1
				FROM unique_constraints uc
				WHERE uc.conrelid = (
					SELECT c.oid FROM pg_catalog.pg_class c
					JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
					WHERE n.nspname = fk.schema AND c.relname = fk.relation
				)
				AND uc.conkey = (
					SELECT array_agg(a.attnum ORDER BY ord)
					FROM unnest(fk.columns) WITH ORDINALITY AS cols(col, ord)
					JOIN pg_catalog.pg_attribute a ON a.attrelid = (
						SELECT c.oid FROM pg_catalog.pg_class c
						JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
						WHERE n.nspname = fk.schema AND c.relname = fk.relation
					) AND a.attname = cols.col
				)
			) AS is_one_to_one
		FROM fk_columns fk
		ORDER BY fk.schema, fk.relation, fk.conname
	`, strings.Join(placeholders, ", "), strings.Join(placeholders, ", "))

	// Duplicate the args for both IN clauses
	allArgs := append(args, args...)

	rows, err := c.query(sql, allArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []PostgresRelationship
	for rows.Next() {
		var r PostgresRelationship
		if err := rows.Scan(
			&r.ID, &r.ConstraintName, &r.Schema, &r.Relation,
			&r.Columns, &r.ReferencedSchema, &r.ReferencedRelation,
			&r.ReferencedColumns, &r.IsOneToOne,
		); err != nil {
			return nil, err
		}
		relationships = append(relationships, r)
	}

	return relationships, nil
}
