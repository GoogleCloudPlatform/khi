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

package structured

import (
	"testing"
	"time"
)

func TestNodeReader(t *testing.T) {
	yamlData := `
string_value: "test string"
bool_value: true
int_value: 42
float_value: 3.14
timestamp_value: 2022-01-01T12:00:00Z
null_value: null
nested:
  inner_value: "nested value"
  inner_bool: false
array:
  - "item1"
  - "item2"
  - "item3"
complex:
  objects:
    - name: "object1"
      value: 100
    - name: "object2"
      value: 200
`
	node, err := FromYAML(yamlData)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	reader := &NodeReader{Node: node}

	pathStringValue := CompileFieldPath("string_value")
	pathBoolValue := CompileFieldPath("bool_value")
	pathIntValue := CompileFieldPath("int_value")
	pathFloatValue := CompileFieldPath("float_value")
	pathTimestampValue := CompileFieldPath("timestamp_value")
	pathNonexistent := CompileFieldPath("nonexistent")
	pathNestedInnerBool := CompileFieldPath("nested.inner_bool")
	pathNestedInnerValue := CompileFieldPath("nested.inner_value")
	pathNested := CompileFieldPath("nested")
	pathArray := CompileFieldPath("array")
	pathComplexObjects := CompileFieldPath("complex.objects")
	pathName := CompileFieldPath("name")
	pathValue := CompileFieldPath("value")

	t.Run("ReadString", func(t *testing.T) {
		value, err := reader.ReadString(pathStringValue)
		if err != nil {
			t.Errorf("ReadString failed: %v", err)
		}
		if value != "test string" {
			t.Errorf("Expected 'test string', got %q", value)
		}

		_, err = reader.ReadString(pathNonexistent)
		if err != ErrFieldNotFound {
			t.Errorf("Expected ErrFieldNotFound, got %v", err)
		}
	})

	t.Run("ReadBool", func(t *testing.T) {
		value, err := reader.ReadBool(pathBoolValue)
		if err != nil {
			t.Errorf("ReadBool failed: %v", err)
		}
		if !value {
			t.Errorf("Expected true, got %v", value)
		}

		value, err = reader.ReadBool(pathNestedInnerBool)
		if err != nil {
			t.Errorf("ReadBool for nested field failed: %v", err)
		}
		if value {
			t.Errorf("Expected false, got %v", value)
		}
	})

	t.Run("ReadInt", func(t *testing.T) {
		value, err := reader.ReadInt(pathIntValue)
		if err != nil {
			t.Errorf("ReadInt failed: %v", err)
		}
		if value != 42 {
			t.Errorf("Expected 42, got %d", value)
		}
	})

	t.Run("ReadFloat", func(t *testing.T) {
		value, err := reader.ReadFloat(pathFloatValue)
		if err != nil {
			t.Errorf("ReadFloat failed: %v", err)
		}
		if value != 3.14 {
			t.Errorf("Expected 3.14, got %f", value)
		}
	})

	t.Run("ReadTimestamp", func(t *testing.T) {
		value, err := reader.ReadTimestamp(pathTimestampValue)
		if err != nil {
			t.Errorf("ReadTimestamp failed: %v", err)
		}
		expected, _ := time.Parse(time.RFC3339, "2022-01-01T12:00:00Z")
		if !value.Equal(expected) {
			t.Errorf("Expected %v, got %v", expected, value)
		}
	})

	t.Run("ReadDefaultValues", func(t *testing.T) {
		// String with existing field
		value := reader.ReadStringOrDefault(pathStringValue, "default")
		if value != "test string" {
			t.Errorf("Expected 'test string', got %q", value)
		}

		// String with non-existent field
		value = reader.ReadStringOrDefault(pathNonexistent, "default")
		if value != "default" {
			t.Errorf("Expected 'default', got %q", value)
		}

		// Bool
		boolValue := reader.ReadBoolOrDefault(pathNonexistent, true)
		if !boolValue {
			t.Errorf("Expected true, got %v", boolValue)
		}

		// Int
		intValue := reader.ReadIntOrDefault(pathNonexistent, 100)
		if intValue != 100 {
			t.Errorf("Expected 100, got %d", intValue)
		}

		// Float
		floatValue := reader.ReadFloatOrDefault(pathNonexistent, 2.71)
		if floatValue != 2.71 {
			t.Errorf("Expected 2.71, got %f", floatValue)
		}
	})

	t.Run("Has", func(t *testing.T) {
		if !reader.Has(pathStringValue) {
			t.Error("Has returned false for existing field")
		}

		if reader.Has(pathNonexistent) {
			t.Error("Has returned true for non-existing field")
		}

		if !reader.Has(pathNestedInnerValue) {
			t.Error("Has returned false for existing nested field")
		}
	})

	t.Run("NestedFields", func(t *testing.T) {
		value, err := reader.ReadString(pathNestedInnerValue)
		if err != nil {
			t.Errorf("ReadString for nested field failed: %v", err)
		}
		if value != "nested value" {
			t.Errorf("Expected 'nested value', got %q", value)
		}
	})

	t.Run("Children", func(t *testing.T) {
		childCount := 0
		reader.Children()(func(key NodeChildrenKey, value NodeReader) bool {
			childCount++
			return true
		})

		// Count top-level fields in YAML
		expectedChildCount := 9
		if childCount != expectedChildCount {
			t.Errorf("Expected %d children, got %d", expectedChildCount, childCount)
		}
	})

	t.Run("GetReader", func(t *testing.T) {
		nestedReader, err := reader.GetReader(pathNested)
		if err != nil {
			t.Errorf("GetReader failed: %v", err)
		}

		value, err := nestedReader.ReadString(CompileFieldPath("inner_value"))
		if err != nil {
			t.Errorf("ReadString on nested reader failed: %v", err)
		}
		if value != "nested value" {
			t.Errorf("Expected 'nested value', got %q", value)
		}

		_, err = reader.GetReader(pathNonexistent)
		if err != ErrFieldNotFound {
			t.Errorf("Expected ErrFieldNotFound, got %v", err)
		}
	})

	t.Run("ArrayChildren", func(t *testing.T) {
		arrayReader, err := reader.GetReader(pathArray)
		if err != nil {
			t.Errorf("GetReader for array failed: %v", err)
		}

		itemCount := 0
		expectedItems := []string{"item1", "item2", "item3"}
		arrayReader.Children()(func(key NodeChildrenKey, childReader NodeReader) bool {
			if key.Index != itemCount {
				t.Errorf("Expected index %d, got %d", itemCount, key.Index)
			}

			value, err := childReader.ReadString(EmptyFieldPath)
			if err != nil {
				t.Errorf("ReadString failed: %v", err)
			}
			if value != expectedItems[itemCount] {
				t.Errorf("Expected %q, got %q", expectedItems[itemCount], value)
			}

			itemCount++
			return true
		})

		if itemCount != len(expectedItems) {
			t.Errorf("Expected %d items, got %d", len(expectedItems), itemCount)
		}
	})

	t.Run("ComplexArrayChildren", func(t *testing.T) {
		objectsReader, err := reader.GetReader(pathComplexObjects)
		if err != nil {
			t.Errorf("GetReader for complex.objects failed: %v", err)
		}

		objectCount := 0
		expectedObjects := []struct {
			name  string
			value int
		}{
			{"object1", 100},
			{"object2", 200},
		}

		objectsReader.Children()(func(key NodeChildrenKey, objReader NodeReader) bool {
			nameValue, err := objReader.ReadString(pathName)
			if err != nil {
				t.Errorf("ReadString for name failed: %v", err)
			}
			if nameValue != expectedObjects[objectCount].name {
				t.Errorf("Expected name %q, got %q", expectedObjects[objectCount].name, nameValue)
			}

			intValue, err := objReader.ReadInt(pathValue)
			if err != nil {
				t.Errorf("ReadInt for value failed: %v", err)
			}
			if intValue != expectedObjects[objectCount].value {
				t.Errorf("Expected value %d, got %d", expectedObjects[objectCount].value, intValue)
			}

			objectCount++
			return true
		})

		if objectCount != len(expectedObjects) {
			t.Errorf("Expected %d objects, got %d", len(expectedObjects), objectCount)
		}
	})
}

