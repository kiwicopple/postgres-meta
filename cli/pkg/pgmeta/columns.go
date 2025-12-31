package pgmeta

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ListColumns returns all columns in the specified schemas
func (c *Client) ListColumns(schemas []string) ([]PostgresColumn, error) {
	if len(schemas) == 0 {
		return []PostgresColumn{}, nil
	}

	placeholders := make([]string, len(schemas))
	args := make([]interface{}, len(schemas))
	for i, s := range schemas {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}

	sql := fmt.Sprintf(`
		SELECT
			c.oid::int8 AS table_id,
			nc.nspname AS schema,
			c.relname AS table,
			(c.oid::text || '.' || a.attnum::text) AS id,
			a.attnum AS ordinal_position,
			a.attname AS name,
			CASE
				WHEN a.atthasdef THEN pg_get_expr(ad.adbin, ad.adrelid)
				ELSE NULL
			END AS default_value,
			CASE
				WHEN t.typtype = 'd' THEN
					CASE
						WHEN bt.typelem <> 0::oid AND bt.typlen = -1
						THEN 'ARRAY'
						WHEN nbt.nspname = 'pg_catalog'
						THEN format_type(t.typbasetype, NULL)
						ELSE 'USER-DEFINED'
					END
				ELSE
					CASE
						WHEN t.typelem <> 0::oid AND t.typlen = -1
						THEN 'ARRAY'
						WHEN nt.nspname = 'pg_catalog'
						THEN format_type(a.atttypid, NULL)
						ELSE 'USER-DEFINED'
					END
			END AS data_type,
			COALESCE(
				CASE
					WHEN t.typtype = 'd' THEN
						CASE
							WHEN bt.typelem <> 0::oid AND bt.typlen = -1
							THEN elem_bt.typname
							WHEN nbt.nspname = 'pg_catalog'
							THEN bt.typname
							ELSE t.typname
						END
					ELSE
						CASE
							WHEN t.typelem <> 0::oid AND t.typlen = -1
							THEN elem.typname
							WHEN nt.nspname = 'pg_catalog'
							THEN t.typname
							ELSE t.typname
						END
				END,
				''
			) AS format,
			CASE WHEN a.attidentity IN ('a', 'd') THEN TRUE ELSE FALSE END AS is_identity,
			CASE
				WHEN a.attidentity = 'a' THEN 'ALWAYS'
				WHEN a.attidentity = 'd' THEN 'BY DEFAULT'
				ELSE NULL
			END AS identity_generation,
			CASE WHEN a.attgenerated IN ('s') THEN TRUE ELSE FALSE END AS is_generated,
			NOT a.attnotnull AS is_nullable,
			CASE
				WHEN c.relkind = 'r' THEN TRUE
				ELSE pg_column_is_updatable(c.oid::regclass, a.attnum, false)
			END AS is_updatable,
			COALESCE(
				(
					SELECT TRUE
					FROM pg_catalog.pg_constraint con
					WHERE con.conrelid = c.oid
					  AND con.contype = 'u'
					  AND con.conkey = ARRAY[a.attnum]
				),
				FALSE
			) AS is_unique,
			CASE
				WHEN t.typtype = 'e' THEN
					(
						SELECT array_agg(e.enumlabel ORDER BY e.enumsortorder)
						FROM pg_catalog.pg_enum e
						WHERE e.enumtypid = t.oid
					)
				ELSE NULL
			END AS enums,
			(
				SELECT pg_get_constraintdef(con.oid)
				FROM pg_catalog.pg_constraint con
				WHERE con.conrelid = c.oid
				  AND con.contype = 'c'
				  AND a.attnum = ANY(con.conkey)
				LIMIT 1
			) AS check,
			pg_catalog.col_description(c.oid, a.attnum) AS comment
		FROM pg_catalog.pg_attribute a
		LEFT JOIN pg_catalog.pg_attrdef ad ON a.attrelid = ad.adrelid AND a.attnum = ad.adnum
		JOIN (
			pg_catalog.pg_class c
			JOIN pg_catalog.pg_namespace nc ON c.relnamespace = nc.oid
		) ON a.attrelid = c.oid
		JOIN (
			pg_catalog.pg_type t
			JOIN pg_catalog.pg_namespace nt ON t.typnamespace = nt.oid
		) ON a.atttypid = t.oid
		LEFT JOIN (
			pg_catalog.pg_type bt
			JOIN pg_catalog.pg_namespace nbt ON bt.typnamespace = nbt.oid
		) ON t.typtype = 'd' AND t.typbasetype = bt.oid
		LEFT JOIN pg_catalog.pg_type elem ON t.typelem = elem.oid AND t.typlen = -1
		LEFT JOIN pg_catalog.pg_type elem_bt ON bt.typelem = elem_bt.oid AND bt.typlen = -1
		WHERE a.attnum > 0
		  AND NOT a.attisdropped
		  AND c.relkind IN ('r', 'v', 'm', 'f', 'p')
		  AND nc.nspname IN (%s)
		ORDER BY nc.nspname, c.relname, a.attnum
	`, strings.Join(placeholders, ", "))

	rows, err := c.query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []PostgresColumn
	for rows.Next() {
		var col PostgresColumn
		var enums []string
		if err := rows.Scan(
			&col.TableID, &col.Schema, &col.Table, &col.ID, &col.OrdinalPosition,
			&col.Name, &col.DefaultValue, &col.DataType, &col.Format,
			&col.IsIdentity, &col.IdentityGeneration, &col.IsGenerated,
			&col.IsNullable, &col.IsUpdatable, &col.IsUnique, &enums,
			&col.Check, &col.Comment,
		); err != nil {
			return nil, err
		}
		if enums == nil {
			col.Enums = []string{}
		} else {
			col.Enums = enums
		}
		columns = append(columns, col)
	}

	return columns, nil
}

// GetColumn returns a single column by table and column name
func (c *Client) GetColumn(schema, table, column string) (*PostgresColumn, error) {
	columns, err := c.ListColumns([]string{schema})
	if err != nil {
		return nil, err
	}

	for _, col := range columns {
		if col.Table == table && col.Name == column {
			return &col, nil
		}
	}

	return nil, pgx.ErrNoRows
}
