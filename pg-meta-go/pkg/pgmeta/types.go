package pgmeta

// PostgresSchema represents a database schema
type PostgresSchema struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

// PostgresTable represents a database table
type PostgresTable struct {
	ID                int64         `json:"id"`
	Schema            string        `json:"schema"`
	Name              string        `json:"name"`
	RLSEnabled        bool          `json:"rls_enabled"`
	RLSForced         bool          `json:"rls_forced"`
	ReplicaIdentity   string        `json:"replica_identity"`
	Bytes             int64         `json:"bytes"`
	Size              string        `json:"size"`
	LiveRowsEstimate  int64         `json:"live_rows_estimate"`
	DeadRowsEstimate  int64         `json:"dead_rows_estimate"`
	Comment           *string       `json:"comment"`
	PrimaryKeys       []PrimaryKey  `json:"primary_keys"`
}

// PrimaryKey represents a primary key column
type PrimaryKey struct {
	TableID int64  `json:"table_id"`
	Name    string `json:"name"`
	Schema  string `json:"schema"`
	Table   string `json:"table_name"`
}

// PostgresColumn represents a database column
type PostgresColumn struct {
	TableID            int64    `json:"table_id"`
	Schema             string   `json:"schema"`
	Table              string   `json:"table"`
	ID                 string   `json:"id"`
	OrdinalPosition    int      `json:"ordinal_position"`
	Name               string   `json:"name"`
	DefaultValue       *string  `json:"default_value"`
	DataType           string   `json:"data_type"`
	Format             string   `json:"format"`
	IsIdentity         bool     `json:"is_identity"`
	IdentityGeneration *string  `json:"identity_generation"`
	IsGenerated        bool     `json:"is_generated"`
	IsNullable         bool     `json:"is_nullable"`
	IsUpdatable        bool     `json:"is_updatable"`
	IsUnique           bool     `json:"is_unique"`
	Enums              []string `json:"enums"`
	Check              *string  `json:"check"`
	Comment            *string  `json:"comment"`
}

// PostgresRelationship represents a foreign key relationship
type PostgresRelationship struct {
	ID                 int64    `json:"id"`
	ConstraintName     string   `json:"constraint_name"`
	Schema             string   `json:"schema"`
	Relation           string   `json:"relation"`
	Columns            []string `json:"columns"`
	ReferencedSchema   string   `json:"referenced_schema"`
	ReferencedRelation string   `json:"referenced_relation"`
	ReferencedColumns  []string `json:"referenced_columns"`
	IsOneToOne         bool     `json:"is_one_to_one"`
}

// PostgresFunction represents a database function
type PostgresFunction struct {
	ID                     int64                  `json:"id"`
	Schema                 string                 `json:"schema"`
	Name                   string                 `json:"name"`
	Language               string                 `json:"language"`
	Definition             string                 `json:"definition"`
	CompleteStatement      string                 `json:"complete_statement"`
	Args                   []FunctionArg          `json:"args"`
	ArgumentTypes          string                 `json:"argument_types"`
	IdentityArgumentTypes  string                 `json:"identity_argument_types"`
	ReturnTypeID           int64                  `json:"return_type_id"`
	ReturnType             string                 `json:"return_type"`
	ReturnTypeRelationID   *int64                 `json:"return_type_relation_id"`
	IsSetReturningFunction bool                   `json:"is_set_returning_function"`
	Behavior               string                 `json:"behavior"`
	SecurityDefiner        bool                   `json:"security_definer"`
	ConfigParams           map[string]string      `json:"config_params"`
	Comment                *string                `json:"comment"`
}

// FunctionArg represents a function argument
type FunctionArg struct {
	Mode       string `json:"mode"`
	Name       string `json:"name"`
	TypeID     int64  `json:"type_id"`
	HasDefault bool   `json:"has_default"`
}

// PostgresType represents a custom type (enum or composite)
type PostgresType struct {
	ID             int64           `json:"id"`
	Name           string          `json:"name"`
	Schema         string          `json:"schema"`
	Format         string          `json:"format"`
	Enums          []string        `json:"enums"`
	Attributes     []TypeAttribute `json:"attributes"`
	Comment        *string         `json:"comment"`
	TypeRelationID *int64          `json:"type_relation_id"`
}

// TypeAttribute represents an attribute of a composite type
type TypeAttribute struct {
	Name   string `json:"name"`
	TypeID int64  `json:"type_id"`
}

// PostgresView represents a database view
type PostgresView struct {
	ID          int64   `json:"id"`
	Schema      string  `json:"schema"`
	Name        string  `json:"name"`
	IsUpdatable bool    `json:"is_updatable"`
	Comment     *string `json:"comment"`
}

// PostgresMaterializedView represents a materialized view
type PostgresMaterializedView struct {
	ID          int64   `json:"id"`
	Schema      string  `json:"schema"`
	Name        string  `json:"name"`
	IsPopulated bool    `json:"is_populated"`
	Comment     *string `json:"comment"`
}

// PostgresForeignTable represents a foreign table
type PostgresForeignTable struct {
	ID      int64   `json:"id"`
	Schema  string  `json:"schema"`
	Name    string  `json:"name"`
	Comment *string `json:"comment"`
}

// GeneratorMetadata contains all metadata needed for code generation
type GeneratorMetadata struct {
	Schemas           []PostgresSchema           `json:"schemas"`
	Tables            []PostgresTable            `json:"tables"`
	ForeignTables     []PostgresForeignTable     `json:"foreign_tables"`
	Views             []PostgresView             `json:"views"`
	MaterializedViews []PostgresMaterializedView `json:"materialized_views"`
	Columns           []PostgresColumn           `json:"columns"`
	Relationships     []PostgresRelationship     `json:"relationships"`
	Functions         []PostgresFunction         `json:"functions"`
	Types             []PostgresType             `json:"types"`
}

// GeneratorOptions contains options for code generation
type GeneratorOptions struct {
	IncludedSchemas              []string
	ExcludedSchemas              []string
	DetectOneToOneRelationships  bool
	PostgrestVersion             string
	SwiftAccessControl           string
}
