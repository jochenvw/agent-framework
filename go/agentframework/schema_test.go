// Copyright (c) Microsoft. All rights reserved.

package agentframework_test

import (
	"encoding/json"
	"testing"

	af "github.com/microsoft/agent-framework/go/agentframework"
)

type weatherArgs struct {
	Location string `json:"location" jsonschema:"description=City name,required"`
	Unit     string `json:"unit"     jsonschema:"description=Temperature unit,enum=celsius|fahrenheit"`
}

func TestGenerateSchema_BasicStruct(t *testing.T) {
	schema := af.GenerateSchema[weatherArgs]()

	var parsed map[string]any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("type = %v, want object", parsed["type"])
	}

	props, ok := parsed["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties not a map: %T", parsed["properties"])
	}

	locProp, ok := props["location"].(map[string]any)
	if !ok {
		t.Fatalf("location property missing or wrong type")
	}
	if locProp["type"] != "string" {
		t.Errorf("location type = %v", locProp["type"])
	}
	if locProp["description"] != "City name" {
		t.Errorf("location description = %v", locProp["description"])
	}

	unitProp, ok := props["unit"].(map[string]any)
	if !ok {
		t.Fatalf("unit property missing or wrong type")
	}
	enumVals, ok := unitProp["enum"].([]any)
	if !ok {
		t.Fatalf("unit enum missing or wrong type: %T", unitProp["enum"])
	}
	if len(enumVals) != 2 {
		t.Errorf("enum len = %d, want 2", len(enumVals))
	}

	required, ok := parsed["required"].([]any)
	if !ok {
		t.Fatalf("required missing or wrong type")
	}
	found := false
	for _, r := range required {
		if r == "location" {
			found = true
		}
	}
	if !found {
		t.Error("location not in required list")
	}
}

type nestedArgs struct {
	Items []string       `json:"items"`
	Tags  map[string]int `json:"tags"`
	Count int            `json:"count"`
	Flag  bool           `json:"flag"`
	Score float64        `json:"score"`
}

func TestGenerateSchema_TypeMapping(t *testing.T) {
	schema := af.GenerateSchema[nestedArgs]()

	var parsed map[string]any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatal(err)
	}

	props := parsed["properties"].(map[string]any)

	// Array of strings
	items := props["items"].(map[string]any)
	if items["type"] != "array" {
		t.Errorf("items type = %v", items["type"])
	}
	itemsInner := items["items"].(map[string]any)
	if itemsInner["type"] != "string" {
		t.Errorf("items inner type = %v", itemsInner["type"])
	}

	// Map
	tags := props["tags"].(map[string]any)
	if tags["type"] != "object" {
		t.Errorf("tags type = %v", tags["type"])
	}

	// Int
	count := props["count"].(map[string]any)
	if count["type"] != "integer" {
		t.Errorf("count type = %v", count["type"])
	}

	// Bool
	flag := props["flag"].(map[string]any)
	if flag["type"] != "boolean" {
		t.Errorf("flag type = %v", flag["type"])
	}

	// Float
	score := props["score"].(map[string]any)
	if score["type"] != "number" {
		t.Errorf("score type = %v", score["type"])
	}
}
