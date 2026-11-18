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

package storybook

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadStories(t *testing.T) {
	sampleIndex := `{
  "v": 5,
  "entries": {
    "timeline-chart--default": {
      "type": "story",
      "id": "timeline-chart--default",
      "title": "Timeline/Main/TimelineChart",
      "name": "Default",
      "importPath": "./src/app/timeline/components/timeline-chart.stories.ts",
      "componentPath": "./src/app/timeline/components/timeline-chart.component.ts",
      "tags": ["dev", "test", "autodocs"]
    },
    "timeline-chart--docs": {
      "type": "docs",
      "id": "timeline-chart--docs",
      "title": "Timeline/Main/TimelineChart",
      "name": "Docs",
      "importPath": "./src/app/timeline/components/timeline-chart.stories.ts"
    },
    "header--default": {
      "type": "story",
      "title": "Header",
      "name": "Default",
      "importPath": "./src/app/header/components/header.component.stories.ts"
    }
  }
}`

	testCases := []struct {
		name      string
		indexJSON string
		filter    string
		want      []StoryEntry
		wantErr   bool
	}{
		{
			name:      "loads story entries sorted by id and skips docs",
			indexJSON: sampleIndex,
			want: []StoryEntry{
				{
					ID:         "header--default",
					Title:      "Header",
					Name:       "Default",
					ImportPath: "./src/app/header/components/header.component.stories.ts",
				},
				{
					ID:            "timeline-chart--default",
					Title:         "Timeline/Main/TimelineChart",
					Name:          "Default",
					ImportPath:    "./src/app/timeline/components/timeline-chart.stories.ts",
					ComponentPath: "./src/app/timeline/components/timeline-chart.component.ts",
					Tags:          []string{"dev", "test", "autodocs"},
				},
			},
		},
		{
			name:      "filters case-insensitively across title and id",
			indexJSON: sampleIndex,
			filter:    "timelinechart",
			want: []StoryEntry{
				{
					ID:            "timeline-chart--default",
					Title:         "Timeline/Main/TimelineChart",
					Name:          "Default",
					ImportPath:    "./src/app/timeline/components/timeline-chart.stories.ts",
					ComponentPath: "./src/app/timeline/components/timeline-chart.component.ts",
					Tags:          []string{"dev", "test", "autodocs"},
				},
			},
		},
		{
			name:      "rejects invalid filter regex",
			indexJSON: sampleIndex,
			filter:    "[unclosed",
			wantErr:   true,
		},
		{
			name:      "rejects malformed json",
			indexJSON: "{invalid",
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(tc.indexJSON), 0o600); err != nil {
				t.Fatalf("failed to write test index.json: %v", err)
			}

			got, err := LoadStories(dir, tc.filter)
			if (err != nil) != tc.wantErr {
				t.Fatalf("LoadStories() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("LoadStories() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadStories_MissingFile(t *testing.T) {
	_, err := LoadStories(t.TempDir(), "")
	if err == nil {
		t.Fatal("LoadStories() expected error for missing index.json, got nil")
	}
}

func TestStartStaticServer(t *testing.T) {
	dir := t.TempDir()
	wantBody := "<html>storybook</html>"
	if err := os.WriteFile(filepath.Join(dir, "iframe.html"), []byte(wantBody), 0o600); err != nil {
		t.Fatalf("failed to write iframe.html: %v", err)
	}

	srv, err := StartStaticServer(dir)
	if err != nil {
		t.Fatalf("StartStaticServer() unexpected error = %v", err)
	}
	defer srv.Close()

	resp, err := http.Get(srv.BaseURL() + "/iframe.html")
	if err != nil {
		t.Fatalf("GET /iframe.html failed: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body failed: %v", err)
	}
	if string(bodyBytes) != wantBody {
		t.Errorf("GET /iframe.html body = %q, want %q", string(bodyBytes), wantBody)
	}
}

func TestStartStaticServer_InvalidPath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	testCases := []struct {
		name string
		path string
	}{
		{name: "missing path", path: filepath.Join(t.TempDir(), "does-not-exist")},
		{name: "regular file path", path: filePath},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := StartStaticServer(tc.path); err == nil {
				t.Errorf("StartStaticServer(%q) expected error, got nil", tc.path)
			}
		})
	}
}
