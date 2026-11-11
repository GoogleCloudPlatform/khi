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

	"github.com/google/go-cmp/cmp"
)

func TestLoadCoverProfileIfPresent(t *testing.T) {
	tmpDir := t.TempDir()
	existingProfile := filepath.Join(tmpDir, "go-cover.output")
	profileContent := `mode: set
github.com/GoogleCloudPlatform/khi/pkg/foo/bar.go:5.12,8.2 1 1
`
	if err := os.WriteFile(existingProfile, []byte(profileContent), 0o644); err != nil {
		t.Fatalf("failed to write test coverprofile: %v", err)
	}

	testCases := []struct {
		name        string
		profilePath string
		want        map[string][]CoverBlock
	}{
		{
			name:        "returns empty map when coverprofile was not generated",
			profilePath: filepath.Join(tmpDir, "nonexistent.output"),
			want:        map[string][]CoverBlock{},
		},
		{
			name:        "parses blocks when coverprofile exists",
			profilePath: existingProfile,
			want: map[string][]CoverBlock{
				"pkg/foo/bar.go": {
					{
						StartLine:     5,
						StartCol:      12,
						EndLine:       8,
						EndCol:        2,
						NumStatements: 1,
						Count:         1,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := loadCoverProfileIfPresent(tc.profilePath, "github.com/GoogleCloudPlatform/khi")
			if err != nil {
				t.Fatalf("loadCoverProfileIfPresent() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("loadCoverProfileIfPresent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
