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

package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteHTMLReport(t *testing.T) {
	dir := t.TempDir()
	imgRel := "after/story/header--default.png"
	unchangedRel := "after/story/nav--default.png"

	rep := ComparisonReport{
		BaseSHA:      "aaaa1111",
		HeadSHA:      "bbbb2222",
		AssetBaseURL: "https://storage.cloud.google.com/khi-vrt/reports/bbbb2222/1",
		Summary:      Summary{Total: 3, Added: 1, Unchanged: 1, Errored: 1},
		Items: []StoryComparison{
			{
				ID:           "story/header--default",
				Title:        "Header",
				Name:         "Default",
				Status:       StatusAdded,
				AfterRelPath: imgRel,
			},
			{
				ID:           "story/nav--default",
				Title:        "Nav",
				Name:         "Default",
				Status:       StatusUnchanged,
				AfterRelPath: unchangedRel,
			},
			{
				ID:     "story/broken",
				Status: StatusErrored,
				Error:  "bad </script> payload",
			},
		},
	}

	outPath := filepath.Join(dir, ReportHTMLFileName)
	if err := WriteHTMLReport(outPath, rep); err != nil {
		t.Fatalf("WriteHTMLReport() unexpected error = %v", err)
	}

	htmlBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read HTML report: %v", err)
	}
	html := string(htmlBytes)

	if !strings.Contains(html, imgRel) || !strings.Contains(html, unchangedRel) {
		t.Error("expected HTML report to reference relative image paths directly")
	}
	if !strings.Contains(html, `"assetBaseUrl":"https://storage.cloud.google.com/khi-vrt/reports/bbbb2222/1"`) {
		t.Error("expected HTML report to preserve assetBaseUrl for GCS absolute image resolution")
	}
	if strings.Contains(html, "data:image/png;base64,") {
		t.Error("expected HTML report to load images externally without base64 inlining")
	}
	if strings.Contains(html, "bad </script> payload") {
		t.Error("expected </script> inside JSON payload to be escaped")
	}
}
