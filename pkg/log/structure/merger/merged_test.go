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

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredata"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

var emptyMergeKeyResolver = &MergeConfigResolver{
	MergeKeys: map[string]string{},
}

func keyResolverFromMap(keys map[string]string) *MergeConfigResolver {
	strategy := map[string]MergeArrayStrategy{}
	for path := range keys {
		strategy[path] = MergeStrategyMerge
	}
	return &MergeConfigResolver{
		MergeKeys:       keys,
		MergeStrategies: strategy,
		Parent:          nil,
	}
}

func mustFromYaml(t *testing.T, yamlStr string) structuredata.StructureData {
	sd, err := structuredata.DataFromYaml(yamlStr)
	if err != nil {
		t.Fatal(err)
	}
	return sd
}

func mustToYaml(t *testing.T, st structuredata.StructureData) string {
	sd, err := structuredata.ToYaml(st)
	if err != nil {
		t.Fatal(err)
	}
	return sd
}

func TestMergedTypes(t *testing.T) {
	testCases := []struct {
		name         string
		prev         string
		patch        string
		expectedType structuredata.StructuredDataFieldType
	}{
		{
			name:         "with same type",
			prev:         `foo: 10`,
			patch:        `foo: 11`,
			expectedType: structuredata.StructuredTypeMap,
		},
		{
			name:         "patching with null",
			prev:         `10`,
			patch:        `~`,
			expectedType: structuredata.StructuredTypeScalar,
		},
		{
			name:         "null patched with actual value",
			prev:         `~`,
			patch:        `10`,
			expectedType: structuredata.StructuredTypeScalar,
		},
		{
			name:         "with $patch:delete attribute",
			prev:         `10`,
			patch:        `$patch: delete`,
			expectedType: structuredata.StructuredTypeScalar,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prev, err := structuredata.DataFromYaml(tc.prev)
			if err != nil {
				t.Fatal(err)
			}
			patch, err := structuredata.DataFromYaml(tc.patch)
			if err != nil {
				t.Fatal(err)
			}
			merged := NewStrategicMergedStructureData("", prev, patch, emptyMergeKeyResolver)
			ty, err := merged.Type()
			if err != nil {
				t.Fatal(err)
			}
			if ty != tc.expectedType {
				t.Errorf("returned data type is not matching with the expected type\nexpected:%s,actual:%s", tc.expectedType, ty)
			}
		})
	}
}

