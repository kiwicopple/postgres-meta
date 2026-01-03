# pg-meta CLI

A Go library and CLI for PostgreSQL database introspection and type generation. This is part of the [postgres-meta](https://github.com/supabase/postgres-meta) project.

## Features

- **Database Introspection**: Extract metadata from PostgreSQL databases including tables, views, columns, relationships, functions, and types
- **Type Generation**: Generate type definitions for multiple languages and frameworks:
  - **TypeScript**: Supabase-compatible types for PostgREST clients
  - **Drizzle ORM**: TypeScript schema definitions for Drizzle ORM
  - **Python**: Pydantic BaseModel classes and TypedDict types
  - **Django**: Django model definitions with ForeignKey relationships
  - **Go**: Struct definitions with JSON tags
  - **Swift**: Struct definitions with Codable conformance
  - **JSON Schema**: JSON Schema (draft 2020-12) definitions for validation

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
# Generate TypeScript types (Supabase client compatible)
pg-meta generate types typescript --db-url "postgresql://user:pass@localhost:5432/mydb"

# Generate Drizzle ORM schema
pg-meta generate types drizzle --db-url "$DATABASE_URL"

# Generate Python Pydantic models
pg-meta generate types python --db mydb --user postgres --password secret

# Generate Django models
pg-meta generate types django --db-url "$DATABASE_URL"

# Generate Go structs
pg-meta generate types go --db-url "$DATABASE_URL"

# Generate Swift structs
pg-meta generate types swift --db-url "$DATABASE_URL" --access-control public

# Generate JSON Schema
pg-meta generate types jsonschema --db-url "$DATABASE_URL"
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

## Generators

### TypeScript
Generates Supabase-compatible type definitions for use with `@supabase/supabase-js`. Includes `Database` type with `Tables`, `Views`, `Functions`, and `Enums`.

### Drizzle ORM
Generates Drizzle ORM schema definitions with `pgTable`, `pgEnum`, and `relations`. Ready to use with Drizzle Kit for migrations.

### Python
Generates Pydantic `BaseModel` classes for row types and `TypedDict` for insert/update operations. Includes proper type hints for all PostgreSQL types.

### Django
Generates Django model classes with:
- `models.Model` base classes
- `TextChoices` for PostgreSQL enums
- `ForeignKey` relationships with `on_delete`
- Proper field types (CharField, IntegerField, JSONField, etc.)
- `Meta` class with `db_table` configuration
- Unmanaged models for views (`managed = False`)

### Go
Generates Go struct definitions with JSON tags. Includes separate structs for Select, Insert, and Update operations.

### Swift
Generates Swift struct definitions with `Codable` conformance. Supports access control modifiers and includes enums as `String` enums.

### JSON Schema
Generates JSON Schema (draft 2020-12) definitions with:
- Separate Row/Insert/Update schemas per table
- Enum definitions in `$defs`
- Nullable types using `anyOf`
- Format annotations (`date-time`, `uuid`, etc.)
- Proper `required` fields based on operation type

## Examples

See the [`examples/`](examples/) directory for a complete example including:

- [`schema.sql`](examples/schema.sql) - An advanced Pokemon database schema demonstrating:
  - Multiple enum types (pokemon_type, status_condition, move_category, etc.)
  - Composite types (pokemon_stats, geo_location)
  - Tables with foreign key relationships
  - Views with computed fields
  - PostgreSQL functions

- [`docker-compose.yaml`](examples/docker-compose.yaml) - Docker Compose configuration to run the example database

- Generated output files in [`output/`](examples/output/):
  - `typescript_types.ts` - TypeScript types
  - `typescript_drizzleorm.ts` - Drizzle ORM schema
  - `python_types.py` - Python Pydantic models
  - `python_django.py` - Django models
  - `go_types.go` - Go structs
  - `swift_types.swift` - Swift structs
  - `jsonschema.json` - JSON Schema definitions

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

    // Generate types (typescript, python, go, swift, drizzle, django, jsonschema)
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

## License

MIT
