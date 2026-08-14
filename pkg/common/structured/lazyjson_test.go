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
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestLazyJSONNode_Scalar(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		wantType NodeType
		wantVal  any
		wantErr  bool
	}{
		{
			name:     "string without escape",
			input:    `"hello world"`,
			wantType: ScalarNodeType,
			wantVal:  "hello world",
		},
		{
			name:     "string with various escapes",
			input:    `"hello \"world\"\n\t\\\/ \b\f \u0041\u0042"`,
			wantType: ScalarNodeType,
			wantVal:  "hello \"world\"\n\t\\/ \b\f AB",
		},
		{
			name:     "positive integer",
			input:    `42`,
			wantType: ScalarNodeType,
			wantVal:  42,
		},
		{
			name:     "negative integer",
			input:    `-100`,
			wantType: ScalarNodeType,
			wantVal:  -100,
		},
		{
			name:     "float number",
			input:    `3.14159`,
			wantType: ScalarNodeType,
			wantVal:  3.14159,
		},
		{
			name:     "float with exponent",
			input:    `1.5e3`,
			wantType: ScalarNodeType,
			wantVal:  1500.0,
		},
		{
			name:     "boolean true",
			input:    `true`,
			wantType: ScalarNodeType,
			wantVal:  true,
		},
		{
			name:     "boolean false",
			input:    `false`,
			wantType: ScalarNodeType,
			wantVal:  false,
		},
		{
			name:     "null value",
			input:    `null`,
			wantType: ScalarNodeType,
			wantVal:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := NewLazyJSONNodeFromBytes([]byte(tc.input))
			if diff := cmp.Diff(tc.wantType, node.Type()); diff != "" {
				t.Errorf("Type() mismatch (-want +got):\n%s", diff)
			}
			gotVal, err := node.NodeScalarValue()
			if (err != nil) != tc.wantErr {
				t.Fatalf("NodeScalarValue() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.wantVal, gotVal); diff != "" {
				t.Errorf("NodeScalarValue() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLazyJSONNode_Sequence(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		wantLen    int
		wantValues []any
	}{
		{
			name:       "empty array",
			input:      `[]`,
			wantLen:    0,
			wantValues: []any{},
		},
		{
			name:       "scalar array",
			input:      `["foo", 123, true, null, 4.5]`,
			wantLen:    5,
			wantValues: []any{"foo", 123, true, nil, 4.5},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := NewLazyJSONNodeFromBytes([]byte(tc.input))
			if diff := cmp.Diff(NodeType(SequenceNodeType), node.Type()); diff != "" {
				t.Errorf("Type() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantLen, node.Len()); diff != "" {
				t.Errorf("Len() mismatch (-want +got):\n%s", diff)
			}

			gotValues := make([]any, 0)
			for key, child := range node.Children() {
				if diff := cmp.Diff("", key.Key); diff != "" {
					t.Errorf("key.Key for sequence must be empty (-want +got):\n%s", diff)
				}
				childVal, err := child.NodeScalarValue()
				if err != nil {
					t.Fatalf("child.NodeScalarValue() error = %v", err)
				}
				gotValues = append(gotValues, childVal)
			}
			if diff := cmp.Diff(tc.wantValues, gotValues); diff != "" {
				t.Errorf("Children values mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLazyJSONNode_Map(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		wantLen    int
		wantKeys   []string
		wantValues []any
	}{
		{
			name:       "empty object",
			input:      `{}`,
			wantLen:    0,
			wantKeys:   []string{},
			wantValues: []any{},
		},
		{
			name:       "flat map",
			input:      `{"str":"value","num":100,"flag":true,"empty":null}`,
			wantLen:    4,
			wantKeys:   []string{"str", "num", "flag", "empty"},
			wantValues: []any{"value", 100, true, nil},
		},
		{
			name:       "map with escaped key",
			input:      `{"key \"1\"":"val1","key\n2":"val2"}`,
			wantLen:    2,
			wantKeys:   []string{`key "1"`, "key\n2"},
			wantValues: []any{"val1", "val2"},
		},
		{
			name:       "map with escaped quotes in values and subsequent keys",
			input:      `{"first":"hello \"world\"","second":"after"}`,
			wantLen:    2,
			wantKeys:   []string{"first", "second"},
			wantValues: []any{`hello "world"`, "after"},
		},
		{
			name:       "map with escaped backslashes in values",
			input:      `{"path":"C:\\dir\\file.txt","second":"ok"}`,
			wantLen:    2,
			wantKeys:   []string{"path", "second"},
			wantValues: []any{`C:\dir\file.txt`, "ok"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := NewLazyJSONNodeFromBytes([]byte(tc.input))
			if diff := cmp.Diff(NodeType(MapNodeType), node.Type()); diff != "" {
				t.Errorf("Type() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantLen, node.Len()); diff != "" {
				t.Errorf("Len() mismatch (-want +got):\n%s", diff)
			}

			gotKeys := make([]string, 0)
			gotValues := make([]any, 0)
			for key, child := range node.Children() {
				gotKeys = append(gotKeys, key.Key)
				childVal, err := child.NodeScalarValue()
				if err != nil {
					t.Fatalf("child.NodeScalarValue() error = %v", err)
				}
				gotValues = append(gotValues, childVal)
			}
			if diff := cmp.Diff(tc.wantKeys, gotKeys); diff != "" {
				t.Errorf("Children keys mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantValues, gotValues); diff != "" {
				t.Errorf("Children values mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLazyJSONNode_NodeReader(t *testing.T) {
	jsonStr := `{
		"name": "test-pod",
		"count": 5,
		"ratio": 0.75,
		"enabled": true,
		"timestamp": "2025-01-02T03:04:05Z",
		"labels": {
			"app": "khi",
			"tier": "backend"
		},
		"items": ["a", "b", "c"]
	}`

	node := NewLazyJSONNodeFromBytes([]byte(jsonStr))
	reader := NewNodeReader(node)

	name, err := reader.ReadString("name")
	if err != nil {
		t.Fatalf("ReadString(name) failed: %v", err)
	}
	if diff := cmp.Diff("test-pod", name); diff != "" {
		t.Errorf("ReadString(name) mismatch (-want +got):\n%s", diff)
	}

	count, err := reader.ReadInt("count")
	if err != nil {
		t.Fatalf("ReadInt(count) failed: %v", err)
	}
	if diff := cmp.Diff(5, count); diff != "" {
		t.Errorf("ReadInt(count) mismatch (-want +got):\n%s", diff)
	}

	ratio, err := reader.ReadFloat("ratio")
	if err != nil {
		t.Fatalf("ReadFloat(ratio) failed: %v", err)
	}
	if diff := cmp.Diff(0.75, ratio); diff != "" {
		t.Errorf("ReadFloat(ratio) mismatch (-want +got):\n%s", diff)
	}

	enabled, err := reader.ReadBool("enabled")
	if err != nil {
		t.Fatalf("ReadBool(enabled) failed: %v", err)
	}
	if diff := cmp.Diff(true, enabled); diff != "" {
		t.Errorf("ReadBool(enabled) mismatch (-want +got):\n%s", diff)
	}

	ts, err := reader.ReadTimestamp("timestamp")
	if err != nil {
		t.Fatalf("ReadTimestamp(timestamp) failed: %v", err)
	}
	expectedTime := time.Date(2025, time.January, 2, 3, 4, 5, 0, time.UTC)
	if !ts.Equal(expectedTime) {
		t.Errorf("ReadTimestamp(timestamp) got %v, want %v", ts, expectedTime)
	}

	app, err := reader.ReadString("labels.app")
	if err != nil {
		t.Fatalf("ReadString(labels.app) failed: %v", err)
	}
	if diff := cmp.Diff("khi", app); diff != "" {
		t.Errorf("ReadString(labels.app) mismatch (-want +got):\n%s", diff)
	}

	if !reader.Has("labels.tier") {
		t.Errorf("Has(labels.tier) expected true, got false")
	}
	if reader.Has("labels.nonexistent") {
		t.Errorf("Has(labels.nonexistent) expected false, got true")
	}
}

func TestNewLazyJSONNode(t *testing.T) {
	stdMap := NewStandardMap(
		[]string{"foo", "bar"},
		[]Node{
			NewStandardScalarNode("hello"),
			NewStandardScalarNode(123),
		},
	)

	lazyNode, err := NewLazyJSONNode(stdMap)
	if err != nil {
		t.Fatalf("NewLazyJSONNode failed: %v", err)
	}

	reader := NewNodeReader(lazyNode)
	fooStr, err := reader.ReadString("foo")
	if err != nil {
		t.Fatalf("ReadString(foo) failed: %v", err)
	}
	if diff := cmp.Diff("hello", fooStr); diff != "" {
		t.Errorf("ReadString(foo) mismatch (-want +got):\n%s", diff)
	}

	barInt, err := reader.ReadInt("bar")
	if err != nil {
		t.Fatalf("ReadInt(bar) failed: %v", err)
	}
	if diff := cmp.Diff(123, barInt); diff != "" {
		t.Errorf("ReadInt(bar) mismatch (-want +got):\n%s", diff)
	}
}

func TestLazyJSONNode_Concurrency(t *testing.T) {
	jsonStr := `{"foo":"bar","nested":{"num":42},"arr":[1,2,3]}`
	node := NewLazyJSONNodeFromBytes([]byte(jsonStr))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reader := NewNodeReader(node)
			for j := 0; j < 100; j++ {
				str, err := reader.ReadString("foo")
				if err != nil || str != "bar" {
					t.Errorf("concurrent ReadString failed: %v, got %s", err, str)
				}
				num, err := reader.ReadInt("nested.num")
				if err != nil || num != 42 {
					t.Errorf("concurrent ReadInt failed: %v, got %d", err, num)
				}
				if node.Len() != 3 {
					t.Errorf("concurrent Len failed, got %d", node.Len())
				}
			}
		}()
	}
	wg.Wait()
}

func TestLazyJSONNode_ChildrenEarlyBreak(t *testing.T) {
	mapNode := NewLazyJSONNodeFromBytes([]byte(`{"a":1,"b":2,"c":3}`))
	mapKeys := make([]string, 0)
	for key := range mapNode.Children() {
		mapKeys = append(mapKeys, key.Key)
		if key.Index == 0 {
			break
		}
	}
	if diff := cmp.Diff([]string{"a"}, mapKeys); diff != "" {
		t.Errorf("map Children early break mismatch (-want +got):\n%s", diff)
	}

	seqNode := NewLazyJSONNodeFromBytes([]byte(`[10,20,30,40]`))
	seqCount := 0
	for key := range seqNode.Children() {
		seqCount++
		if key.Index == 1 {
			break
		}
	}
	if diff := cmp.Diff(2, seqCount); diff != "" {
		t.Errorf("sequence Children early break count mismatch (-want +got):\n%s", diff)
	}
}

func TestLazyJSONNode_Serialization(t *testing.T) {
	inputJSON := `{"foo":"bar","items":[1,2,3]}`
	lazyNode := NewLazyJSONNodeFromBytes([]byte(inputJSON))

	yamlSerializer := &YAMLNodeSerializer{}
	yamlBytes, err := yamlSerializer.Serialize(lazyNode)
	if err != nil {
		t.Fatalf("YAMLNodeSerializer.Serialize failed: %v", err)
	}
	expectedYAML := "foo: bar\nitems:\n  - 1\n  - 2\n  - 3\n"
	if diff := cmp.Diff(expectedYAML, string(yamlBytes)); diff != "" {
		t.Errorf("YAML serialization mismatch (-want +got):\n%s", diff)
	}

	jsonSerializer := &JSONNodeSerializer{}
	jsonBytes, err := jsonSerializer.Serialize(lazyNode)
	if err != nil {
		t.Fatalf("JSONNodeSerializer.Serialize failed: %v", err)
	}
	expectedJSON := `{"foo":"bar","items":[1,2,3]}`
	if diff := cmp.Diff(expectedJSON, string(jsonBytes)); diff != "" {
		t.Errorf("JSON serialization mismatch (-want +got):\n%s", diff)
	}
}

func TestLazyJSONNode_ReadReflectWithEscapes(t *testing.T) {
	type NestedStruct struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}
	type SampleStruct struct {
		Title  string       `json:"title"`
		Nested NestedStruct `json:"nested"`
	}

	inputJSON := `{"title":"hello \"world\" \n test","nested":{"message":"inner \"quotes\" and \\ slashes","code":200}}`
	node := NewLazyJSONNodeFromBytes([]byte(inputJSON))
	reader := NewNodeReader(node)

	var target SampleStruct
	err := ReadReflect(reader, "", &target)
	if err != nil {
		t.Fatalf("ReadReflect failed: %v", err)
	}

	expected := SampleStruct{
		Title: "hello \"world\" \n test",
		Nested: NestedStruct{
			Message: "inner \"quotes\" and \\ slashes",
			Code:    200,
		},
	}
	if diff := cmp.Diff(expected, target); diff != "" {
		t.Errorf("ReadReflect mismatch (-want +got):\n%s", diff)
	}
}

func TestLazyJSONNode_MergeNode(t *testing.T) {
	prev := NewLazyJSONNodeFromBytes([]byte(`{"name":"test","count":1,"labels":{"env":"prod"}}`))
	patch := NewLazyJSONNodeFromBytes([]byte(`{"count":2,"labels":{"tier":"frontend"}}`))

	merged, err := MergeNode(prev, patch, MergeConfiguration{
		MergeMapOrderStrategy: &DefaultMergeMapOrderStrategy{},
	})
	if err != nil {
		t.Fatalf("MergeNode failed: %v", err)
	}

	reader := NewNodeReader(merged)
	name, err := reader.ReadString("name")
	if err != nil || name != "test" {
		t.Errorf("merged name mismatch: %v, %s", err, name)
	}
	count, err := reader.ReadInt("count")
	if err != nil || count != 2 {
		t.Errorf("merged count mismatch: %v, %d", err, count)
	}
	env, err := reader.ReadString("labels.env")
	if err != nil || env != "prod" {
		t.Errorf("merged labels.env mismatch: %v, %s", err, env)
	}
	tier, err := reader.ReadString("labels.tier")
	if err != nil || tier != "frontend" {
		t.Errorf("merged labels.tier mismatch: %v, %s", err, tier)
	}
}

func BenchmarkLazyJSONNodeVsStandardMap(b *testing.B) {
	rawJSON := `{"insertId":"123","logName":"projects/p/logs/l","labels":{"k1":"v1","k2":"v2"},"resource":{"type":"gce_instance","labels":{"zone":"us-central1-a"}}}`
	nodeFromJSON, err := FromYAML(rawJSON)
	if err != nil {
		b.Fatal(err)
	}
	lazyNode := NewLazyJSONNodeFromBytes([]byte(rawJSON))

	b.Run("StandardMap_ReadString", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			r := NewNodeReader(nodeFromJSON)
			_, _ = r.ReadString("resource.labels.zone")
		}
	})

	b.Run("LazyJSONNode_ReadString", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			r := NewNodeReader(lazyNode)
			_, _ = r.ReadString("resource.labels.zone")
		}
	})
}
