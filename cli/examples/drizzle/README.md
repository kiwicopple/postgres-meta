# Drizzle ORM Schema

Generated Drizzle ORM schema definitions for the Pokemon database.

## Usage

```typescript
import { drizzle } from 'drizzle-orm/node-postgres';
import { Pool } from 'pg';
import * as schema from './schema';

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
});

const db = drizzle(pool, { schema });

// Type-safe queries
const allPokemon = await db.query.pokemon.findMany({
  with: {
    pokemonSpecies: true,
    trainer: true,
  },
});

// Insert with full type safety
await db.insert(schema.trainers).values({
  id: 'trainer-1',
  name: 'Ash Ketchum',
  email: 'ash@pokemon.com',
});
```

## Features

- Full type inference for all tables and columns
- Relation definitions for foreign keys
- Enum types as TypeScript unions
- Support for arrays, JSON, and all PostgreSQL types

## Regenerate

```bash
pg-meta generate types drizzle \
  --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" \
  > schema.ts
```

## Learn More

- [Drizzle ORM Documentation](https://orm.drizzle.team/)
- [Drizzle with PostgreSQL](https://orm.drizzle.team/docs/get-started-postgresql)
