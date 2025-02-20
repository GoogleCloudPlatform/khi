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

package testlog

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestStringField(t *testing.T) {
	testCases := []struct {
		name        string
		opts        []TestLogOpt
		outputYaml  string
		expectError bool
	}{
		{
			name: "rewriting root level field",
			opts: []TestLogOpt{
				BaseYaml(`foo: bar
qux: quux`),
				StringField("qux", "quux2"),
			},
			outputYaml: `foo: bar
qux: quux2
`,
			expectError: false,
		},
		{
			name: "rewriting non existing root level field",
			opts: []TestLogOpt{
				BaseYaml(`foo: bar`),
				StringField("qux", "quux"),
			},
			outputYaml: `foo: bar
qux: quux
`,
			expectError: false,
		},
		{
			name: "rewriting deeper level field",
			opts: []TestLogOpt{
				BaseYaml(`foo:
  bar: quux`),
				StringField("foo.bar", "quux2"),
			},
			outputYaml: `foo:
  bar: quux2
`,
			expectError: false,
		},
		{
			name: "rewriting non existing deeper level field",
			opts: []TestLogOpt{
				BaseYaml(`qux: foo`),
				StringField("foo.bar", "quux2"),
			},
			outputYaml: `qux: foo
foo:
  bar: quux2
`,
			expectError: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tl := New(tc.opts...)
			reader, err := tl.BuildReader()
			if tc.expectError {
				if err == nil {
					t.Errorf("Expecting an error but no error returned.")
				}
			} else {
				if err != nil {
					t.Fatal(err.Error())
				}
				yamlStr, err := reader.ToYaml("")
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if diff := cmp.Diff(yamlStr, tc.outputYaml); diff != "" {
					t.Errorf("Yaml string mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
