package pgmeta

// ListTables returns all tables in the specified schemas
func (c *Client) ListTables(schemas []string) ([]PostgresTable, error) {
	if len(schemas) == 0 {
		return []PostgresTable{}, nil
	}

	rows, err := c.query(tablesSQL, schemas)
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

	rows, err := c.query(primaryKeysSQL, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pks []PrimaryKey
	for rows.Next() {
		var pk PrimaryKey
		if err := rows.Scan(&pk.TableID, &pk.Schema, &pk.Table, &pk.Name); err != nil {
			return nil, err
		}
		pks = append(pks, pk)
	}

	return pks, nil
}
