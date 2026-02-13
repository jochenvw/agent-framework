// Copyright (c) Microsoft. All rights reserved.

package agentframework

import (
	"encoding/json"
	"reflect"
	"strings"
)

// generateSchemaFromType uses reflection to produce a JSON Schema for a struct.
func generateSchemaFromType(v any) json.RawMessage {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	schema := schemaForType(t)
	b, _ := json.Marshal(schema)
	return b
}

func schemaForType(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": schemaForType(t.Elem()),
		}
	case reflect.Ptr:
		return schemaForType(t.Elem())
	case reflect.Struct:
		return schemaForStruct(t)
	case reflect.Map:
		if t.Key().Kind() == reflect.String {
			return map[string]any{
				"type":                 "object",
				"additionalProperties": schemaForType(t.Elem()),
			}
		}
		return map[string]any{"type": "object"}
	default:
		return map[string]any{"type": "string"}
	}
}

func schemaForStruct(t reflect.Type) map[string]any {
	properties := make(map[string]any)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Determine JSON field name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		name := field.Name
		if jsonTag != "" {
			parts := strings.SplitN(jsonTag, ",", 2)
			if parts[0] != "" {
				name = parts[0]
			}
		}

		prop := schemaForType(field.Type)

		// Parse jsonschema tag
		jsTag := field.Tag.Get("jsonschema")
		if jsTag != "" {
			for _, part := range strings.Split(jsTag, ",") {
				kv := strings.SplitN(part, "=", 2)
				key := strings.TrimSpace(kv[0])
				val := ""
				if len(kv) == 2 {
					val = strings.TrimSpace(kv[1])
				}
				switch key {
				case "description":
					prop["description"] = val
				case "required":
					required = append(required, name)
				case "enum":
					enumVals := strings.Split(val, "|")
					anyVals := make([]any, len(enumVals))
					for j, ev := range enumVals {
						anyVals[j] = strings.TrimSpace(ev)
					}
					prop["enum"] = anyVals
				}
			}
		}

		properties[name] = prop
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
