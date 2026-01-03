# pg-meta CLI

A Go library and CLI for PostgreSQL database introspection and type generation. This is part of the [postgres-meta](https://github.com/supabase/postgres-meta) project.

## Features

- **Database Introspection**: Extract metadata from PostgreSQL databases including tables, views, columns, relationships, functions, and types
- **Type Generation**: Generate type definitions for multiple languages and frameworks:
  - **TypeScript**
    - Supabase client types
    - Drizzle ORM schemas
    - Kysely type definitions
  - **Python**
    - Pydantic models
    - Django models
  - **Go**: Struct definitions with JSON tags
  - **Swift**: Struct definitions with Codable conformance
  - **JSON Schema**: Validation schemas (draft 2020-12)

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
# TypeScript - Supabase client types
pg-meta generate types typescript --db-url "postgresql://user:pass@localhost:5432/mydb"

# TypeScript - Drizzle ORM schema
pg-meta generate types drizzle --db-url "$DATABASE_URL"

# TypeScript - Kysely types
pg-meta generate types kysely --db-url "$DATABASE_URL"

# Python - Pydantic models
pg-meta generate types python --db mydb --user postgres --password secret

# Python - Django models
pg-meta generate types django --db-url "$DATABASE_URL"

# Go structs
pg-meta generate types go --db-url "$DATABASE_URL"

# Swift structs
pg-meta generate types swift --db-url "$DATABASE_URL" --access-control public

# JSON Schema
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

#### Supabase (`typescript`)
Generates Supabase-compatible type definitions for use with `@supabase/supabase-js`. Includes `Database` type with `Tables`, `Views`, `Functions`, and `Enums`.

#### Drizzle ORM (`drizzle`)
Generates Drizzle ORM schema definitions with `pgTable`, `pgEnum`, and `relations`. Ready to use with Drizzle Kit for migrations.

#### Kysely (`kysely`)
Generates Kysely type definitions with:
- Table interfaces using `ColumnType` for insert/update behavior
- `Generated<T>` for auto-increment columns
- `Selectable`, `Insertable`, `Updateable` type aliases
- `Database` interface for type-safe queries

### Python

#### Pydantic (`python`)
Generates Pydantic `BaseModel` classes for row types and `TypedDict` for insert/update operations. Includes proper type hints for all PostgreSQL types.

#### Django (`django`)
Generates Django model classes with:
- `models.Model` base classes
- `TextChoices` for PostgreSQL enums
- `ForeignKey` relationships with `on_delete`
- Proper field types (CharField, IntegerField, JSONField, etc.)
- `Meta` class with `db_table` configuration
- Unmanaged models for views (`managed = False`)

### Go (`go`)
Generates Go struct definitions with JSON tags. Includes separate structs for Select, Insert, and Update operations.

### Swift (`swift`)
Generates Swift struct definitions with `Codable` conformance. Supports access control modifiers and includes enums as `String` enums.

### JSON Schema (`jsonschema`)
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
  - **TypeScript**
    - `typescript_types.ts` - Supabase client types
    - `typescript_drizzleorm.ts` - Drizzle ORM schema
    - `typescript_kysely.ts` - Kysely types
  - **Python**
    - `python_types.py` - Pydantic models
    - `python_django.py` - Django models
  - **Go**
    - `go_types.go` - Go structs
  - **Swift**
    - `swift_types.swift` - Swift structs
  - **JSON Schema**
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

    // Available generators:
    // typescript, drizzle, kysely, python, django, go, swift, jsonschema
    gen := generators.GetGenerator("kysely")
    output, err := gen.Generate(meta, generators.Options{})
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