func TestMergedKeys(t *testing.T) {
	testCases := []struct {
		name         string
		prev         string
		patch        string
		childField   string
		expectedKeys []string
		keyResolver  *MergeConfigResolver
	}{
		{
			name: "basic map merge",
			prev: `foo: 10
bar: 12
`,
			patch: `foo: 11
qux: 13
`,
			expectedKeys: []string{"foo", "bar", "qux"},
		},
		{
			name: "map merge with multiple layers",
			prev: `foo:
  bar:
    qux:
      quux:
        apple: 1
    banana: 2
`,
			patch: `foo:
  bar:
    qux:
      quux:
        apple: 1
        grape: 2
`,
			expectedKeys: []string{"foo"},
		},
		{
			name:         "merge scalar value",
			prev:         `10`,
			patch:        `12`,
			expectedKeys: []string{""},
		},
		{
			name: "merge map with childPath",
			prev: `foo:
  bar: 10
  qux: 12
`,
			patch: `foo:
  bar: 10
  quux: 13`,
			childField:   "foo",
			expectedKeys: []string{"bar", "qux", "quux"},
		},
		{
			name: "merge array without mergekey",
			prev: `foo:
  - name: foo
  - name: bar
`,
			patch: `foo:
  - name: bar
  - name: qux`,
			childField:   "foo",
			expectedKeys: []string{"0", "1"},
		},
		{
			name: "merge array with mergekey",
			prev: `foo:
  - name: foo
  - name: bar
`,
			patch: `foo:
  - name: bar
  - name: qux`,
			childField:   "foo",
			expectedKeys: []string{"0", "1", "2"},
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "name",
			}),
		},
		{
			name: "merge array with mergekey and child properties",
			prev: `foo:
  - name: foo
    value: 0
  - name: bar
    value: 2
`,
			patch: `foo:
  - name: bar
    value: 3
  - name: qux
    value: 4`,
			childField:   "foo",
			expectedKeys: []string{"0", "1", "2"},
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "name",
			}),
		},
		{
			name: "merge primitive array without strategic merge directives and replace mode",
			prev: `foo:
  - 1
  - 2
`,
			patch: `foo:
  - 3
  - 4`,
			childField:   "foo",
			expectedKeys: []string{"0", "1"},
		},
		{
			name: "merge primitive array without strategic merge directives in merge mode",
			prev: `foo:
  - 1
  - 2
`,
			patch: `foo:
  - 3
  - 4`,
			childField:   "foo",
			expectedKeys: []string{"0", "1", "2", "3"},
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array without strategic merge directives in merge mode with duplicated elements",
			prev: `foo:
  - 1
  - 2
`,
			patch: `foo:
  - 2
  - 3`,
			childField:   "foo",
			expectedKeys: []string{"0", "1", "2"},
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge map with $patch=replace",
			prev: `foo:
  bar: 10
  qux: 12
`,
			patch: `foo:
  bar: 10
  quux: 13
  $patch: replace`,
			childField:   "foo",
			expectedKeys: []string{"bar", "quux"},
		},
		{
			name: "merge array with $patch=replace",
			prev: `- name: foo
- name: bar
- name: qux
`,
			patch: `- name: foo
- name: bar2
- name: quux
- $patch: replace
`,
			expectedKeys: []string{"0", "1", "2"},
			keyResolver: keyResolverFromMap(map[string]string{
				"": "name",
			}),
		},
		{
			name: "merge array with $patch=replace in middle",
			prev: `- name: foo
- name: bar
- name: qux
`,
			patch: `- name: foo
- name: bar2
- $patch: replace
- name: quux
`,
			expectedKeys: []string{"0", "1", "2"},
			keyResolver: keyResolverFromMap(map[string]string{
				"": "name",
			}),
		},
		{
			name: "merge array with $patch=delete",
			prev: `- name: foo
- name: bar
- name: qux
`,
			patch: `- name: foo
- name: bar
  $patch: delete
- name: qux
`,
			expectedKeys: []string{"0", "1"},
			keyResolver: keyResolverFromMap(map[string]string{
				"": "name",
			}),
		},
		{
			name: "merge array with $deleteFromPrimitiveList with patching value",
			prev: `foo:
  - a
  - b
  - c`,
			patch: `foo:
  - b
$deleteFromPrimitiveList/foo:
  - a
  - c
`,
			expectedKeys: []string{"0"},
			childField:   "foo",
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array with $deleteFromPrimitiveList without patching value",
			prev: `foo:
  - a
  - b
  - c`,
			patch: `
$deleteFromPrimitiveList/foo:
  - a
  - c
`,
			expectedKeys: []string{"0"},
			childField:   "foo",
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array with $deleteFromPrimitiveList without patching value in int values",
			prev: `foo:
  - 1
  - 2
  - 3`,
			patch: `
$deleteFromPrimitiveList/foo:
  - 2
  - 3
`,
			expectedKeys: []string{"0"},
			childField:   "foo",
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array with $setElementOrder without patching value",
			prev: `foo:
  - a
  - b
  - c`,
			patch: `
$setElementOrder/foo:
  - c
  - b
  - a
`,
			expectedKeys: []string{"0", "1", "2"},
			childField:   "foo",
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array with $setElementOrder with patching value",
			prev: `foo:
  - a
  - b
  - c`,
			patch: `foo:
  - d
$setElementOrder/foo:
  - c
  - b
  - a
  - d

`,
			expectedKeys: []string{"0", "1", "2", "3"},
			childField:   "foo",
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array of array (with replace policy)",
			prev: `foo:
  - ['apple','banana']
  - ['grape','pinapple']
`,
			patch: `foo:
  - ['fish','meat']
`,
			expectedKeys: []string{"0"},
			childField:   "foo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prev, err := structuredata.DataFromYaml(tc.prev)
			if err != nil {
				t.Fatal(err)
			}
			patch, err := structuredata.DataFromYaml(tc.patch)
			if err != nil {
				t.Fatal(err)
			}
			keyResolver := emptyMergeKeyResolver
			if tc.keyResolver != nil {
				keyResolver = tc.keyResolver
			}
			merged := (structuredata.StructureData)(NewStrategicMergedStructureData("", prev, patch, keyResolver))
			if tc.childField != "" {
				child, err := merged.Value(tc.childField)
				if err != nil {
					t.Fatal(err)
				}
				childStructured, convertible := child.(structuredata.StructureData)
				if !convertible {
					t.Errorf("child node %s can't be converted to structuredata.StructureData", tc.childField)
				}
				merged = childStructured
			}
			ty, err := merged.Keys()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expectedKeys, ty); diff != "" {
				t.Errorf("returned key array is not matching with the expected array\n%s\n%s", diff, mustToYaml(t, merged))
			}
		})
	}
}

