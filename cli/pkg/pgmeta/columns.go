package pgmeta

import (
	"github.com/jackc/pgx/v5"
)

// ListColumns returns all columns in the specified schemas
func (c *Client) ListColumns(schemas []string) ([]PostgresColumn, error) {
	if len(schemas) == 0 {
		return []PostgresColumn{}, nil
	}

	rows, err := c.query(columnsSQL, schemas)
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
