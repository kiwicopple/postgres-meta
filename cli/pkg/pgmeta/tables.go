package pgmeta

import (
	"fmt"
	"strings"
)

// ListTables returns all tables in the specified schemas
func (c *Client) ListTables(schemas []string) ([]PostgresTable, error) {
	if len(schemas) == 0 {
		return []PostgresTable{}, nil
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
			c.relrowsecurity AS rls_enabled,
			c.relforcerowsecurity AS rls_forced,
			CASE c.relreplident
				WHEN 'd' THEN 'DEFAULT'
				WHEN 'n' THEN 'NOTHING'
				WHEN 'f' THEN 'FULL'
				WHEN 'i' THEN 'INDEX'
			END AS replica_identity,
			COALESCE(pg_total_relation_size(c.oid), 0)::int8 AS bytes,
			pg_size_pretty(pg_total_relation_size(c.oid)) AS size,
			COALESCE(pg_stat_get_live_tuples(c.oid), 0)::int8 AS live_rows_estimate,
			COALESCE(pg_stat_get_dead_tuples(c.oid), 0)::int8 AS dead_rows_estimate,
			pg_catalog.obj_description(c.oid, 'pg_class') AS comment
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace nc ON nc.oid = c.relnamespace
		WHERE c.relkind = 'r'
		  AND nc.nspname IN (%s)
		ORDER BY nc.nspname, c.relname
	`, strings.Join(placeholders, ", "))

	rows, err := c.query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []PostgresTable
	for rows.Next() {
		var t PostgresTable
		if err := rows.Scan(
			&t.ID, &t.Schema, &t.Name, &t.RLSEnabled, &t.RLSForced,
			&t.ReplicaIdentity, &t.Bytes, &t.Size, &t.LiveRowsEstimate,
			&t.DeadRowsEstimate, &t.Comment,
		); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}

	// Fetch primary keys for all tables
	if len(tables) > 0 {
		pks, err := c.listPrimaryKeys(schemas)
		if err != nil {
			return nil, err
		}

		pkMap := make(map[int64][]PrimaryKey)
		for _, pk := range pks {
			pkMap[pk.TableID] = append(pkMap[pk.TableID], pk)
		}

		for i := range tables {
			if pks, ok := pkMap[tables[i].ID]; ok {
				tables[i].PrimaryKeys = pks
			} else {
				tables[i].PrimaryKeys = []PrimaryKey{}
			}
		}
	}

	return tables, nil
}

// listPrimaryKeys returns all primary keys in the specified schemas
func (c *Client) listPrimaryKeys(schemas []string) ([]PrimaryKey, error) {
	if len(schemas) == 0 {
		return []PrimaryKey{}, nil
	}

	placeholders := make([]string, len(schemas))
	args := make([]interface{}, len(schemas))
	for i, s := range schemas {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}

	sql := fmt.Sprintf(`
		SELECT
			c.conrelid::int8 AS table_id,
			a.attname AS name,
			nc.nspname AS schema,
			rel.relname AS table_name
		FROM pg_catalog.pg_constraint c
		JOIN pg_catalog.pg_class rel ON rel.oid = c.conrelid
		JOIN pg_catalog.pg_namespace nc ON nc.oid = rel.relnamespace
		JOIN pg_catalog.pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
		WHERE c.contype = 'p'
		  AND nc.nspname IN (%s)
		ORDER BY nc.nspname, rel.relname, a.attnum
	`, strings.Join(placeholders, ", "))

	rows, err := c.query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pks []PrimaryKey
	for rows.Next() {
		var pk PrimaryKey
		if err := rows.Scan(&pk.TableID, &pk.Name, &pk.Schema, &pk.Table); err != nil {
			return nil, err
		}
		pks = append(pks, pk)
	}

	return pks, nil
}
