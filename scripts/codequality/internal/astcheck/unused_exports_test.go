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

package astcheck

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCheckUnusedExports(t *testing.T) {
	testCases := []struct {
		name     string
		pkgsFlag string
		files    map[string]string
		want     []ConventionViolation
	}{
		{
			name:     "empty pkgs flag returns nil",
			pkgsFlag: " , ",
			files: map[string]string{
				"go.mod": "module example.com/testmod\n\ngo 1.24\n",
			},
			want: nil,
		},
		{
			name:     "detects unused exported decls and respects cross-pkg test generated and field selector boundaries",
			pkgsFlag: "./...",
			files: map[string]string{
				"go.mod": "module example.com/testmod\n\ngo 1.24\n",
				"pkg/a/a.go": `package a

const (
	UsedConst   = 1
	UnusedConst = 2
)

var (
	UsedVar   = "ok"
	UnusedVar = "dead"
)

type UsedType struct {
	ShadowName int
}

type ShadowName struct{}

type UnusedType struct{}

type TestOnlyType struct{}

type GeneratedUsedType struct{}

func UsedFunc() int { return UsedConst }

func UnusedFunc() {}

func (UsedType) UnusedMethodIgnored() {}
`,
				"pkg/a/a_test.go": `package a

import "testing"

func TestA(t *testing.T) {
	var _ TestOnlyType
}
`,
				"pkg/a/zzz_gen.go": `package a

var GenRef GeneratedUsedType
var UnusedGenDecl = 1
`,
				"pkg/b/b.go": `package b

import "example.com/testmod/pkg/a"

func UseA(x a.UsedType) int {
	_ = a.UsedVar
	_ = x.ShadowName
	return a.UsedFunc()
}
`,
				"cmd/app/main.go": `package main

import (
	"example.com/testmod/pkg/a"
	"example.com/testmod/pkg/b"
)

func main() {
	_ = b.UseA(a.UsedType{})
}
`,
				"pkg/generated/g.go": `package generated

type IgnoredGenerated int
`,
				"pkg/testutil/helper.go": `package testutil

type IgnoredTestutil int
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/a/a.go",
					Line:    5,
					Column:  2,
					Rule:    RuleUnusedExportedDecl,
					Message: "exported const UnusedConst has no references in the repository",
				},
				{
					File:    "pkg/a/a.go",
					Line:    10,
					Column:  2,
					Rule:    RuleUnusedExportedDecl,
					Message: "exported var UnusedVar has no references in the repository",
				},
				{
					File:    "pkg/a/a.go",
					Line:    17,
					Column:  6,
					Rule:    RuleUnusedExportedDecl,
					Message: "exported type ShadowName has no references in the repository",
				},
				{
					File:    "pkg/a/a.go",
					Line:    19,
					Column:  6,
					Rule:    RuleUnusedExportedDecl,
					Message: "exported type UnusedType has no references in the repository",
				},
				{
					File:    "pkg/a/a.go",
					Line:    27,
					Column:  6,
					Rule:    RuleUnusedExportedDecl,
					Message: "exported func UnusedFunc has no references in the repository",
				},
			},
		},
		{
			name:     "detects unregistered task variables in _impl packages even when tested",
			pkgsFlag: "./...",
			files: map[string]string{
				"go.mod": "module example.com/testmod\n\ngo 1.24\n",
				"pkg/core/task.go": `package core

func DefineTask[T any](name string) T { var z T; return z }
`,
				"pkg/feature/impl/tasks.go": `package feature_impl

import "example.com/testmod/pkg/core"

var registeredTask = core.DefineTask[int]("reg")
var unregisteredTask = core.DefineTask[int]("unreg")
var ExportedUnregisteredTask = core.DefineTask[int]("exp-unreg")

func Register() int {
	return registeredTask
}
`,
				"pkg/feature/impl/tasks_test.go": `package feature_impl

import "testing"

func TestTasks(t *testing.T) {
	_ = unregisteredTask
	_ = ExportedUnregisteredTask
}
`,
				"cmd/app/main.go": `package main

import feature_impl "example.com/testmod/pkg/feature/impl"

func main() {
	_ = feature_impl.Register()
}
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/feature/impl/tasks.go",
					Line:    6,
					Column:  5,
					Rule:    RuleUnregisteredImplTask,
					Message: "task variable unregisteredTask in feature_impl is never referenced in production code (missing registration?)",
				},
				{
					File:    "pkg/feature/impl/tasks.go",
					Line:    7,
					Column:  5,
					Rule:    RuleUnregisteredImplTask,
					Message: "task variable ExportedUnregisteredTask in feature_impl is never referenced in production code (missing registration?)",
				},
			},
		},
		{
			name:     "pkg pattern scopes reported violations while still indexing cross-package references",
			pkgsFlag: "./pkg/a",
			files: map[string]string{
				"go.mod": "module example.com/testmod\n\ngo 1.24\n",
				"pkg/a/a.go": `package a

const UsedByB = 1
const UnusedInA = 2
`,
				"pkg/b/b.go": `package b

import "example.com/testmod/pkg/a"

var UnusedInB = a.UsedByB
`,
			},
			want: []ConventionViolation{
				{
					File:    "pkg/a/a.go",
					Line:    4,
					Column:  7,
					Rule:    RuleUnusedExportedDecl,
					Message: "exported const UnusedInA has no references in the repository",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeConvFixtureRepo(t, tc.files)
			got, err := CheckUnusedExports(root, tc.pkgsFlag)
			if err != nil {
				t.Fatalf("CheckUnusedExports() error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("CheckUnusedExports() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCheckUnusedExportsErrors(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(t *testing.T) string
	}{
		{
			name: "missing go.mod returns error",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing")
			},
		},
		{
			name: "go.mod without module directive returns error",
			setup: func(t *testing.T) string {
				return writeConvFixtureRepo(t, map[string]string{
					"go.mod": "go 1.24\n",
				})
			},
		},
		{
			name: "invalid Go syntax returns parse error",
			setup: func(t *testing.T) string {
				return writeConvFixtureRepo(t, map[string]string{
					"go.mod":        "module example.com/broken\n\ngo 1.24\n",
					"pkg/broken.go": "package broken\nfunc (",
				})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := tc.setup(t)
			if _, err := CheckUnusedExports(root, "./..."); err == nil {
				t.Errorf("expected error for %s, got nil", tc.name)
			}
		})
	}
}
