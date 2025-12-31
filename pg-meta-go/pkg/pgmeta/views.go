package pgmeta

import (
	"fmt"
	"strings"
)

// ListViews returns all views in the specified schemas
func (c *Client) ListViews(schemas []string) ([]PostgresView, error) {
	if len(schemas) == 0 {
		return []PostgresView{}, nil
	}

	placeholders := make([]string, len(schemas))
	args := make([]interface{}, len(schemas))
	for i, s := range schemas {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}

	sql := fmt.Sprintf(`
		SELECT
			c.oid::int8 AS id,
			nc.nspname AS schema,
			c.relname AS name,
			pg_catalog.pg_relation_is_updatable(c.oid, true) > 0 AS is_updatable,
			pg_catalog.obj_description(c.oid, 'pg_class') AS comment
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace nc ON nc.oid = c.relnamespace
		WHERE c.relkind = 'v'
		  AND nc.nspname IN (%s)
		ORDER BY nc.nspname, c.relname
	`, strings.Join(placeholders, ", "))

	rows, err := c.query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []PostgresView
	for rows.Next() {
		var v PostgresView
		if err := rows.Scan(&v.ID, &v.Schema, &v.Name, &v.IsUpdatable, &v.Comment); err != nil {
			return nil, err
		}
		views = append(views, v)
	}

	return views, nil
}

// ListMaterializedViews returns all materialized views in the specified schemas
func (c *Client) ListMaterializedViews(schemas []string) ([]PostgresMaterializedView, error) {
	if len(schemas) == 0 {
		return []PostgresMaterializedView{}, nil
	}

	placeholders := make([]string, len(schemas))
	args := make([]interface{}, len(schemas))
	for i, s := range schemas {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}

	sql := fmt.Sprintf(`
		SELECT
			c.oid::int8 AS id,
			nc.nspname AS schema,
			c.relname AS name,
			c.relispopulated AS is_populated,
			pg_catalog.obj_description(c.oid, 'pg_class') AS comment
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace nc ON nc.oid = c.relnamespace
		WHERE c.relkind = 'm'
		  AND nc.nspname IN (%s)
		ORDER BY nc.nspname, c.relname
	`, strings.Join(placeholders, ", "))

	rows, err := c.query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []PostgresMaterializedView
	for rows.Next() {
		var v PostgresMaterializedView
		if err := rows.Scan(&v.ID, &v.Schema, &v.Name, &v.IsPopulated, &v.Comment); err != nil {
			return nil, err
		}
		views = append(views, v)
	}

	return views, nil
}

// ListForeignTables returns all foreign tables in the specified schemas
func (c *Client) ListForeignTables(schemas []string) ([]PostgresForeignTable, error) {
	if len(schemas) == 0 {
		return []PostgresForeignTable{}, nil
	}

	placeholders := make([]string, len(schemas))
	args := make([]interface{}, len(schemas))
	for i, s := range schemas {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}

	sql := fmt.Sprintf(`
		SELECT
			c.oid::int8 AS id,
			nc.nspname AS schema,
			c.relname AS name,
			pg_catalog.obj_description(c.oid, 'pg_class') AS comment
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace nc ON nc.oid = c.relnamespace
		WHERE c.relkind = 'f'
		  AND nc.nspname IN (%s)
		ORDER BY nc.nspname, c.relname
	`, strings.Join(placeholders, ", "))

	rows, err := c.query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []PostgresForeignTable
	for rows.Next() {
		var t PostgresForeignTable
		if err := rows.Scan(&t.ID, &t.Schema, &t.Name, &t.Comment); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}

	return tables, nil
}
