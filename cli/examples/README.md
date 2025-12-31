# Examples

This directory contains an example Pokemon database schema and generated types for each supported language.

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

# Generate types for each language
../pg-meta generate types typescript --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > typescript/types.ts
../pg-meta generate types python --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > python/types.py
../pg-meta generate types go --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > go/types.go
../pg-meta generate types swift --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" > swift/types.swift
```

### 3. Cleanup

```bash
docker compose down -v
```

## Files

- `schema.sql` - Pokemon database schema with tables, enums, views, and functions
- `docker-compose.yaml` - Docker Compose configuration to run PostgreSQL
- `typescript/` - TypeScript type definitions
- `python/` - Python Pydantic models
- `go/` - Go struct definitions
- `swift/` - Swift struct definitions

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
