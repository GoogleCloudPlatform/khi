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

package langgo

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/simcheck"
)

func TestAnalyzerMetadataCorpusAndBaseKeys(t *testing.T) {
	analyzer := NewAnalyzer()
	if analyzer.ID() != simcheck.LangGo || analyzer.DisplayName() != "Go" || analyzer.CodeFence() != "go" {
		t.Fatalf("unexpected Analyzer metadata: %s / %s / %s", analyzer.ID(), analyzer.DisplayName(), analyzer.CodeFence())
	}

	tmp := t.TempDir()
	filesToCreate := map[string]string{
		"pkg/alpha/run.go":           "package alpha\n",
		"pkg/alpha/run_test.go":      "package alpha\n",
		"pkg/common/collection.go":   "package common\n",
		"pkg/generated/gen.go":       "package generated\n",
		"pkg/model/enum/zzz_enum.go": "package enum\n",
		"pkg/api/v1/service.pb.go":   "package v1\n",
		"pkg/testutil/helper.go":     "package testutil\n",
		"scripts/tool/main.go":       "package main\n",
		"scripts/tool/testdata/x.go": "package testdata\n",
		"scripts/tool/.hidden/x.go":  "package hidden\n",
	}
	for rel, content := range filesToCreate {
		full := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}

	got, err := analyzer.DiscoverCorpusFiles(tmp)
	if err != nil {
		t.Fatalf("DiscoverCorpusFiles() failed: %v", err)
	}
	want := []string{"pkg/alpha/run.go", "pkg/common/collection.go", "scripts/tool/main.go"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DiscoverCorpusFiles() mismatch (-want +got):\n%s", diff)
	}

	keys, err := analyzer.ExtractBaseKeys(context.Background(), []simcheck.FileInput{
		{Path: "pkg/a/a.go", Content: []byte("package a\nfunc Small() int { return 1 }\n")},
	})
	if err != nil {
		t.Fatalf("ExtractBaseKeys() failed: %v", err)
	}
	wantKeys := map[string]map[string]struct{}{
		"pkg/a/a.go": {"func:Small": {}},
	}
	if diff := cmp.Diff(wantKeys, keys); diff != "" {
		t.Errorf("ExtractBaseKeys() mismatch (-want +got):\n%s", diff)
	}
}

func TestAnalyzerIsSharedModule(t *testing.T) {
	analyzer := NewAnalyzer()
	testCases := []struct {
		mod  string
		want bool
	}{
		{"pkg/common", true},
		{"pkg/common/typedmap", true},
		{"scripts/gotestutil", true},
		{"scripts/gotestutil/sub", true},
		{"pkg/task/inspection/googlecloud/gcpcommon", true},
		{"pkg/task/inspection/common/k8saudit", true},
		{"pkg/task/inspection/googlecloud/gcpcommon/impl", false},
		{"pkg/task/inspection/googlecloud/csm", false},
	}
	for _, tc := range testCases {
		t.Run(tc.mod, func(t *testing.T) {
			if got := analyzer.IsSharedModule(tc.mod); got != tc.want {
				t.Errorf("IsSharedModule(%q) = %v, want %v", tc.mod, got, tc.want)
			}
		})
	}
}
