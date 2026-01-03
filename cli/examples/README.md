# Examples

This directory contains an example Pokemon database schema and generated types for each supported language/framework.

## Quick Start

### 1. Start the database

```bash
docker compose up -d
```

This starts PostgreSQL and automatically loads `schema.sql`.

### 2. Generate types

```bash
# Build the CLI (from the cli/ directory)
cd .. && go build -o pg-meta ./cmd/pg-meta && cd examples

# Generate all output types
# TypeScript
../pg-meta generate types typescript --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > output/typescript_types.ts
../pg-meta generate types drizzle --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > output/typescript_drizzleorm.ts
../pg-meta generate types kysely --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > output/typescript_kysely.ts

# Python
../pg-meta generate types python --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > output/python_types.py
../pg-meta generate types django --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > output/python_django.py

# Go
../pg-meta generate types go --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > output/go_types.go

# Swift
../pg-meta generate types swift --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > output/swift_types.swift

# JSON Schema
../pg-meta generate types jsonschema --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > output/jsonschema.json
```

### 3. Cleanup

```bash
docker compose down -v
```

## Files

- `schema.sql` - Pokemon database schema with tables, enums, views, and functions
- `docker-compose.yaml` - Docker Compose configuration to run PostgreSQL
- `output/` - Generated type definitions:
  - **TypeScript**
    - `typescript_types.ts` - Supabase client types
    - `typescript_drizzleorm.ts` - Drizzle ORM schema definitions
    - `typescript_kysely.ts` - Kysely type definitions
  - **Python**
    - `python_types.py` - Pydantic models
    - `python_django.py` - Django model definitions
  - **Go**
    - `go_types.go` - Go struct definitions
  - **Swift**
    - `swift_types.swift` - Swift struct definitions
  - **JSON Schema**
    - `jsonschema.json` - JSON Schema definitions

## Schema Overview

The Pokemon schema includes:

### Enums
- `pokemon_type` - The 18 Pokemon elemental types
- `status_condition` - Status effects (burned, frozen, etc.)
- `move_category` - Physical, special, or status moves
- `evolution_method` - How Pokemon evolve
- `pokemon_nature` - The 25 Pokemon natures
- `battle_terrain` - Battle field conditions
- `weather_condition` - Weather effects

### Composite Types
- `pokemon_stats` - HP, Attack, Defense, Sp. Atk, Sp. Def, Speed
- `geo_location` - Latitude, longitude, altitude

### Tables
- `regions` - Kanto, Johto, etc.
- `locations` - Cities, routes, caves
- `abilities` - Pokemon abilities
- `pokemon_species` - Pokedex entries
- `pokemon` - Individual Pokemon instances
- `trainers` - Pokemon trainers
- `moves` - Attack moves
- `teams` - Trainer teams
- `battles` - Battle records
- `items` - Held items and consumables
- And more...

### Views
- `trainer_stats` - Trainer statistics with win rate
- `pokemon_full` - Pokemon with species and trainer info
- `type_matchups` - Type effectiveness analysis

### Functions
- `calculate_stat()` - Calculate effective stats
- `get_pokemon_calculated_stats()` - Get full stats for a Pokemon
- `search_pokemon_by_type()` - Find Pokemon by type
- `get_evolution_chain()` - Get evolution chain for a species

## Generator Output Examples

### TypeScript

#### Supabase (`typescript`)
```typescript
export type Database = {
  public: {
    Tables: {
      pokemon: {
        Row: { id: number; name: string; type1: Database["public"]["Enums"]["pokemon_type"]; ... }
        Insert: { id?: number; name: string; ... }
        Update: { id?: number; name?: string; ... }
      }
    }
  }
}
```

#### Drizzle ORM (`drizzle`)
```typescript
export const pokemonTypeEnum = pgEnum('pokemon_type', ['normal', 'fire', 'water', ...]);

export const pokemon = pgTable('pokemon', {
  id: serial('id').primaryKey(),
  name: varchar('name', { length: 255 }).notNull(),
  type1: pokemonTypeEnum('type1').notNull(),
});
```

#### Kysely (`kysely`)
```typescript
export interface PokemonTable {
  id: Generated<number>;
  name: string;
  type1: PokemonType;
}

export type Pokemon = Selectable<PokemonTable>;
export type NewPokemon = Insertable<PokemonTable>;
export type PokemonUpdate = Updateable<PokemonTable>;

export interface Database {
  pokemon: PokemonTable;
}
```

### Python

#### Pydantic (`python`)
```python
class Pokemon(BaseModel):
    id: int = Field(alias="id")
    name: str = Field(alias="name")
    type1: PokemonType = Field(alias="type1")
```

#### Django (`django`)
```python
class Pokemon(models.Model):
    id = models.AutoField(primary_key=True)
    name = models.CharField(max_length=255)
    type1 = models.CharField(max_length=255, choices=PokemonType.choices)

    class Meta:
        db_table = 'pokemon'
```

### Go (`go`)
```go
type PokemonSelect struct {
    Id    int32  `json:"id"`
    Name  string `json:"name"`
    Type1 string `json:"type1"`
}
```

### Swift (`swift`)
```swift
struct Pokemon: Codable, Hashable, Sendable {
    let id: Int32
    let name: String
    let type1: PokemonType
}
```

### JSON Schema (`jsonschema`)
```json
{
  "$defs": {
    "PokemonRow": {
      "type": "object",
      "properties": {
        "id": { "type": "integer" },
        "name": { "type": "string" },
        "type1": { "type": "string", "enum": ["normal", "fire", "water", ...] }
      },
      "required": ["id", "name", "type1"]
    }
  }
}
```
