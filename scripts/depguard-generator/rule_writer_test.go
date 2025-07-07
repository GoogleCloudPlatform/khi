// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestFileSystemRuleWriter_Write(t *testing.T) {
	testCases := []struct {
		name            string
		initialContent  string
		rulesToWrite    []*GeneratedRule
		expectedContent string
		expectError     bool
	}{
		{
			name:           "Create new file with one rule",
			initialContent: "",
			rulesToWrite: []*GeneratedRule{
				{
					RuleName:    "test-rule-1",
					TargetFiles: []string{"pkg/a/*"},
					DeniedPkgs:  []map[string]string{{"pkg": "pkg/b", "desc": "reason"}},
				},
			},
			expectedContent: `linters:
  settings:
    depguard:
      rules:
        test-rule-1:
          deny:
            - desc: reason
              pkg: pkg/b
          files:
            - pkg/a/*
`,
		},
		{
			name: "Add rules to existing file",
			initialContent: `
run:
  timeout: 5m
`,
			rulesToWrite: []*GeneratedRule{
				{
					RuleName:    "test-rule-2",
					TargetFiles: []string{"pkg/c/*"},
					DeniedPkgs:  []map[string]string{{"pkg": "pkg/d", "desc": "another reason"}},
				},
			},
			expectedContent: `run:
  timeout: 5m
linters:
  settings:
    depguard:
      rules:
        test-rule-2:
          deny:
            - desc: another reason
              pkg: pkg/d
          files:
            - pkg/c/*
`,
		},
		{
			name: "Overwrite existing rules",
			initialContent: `
linters-settings:
  depguard:
    rules:
      old-rule:
        files:
          - "old/path"
        deny:
          - pkg: "old/pkg"
            desc: "old reason"
`,
			rulesToWrite: []*GeneratedRule{
				{
					RuleName:    "new-rule",
					TargetFiles: []string{"new/path"},
					DeniedPkgs:  []map[string]string{{"pkg": "new/pkg", "desc": "new reason"}},
				},
			},
			expectedContent: `linters:
  settings:
    depguard:
      rules:
        new-rule:
          deny:
            - desc: new reason
              pkg: new/pkg
          files:
            - new/path
linters-settings:
  depguard:
    rules:
      old-rule:
        deny:
          - desc: old reason
            pkg: old/pkg
        files:
          - old/path
`,
		},
		{
			name:           "Write empty rules",
			initialContent: ``,
			rulesToWrite:   []*GeneratedRule{},
			expectedContent: `linters:
  settings:
    depguard:
      rules: {}
`,
		},
		{
			name:           "Create new file with allow rule",
			initialContent: "",
			rulesToWrite: []*GeneratedRule{
				{
					RuleName:    "test-rule-with-allow",
					TargetFiles: []string{"pkg/a/*"},
					DeniedPkgs:  []map[string]string{{"pkg": "pkg/b", "desc": "reason"}},
					AllowedPkgs: []string{"pkg/c", "pkg/d"},
				},
			},
			expectedContent: `linters:
  settings:
    depguard:
      rules:
        test-rule-with-allow:
          allow:
            - pkg/c
            - pkg/d
          deny:
            - desc: reason
              pkg: pkg/b
          files:
            - pkg/a/*
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, ".golangci-test.yaml")

			if tc.initialContent != "" {
				err := os.WriteFile(filePath, []byte(tc.initialContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write initial content: %v", err)
				}
			}

			writer := &FileSystemRuleWriter{Path: filePath}
			err := writer.Write(tc.rulesToWrite...)

			if (err != nil) != tc.expectError {
				t.Fatalf("Expected error: %v, got: %v", tc.expectError, err)
			}

			if !tc.expectError {
				fileBytes, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read file after write: %v", err)
				}

				var actual, expected interface{}
				if err := yaml.Unmarshal(fileBytes, &actual); err != nil {
					t.Fatalf("Failed to unmarshal actual content: %v", err)
				}
				if err := yaml.Unmarshal([]byte(tc.expectedContent), &expected); err != nil {
					t.Fatalf("Failed to unmarshal expected content: %v", err)
				}

				if diff := cmp.Diff(expected, actual); diff != "" {
					t.Errorf("Content mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