func TestFieldPath(t *testing.T) {
	yamlData := `
string_value: "test string"
nested:
  inner_value: "nested value"
  inner_int: 123
  inner_bool: true
`
	node, err := FromYAML(yamlData)
	if err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	reader := NewNodeReader(node)

	pathString := CompileFieldPath("string_value")
	pathNestedStr := CompileFieldPath("nested.inner_value")
	pathNestedInt := CompileFieldPath("nested.inner_int")
	pathNestedBool := CompileFieldPath("nested.inner_bool")
	pathNonExistent := CompileFieldPath("nested.non_existent")

	if got := reader.ReadStringOrDefault(pathString, ""); got != "test string" {
		t.Errorf("ReadStringOrDefault() mismatch: want %q, got %q", "test string", got)
	}
	if got := reader.ReadStringOrDefault(pathNestedStr, ""); got != "nested value" {
		t.Errorf("ReadStringOrDefault() mismatch: want %q, got %q", "nested value", got)
	}
	if got := reader.ReadIntOrDefault(pathNestedInt, 0); got != 123 {
		t.Errorf("ReadIntOrDefault() mismatch: want %d, got %d", 123, got)
	}
	if got := reader.ReadBoolOrDefault(pathNestedBool, false); !got {
		t.Errorf("ReadBoolOrDefault() mismatch: want true, got %v", got)
	}
	if got := reader.ReadStringOrDefault(pathNonExistent, "default"); got != "default" {
		t.Errorf("ReadStringOrDefault() fallback mismatch: want %q, got %q", "default", got)
	}
}
