# Python Types

Generated Python type definitions for the Pokemon database schema using Pydantic.

## Usage

```python
from types import Pokemon, Trainers, PokemonSpecies

# Type hints for your data
def get_pokemon(data: dict) -> Pokemon:
    return Pokemon(**data)

# Use with Supabase Python client
from supabase import create_client
import os

supabase = create_client(
    os.environ["SUPABASE_URL"],
    os.environ["SUPABASE_ANON_KEY"]
)

# Query returns typed data
result = supabase.table("pokemon").select("*").execute()
pokemon_list: list[Pokemon] = [Pokemon(**p) for p in result.data]
```

## Regenerate

```bash
pg-meta generate types python \
  --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" \
  > types.py
```
