# Swift Types

Generated Swift struct definitions for the Pokemon database schema with Codable conformance.

## Usage

```swift
import Foundation

// The generated types conform to Codable, Hashable, and Sendable
// Use them with Supabase Swift client or URLSession

let decoder = JSONDecoder()
decoder.keyDecodingStrategy = .convertFromSnakeCase

// Decode Pokemon from JSON
let jsonData = """
{"id": 1, "nickname": "Sparky", "level": 25}
""".data(using: .utf8)!

let pokemon = try decoder.decode(Pokemon.self, from: jsonData)
print("Pokemon: \(pokemon.nickname ?? "Unknown") (Level \(pokemon.level ?? 0))")
```

## With Supabase Swift

```swift
import Supabase

let client = SupabaseClient(
    supabaseURL: URL(string: "YOUR_SUPABASE_URL")!,
    supabaseKey: "YOUR_SUPABASE_KEY"
)

// Fully typed queries
let pokemon: [Pokemon] = try await client
    .from("pokemon")
    .select()
    .execute()
    .value
```

## Regenerate

```bash
pg-meta generate types swift \
  --db-url "postgresql://postgres:postgres@localhost:5432/pokemon" \
  > types.swift
```
