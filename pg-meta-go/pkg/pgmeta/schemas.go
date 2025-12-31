package pgmeta

import (
	"fmt"
	"strings"
)

// ListSchemas returns all schemas in the database
func (c *Client) ListSchemas(opts GeneratorOptions) ([]PostgresSchema, error) {
	sql := `
		SELECT
			n.oid::int8 AS id,
			n.nspname AS name,
			pg_catalog.pg_get_userbyid(n.nspowner) AS owner
		FROM pg_catalog.pg_namespace n
		WHERE n.nspname !~ '^pg_'
		  AND n.nspname <> 'information_schema'
	`

	// Apply included/excluded schemas filter
	if len(opts.IncludedSchemas) > 0 {
		placeholders := make([]string, len(opts.IncludedSchemas))
		for i := range opts.IncludedSchemas {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		sql += fmt.Sprintf(" AND n.nspname IN (%s)", strings.Join(placeholders, ", "))
	}

	sql += " ORDER BY n.nspname"

	var args []interface{}
	for _, s := range opts.IncludedSchemas {
		args = append(args, s)
	}

	rows, err := c.query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []PostgresSchema
	for rows.Next() {
		var s PostgresSchema
		if err := rows.Scan(&s.ID, &s.Name, &s.Owner); err != nil {
			return nil, err
		}
		schemas = append(schemas, s)
	}

	// Filter out excluded schemas
	if len(opts.ExcludedSchemas) > 0 {
		excluded := make(map[string]bool)
		for _, s := range opts.ExcludedSchemas {
			excluded[s] = true
		}
		filtered := make([]PostgresSchema, 0)
		for _, s := range schemas {
			if !excluded[s.Name] {
				filtered = append(filtered, s)
			}
		}
		schemas = filtered
	}

	return schemas, nil
}
