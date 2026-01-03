package pgmeta

import (
	"encoding/json"
	"strings"
)

// ListFunctions returns all functions in the specified schemas (excluding triggers)
func (c *Client) ListFunctions(schemas []string) ([]PostgresFunction, error) {
	if len(schemas) == 0 {
		return []PostgresFunction{}, nil
	}

	rows, err := c.query(functionsSQL, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []PostgresFunction
	for rows.Next() {
		var f PostgresFunction
		var argsJSON []byte
		var comment *string
		var configParams []string

		if err := rows.Scan(
			&f.ID, &f.Schema, &f.Name, &f.Language, &f.Definition,
			&f.ArgumentTypes, &f.IdentityArgumentTypes, &f.ReturnTypeID, &f.ReturnType,
			&f.ReturnTypeRelationID, &f.IsSetReturningFunction, &f.Behavior,
			&f.SecurityDefiner, &argsJSON, &comment, &configParams,
		); err != nil {
			return nil, err
		}

		f.Comment = comment
		f.Args = make([]FunctionArg, 0)

		// Parse args from JSONB
		if argsJSON != nil {
			var args []map[string]interface{}
			if err := json.Unmarshal(argsJSON, &args); err == nil {
				for _, arg := range args {
					fa := FunctionArg{
						Mode:       getString(arg, "mode"),
						Name:       getString(arg, "name"),
						TypeID:     getInt64(arg, "type_id"),
						HasDefault: getBool(arg, "has_default"),
					}
					f.Args = append(f.Args, fa)
				}
			}
		}

		f.ConfigParams = make(map[string]string)
		if configParams != nil {
			for _, param := range configParams {
				parts := strings.SplitN(param, "=", 2)
				if len(parts) == 2 {
					f.ConfigParams[parts[0]] = parts[1]
				}
			}
		}

		functions = append(functions, f)
	}

	return functions, nil
}

// Helper functions for JSON parsing
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return int64(f)
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
