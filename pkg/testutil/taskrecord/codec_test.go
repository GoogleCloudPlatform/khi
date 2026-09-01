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

package taskrecord

import (
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
)

var (
	pathTextPayload            = structured.CompileFieldPath("textPayload")
	pathProtoPayloadMethodName = structured.CompileFieldPath("protoPayload.methodName")
)

type testCustomStruct struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestDefaultJSONCodec(t *testing.T) {
	codec := &DefaultJSONCodec{}

	testCases := []struct {
		name  string
		input any
	}{
		{
			name: "map input",
			input: map[string]any{
				"foo": "bar",
			},
		},
		{
			name: "struct input",
			input: testCustomStruct{
				Name:  "test",
				Count: 42,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			serialized, err := codec.Serialize(tc.input)
			if err != nil {
				t.Fatalf("failed to serialize: %v", err)
			}

			deserialized, err := codec.Deserialize(serialized)
			if err != nil {
				t.Fatalf("failed to deserialize: %v", err)
			}

			if deserialized == nil {
				t.Error("expected non-nil deserialized value")
			}
		})
	}
}

func TestLogListCodec(t *testing.T) {
	testCases := []struct {
		name      string
		inputYAML []string
	}{
		{
			name: "single log",
			inputYAML: []string{
				"textPayload: sample log message\nseverity: INFO",
			},
		},
		{
			name: "multiple logs with nested fields",
			inputYAML: []string{
				"protoPayload:\n  methodName: v1.createPod\n  resourceName: pods/nginx",
				"textPayload: second log\nseverity: ERROR",
			},
		},
	}

	codec := &LogListCodec{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var logs []*log.Log
			for _, y := range tc.inputYAML {
				l, err := log.NewLogFromYAMLString(y)
				if err != nil {
					t.Fatalf("failed to create log from yaml: %v", err)
				}
				logs = append(logs, l)
			}

			serialized, err := codec.Serialize(logs)
			if err != nil {
				t.Fatalf("failed to serialize logs: %v", err)
			}

			deserializedAny, err := codec.Deserialize(serialized)
			if err != nil {
				t.Fatalf("failed to deserialize logs: %v", err)
			}

			deserializedLogs, ok := deserializedAny.([]*log.Log)
			if !ok {
				t.Fatalf("expected []*log.Log, got %T", deserializedAny)
			}

			if len(deserializedLogs) != len(logs) {
				t.Fatalf("length mismatch: got %d, want %d", len(deserializedLogs), len(logs))
			}

			for i, origLog := range logs {
				deserializedLog := deserializedLogs[i]
				if origLog.Has(pathTextPayload) {
					want := origLog.ReadStringOrDefault(pathTextPayload, "")
					got := deserializedLog.ReadStringOrDefault(pathTextPayload, "")
					if diff := cmp.Diff(want, got); diff != "" {
						t.Errorf("textPayload[%d] mismatch (-want +got):\n%s", i, diff)
					}
				}
				if origLog.Has(pathProtoPayloadMethodName) {
					want := origLog.ReadStringOrDefault(pathProtoPayloadMethodName, "")
					got := deserializedLog.ReadStringOrDefault(pathProtoPayloadMethodName, "")
					if diff := cmp.Diff(want, got); diff != "" {
						t.Errorf("protoPayload.methodName[%d] mismatch (-want +got):\n%s", i, diff)
					}
				}
			}
		})
	}
}

func TestCodecRegistry_DeserializeForType(t *testing.T) {
	registry := NewCodecRegistry()

	testCases := []struct {
		name       string
		targetType reflect.Type
		jsonInput  string
		want       any
	}{
		{
			name:       "struct type",
			targetType: reflect.TypeOf(testCustomStruct{}),
			jsonInput:  `{"name":"sample","count":10}`,
			want: testCustomStruct{
				Name:  "sample",
				Count: 10,
			},
		},
		{
			name:       "string slice type",
			targetType: reflect.TypeOf([]string{}),
			jsonInput:  `["a","b","c"]`,
			want:       []string{"a", "b", "c"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := registry.DeserializeForType([]byte(tc.jsonInput), tc.targetType)
			if err != nil {
				t.Fatalf("failed to deserialize for type: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
