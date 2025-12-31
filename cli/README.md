# pg-meta CLI

A Go library and CLI for PostgreSQL database introspection and type generation. This is part of the [postgres-meta](https://github.com/supabase/postgres-meta) project.

## Features

- **Database Introspection**: Extract metadata from PostgreSQL databases including tables, views, columns, relationships, functions, and types
- **Type Generation**: Generate type definitions for multiple languages:
  - **TypeScript**: Supabase-compatible types for PostgREST clients
  - **Python**: Pydantic BaseModel classes and TypedDict types
  - **Go**: Struct definitions with JSON tags
  - **Swift**: Struct definitions with Codable conformance

## Installation

```bash
go install github.com/supabase/postgres-meta/cli/cmd/pg-meta@latest
```

Or build from source:

```bash
cd cli
go build -v -o pg-meta ./cmd/pg-meta
```

## CLI Usage

### Generate Types

```bash
# Generate TypeScript types
pg-meta generate types typescript --db-url "postgresql://user:pass@localhost:5432/mydb"

# Generate Python types
pg-meta generate types python --db mydb --user postgres --password secret

# Generate Go types
pg-meta generate types go --db-url "$DATABASE_URL"

# Generate Swift types
pg-meta generate types swift --db-url "$DATABASE_URL" --access-control public
```

### Options

**Database Connection:**
- `--db-url`: Full connection URL (overrides other connection flags)
- `--host`: Database host (default: localhost)
- `--port`: Database port (default: 5432)
- `--db`: Database name
- `--user`: Database user
- `--password`: Database password
- `--ssl-mode`: SSL mode (disable, prefer, require)

**Generator Options:**
- `--included-schemas`: Comma-separated list of schemas to include
- `--excluded-schemas`: Comma-separated list of schemas to exclude

**TypeScript-specific:**
- `--detect-one-to-one`: Detect one-to-one relationships
- `--postgrest-version`: PostgREST version for type generation

**Swift-specific:**
- `--access-control`: Access control modifier (internal, public, private, package)

## Examples

See the [`examples/`](examples/) directory for a complete example including:

- [`schema.sql`](examples/schema.sql) - An advanced Pokemon database schema demonstrating:
  - Multiple enum types (pokemon_type, status_condition, move_category, etc.)
  - Composite types (pokemon_stats, geo_location)
  - Tables with foreign key relationships
  - Views with computed fields
  - PostgreSQL functions

- [`docker-compose.yaml`](examples/docker-compose.yaml) - Docker Compose configuration to run the example database

- Generated type files by language:
  - [`typescript/types.ts`](examples/typescript/types.ts) - TypeScript types
  - [`python/types.py`](examples/python/types.py) - Python Pydantic models
  - [`go/types.go`](examples/go/types.go) - Go structs
  - [`swift/types.swift`](examples/swift/types.swift) - Swift structs

## Library Usage

The library can be used directly in Go applications:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/supabase/postgres-meta/cli/pkg/pgmeta"
    "github.com/supabase/postgres-meta/cli/pkg/generators"
)

func main() {
    ctx := context.Background()

    // Create a client
    client, err := pgmeta.NewClientFromConnectionString(ctx, "postgresql://user:pass@localhost:5432/mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Get metadata
    opts := pgmeta.GeneratorOptions{
        IncludedSchemas: []string{"public"},
    }
    meta, err := client.GetGeneratorMetadata(opts)
    if err != nil {
        log.Fatal(err)
    }

    // Generate TypeScript types
    gen := generators.GetGenerator("typescript")
    genOpts := generators.Options{
        DetectOneToOneRelationships: true,
    }
    output, err := gen.Generate(meta, genOpts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Print(output)
}
```

## Supported PostgreSQL Features

- Tables, Views, Materialized Views, Foreign Tables
- Columns with all metadata (nullable, default, identity, generated)
- Custom types (enums and composite types)
- Foreign key relationships
- Functions (excluding triggers)
- Primary keys and unique constraints

## Future Work

- `pg-meta scaffold` command for generating framework-specific code:
  - `typescript --drizzle-orm`
  - `python --django`
  - `elixir --phoenix`

## License

MIT
