package pgmeta

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Client represents a connection to a PostgreSQL database for metadata extraction
type Client struct {
	pool *pgxpool.Pool
	ctx  context.Context
}

// Config holds connection configuration
type Config struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

// NewClient creates a new PostgresMeta client
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "prefer"
	}

	connString := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password, sslMode,
	)

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Client{
		pool: pool,
		ctx:  ctx,
	}, nil
}

// NewClientFromConnectionString creates a new client from a connection string
func NewClientFromConnectionString(ctx context.Context, connString string) (*Client, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Client{
		pool: pool,
		ctx:  ctx,
	}, nil
}

// Close closes the database connection
func (c *Client) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}

// query executes a query and returns rows
func (c *Client) query(sql string, args ...interface{}) (pgx.Rows, error) {
	return c.pool.Query(c.ctx, sql, args...)
}

// queryRow executes a query and returns a single row
func (c *Client) queryRow(sql string, args ...interface{}) pgx.Row {
	return c.pool.QueryRow(c.ctx, sql, args...)
}

// GetGeneratorMetadata fetches all metadata needed for code generation
func (c *Client) GetGeneratorMetadata(opts GeneratorOptions) (*GeneratorMetadata, error) {
	schemas, err := c.ListSchemas(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}

	schemaNames := make([]string, len(schemas))
	for i, s := range schemas {
		schemaNames[i] = s.Name
	}

	tables, err := c.ListTables(schemaNames)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	foreignTables, err := c.ListForeignTables(schemaNames)
	if err != nil {
		return nil, fmt.Errorf("failed to list foreign tables: %w", err)
	}

	views, err := c.ListViews(schemaNames)
	if err != nil {
		return nil, fmt.Errorf("failed to list views: %w", err)
	}

	materializedViews, err := c.ListMaterializedViews(schemaNames)
	if err != nil {
		return nil, fmt.Errorf("failed to list materialized views: %w", err)
	}

	columns, err := c.ListColumns(schemaNames)
	if err != nil {
		return nil, fmt.Errorf("failed to list columns: %w", err)
	}

	relationships, err := c.ListRelationships(schemaNames)
	if err != nil {
		return nil, fmt.Errorf("failed to list relationships: %w", err)
	}

	functions, err := c.ListFunctions(schemaNames)
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}

	types, err := c.ListTypes(schemaNames)
	if err != nil {
		return nil, fmt.Errorf("failed to list types: %w", err)
	}

	return &GeneratorMetadata{
		Schemas:           schemas,
		Tables:            tables,
		ForeignTables:     foreignTables,
		Views:             views,
		MaterializedViews: materializedViews,
		Columns:           columns,
		Relationships:     relationships,
		Functions:         functions,
		Types:             types,
	}, nil
}
