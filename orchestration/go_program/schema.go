package go_program

import (
	"encoding/json"
	"fmt"
)

func ValidateJSONInput(schema ObjectSchema, data []byte) error {
	return validateJSONObject("input", schema, data)
}

func ValidateJSONOutput(schema ObjectSchema, data []byte) error {
	return validateJSONObject("output", schema, data)
}

func validateJSONObject(label string, schema ObjectSchema, data []byte) error {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("go_program: %s JSON invalid: %w", label, err)
	}
	for _, key := range schema.Required {
		if _, ok := obj[key]; !ok {
			return fmt.Errorf("go_program: %s JSON missing required field %q", label, key)
		}
	}
	return nil
}