func TestMergedValues(t *testing.T) {
	testCases := []struct {
		name        string
		prev        string
		patch       string
		expected    string
		keyResolver *MergeConfigResolver
	}{
		{
			name:     "merge scalar",
			prev:     "0",
			patch:    "10",
			expected: `10`,
		},
		{
			name: "merge map",
			prev: `foo:
  bar: 10
`,
			patch: `foo:
  qux: 11
`,
			expected: `foo:
  bar: 10
  qux: 11
`,
		},
		{
			name: "",
			prev: `foo:
  bar: abcdefg
qux:
  hoge: abc
  fuga:
    - a
    - b
    - c`,
			patch: `foo:
  bar: abcdefg
qux:
  hoge: null
  fuga: null`,
			expected: `foo:
  bar: abcdefg
qux:
  hoge: null
  fuga: null`,
		},
		{
			name: "map merge with multiple layers",
			prev: `foo:
  bar:
    qux:
      quux:
        apple: 1
    banana: 2
`,
			patch: `foo:
  bar:
    qux:
      quux:
        apple: 2
        grape: 3
`,
			expected: `foo:
  bar:
    qux:
      quux:
        apple: 2
        grape: 3
    banana: 2`,
		},
		{
			name: "merge map with $patch=replace",
			prev: `foo:
  bar: 10
`,
			patch: `foo:
  $patch: replace
  qux: 11
`,
			expected: `foo:
  qux: 11
`,
		},
		{
			name: "merge map with $patch=merge",
			prev: `foo:
  bar: 10
`,
			patch: `foo:
  $patch: merge
  qux: 11
`,
			expected: `foo:
  bar: 10
  qux: 11
`,
		},
		{
			name: "merge map with $patch=delete",
			prev: `foo:
  bar: 10
`,
			patch: `foo:
  $patch: delete
  qux: 11
`,
			expected: `foo: ~`,
		},
		{
			name: "merge map with $retainKeys",
			prev: `foo:
  bar: 10
`,
			patch: `foo:
  $retainKeys:
    - qux
  qux: 11
`,
			expected: `foo:
  qux: 11
`,
		},
		{
			name: "merge map with $retainKeys to keep prev value",
			prev: `foo:
  bar: 10
`,
			patch: `foo:
  $retainKeys:
    - bar
    - qux
  qux: 11
`,
			expected: `foo:
  bar: 10
  qux: 11
`,
		},
		{
			name: "merge primitive array without strategic merge directives",
			prev: `foo:
  - 1
  - 2
`,
			patch: `foo:
  - 3
  - 4`,
			expected: `foo:
  - 1
  - 2
  - 3
  - 4`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge primitive array without strategic merge directives",
			prev: `foo:
  - 1
  - 2
`,
			patch: `foo:
  - 2
  - 3`,
			expected: `foo:
 - 1
 - 2
 - 3
 `,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array with replace strategy",
			prev: `foo:
  - name: apple
  - name: banana
`,
			patch: `foo:
  - name: apple
  - name: grape
`,
			expected: `foo:
  - name: apple
  - name: grape
`,
		},
		{
			name: "merge array with merge strategy(prioritize the patch order)",
			prev: `foo:
  - name: apple
  - name: banana
`,
			patch: `foo:
  - name: apple
  - name: grape
`,
			expected: `foo:
  - name: banana
  - name: apple
  - name: grape
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "name",
			}),
		},
		{
			name: "merge array with merge strategy",
			prev: `foo:
  - name: apple
  - name: banana
`,
			patch: `foo:
  - name: grape
  - name: apple
`,
			expected: `foo:
  - name: banana
  - name: grape
  - name: apple
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "name",
			}),
		},
		{
			name: "merge array with $patch=replace",
			prev: `foo:
  - name: apple
  - name: banana
`,
			patch: `foo:
  - name: apple
  - name: grape
  - $patch: replace
`,
			expected: `foo:
  - name: apple
  - name: grape
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "name",
			}),
		},
		{
			name: "merge array with $patch=delete",
			prev: `foo:
  - name: apple
  - name: banana
`,
			patch: `foo:
  - name: pinapple
  - name: grape
  - $patch: delete
    name: banana
`,
			expected: `foo:
  - name: apple
  - name: pinapple
  - name: grape
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "name",
			}),
		},
		{
			name: "merge array with $deleteFromPrimitiveList in merge mode",
			prev: `foo:
  - apple
  - banana
  - grape
  - pinapple
`,
			patch: `$deleteFromPrimitiveList/foo:
  - apple
  - grape
`,
			expected: `foo:
  - banana
  - pinapple
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array with $deleteFromPrimitiveList for non existing values",
			prev: `foo:
  - apple
  - banana
  - grape
  - pinapple
`,
			patch: `$deleteFromPrimitiveList/foo:
  - fish
  - meat
`,
			expected: `foo:
- apple
- banana
- grape
- pinapple
`,
		},
		{
			name: "merge array with $setElementOrder for primitives",
			prev: `foo:
  - apple
  - banana
  - grape
  - pinapple
`,
			patch: `$setElementOrder/foo:
  - apple
  - grape
  - pinapple
  - banana
`,
			expected: `foo:
  - apple
  - grape
  - pinapple
  - banana
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array with non-null value over the nil value previous",
			prev: `foo: ~`,
			patch: `foo:
  - apple
  - grape
  - pinapple
  - banana
`,
			expected: `foo:
  - apple
  - grape
  - pinapple
  - banana
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array with non-null value on struct with nil field",
			prev: `foo: ~`,
			patch: `bar:
  - apple
  - grape
  - pinapple
  - banana
`,
			expected: `foo: ~
bar:
  - apple
  - grape
  - pinapple
  - banana
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge object with non-null value over the nil value previous",
			prev: `foo: ~`,
			patch: `foo:
  apple : 0
  banana : 1
  grape : 2
`,
			expected: `foo:
  apple : 0
  banana : 1
  grape : 2
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "",
			}),
		},
		{
			name: "merge array with $setElementOrder for non primitives",
			prev: `foo:
- name: apple
  index: 0
- name: banana
  index: 1
- name: grape
  index: 2
- name: pinapple
  index: 3
`,
			patch: `$setElementOrder/foo:
- name: apple
- name: grape
- name: pinapple
- name: banana
`,
			expected: `foo:
- name: apple
  index: 0
- name: grape
  index: 2
- name: pinapple
  index: 3
- name: banana
  index: 1
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "name",
			}),
		},
		{
			name: "merge array with $setElementOrder for non primitives and missing prev fields",
			prev: `foo:
- name: apple
  index: 0
- name: banana
  index: 1
- name: grape
  index: 2
- name: pinapple
  index: 3
`,
			patch: `$setElementOrder/foo:
- name: apple
- name: grape
- name: pinapple
- name: banana
- name: fish
`,
			expected: `foo:
- name: apple
  index: 0
- name: grape
  index: 2
- name: pinapple
  index: 3
- name: banana
  index: 1
- name: fish
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo": "name",
			}),
		},
		{
			name: "mix $deleteFromPrimitiveList and $setElementOrder",
			prev: `foo:
  bar:
    baz1: hello
    baz2: true
  qux:
    quux:
      - apple
      - banana
      - grape
`,
			patch: `foo:
  bar:
    baz2: false
    baz1: hello2
  qux:
    $deleteFromPrimitiveList/quux:
      - banana
    $setElementOrder/quux:
      - grape
      - lemon
    quux:
    - lemon
`,
			expected: `foo:
  bar:
    baz1: hello2
    baz2: false
  qux:
    quux:
    - apple
    - grape
    - lemon
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo.qux.quux": "",
			}),
		},
		{
			name: "mix $patch=delete and $setElementOrder",
			prev: `foo:
  bar:
    baz1: hello
    baz2: true
  qux:
    quux:
      - name: apple
        value: 1
        value2: 10
      - name: banana
        value: 2
        value2: 20
      - name: grape
        value: 3
        value2: 30
`,
			patch: `foo:
  bar:
    baz2: false
    baz1: hello2
  qux:
    $setElementOrder/quux:
      - name: grape
      - name: lemon
    quux:
    - name: banana
      $patch: delete
    - name: grape
      value: 4
    - name: lemon
      value: 4
      value2: 40
`,
			expected: `foo:
  bar:
    baz1: hello2
    baz2: false
  qux:
    quux:
    - name: apple
      value: 1
      value2: 10
    - name: grape
      value: 4
      value2: 30
    - name: lemon
      value: 4
      value2: 40
`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo.qux.quux": "name",
			}),
		},
		{
			name: "merge array of array (with replace policy)",
			prev: `foo:
  - ['apple','banana']
  - ['grape','pinapple']
`,
			patch: `foo:
  - ['fish','meat']
`,
			expected: `foo:
  - ['fish','meat']
`,
		},
		{
			name: "merge array of an object in array",
			prev: `foo:
- name: apple
  values:
    - name: fish
      value: 1
    - name: meat
      value: 2
- name: banana
  values:
    - name: dog
      value: 3
    - name: cat
      value: 4
`,
			patch: `foo:
- name: banana
  $setElementOrder/values:
    - name: dog
    - name: cat
    - name: unicorn
  values:
    - name: cat
      value: 5
    - name: unicorn
      value: 6
- name: pinapple
  values:
    - name: moon
      value: 7
    - name: sun
      value: 8
`,
			expected: `foo:
  - name: apple
    values:
      - name: fish
        value: 1
      - name: meat
        value: 2
  - name: banana
    values:
      - name: dog
        value: 3
      - name: cat
        value: 5
      - name: unicorn
        value: 6
  - name: pinapple
    values:
      - name: moon
        value: 7
      - name: sun
        value: 8`,
			keyResolver: keyResolverFromMap(map[string]string{
				".foo":          "name",
				".foo[].values": "name",
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prev, err := structuredata.DataFromYaml(tc.prev)
			if err != nil {
				t.Fatal(err)
			}
			patch, err := structuredata.DataFromYaml(tc.patch)
			if err != nil {
				t.Fatal(err)
			}
			keyResolver := emptyMergeKeyResolver
			if tc.keyResolver != nil {
				keyResolver = tc.keyResolver
			}

			merged := NewStrategicMergedStructureData("", prev, patch, keyResolver)
			expected := mustFromYaml(t, tc.expected)

			eq, err := structuredata.EqualStructureData(merged, expected)
			if err != nil {
				t.Fatal(err)
			}
			if !eq {
				t.Errorf("result is not matching with the expected result\nactual:\n%s\n\nexpected:\n%s", mustToYaml(t, merged), mustToYaml(t, expected))
			}
		})
	}
}

