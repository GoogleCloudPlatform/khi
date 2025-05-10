// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package structurev2

import (
	"testing"
)

func TestFromGoValue(t *testing.T) {
	orderProvider := &AlphabeticalGoMapKeyOrderProvider{}

	t.Run("ScalarValue", func(t *testing.T) {
		testScalarValue(t, orderProvider)
	})

	t.Run("SliceValue", func(t *testing.T) {
		testSliceValue(t, orderProvider)
	})

	t.Run("MapValue", func(t *testing.T) {
		testMapValue(t, orderProvider)
	})

	t.Run("ComplexValue", func(t *testing.T) {
		testComplexValue(t, orderProvider)
	})
}

func testScalarValue(t *testing.T, orderProvider *AlphabeticalGoMapKeyOrderProvider) {
	input := "test value"

	result, err := FromGoValue(input, orderProvider)

	if err != nil {
		t.Fatalf("FromGoValue returned error: %v", err)
	}

	if result.Type() != ScalarNodeType {
		t.Errorf("Expected ScalarNodeType, got %v", result.Type())
	}

	value, err := result.NodeScalarValue()
	if err != nil {
		t.Fatalf("NodeScalarValue returned error: %v", err)
	}

	if value != input {
		t.Errorf("Expected value %v, got %v", input, value)
	}
}

func testSliceValue(t *testing.T, orderProvider *AlphabeticalGoMapKeyOrderProvider) {
	input := []any{1, "test", true}

	result, err := FromGoValue(input, orderProvider)

	if err != nil {
		t.Fatalf("FromGoValue returned error: %v", err)
	}

	if result.Type() != SequenceNodeType {
		t.Errorf("Expected SequenceNodeType, got %v", result.Type())
	}

	childCount := 0
	result.Children()(func(key NodeChildrenKey, value Node) bool {
		childCount++

		rawValue, err := value.NodeScalarValue()
		if err != nil {
			t.Fatalf("NodeScalarValue returned error: %v", err)
		}

		if input[key.Index] != rawValue {
			t.Errorf("Expected value %v, got %v", input[key.Index], rawValue)
		}

		return true
	})

	if childCount != len(input) {
		t.Errorf("Expected %d children, got %d", len(input), childCount)
	}
}

func testMapValue(t *testing.T, orderProvider *AlphabeticalGoMapKeyOrderProvider) {
	input := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}

	result, err := FromGoValue(input, orderProvider)

	if err != nil {
		t.Fatalf("FromGoValue returned error: %v", err)
	}

	if result.Type() != MapNodeType {
		t.Errorf("Expected MapNodeType, got %v", result.Type())
	}

	childCount := 0
	result.Children()(func(key NodeChildrenKey, value Node) bool {
		childCount++

		rawValue, err := value.NodeScalarValue()
		if err != nil {
			t.Fatalf("NodeScalarValue returned error: %v", err)
		}

		if input[key.Key] != rawValue {
			t.Errorf("Expected value %v, got %v", input[key.Key], rawValue)
		}

		return true
	})

	if childCount != len(input) {
		t.Errorf("Expected %d children, got %d", len(input), childCount)
	}
}

func testComplexValue(t *testing.T, orderProvider *AlphabeticalGoMapKeyOrderProvider) {
	input := map[string]any{
		"scalar": "value",
		"array":  []any{1, 2, 3},
		"map": map[string]any{
			"nested": "nestedValue",
		},
	}

	result, err := FromGoValue(input, orderProvider)

	if err != nil {
		t.Fatalf("FromGoValue returned error: %v", err)
	}

	if result.Type() != MapNodeType {
		t.Errorf("Expected MapNodeType, got %v", result.Type())
	}

	childCount := 0
	nodeTypes := make(map[string]NodeType)

	result.Children()(func(key NodeChildrenKey, value Node) bool {
		childCount++
		nodeTypes[key.Key] = value.Type()
		return true
	})

	if childCount != 3 {
		t.Errorf("Expected 3 children, got %d", childCount)
	}

	expectedTypes := map[string]NodeType{
		"scalar": ScalarNodeType,
		"array":  SequenceNodeType,
		"map":    MapNodeType,
	}

	for key, expectedType := range expectedTypes {
		if nodeType, ok := nodeTypes[key]; !ok {
			t.Errorf("Missing expected key %q", key)
		} else if nodeType != expectedType {
			t.Errorf("Key %q: expected %v, got %v", key, expectedType, nodeType)
		}
	}
}

func TestAlphabeticalGoMapKeyOrderProvider(t *testing.T) {
	provider := &AlphabeticalGoMapKeyOrderProvider{}

	input := map[string]any{
		"c": 3,
		"a": 1,
		"b": 2,
	}

	keys, err := provider.GetOrderedKeys([]string{"test", "path"}, input)

	if err != nil {
		t.Fatalf("GetOrderedKeys returned error: %v", err)
	}

	if len(keys) != len(input) {
		t.Errorf("Expected %d keys, got %d", len(input), len(keys))
	}

	expectedKeys := []string{"a", "b", "c"}
	for i, expectedKey := range expectedKeys {
		if i >= len(keys) {
			t.Errorf("Missing expected key at index %d", i)
			continue
		}

		if keys[i] != expectedKey {
			t.Errorf("Expected key %q at index %d, got %q", expectedKey, i, keys[i])
		}
	}
}
