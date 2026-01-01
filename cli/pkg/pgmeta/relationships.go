package pgmeta

// ListRelationships returns all foreign key relationships in the specified schemas
func (c *Client) ListRelationships(schemas []string) ([]PostgresRelationship, error) {
	if len(schemas) == 0 {
		return []PostgresRelationship{}, nil
	}

	rows, err := c.query(relationshipsSQL, schemas)
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
