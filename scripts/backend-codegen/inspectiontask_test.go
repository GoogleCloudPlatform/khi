// Copyright 2026 Google LLC
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

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasModuleDeclaration(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  bool
	}{
		{
			name: "single var declaration",
			files: map[string]string{
				"module.go": "package impl\n\nvar Module = struct{}{}\n",
			},
			want: true,
		},
		{
			name: "var declared in a group",
			files: map[string]string{
				"module.go": "package impl\n\nvar (\n\tother = 1\n\tModule = struct{}{}\n)\n",
			},
			want: true,
		},
		{
			name: "only Register function",
			files: map[string]string{
				"registration.go": "package impl\n\nfunc Register() error { return nil }\n",
			},
			want: false,
		},
		{
			name: "Module declared as a function or a constant",
			files: map[string]string{
				"module.go": "package impl\n\nconst Module = 1\n\nfunc init() {}\n",
				"func.go":   "package impl\n\ntype T struct{}\n\nfunc (T) Module() {}\n",
			},
			want: false,
		},
		{
			name: "unexported module variable",
			files: map[string]string{
				"module.go": "package impl\n\nvar module = struct{}{}\n",
			},
			want: false,
		},
		{
			name: "Module declared only in a test file",
			files: map[string]string{
				"registration.go": "package impl\n\nfunc Register() error { return nil }\n",
				"module_test.go":  "package impl\n\nvar Module = struct{}{}\n",
			},
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
					t.Fatalf("failed to write %s: %v", name, err)
				}
			}
			got, err := hasModuleDeclaration(dir)
			if err != nil {
				t.Fatalf("hasModuleDeclaration() returned unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("hasModuleDeclaration() = %t, want %t", got, tc.want)
			}
		})
	}
}
