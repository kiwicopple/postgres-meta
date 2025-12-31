# Go Types

Generated Go struct definitions for the Pokemon database schema.

## Usage

```go
package main

import (
    "encoding/json"
    "fmt"
)

// The generated types include Select, Insert, and Update variants
// Use them with your preferred database library

func main() {
    // Example: Unmarshal JSON response from PostgREST
    jsonData := `{"id": 1, "nickname": "Sparky", "level": 25}`

    var pokemon PokemonSelect
    json.Unmarshal([]byte(jsonData), &pokemon)

    fmt.Printf("Pokemon: %s (Level %d)\n", *pokemon.Nickname, *pokemon.Level)
}
```

## Regenerate

```bash
pg-meta generate types go \
  --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" \
  > types.go
```
