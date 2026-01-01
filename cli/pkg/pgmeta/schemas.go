package pgmeta

// ListSchemas returns all schemas in the database
func (c *Client) ListSchemas(opts GeneratorOptions) ([]PostgresSchema, error) {
	// Ensure we pass an empty slice, not nil (nil becomes NULL in PostgreSQL,
	// but empty slice becomes an empty array which works with cardinality())
	includedSchemas := opts.IncludedSchemas
	if includedSchemas == nil {
		includedSchemas = []string{}
	}
	rows, err := c.query(schemasSQL, includedSchemas)
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
