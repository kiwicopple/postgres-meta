# TypeScript Types

Generated TypeScript type definitions for the Pokemon database schema.

## Usage with Supabase Client

```typescript
import { createClient } from '@supabase/supabase-js'
import { Database } from './types'

const supabase = createClient<Database>(
  process.env.SUPABASE_URL!,
  process.env.SUPABASE_ANON_KEY!
)

// Fully typed queries
const { data: pokemon } = await supabase
  .from('pokemon')
  .select('*, pokemon_species(*)')
  .eq('is_shiny', true)

// Type-safe inserts
const { data: newTrainer } = await supabase
  .from('trainers')
  .insert({
    username: 'ash_ketchum',
    email: 'ash@pokemon.com',
  })
  .select()
  .single()
```

## Regenerate

```bash
pg-meta generate types typescript \
  --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" \
  > types.ts
```

## Future: Drizzle ORM

Coming soon - Drizzle schema generation:

```bash
pg-meta scaffold typescript --drizzle-orm
```
