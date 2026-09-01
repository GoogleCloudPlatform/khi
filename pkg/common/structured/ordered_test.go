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

	"github.com/google/go-cmp/cmp"
)

func TestWithKeyOrder(t *testing.T) {
	testCases := []struct {
		name         string
		jsonInput    string
		priorityKeys []string
		wantKeys     []string
	}{
		{
			name:         "reorders specified keys to the front",
			jsonInput:    `{"zebra": 1, "apple": 2, "mango": 3, "banana": 4}`,
			priorityKeys: []string{"mango", "apple"},
			wantKeys:     []string{"mango", "apple", "zebra", "banana"},
		},
		{
			name:         "skips nonexistent priority keys gracefully",
			jsonInput:    `{"a": 1, "b": 2, "c": 3}`,
			priorityKeys: []string{"nonexistent", "c", "missing"},
			wantKeys:     []string{"c", "a", "b"},
		},
		{
			name:         "empty priority keys returns original order",
			jsonInput:    `{"a": 1, "b": 2, "c": 3}`,
			priorityKeys: []string{},
			wantKeys:     []string{"a", "b", "c"},
		},
		{
			name:         "all keys prioritized in reverse order",
			jsonInput:    `{"a": 1, "b": 2, "c": 3}`,
			priorityKeys: []string{"c", "b", "a"},
			wantKeys:     []string{"c", "b", "a"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := NewLazyJSONNodeFromBytes([]byte(tc.jsonInput))
			ordered := WithKeyOrder(node, tc.priorityKeys...)

			var gotKeys []string
			for k := range ordered.Children() {
				gotKeys = append(gotKeys, k.Key)
			}

			if diff := cmp.Diff(tc.wantKeys, gotKeys); diff != "" {
				t.Errorf("WithKeyOrder() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWithKeyOrderNonMap(t *testing.T) {
	scalar := NewStandardScalarNode("hello")
	orderedScalar := WithKeyOrder(scalar, "a", "b")
	if orderedScalar != scalar {
		t.Errorf("WithKeyOrder on scalar should return original node")
	}

	nilOrdered := WithKeyOrder(nil, "a")
	if nilOrdered != nil {
		t.Errorf("WithKeyOrder on nil should return nil")
	}
}

func TestNodeReaderWithKeyOrder(t *testing.T) {
	jsonBytes := []byte(`{"logName":"projects/p/logs/l","insertId":"ins-123","severity":"INFO"}`)
	node := NewLazyJSONNodeFromBytes(jsonBytes)
	reader := NewNodeReader(node)

	orderedReader := reader.WithKeyOrder("insertId", "logName")
	yamlBytes, err := orderedReader.Serialize(EmptyFieldPath, &YAMLNodeSerializer{})
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	wantYAML := "insertId: ins-123\nlogName: projects/p/logs/l\nseverity: INFO\n"
	if diff := cmp.Diff(wantYAML, string(yamlBytes)); diff != "" {
		t.Errorf("Serialize mismatch (-want +got):\n%s", diff)
	}
}

func TestWithKeyOrderNested(t *testing.T) {
	jsonBytes := []byte(`{"a": 1, "b": 2, "c": 3}`)
	node := NewLazyJSONNodeFromBytes(jsonBytes)
	ordered1 := WithKeyOrder(node, "b")
	ordered2 := WithKeyOrder(ordered1, "c")

	orderedMap, ok := ordered2.(*orderedMapNode)
	if !ok {
		t.Fatalf("expected *orderedMapNode, got %T", ordered2)
	}
	if _, isNested := orderedMap.inner.(*orderedMapNode); isNested {
		t.Errorf("expected inner node not to be *orderedMapNode, but got nested *orderedMapNode")
	}

	var gotKeys []string
	for k := range ordered2.Children() {
		gotKeys = append(gotKeys, k.Key)
	}
	wantKeys := []string{"c", "a", "b"}
	if diff := cmp.Diff(wantKeys, gotKeys); diff != "" {
		t.Errorf("WithKeyOrder nested keys mismatch (-want +got):\n%s", diff)
	}
}
