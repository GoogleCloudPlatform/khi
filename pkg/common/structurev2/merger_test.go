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
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/merger"
	"github.com/google/go-cmp/cmp"
)

func TestMergeNode(t *testing.T) {
	testCases := []struct {
		Name               string
		Prev               string
		Patch              string
		Expected           string
		ArrayMergeStrategy *merger.MergeConfigResolver
	}{
		{
			Name: "merge to update a field",
			Prev: `foo: bar
qux: 2
`,
			Patch: `foo: baz
`,
			Expected: `foo: baz
qux: 2
`,
		},
		{
			Name: "merge to add a field",
			Prev: `foo: bar
qux: 2
`,
			Patch: `quux: 3
`,
			Expected: `foo: bar
qux: 2
quux: 3
`,
		},
		{
			Name: "merge sequence of sequence should be handled with replace strategy",
			Prev: `foo:
- - 1
  - 2
- - 3
- - 4
`,
			Patch: `foo:
- - 1
  - 2
  - 3
- - 4
`,
			Expected: `foo:
    - - 1
      - 2
      - 3
    - - 4
`,
		},
		{
			Name: "$patch:replace directive at root",
			Prev: `foo: bar
qux: 2
`,
			Patch: `$patch: replace
quux: 3
`,
			Expected: `quux: 3
`,
		},
		{
			Name: "$patch:replace directive in child map",
			Prev: `foo:
  bar: baz
  qux: 
    quux: 42
`,
			Patch: `foo:
  bar: baz2
  qux:
    $patch: replace
    quux2: 42
`,
			Expected: `foo:
    bar: baz2
    qux:
        quux2: 42
`,
		},
		{
			Name: "$patch:delete directive in map",
			Prev: `foo:
  bar: baz
  qux: 
    quux: 42
`,
			Patch: `foo:
  bar: baz2
  qux:
    $patch: delete
`,
			Expected: `foo:
    bar: baz2
`,
		},
		{
			Name: "$patch: merge must be ignored",
			Prev: `foo:
  bar: baz
`,
			Patch: `foo:
  bar: baz2
  $patch: merge
`,
			Expected: `foo:
    bar: baz2
`,
		},
		{
			Name: "$deleteFromPrimitiveList on a string array",
			Prev: `foo:
- a
- b
- c
`,
			Patch: `$deleteFromPrimitiveList/foo:
  - a
  - b
`,
			Expected: `foo:
    - c
`,
		},
		{
			Name: "$deleteFromPrimitiveList on an int array",
			Prev: `foo:
- 1
- 2
- 3
`,
			Patch: `$deleteFromPrimitiveList/foo:
  - 1
  - 3
`,
			Expected: `foo:
    - 2
`,
		},
		{
			Name: "$retainKeys on a map",
			Prev: `foo:
  apple: 1
  banana: 2
  grape: 3
`,
			Patch: `$retainKeys/foo:
  - apple
  - grape
`,
			Expected: `foo:
    apple: 1
    grape: 3
`,
		},
		{
			Name: "$retainKeys on a map with int key",
			Prev: `foo:
  1: apple
  2: banana
  3: grape
`,
			Patch: `$retainKeys/foo:
  - 1
  - 3
`,
			Expected: `foo:
    "1": apple
    "3": grape
`,
		},
		{
			Name: "$setElementOrder with primitives",
			Prev: `foo:
- a
- b
- c
`,
			Patch: `$setElementOrder/foo:
  - c
  - b
  - a
`,
			Expected: `foo:
    - c
    - b
    - a
`,
		},
		{
			Name: "merge sequence of map with merge strategy without $setElementOrder directive",
			Prev: `foo:
- key: apple
  value: 1
- key: banana
  value: 2
`,
			Patch: `foo:
- key: grape
  value: 3
- key: banana
  value: 4
`,
			Expected: `foo:
    - key: apple
      value: 1
    - key: grape
      value: 3
    - key: banana
      value: 4
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"foo": "key",
				},
			},
		},
		{
			Name: "merge sequence of map with merge strategy and int item key without $setElementOrder directive",
			Prev: `foo:
- key: 1
  value: apple
- key: 2
  value: banana
`,
			Patch: `foo:
- key: 3
  value: grape
- key: 1
  value: pinapple
`,
			Expected: `foo:
    - key: 2
      value: banana
    - key: 3
      value: grape
    - key: 1
      value: pinapple
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"foo": "key",
				},
			},
		},
		{
			Name: "$setElementOrder with maps",
			Prev: `foo:
- key: apple
  value: 1
- key: banana
  value: 2
- key: grape
  value: 3
`,
			Patch: `$setElementOrder/foo:
  - key: grape
  - key: banana
  - key: apple
`,
			Expected: `foo:
    - key: grape
      value: 3
    - key: banana
      value: 2
    - key: apple
      value: 1
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"foo": "key",
				},
			},
		},
		{
			Name: "$setElementOrder with maps and int item key",
			Prev: `foo:
- key: 1
  value: apple
- key: 2
  value: banana
- key: 3
  value: grape
`,
			Patch: `$setElementOrder/foo:
  - key: 2
  - key: 3
  - key: 1
`,
			Expected: `foo:
    - key: 2
      value: banana
    - key: 3
      value: grape
    - key: 1
      value: apple
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"foo": "key",
				},
			},
		},
		{
			Name: "$setElementOrder item not found in the prev map should be complemented",
			Prev: `foo:
- key: apple
  value: 1
- key: banana
  value: 2
`,
			Patch: `$setElementOrder/foo:
  - key: grape
  - key: banana
  - key: apple
`,
			Expected: `foo:
    - key: grape
    - key: banana
      value: 2
    - key: apple
      value: 1
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"foo": "key",
				},
			},
		},
		{
			Name: "$setElementOrder item not found in the prev map should be complemented for int item eky",
			Prev: `foo:
- key: 2
  value: apple
- key: 3
  value: banana
`,
			Patch: `$setElementOrder/foo:
  - key: 1
  - key: 2
  - key: 3
`,
			Expected: `foo:
    - key: 1
    - key: 2
      value: apple
    - key: 3
      value: banana
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"foo": "key",
				},
			},
		},
		{
			Name: "merge array items with $setElementOrder and patching",
			Prev: `foo:
- key: apple
  value: 1
- key: banana
  value: 2"
- key: grape
  value: 3
`,
			Patch: `$setElementOrder/foo:
  - key: grape
  - key: banana
  - key: apple
foo:
  - key: banana
    value: 4
`,
			Expected: `foo:
    - key: grape
      value: 3
    - key: banana
      value: 4
    - key: apple
      value: 1
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"foo": "key",
				},
			},
		},
		{
			Name: "merge sequence of map with replace strategy",
			Prev: `foo:
- key: apple
  value: 1
- key: banana
  value: 2
`,
			Patch: `foo:
- key: grape
  value: 3
- key: banana
  value: 4
`,
			Expected: `foo:
    - key: grape
      value: 3
    - key: banana
      value: 4
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyReplace,
				},
				MergeKeys: map[string]string{},
			},
		},
		{
			Name: "merge sequence of map with replace strategy but shouldn't delete when the patch not containing the previous array field",
			Prev: `foo:
- key: apple
  value: 1
- key: banana
  value: 2
`,
			Patch: `bar: qux
`,
			Expected: `foo:
    - key: apple
      value: 1
    - key: banana
      value: 2
bar: qux
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyReplace,
				},
				MergeKeys: map[string]string{},
			},
		},
		{
			Name: "merge sequence of map with nil patch with replace strategy",
			Prev: `- key: apple
  value: 1
- key: banana
  value: 2
`,
			Patch: "",
			Expected: `- key: apple
  value: 1
- key: banana
  value: 2
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"": merger.MergeStrategyReplace,
				},
			},
		},
		{
			Name: "merge sequence of map with nil prev with replace strategy",
			Prev: ``,
			Patch: `- key: apple
  value: 1
- key: banana
  value: 2`,
			Expected: `- key: apple
  value: 1
- key: banana
  value: 2
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"": merger.MergeStrategyReplace,
				},
			},
		},
		{
			Name: "merge sequence of map with nil patch with merge strategy",
			Prev: `- key: apple
  value: 1
- key: banana
  value: 2
`,
			Patch: "",
			Expected: `- key: apple
  value: 1
- key: banana
  value: 2
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"": "key",
				},
			},
		},
		{
			Name: "merge sequence of map with nil prev with merge strategy",
			Prev: ``,
			Patch: `- key: apple
  value: 1
- key: banana
  value: 2`,
			Expected: `- key: apple
  value: 1
- key: banana
  value: 2
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"": "key",
				},
			},
		},
		{
			Name: "merge sequence of map with $setElementOrder but the array is not found in prev and patch",
			Prev: ``,
			Patch: `$setElementOrder/foo:
  - key: apple
  - key: banana
`,
			Expected: `foo:
    - key: apple
    - key: banana
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"foo": "key",
				},
			},
		},
		{
			Name: "merge sequence of map with $setElementOrder and int key but the array is not found in prev and patch",
			Prev: ``,
			Patch: `$setElementOrder/foo:
  - key: 1
  - key: 2
`,
			Expected: `foo:
    - key: 1
    - key: 2
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"foo": "key",
				},
			},
		},
		{
			Name: "merge sequence of primitive with $setElementOrder but the array is not found in prev and patch",
			Prev: ``,
			Patch: `$setElementOrder/foo:
  - apple
  - banana
`,
			Expected: `foo:
    - apple
    - banana
`,
			ArrayMergeStrategy: &merger.MergeConfigResolver{
				MergeStrategies: map[string]merger.MergeArrayStrategy{
					"foo": merger.MergeStrategyMerge,
				},
				MergeKeys: map[string]string{
					"foo": "",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var prevNode, patchNode Node
			var err error
			if tc.Prev != "" {
				prevNode, err = FromYAML(tc.Prev)
				if err != nil {
					t.Fatalf("failed to parse prev node yaml %v", err)
				}
			}
			if tc.Patch != "" {
				patchNode, err = FromYAML(tc.Patch)
				if err != nil {
					t.Fatalf("failed to parse patch node yaml %v", err)
				}
			}
			got, err := MergeNode(prevNode, patchNode, MergeConfiguration{MergeMapOrderStrategy: &DefaultMergeMapOrderStrategy{}, ArrayMergeConfigResolver: tc.ArrayMergeStrategy})
			if err != nil {
				t.Fatalf("failed to merge nodes %v", err)
			}
			reader := NewNodeReader(got)

			yamlMergedResult, err := reader.Serialize("", &YAMLNodeSerializer{})
			if err != nil {
				t.Fatalf("failed to serialize result to yaml %v", err)
			}
			yamlMergeResultStr := string(yamlMergedResult)
			if diff := cmp.Diff(tc.Expected, yamlMergeResultStr); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}

			// all nodes must be instanciated as new node after the merge
			t.Run("all Nodes shoulnd't share its instances", func(t *testing.T) {
				foundAddress := map[string]struct{}{}
				verifyTarget := []Node{prevNode, patchNode, got}
				for _, rootNode := range verifyTarget {
					if rootNode == nil {
						continue
					}
					for n := range walkAllNodes(rootNode) {
						addressStr := fmt.Sprintf("%p", n)
						if _, found := foundAddress[addressStr]; found {
							t.Errorf("duplicated node instance was found")
						}
					}
				}
			})
		})
	}
}

// walkAllNodes return the iterator for node listing all descendants and itself.
func walkAllNodes(node Node) func(func(n Node) bool) {
	return func(yield func(n Node) bool) {
		if !yield(node) {
			return
		}
		for _, child := range node.Children() {
			walkAllNodes(child)(yield)
		}
	}
}
