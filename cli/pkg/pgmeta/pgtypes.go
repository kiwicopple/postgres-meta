package pgmeta

import (
	"encoding/json"
)

// ListTypes returns all custom types (enums and composites) in the specified schemas
func (c *Client) ListTypes(schemas []string) ([]PostgresType, error) {
	if len(schemas) == 0 {
		return []PostgresType{}, nil
	}

	rows, err := c.query(typesSQL, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []PostgresType
	for rows.Next() {
		var t PostgresType
		var enumsJSON []byte
		var attributesJSON []byte

		if err := rows.Scan(
			&t.ID, &t.Name, &t.Schema, &t.Format,
			&enumsJSON, &attributesJSON, &t.Comment,
		); err != nil {
			return nil, err
		}

		// Parse enums from JSONB
		t.Enums = []string{}
		if enumsJSON != nil {
			json.Unmarshal(enumsJSON, &t.Enums)
		}

		// Parse attributes from JSONB
		t.Attributes = make([]TypeAttribute, 0)
		if attributesJSON != nil {
			var attrs []map[string]interface{}
			if err := json.Unmarshal(attributesJSON, &attrs); err == nil {
				for _, attr := range attrs {
					t.Attributes = append(t.Attributes, TypeAttribute{
						Name:   getString(attr, "name"),
						TypeID: getInt64(attr, "type_id"),
					})
				}
			}
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
