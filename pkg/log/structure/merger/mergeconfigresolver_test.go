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

package merger

import (
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestMergeConfigResolverWithoutParent(t *testing.T) {
	type resolverTestCase struct {
		name             string
		fieldPath        string
		expectedStrategy MergeArrayStrategy
		mergeKey         string
		mergeKeyError    bool
	}
	resolver := MergeConfigResolver{
		Parent: nil,
		MergeStrategies: map[string]MergeArrayStrategy{
			".merge":   MergeStrategyMerge,
			".replace": MergeStrategyReplace,
		},
		MergeKeys: map[string]string{
			".merge": "merge-key",
		},
	}
	resolverParent := MergeConfigResolver{
		Parent: nil,
		MergeStrategies: map[string]MergeArrayStrategy{
			".merge1":   MergeStrategyMerge,
			".merge2":   MergeStrategyMerge,
			".replace1": MergeStrategyReplace,
			".replace2": MergeStrategyReplace,
		},
		MergeKeys: map[string]string{
			".merge1": "merge-key-1-parent",
			".merge2": "merge-key-2-parent",
		},
	}
	resolverChild := MergeConfigResolver{
		Parent: &resolverParent,
		MergeStrategies: map[string]MergeArrayStrategy{
			".replace2": MergeStrategyMerge,
		},
		MergeKeys: map[string]string{
			".merge1":   "merge-key-1-child",
			".replace2": "replace-key-1",
		},
	}
	testCases := []struct {
		name     string
		resolver *MergeConfigResolver
		cases    []resolverTestCase
	}{
		{
			name:     "without parent config",
			resolver: &resolver,
			cases: []resolverTestCase{
				{
					name:             "merge strategy",
					fieldPath:        ".merge",
					expectedStrategy: MergeStrategyMerge,
					mergeKey:         "merge-key",
					mergeKeyError:    false,
				},
				{
					name:             "replace strategy",
					fieldPath:        ".replace",
					expectedStrategy: MergeStrategyReplace,
					mergeKey:         "",
					mergeKeyError:    true,
				},
				{
					name:             "unknown field",
					fieldPath:        ".unknown",
					expectedStrategy: MergeStrategyReplace, // unknown field must be merged as replace strategy
					mergeKey:         "",
					mergeKeyError:    true,
				},
			},
		},
		{
			name:     "with parent config",
			resolver: &resolverChild,
			cases: []resolverTestCase{
				{
					name:             "merge strategy overwriting with child",
					fieldPath:        ".merge1",
					expectedStrategy: MergeStrategyMerge,
					mergeKey:         "merge-key-1-child",
					mergeKeyError:    false,
				},
				{
					name:             "merge strategy non overwriting with child",
					fieldPath:        ".merge2",
					expectedStrategy: MergeStrategyMerge,
					mergeKey:         "merge-key-2-parent",
					mergeKeyError:    false,
				},
				{
					name:             "replace strategy not overwritten with child",
					fieldPath:        ".replace1",
					expectedStrategy: MergeStrategyReplace,
					mergeKey:         "",
					mergeKeyError:    true,
				},
				{
					name:             "replace strategy overwritten with child",
					fieldPath:        ".replace2",
					expectedStrategy: MergeStrategyMerge,
					mergeKey:         "replace-key-1",
					mergeKeyError:    false,
				},
				{
					name:             "unknown field",
					fieldPath:        ".unknown",
					expectedStrategy: MergeStrategyReplace, // unknown field must be merged as replace strategy
					mergeKey:         "",
					mergeKeyError:    true,
				},
			},
		}}
	for _, resolverCase := range testCases {
		for _, tc := range resolverCase.cases {
			resolver := resolverCase.resolver
			t.Run(tc.name, func(t *testing.T) {
				t.Run("GetMergeArrayStrategy", func(t *testing.T) {
					strategy := resolver.GetMergeArrayStrategy(tc.fieldPath)
					if strategy != tc.expectedStrategy {
						t.Errorf("expected %s, but %s was given", tc.expectedStrategy, strategy)
					}
				})
				t.Run("GetMergeKey", func(t *testing.T) {
					key, err := resolver.GetMergeKey(tc.fieldPath)
					if tc.mergeKeyError {
						if err == nil {
							t.Errorf("expected an error returned, but the error was nil")
						}
					} else {
						if err != nil {
							t.Errorf(err.Error())
						}
						if key != tc.mergeKey {
							t.Errorf("expected %s, but %s was given", tc.mergeKey, key)
						}
					}
				})
			})
		}
	}
}
