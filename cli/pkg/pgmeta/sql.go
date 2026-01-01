package pgmeta

import (
	_ "embed"
)

// Embedded SQL queries from the shared sql/ directory
var (
	//go:embed sql/schemas.sql
	schemasSQL string

	//go:embed sql/tables.sql
	tablesSQL string

	//go:embed sql/primary_keys.sql
	primaryKeysSQL string

	//go:embed sql/columns.sql
	columnsSQL string

	//go:embed sql/relationships.sql
	relationshipsSQL string

	//go:embed sql/types.sql
	typesSQL string

	//go:embed sql/functions.sql
	functionsSQL string

	//go:embed sql/views.sql
	viewsSQL string

	//go:embed sql/materialized_views.sql
	materializedViewsSQL string

	//go:embed sql/foreign_tables.sql
	foreignTablesSQL string
)
