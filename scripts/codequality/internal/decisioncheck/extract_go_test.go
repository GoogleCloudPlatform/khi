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

package decisioncheck

import (
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractGoSubstantiveUnits(t *testing.T) {
	testCases := []struct {
		name      string
		target    string
		files     []BundledFile
		wantUnits []ExtractedItem
		wantIndex string
	}{
		// Extracts only bare Ident entries under Tasks composite literals and ignores non-Go files or non-Tasks fields.
		{
			name:   "task module package extracts registered Module.Tasks",
			target: "pkg/task/demo",
			files: []BundledFile{
				{
					RelPath: "pkg/task/demo/README.md",
					Content: []byte(`# Demo`),
				},
				{
					RelPath: "pkg/task/demo/types.go",
					Content: []byte(`package demo

type FieldSet struct {
	ID string
}
`),
				},
				{
					RelPath: "pkg/task/demo/impl/module.go",
					Content: []byte(`package demo_impl

var existingTasks = []Untyped{}
var lookup = map[string]int{"k": 1}

var OtherModule = ModuleSpec{
	Other: []Untyped{ignoredTask},
	Tasks: existingTasks,
}

var Module = ModuleSpec{
	Tasks: []Untyped{
		ingesterTask,
		pkg.ExternalTask,
		mapperTask,
	},
}
`),
				},
			},
			wantUnits: []ExtractedItem{
				{Target: "pkg/task/demo", File: "pkg/task/demo/impl/module.go", Line: 13, StartLine: 13, EndLine: 13, Name: "ingesterTask", Kind: "registered DAG task"},
				{Target: "pkg/task/demo", File: "pkg/task/demo/impl/module.go", Line: 15, StartLine: 15, EndLine: 15, Name: "mapperTask", Kind: "registered DAG task"},
			},
			wantIndex: `- pkg/task/demo/types.go (package demo)
  type FieldSet (pkg/task/demo/types.go:3)
    field ID string
- pkg/task/demo/impl/module.go (package demo_impl)
`,
		},
		// Summarizes non-module Go files, clips exported file labels past 6 symbols while keeping exact-6 unclipped, and truncates struct fields past 16.
		{
			name:   "general Go package caps file symbols over 6 and truncates struct fields at 16",
			target: "pkg/model/store",
			files: []BundledFile{
				{
					RelPath: "pkg/model/store/store.go",
					Content: []byte(`package store

type Reader interface{ Read() }

type internalCache struct {
	size int
}

type Store struct {
	Reader
	left, right int
	F3          int
	F4          int
	F5          int
	F6          int
	F7          int
	F8          int
	F9          int
	F10         int
	F11         int
	F12         int
	F13         int
	F14         int
	F15         int
	F16         int
	F17         int
}

func (s *Store) Flush() {}
func NewStore() *Store  { return nil }
func OpenStore()        {}
func CloseStore()       {}
func ResetStore()       {}
func SnapshotStore()    {}
`),
				},
				{
					RelPath: "pkg/model/store/exact_six.go",
					Content: []byte(`package store

func A() {}
func B() {}
func C() {}
func D() {}
func E() {}
func F() {}
`),
				},
				{
					RelPath: "pkg/model/store/private.go",
					Content: []byte(`package store

func internalHelper() {}
`),
				},
				{
					RelPath: "pkg/model/store/store_test.go",
					Content: []byte(`package store
`),
				},
			},
			wantUnits: []ExtractedItem{
				{Target: "pkg/model/store", File: "pkg/model/store/store.go", Line: 1, StartLine: 1, EndLine: 34, Name: "store.go (Reader, Store, NewStore, OpenStore, CloseStore, ResetStore, ...)", Kind: "Go source file"},
				{Target: "pkg/model/store", File: "pkg/model/store/exact_six.go", Line: 1, StartLine: 1, EndLine: 8, Name: "exact_six.go (A, B, C, D, E, F)", Kind: "Go source file"},
			},
			wantIndex: `- pkg/model/store/store.go (package store)
  type Reader (pkg/model/store/store.go:3)
  type internalCache (pkg/model/store/store.go:5)
    field size int
  type Store (pkg/model/store/store.go:9)
    embedded Reader
    field left, right int
    field F3 int
    field F4 int
    field F5 int
    field F6 int
    field F7 int
    field F8 int
    field F9 int
    field F10 int
    field F11 int
    field F12 int
    field F13 int
    field F14 int
    field F15 int
    field F16 int
    ...
  func Store.Flush (pkg/model/store/store.go:29)
  func NewStore (pkg/model/store/store.go:30)
  func OpenStore (pkg/model/store/store.go:31)
  func CloseStore (pkg/model/store/store.go:32)
  func ResetStore (pkg/model/store/store.go:33)
  func SnapshotStore (pkg/model/store/store.go:34)
- pkg/model/store/exact_six.go (package store)
  func A (pkg/model/store/exact_six.go:3)
  func B (pkg/model/store/exact_six.go:4)
  func C (pkg/model/store/exact_six.go:5)
  func D (pkg/model/store/exact_six.go:6)
  func E (pkg/model/store/exact_six.go:7)
  func F (pkg/model/store/exact_six.go:8)
- pkg/model/store/private.go (package store)
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotUnits, gotIndex, err := ExtractGoSubstantiveUnits(tc.target, tc.files)

			if err != nil {
				t.Fatalf("ExtractGoSubstantiveUnits() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantUnits, gotUnits); diff != "" {
				t.Errorf("ExtractGoSubstantiveUnits() units mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantIndex, gotIndex); diff != "" {
				t.Errorf("ExtractGoSubstantiveUnits() index mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractChangedGoDecls(t *testing.T) {
	sampleSrc := `package sample

import "errors"

var _ error = (*Impl)(nil)

// SingleType documents an unparenthesized type declaration.
type SingleType struct{}

// GroupDoc does not attach to individual specs when multiple specs exist.
type (
	// FirstGrouped documents the first grouped type.
	FirstGrouped struct{}
	UndocumentedGrouped struct{}
)

// FeatureTask defines the primary feature entrypoint.
var FeatureTask = defineTask(
	"feature",
)

const (
	//nolint
	// FirstConst documents the first constant.
	FirstConst = 1
	//go:noinline
	// SecondConst documents the second constant.
	SecondConst = 2
	// CODEQUALITY_ALLOW(func_cognitive, equivalent): synthetic fixture.
	ThirdConst = 3
	//go:noinline
	OnlyDirectiveConst = 4
	UndocumentedConst = 5
)

type Impl struct{}

// Run executes the implementation workflow.
func (i *Impl) Run() error {
	return errors.New("boom")
}

// Validate checks value receiver naming.
func (i Impl) Validate() bool {
	return true
}

// Clear checks generic receiver fallback naming.
func (b *GenericBox[T]) Clear() {}

// Helper performs standalone work.
func Helper() {}

func untouchedHelper() {}
`
	files := []BundledFile{
		{RelPath: "pkg/sample/sample.go", Content: []byte(sampleSrc)},
		{
			RelPath: "pkg/sample/other.go",
			Content: []byte(`package sample

func UntouchedFileFunc() {}
`),
		},
	}

	testCases := []struct {
		name         string
		changedLines map[string]map[int]struct{}
		wantDecls    []ExtractedItem
	}{
		// Extracts all non-import declarations across files when changedLines is nil, stripping directive comments and resolving single/grouped docs and receiver kinds.
		{
			name:         "full mode extracts all decls and strips directives",
			changedLines: nil,
			wantDecls: []ExtractedItem{
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 8, StartLine: 7, EndLine: 8, Name: "SingleType", Kind: "type", Doc: "SingleType documents an unparenthesized type declaration."},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 13, StartLine: 12, EndLine: 13, Name: "FirstGrouped", Kind: "type", Doc: "FirstGrouped documents the first grouped type."},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 14, StartLine: 14, EndLine: 14, Name: "UndocumentedGrouped", Kind: "type"},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 18, StartLine: 17, EndLine: 20, Name: "FeatureTask", Kind: "var", Doc: "FeatureTask defines the primary feature entrypoint."},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 25, StartLine: 23, EndLine: 25, Name: "FirstConst", Kind: "const", Doc: "FirstConst documents the first constant."},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 28, StartLine: 26, EndLine: 28, Name: "SecondConst", Kind: "const", Doc: "SecondConst documents the second constant."},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 30, StartLine: 29, EndLine: 30, Name: "ThirdConst", Kind: "const"},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 32, StartLine: 31, EndLine: 32, Name: "OnlyDirectiveConst", Kind: "const"},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 33, StartLine: 33, EndLine: 33, Name: "UndocumentedConst", Kind: "const"},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 36, StartLine: 36, EndLine: 36, Name: "Impl", Kind: "type"},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 39, StartLine: 38, EndLine: 41, Name: "Impl.Run", Kind: "method", Doc: "Run executes the implementation workflow."},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 44, StartLine: 43, EndLine: 46, Name: "Impl.Validate", Kind: "method", Doc: "Validate checks value receiver naming."},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 49, StartLine: 48, EndLine: 49, Name: "Clear", Kind: "method", Doc: "Clear checks generic receiver fallback naming."},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 52, StartLine: 51, EndLine: 52, Name: "Helper", Kind: "func", Doc: "Helper performs standalone work."},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 54, StartLine: 54, EndLine: 54, Name: "untouchedHelper", Kind: "func"},
				{Target: "pkg/sample", File: "pkg/sample/other.go", Line: 3, StartLine: 3, EndLine: 3, Name: "UntouchedFileFunc", Kind: "func"},
			},
		},
		// Filters declarations strictly to those overlapping changed line numbers including exact EndLine matches and skips files absent from changedLines.
		{
			name: "differential mode matches exact EndLine and skips unchanged files",
			changedLines: map[string]map[int]struct{}{
				"pkg/sample/sample.go": {5: {}, 20: {}, 52: {}},
			},
			wantDecls: []ExtractedItem{
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 18, StartLine: 17, EndLine: 20, Name: "FeatureTask", Kind: "var", Doc: "FeatureTask defines the primary feature entrypoint."},
				{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 52, StartLine: 51, EndLine: 52, Name: "Helper", Kind: "func", Doc: "Helper performs standalone work."},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotDecls, err := ExtractChangedGoDecls("pkg/sample", files, tc.changedLines)

			if err != nil {
				t.Fatalf("ExtractChangedGoDecls() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantDecls, gotDecls); diff != "" {
				t.Errorf("ExtractChangedGoDecls() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestExtractChangedGoFuncs verifies that ExtractChangedGoFuncs skips non-function declarations and extracts only functions and methods overlapping changedLines.
func TestExtractChangedGoFuncs(t *testing.T) {
	src := `package sample

var FeatureTask = 1

// Run executes pointer-receiver work.
func (i *Impl) Run() error {
	return nil
}

// Helper performs standalone work.
func Helper() {}

func untouchedHelper() {}
`
	files := []BundledFile{{RelPath: "pkg/sample/sample.go", Content: []byte(src)}}
	changedLines := map[string]map[int]struct{}{
		"pkg/sample/sample.go": {3: {}, 7: {}, 11: {}},
	}

	gotFuncs, err := ExtractChangedGoFuncs("pkg/sample", files, changedLines)

	if err != nil {
		t.Fatalf("ExtractChangedGoFuncs() unexpected error: %v", err)
	}
	wantFuncs := []ExtractedItem{
		{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 6, StartLine: 5, EndLine: 8, Name: "Impl.Run", Kind: "method", Doc: "Run executes pointer-receiver work."},
		{Target: "pkg/sample", File: "pkg/sample/sample.go", Line: 11, StartLine: 10, EndLine: 11, Name: "Helper", Kind: "func", Doc: "Helper performs standalone work."},
	}
	if diff := cmp.Diff(wantFuncs, gotFuncs); diff != "" {
		t.Errorf("ExtractChangedGoFuncs() mismatch (-want +got):\n%s", diff)
	}
}

// TestParseTestGoFiles verifies that parseTestGoFiles parses only _test.go files and skips production Go and Markdown files.
func TestParseTestGoFiles(t *testing.T) {
	fset := token.NewFileSet()
	files := []BundledFile{
		{RelPath: "pkg/sample/prod.go", Content: []byte(`package sample`)},
		{RelPath: "pkg/sample/README.md", Content: []byte(`# Sample`)},
		{RelPath: "pkg/sample/sample_test.go", Content: []byte(`package sample_test`)},
	}

	gotParsed, err := parseTestGoFiles(fset, files)

	if err != nil {
		t.Fatalf("parseTestGoFiles() unexpected error: %v", err)
	}
	type parsedSummary struct {
		RelPath string
		PkgName string
	}
	gotSummaries := make([]parsedSummary, len(gotParsed))
	for i, parsed := range gotParsed {
		gotSummaries[i] = parsedSummary{
			RelPath: parsed.relPath,
			PkgName: parsed.astFile.Name.Name,
		}
	}
	wantSummaries := []parsedSummary{
		{RelPath: "pkg/sample/sample_test.go", PkgName: "sample_test"},
	}
	if diff := cmp.Diff(wantSummaries, gotSummaries); diff != "" {
		t.Errorf("parseTestGoFiles() mismatch (-want +got):\n%s", diff)
	}
}
