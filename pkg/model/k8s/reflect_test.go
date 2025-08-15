// Copyright 2024 Google LLC
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

package k8s

import (
	"fmt"
	"log/slog"
	"testing"

	corev1 "k8s.io/api/core/v1"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

func TestFromResourceTypeReflection(t *testing.T) {
	type TestCaseMapField struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	type InlineFields struct {
		InlineField1       string             `json:"inlineField1"`
		InlineArrayMerge   []TestCaseMapField `json:"inlineArrayMerge" patchStrategy:"merge" patchMergeKey:"name"`
		InlineArrayReplace []TestCaseMapField `json:"inlineArrayReplace"`
	}
	type testStructSecondLayer struct {
		Name   string             `json:"name"`
		Values []TestCaseMapField `json:"values" patchStrategy:"merge" patchMergeKey:"name"`
	}
	type testStructSecondLayerReplace struct {
		Name   string             `json:"name"`
		Values []TestCaseMapField `json:"values"`
	}
	type testStruct struct {
		Scalar         int                            `json:"scalar"`
		MapType        map[string]string              `json:"mapType"`
		PrimitiveArray []string                       `json:"primitive,omitempty" patchStrategy:"merge"`
		MergeWithName  []TestCaseMapField             `json:"mergename,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
		Replace        []TestCaseMapField             `json:"replacearray,omitempty"`
		MergeMerge     []testStructSecondLayer        `json:"mergemerge,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
		ReplaceMerge   []testStructSecondLayer        `json:"replacemerge,omitempty"`
		MergeReplace   []testStructSecondLayerReplace `json:"mergereplace,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
		ReplaceReplace []testStructSecondLayerReplace `json:"replacereplace,omitempty"`
		Inline         InlineFields                   `json:",inline"`
		PointerType    *testStructSecondLayer         `json:"pointerType"`
	}
	type recursiveStruct struct {
		Name      string            `json:"name"`
		Recursive []recursiveStruct `json:"recursive,omitempty" patchStategy:"merge" patchMergeKey:"name"`
	}
	type fieldTestCase struct {
		path     string
		strategy structured.MergeArrayStrategy
		mergeKey string
	}
	testCase := []struct {
		name                        string
		resourceType                interface{}
		fieldTestCases              []fieldTestCase
		wantErrorOnGenerateResolver bool
	}{
		{
			name:         "core v1 Pod can be registered",
			resourceType: corev1.Pod{},
		},
		{
			name:         "simple",
			resourceType: testStruct{},
			fieldTestCases: []fieldTestCase{
				{
					path:     "primitive",
					strategy: structured.MergeStrategyMerge,
					mergeKey: "",
				}, {
					path:     "mergename",
					strategy: structured.MergeStrategyMerge,
					mergeKey: "name",
				}, {
					path:     "replacearray",
					strategy: structured.MergeStrategyReplace,
					mergeKey: "",
				}, {
					path:     "mergemerge",
					strategy: structured.MergeStrategyMerge,
					mergeKey: "name",
				},
				{
					path:     "mergemerge.[].values",
					strategy: structured.MergeStrategyMerge,
					mergeKey: "name",
				}, {
					path:     "mergereplace",
					strategy: structured.MergeStrategyMerge,
					mergeKey: "name",
				},
				{
					path:     "mergereplace.[].values",
					strategy: structured.MergeStrategyReplace,
					mergeKey: "",
				},
				{
					path:     "replacemerge",
					strategy: structured.MergeStrategyReplace,
					mergeKey: "",
				},
				{
					path:     "replacemerge.[].values",
					strategy: structured.MergeStrategyMerge,
					mergeKey: "name",
				},
				{
					path:     "replacereplace",
					strategy: structured.MergeStrategyReplace,
					mergeKey: "",
				},
				{
					path:     "replacereplace.[].values",
					strategy: structured.MergeStrategyReplace,
					mergeKey: "",
				},
				{
					path:     "replacereplace.[].values",
					strategy: structured.MergeStrategyReplace,
					mergeKey: "",
				},
				{
					path:     "replacereplace.[].values",
					strategy: structured.MergeStrategyReplace,
					mergeKey: "",
				},
				{
					path:     "inlineArrayMerge",
					strategy: structured.MergeStrategyMerge,
					mergeKey: "name",
				},
				{
					path:     "inlineArrayReplace",
					strategy: structured.MergeStrategyReplace,
					mergeKey: "",
				},
				{
					path:     "pointerType.values",
					strategy: structured.MergeStrategyMerge,
					mergeKey: "name",
				},
			},
		},
		// Errornous
		{
			name:                        "recursive struct",
			resourceType:                recursiveStruct{},
			wantErrorOnGenerateResolver: true,
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			resolver, err := FromResourceTypeReflection(tc.resourceType)
			if tc.wantErrorOnGenerateResolver {
				if err == nil {
					t.Errorf("an error was expected but no error returned")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for key, strategy := range resolver.MergeStrategies {
				if strategy == structured.MergeStrategyReplace {
					slog.Info(fmt.Sprintf("%s -> %s\n", key, strategy))
				} else {
					mergeKey, err := resolver.GetMergeKey(key)
					if err != nil {
						t.Fatal(err)
					}
					slog.Info(fmt.Sprintf("%s -> %s (%s)\n", key, strategy, mergeKey))
				}
			}
			for _, field := range tc.fieldTestCases {
				t.Run(field.path, func(t *testing.T) {
					t.Run("GetMergeArrayStrategy", func(t *testing.T) {
						strategy := resolver.GetMergeArrayStrategy(field.path)
						if strategy != field.strategy {
							t.Errorf("expected %s, actual %s", field.strategy, strategy)
						}
					})
					t.Run("GetMergeKey", func(t *testing.T) {
						mergeKey, err := resolver.GetMergeKey(field.path)
						if field.strategy == structured.MergeStrategyReplace {
							if err == nil {
								t.Errorf("GetMergeKey in the array field with replace merge policy should return an error but no error returned")
							}
						} else {
							if err != nil {
								t.Fatal(err)
							}
							if mergeKey != field.mergeKey {
								t.Errorf("expected %s, actual %s", field.mergeKey, mergeKey)
							}
						}
					})
				})
			}
		})
	}
}
