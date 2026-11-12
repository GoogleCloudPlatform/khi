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

package gotestutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadModulePath(t *testing.T) {
	tmpDir := t.TempDir()
	validMod := filepath.Join(tmpDir, "go.mod")
	validModContent := `module github.com/example/sample

go 1.27
`
	if err := os.WriteFile(validMod, []byte(validModContent), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	emptyMod := filepath.Join(tmpDir, "empty.mod")
	if err := os.WriteFile(emptyMod, []byte("go 1.27\n"), 0o644); err != nil {
		t.Fatalf("failed to write empty.mod: %v", err)
	}

	testCases := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "reads module path from valid go.mod",
			path: validMod,
			want: "github.com/example/sample",
		},
		{
			name:    "returns error when module directive is missing",
			path:    emptyMod,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ReadModulePath(tc.path)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ReadModulePath() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ReadModulePath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseCoverProfile(t *testing.T) {
	profile := `mode: set

github.com/GoogleCloudPlatform/khi/pkg/foo/bar.go:10.15,14.2 2 0
github.com/GoogleCloudPlatform/khi/pkg/foo/bar.go:10.15,14.2 2 1
github.com/GoogleCloudPlatform/khi/pkg/foo/bar.go:5.20,8.2 1 1
github.com/GoogleCloudPlatform/khi/pkg/foo/bar.go:5.12,8.2 1 0
./pkg/foo/rel.go:3.1,4.2 1 1
github.com/GoogleCloudPlatform/khi/pkg/generated/gen.go:1.1,5.2 3 1
github.com/GoogleCloudPlatform/khi/pkg/testutil/helper.go:1.1,5.2 2 1
github.com/GoogleCloudPlatform/khi/pkg/foo/zzz_codegen.go:1.1,5.2 2 1
`
	want := map[string][]CoverBlock{
		"pkg/foo/bar.go": {
			{
				StartLine:     5,
				StartCol:      12,
				EndLine:       8,
				EndCol:        2,
				NumStatements: 1,
				Count:         0,
			},
			{
				StartLine:     5,
				StartCol:      20,
				EndLine:       8,
				EndCol:        2,
				NumStatements: 1,
				Count:         1,
			},
			{
				StartLine:     10,
				StartCol:      15,
				EndLine:       14,
				EndCol:        2,
				NumStatements: 2,
				Count:         1,
			},
		},
		"pkg/foo/rel.go": {
			{
				StartLine:     3,
				StartCol:      1,
				EndLine:       4,
				EndCol:        2,
				NumStatements: 1,
				Count:         1,
			},
		},
	}

	got, err := ParseCoverProfile(strings.NewReader(profile), "github.com/GoogleCloudPlatform/khi")
	if err != nil {
		t.Fatalf("ParseCoverProfile() unexpected error: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ParseCoverProfile() mismatch (-want +got):\n%s", diff)
	}
}

func TestParseCoverProfileInvalidLineReportsLineNumber(t *testing.T) {
	profile := `mode: set
:1.1,2.2 1 1
`
	_, err := ParseCoverProfile(strings.NewReader(profile), "github.com/GoogleCloudPlatform/khi")
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("ParseCoverProfile() error = %v, want line 2 parse error", err)
	}
}

func TestLoadCoverProfileIfPresent(t *testing.T) {
	tmpDir := t.TempDir()
	existing := filepath.Join(tmpDir, "go-cover.output")
	content := `mode: set
github.com/GoogleCloudPlatform/khi/pkg/foo/bar.go:5.12,7.2 1 2
`
	if err := os.WriteFile(existing, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write coverprofile: %v", err)
	}

	testCases := []struct {
		name string
		path string
		want map[string][]CoverBlock
	}{
		{
			name: "missing file returns empty map without error",
			path: filepath.Join(tmpDir, "missing.output"),
			want: map[string][]CoverBlock{},
		},
		{
			name: "existing file returns parsed blocks",
			path: existing,
			want: map[string][]CoverBlock{
				"pkg/foo/bar.go": {
					{
						StartLine:     5,
						StartCol:      12,
						EndLine:       7,
						EndCol:        2,
						NumStatements: 1,
						Count:         2,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LoadCoverProfileIfPresent(tc.path, "github.com/GoogleCloudPlatform/khi", FilterOptions{})
			if err != nil {
				t.Fatalf("LoadCoverProfileIfPresent() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("LoadCoverProfileIfPresent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCoveredLineSet(t *testing.T) {
	blocks := []CoverBlock{
		{StartLine: 3, EndLine: 5, Count: 0},
		{StartLine: 8, EndLine: 10, Count: 1},
	}
	want := map[int]struct{}{
		8:  {},
		9:  {},
		10: {},
	}
	got := CoveredLineSet(blocks)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CoveredLineSet() mismatch (-want +got):\n%s", diff)
	}
}
