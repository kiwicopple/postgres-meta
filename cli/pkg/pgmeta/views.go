package pgmeta

// ListViews returns all views in the specified schemas
func (c *Client) ListViews(schemas []string) ([]PostgresView, error) {
	if len(schemas) == 0 {
		return []PostgresView{}, nil
	}

	rows, err := c.query(viewsSQL, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []PostgresView
	for rows.Next() {
		var v PostgresView
		if err := rows.Scan(&v.ID, &v.Schema, &v.Name, &v.Definition, &v.Comment, &v.IsUpdatable); err != nil {
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

	rows, err := c.query(materializedViewsSQL, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []PostgresMaterializedView
	for rows.Next() {
		var v PostgresMaterializedView
		if err := rows.Scan(&v.ID, &v.Schema, &v.Name, &v.Definition, &v.Comment, &v.IsPopulated); err != nil {
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

	rows, err := c.query(foreignTablesSQL, schemas)
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