func TestSplitPatchKeysByFieldsOrMergeAttributes(t *testing.T) {
	testCases := []struct {
		name      string
		patchKeys []string
		expected  *strategicMergePatchKeys
	}{
		{
			name: "basic case",
			patchKeys: []string{
				"name",
				"image",
				"patch",
				"$patch",
				"$retainKeys",
				"$unknownAttribute",
				"$setElementOrder/containers",
				"$deleteFromPrimitiveList/command",
			},
			expected: &strategicMergePatchKeys{
				FieldKeys: []string{
					"name",
					"image",
					"patch",
					"$unknownAttribute",
				},
				StrategicMergeAttibutes: []string{
					"$patch",
					"$retainKeys",
					"$setElementOrder/containers",
					"$deleteFromPrimitiveList/command",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := splitPatchKeysByFieldsOrMergeAttributes(tc.patchKeys)
			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Errorf("Unexpected result (-expected +actual):\n%s", diff)
			}
		})
	}
}

func TestRemoveSetElements(t *testing.T) {
	type testCase struct {
		name     string
		list     []any
		patch    []any
		remove   []any
		expected []any
	}

	testCases := []testCase{
		{
			name:     "Empty lists",
			list:     []any{},
			patch:    []any{},
			remove:   []any{},
			expected: []any{},
		},
		{
			name:     "Remove an element",
			list:     []any{"apple", "banana"},
			patch:    []any{},
			remove:   []any{"banana"},
			expected: []any{"apple"},
		},
		{
			name:     "Remove an element in int",
			list:     []any{1, 2, 3, 4},
			patch:    []any{5, 6, 7, 8},
			remove:   []any{2, 3, 6, 7},
			expected: []any{1, 4, 5, 8},
		},
		{
			name:     "Add an element via patch",
			list:     []any{"apple"},
			patch:    []any{"banana"},
			remove:   []any{},
			expected: []any{"apple", "banana"},
		},
		{
			name:     "Multiple operations",
			list:     []any{"apple", "banana", "orange"},
			patch:    []any{"grape"},
			remove:   []any{"banana"},
			expected: []any{"apple", "orange", "grape"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := removeSetElements(tc.list, tc.patch, tc.remove)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf("removeSetElements() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReorderKeys(t *testing.T) {
	testCases := []struct {
		name                     string
		prev                     []string
		patch                    []string
		setElementOrderDirective []string
		expectedOrder            []string
	}{
		{
			name:                     "without the directive 1",
			prev:                     []string{"apple", "banana"},
			patch:                    []string{"banana", "pinapple"},
			setElementOrderDirective: []string{},
			expectedOrder:            []string{"apple", "banana", "pinapple"},
		},
		{
			name:                     "without the directive 2",
			prev:                     []string{"apple", "banana"},
			patch:                    []string{"pinapple", "banana"},
			setElementOrderDirective: []string{},
			expectedOrder:            []string{"apple", "pinapple", "banana"},
		},
		{
			name:                     "without the directive 3",
			prev:                     []string{"banana", "apple"},
			patch:                    []string{"pinapple", "banana"},
			setElementOrderDirective: []string{},
			expectedOrder:            []string{"apple", "pinapple", "banana"},
		},
		{
			name:                     "with partial setElementOrder directive",
			prev:                     []string{"prev", "both"},
			patch:                    []string{"both", "patch"},
			setElementOrderDirective: []string{"both", "prev"},
			expectedOrder:            []string{"both", "prev", "patch"},
		},
		{
			name:                     "with full setElementOrder directive",
			prev:                     []string{"prev", "both", "prev2"},
			patch:                    []string{"both", "patch"},
			setElementOrderDirective: []string{"prev2", "both", "prev", "patch"},
			expectedOrder:            []string{"prev2", "both", "prev", "patch"},
		},
		{
			// Assume the element missing in prev was partial list of the actual live list in the server.
			name:                     "with elements only contained in setElementOrder directive",
			prev:                     []string{"prev", "both"},
			patch:                    []string{"both", "patch"},
			setElementOrderDirective: []string{"both", "new", "prev", "patch"},
			expectedOrder:            []string{"both", "new", "prev", "patch"},
		},
		{
			name:                     "with elements only contained in setElementOrder directive 2",
			prev:                     []string{"prev", "both"},
			patch:                    []string{"both", "patch"},
			setElementOrderDirective: []string{"both", "new"},
			expectedOrder:            []string{"prev", "both", "new", "patch"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := reorderArrayKeysForMerge(tc.prev, tc.patch, tc.setElementOrderDirective)
			if diff := cmp.Diff(tc.expectedOrder, result); diff != "" {
				t.Errorf("generated order is not matching with the expected result\n%s\n\nexpected:\n%v\nactual:\n%v", diff, tc.expectedOrder, result)
			}
		})
	}
}
