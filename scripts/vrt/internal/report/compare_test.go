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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/capture"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/imagediff"
)

func writeSolidStoryPNG(t *testing.T, rootDir string, storyID string, c color.NRGBA) capture.SnapshotEntry {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	snapID := capture.StorySnapshotID(storyID)
	rel := capture.ImageRelPathForID(snapID)
	abs := filepath.Join(rootDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(abs, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write png failed: %v", err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return capture.SnapshotEntry{
		ID:           snapID,
		StoryID:      storyID,
		Title:        "Suite/" + storyID,
		Name:         "Default",
		ImportPath:   "./" + storyID + ".stories.ts",
		ImageRelPath: rel,
		SHA256:       hex.EncodeToString(sum[:]),
		Width:        16,
		Height:       16,
	}
}

func TestCompareDirectories(t *testing.T) {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	red := color.NRGBA{R: 220, G: 20, B: 20, A: 255}
	blue := color.NRGBA{R: 20, G: 40, B: 220, A: 255}

	baseDir := t.TempDir()
	headDir := t.TempDir()
	reportDir := t.TempDir()

	baseManifest := capture.Manifest{
		Version:            capture.ManifestVersion,
		CommitSHA:          "base11111111",
		BuilderFingerprint: "chrome-136-trixie",
		Entries: map[string]capture.SnapshotEntry{
			"story/unchanged": writeSolidStoryPNG(t, baseDir, "unchanged", white),
			"story/changed":   writeSolidStoryPNG(t, baseDir, "changed", white),
			"story/removed":   writeSolidStoryPNG(t, baseDir, "removed", blue),
		},
	}
	headManifest := capture.Manifest{
		Version:            capture.ManifestVersion,
		CommitSHA:          "head22222222",
		BuilderFingerprint: "chrome-136-trixie",
		Entries: map[string]capture.SnapshotEntry{
			"story/unchanged": writeSolidStoryPNG(t, headDir, "unchanged", white),
			"story/changed":   writeSolidStoryPNG(t, headDir, "changed", red),
			"story/added":     writeSolidStoryPNG(t, headDir, "added", blue),
			"story/errored": {
				ID:      "story/errored",
				StoryID: "errored",
				Title:   "Suite/errored",
				Name:    "Default",
				Error:   "uncaught error: boom",
			},
			"story/skipped": {
				ID:      "story/skipped",
				StoryID: "skipped",
				Title:   "Suite/skipped",
				Name:    "Default",
				Skipped: true,
			},
		},
	}

	if err := capture.WriteManifest(filepath.Join(baseDir, capture.ManifestFileName), baseManifest); err != nil {
		t.Fatalf("WriteManifest(base) failed: %v", err)
	}
	if err := capture.WriteManifest(filepath.Join(headDir, capture.ManifestFileName), headManifest); err != nil {
		t.Fatalf("WriteManifest(head) failed: %v", err)
	}

	rep, err := CompareDirectories(CompareConfig{
		BaseDir:      baseDir,
		HeadDir:      headDir,
		ReportDir:    reportDir,
		AssetBaseURL: "https://storage.cloud.google.com/khi-vrt/reports/head/1/",
		DiffOptions:  imagediff.Options{IncludeAntialiasing: true},
		Workers:      2,
	})
	if err != nil {
		t.Fatalf("CompareDirectories() unexpected error = %v", err)
	}
	if rep.AssetBaseURL != "https://storage.cloud.google.com/khi-vrt/reports/head/1" {
		t.Errorf("AssetBaseURL = %q, want trailing slash trimmed", rep.AssetBaseURL)
	}

	wantSummary := Summary{
		Total:     6,
		Changed:   1,
		Added:     1,
		Removed:   1,
		Errored:   1,
		Skipped:   1,
		Unchanged: 1,
	}
	if diff := cmp.Diff(wantSummary, rep.Summary); diff != "" {
		t.Errorf("Summary mismatch (-want +got):\n%s", diff)
	}
	if !rep.HasVisualDiffs() {
		t.Error("HasVisualDiffs() = false, want true")
	}
	if !rep.HasErrors() {
		t.Error("HasErrors() = false, want true")
	}

	wantItems := []StoryComparison{
		{
			ID:      "story/errored",
			StoryID: "errored",
			Title:   "Suite/errored",
			Name:    "Default",
			Status:  StatusErrored,
			Error:   "uncaught error: boom",
		},
		{
			ID:               "story/changed",
			StoryID:          "changed",
			Title:            "Suite/changed",
			Name:             "Default",
			ImportPath:       "./changed.stories.ts",
			Status:           StatusChanged,
			DiffPixels:       256,
			TotalPixels:      256,
			DiffRatio:        1.0,
			BeforeWidth:      16,
			BeforeHeight:     16,
			AfterWidth:       16,
			AfterHeight:      16,
			BeforeRelPath:    "before/story/changed.png",
			AfterRelPath:     "after/story/changed.png",
			DiffRelPath:      "diff/story/changed.png",
			CompositeRelPath: "composite/story/changed.png",
		},
		{
			ID:           "story/added",
			StoryID:      "added",
			Title:        "Suite/added",
			Name:         "Default",
			ImportPath:   "./added.stories.ts",
			Status:       StatusAdded,
			AfterWidth:   16,
			AfterHeight:  16,
			AfterRelPath: "after/story/added.png",
		},
		{
			ID:            "story/removed",
			StoryID:       "removed",
			Title:         "Suite/removed",
			Name:          "Default",
			ImportPath:    "./removed.stories.ts",
			Status:        StatusRemoved,
			BeforeWidth:   16,
			BeforeHeight:  16,
			BeforeRelPath: "before/story/removed.png",
		},
		{
			ID:           "story/unchanged",
			StoryID:      "unchanged",
			Title:        "Suite/unchanged",
			Name:         "Default",
			ImportPath:   "./unchanged.stories.ts",
			Status:       StatusUnchanged,
			TotalPixels:  256,
			BeforeWidth:  16,
			BeforeHeight: 16,
			AfterWidth:   16,
			AfterHeight:  16,
			AfterRelPath: "after/story/unchanged.png",
		},
		{
			ID:      "story/skipped",
			StoryID: "skipped",
			Title:   "Suite/skipped",
			Name:    "Default",
			Status:  StatusSkipped,
		},
	}
	if diff := cmp.Diff(wantItems, rep.Items); diff != "" {
		t.Errorf("rep.Items mismatch (-want +got):\n%s", diff)
	}

	expectedFiles := []string{
		ReportJSONFileName,
		ReportHTMLFileName,
		"before/story/changed.png",
		"after/story/changed.png",
		"diff/story/changed.png",
		"composite/story/changed.png",
		"after/story/added.png",
		"before/story/removed.png",
		"after/story/unchanged.png",
	}
	for _, rel := range expectedFiles {
		if _, err := os.Stat(filepath.Join(reportDir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected report artifact %q to exist: %v", rel, err)
		}
	}
	rawJSON, err := os.ReadFile(filepath.Join(reportDir, ReportJSONFileName))
	if err != nil {
		t.Fatalf("failed to read %s: %v", ReportJSONFileName, err)
	}
	if len(rawJSON) == 0 || rawJSON[len(rawJSON)-1] != '\n' {
		t.Errorf("report.json missing trailing newline")
	}
}

func writePartialDiffStoryPNG(t *testing.T, rootDir string, storyID string, bg color.NRGBA, fg color.NRGBA, halfRows int) capture.SnapshotEntry {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			if y < halfRows {
				img.SetNRGBA(x, y, fg)
			} else {
				img.SetNRGBA(x, y, bg)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	snapID := capture.StorySnapshotID(storyID)
	rel := capture.ImageRelPathForID(snapID)
	abs := filepath.Join(rootDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(abs, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write png failed: %v", err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return capture.SnapshotEntry{
		ID:           snapID,
		StoryID:      storyID,
		Title:        "Suite/" + storyID,
		Name:         "Default",
		ImportPath:   "./" + storyID + ".stories.ts",
		ImageRelPath: rel,
		SHA256:       hex.EncodeToString(sum[:]),
		Width:        16,
		Height:       16,
	}
}

func TestCompareDirectories_EdgeCasesAndOrdering(t *testing.T) {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	red := color.NRGBA{R: 220, G: 20, B: 20, A: 255}
	nearWhite := color.NRGBA{R: 254, G: 254, B: 254, A: 255}

	baseDir := t.TempDir()
	headDir := t.TempDir()
	reportDir := t.TempDir()

	headWithTolerance := writeSolidStoryPNG(t, headDir, "tolerant", red)
	headWithTolerance.DiffPixelRatio = 1.0

	baseNoHash := writeSolidStoryPNG(t, baseDir, "no-hash", white)
	baseNoHash.SHA256 = ""
	headNoHash := writeSolidStoryPNG(t, headDir, "no-hash", red)
	headNoHash.SHA256 = ""

	sizeMismatchHead := writePartialDiffStoryPNG(t, headDir, "size-diff", white, white, 0)
	// Replace size-diff head PNG with 20x16 so SizeMismatch is true.
	bigImg := image.NewNRGBA(image.Rect(0, 0, 20, 16))
	var bigBuf bytes.Buffer
	if err := png.Encode(&bigBuf, bigImg); err != nil {
		t.Fatalf("png.Encode(bigImg) failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(headDir, filepath.FromSlash(sizeMismatchHead.ImageRelPath)), bigBuf.Bytes(), 0o600); err != nil {
		t.Fatalf("write size-diff png failed: %v", err)
	}
	sizeMismatchHead.SHA256 = "different-hash"
	sizeMismatchHead.Width = 20

	baseManifest := capture.Manifest{
		Version: capture.ManifestVersion,
		Entries: map[string]capture.SnapshotEntry{
			"story/tolerant":     writeSolidStoryPNG(t, baseDir, "tolerant", white),
			"story/near":         writeSolidStoryPNG(t, baseDir, "near", white),
			"story/changed-0-lo": writeSolidStoryPNG(t, baseDir, "changed-0-lo", white),
			"story/changed-a":    writeSolidStoryPNG(t, baseDir, "changed-a", white),
			"story/changed-b":    writeSolidStoryPNG(t, baseDir, "changed-b", white),
			"story/no-hash":      baseNoHash,
			"story/size-diff":    writeSolidStoryPNG(t, baseDir, "size-diff", white),
			"story/base-skip":    {ID: "story/base-skip", StoryID: "base-skip", Skipped: true},
			"story/base-err":     {ID: "story/base-err", StoryID: "base-err", Error: "failed", ImageRelPath: "images/x.png"},
			"story/base-noimg":   {ID: "story/base-noimg", StoryID: "base-noimg"},
		},
	}
	headManifest := capture.Manifest{
		Version: capture.ManifestVersion,
		Entries: map[string]capture.SnapshotEntry{
			"story/tolerant":     headWithTolerance,
			"story/near":         writeSolidStoryPNG(t, headDir, "near", nearWhite),
			"story/changed-0-lo": writePartialDiffStoryPNG(t, headDir, "changed-0-lo", white, red, 4),
			"story/changed-a":    writeSolidStoryPNG(t, headDir, "changed-a", red),
			"story/changed-b":    writeSolidStoryPNG(t, headDir, "changed-b", red),
			"story/no-hash":      headNoHash,
			"story/size-diff":    sizeMismatchHead,
			"story/base-err":     writeSolidStoryPNG(t, headDir, "base-err", red),
		},
	}
	if err := capture.WriteManifest(filepath.Join(baseDir, capture.ManifestFileName), baseManifest); err != nil {
		t.Fatalf("WriteManifest(base) failed: %v", err)
	}
	if err := capture.WriteManifest(filepath.Join(headDir, capture.ManifestFileName), headManifest); err != nil {
		t.Fatalf("WriteManifest(head) failed: %v", err)
	}

	rep, err := CompareDirectories(CompareConfig{
		BaseDir:     baseDir,
		HeadDir:     headDir,
		ReportDir:   reportDir,
		DiffOptions: imagediff.Options{IncludeAntialiasing: true, MaxDiffPixelRatio: 0.1},
		Workers:     0,
	})
	if err != nil {
		t.Fatalf("CompareDirectories() error = %v", err)
	}
	wantSummary := Summary{Total: 10, Changed: 5, Added: 1, Unchanged: 2, Skipped: 2}
	if diff := cmp.Diff(wantSummary, rep.Summary); diff != "" {
		t.Errorf("Summary mismatch (-want +got):\n%s", diff)
	}
	// Verify changed items order by higher diff ratio first (changed-0-lo sorts AFTER changed-a/b despite earlier ID),
	// and unchanged items sort purely by ID (near before tolerant despite tolerant having higher DiffRatio=1.0).
	var gotOrder []string
	var sizeDiffFound bool
	for _, item := range rep.Items {
		gotOrder = append(gotOrder, string(item.Status)+":"+item.ID)
		if item.ID == "story/size-diff" && item.SizeMismatch {
			sizeDiffFound = true
		}
	}
	wantOrder := []string{
		"changed:story/changed-a",
		"changed:story/changed-b",
		"changed:story/no-hash",
		"changed:story/changed-0-lo",
		"changed:story/size-diff",
		"added:story/base-err",
		"unchanged:story/near",
		"unchanged:story/tolerant",
		"skipped:story/base-noimg",
		"skipped:story/base-skip",
	}
	if diff := cmp.Diff(wantOrder, gotOrder); diff != "" {
		t.Errorf("item order mismatch (-want +got):\n%s", diff)
	}
	if !sizeDiffFound {
		t.Error("expected story/size-diff to have SizeMismatch=true")
	}
	for _, rel := range []string{"after/story/near.png", "after/story/tolerant.png"} {
		if _, err := os.Stat(filepath.Join(reportDir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected unchanged report artifact %q to exist: %v", rel, err)
		}
	}
	for _, item := range rep.Items {
		if item.Status == StatusUnchanged && item.AfterRelPath != "after/"+item.ID+".png" {
			t.Errorf("unchanged item %q AfterRelPath = %q, want %q", item.ID, item.AfterRelPath, "after/"+item.ID+".png")
		}
	}

	// Verify global DiffOptions.MaxDiffPixelRatio is NOT zeroed out when head.DiffPixelRatio == 0.
	tolerantGlobalDir := t.TempDir()
	repGlobal, err := CompareDirectories(CompareConfig{
		BaseDir:     baseDir,
		HeadDir:     headDir,
		ReportDir:   tolerantGlobalDir,
		DiffOptions: imagediff.Options{IncludeAntialiasing: true, MaxDiffPixelRatio: 0.5},
		Workers:     1,
	})
	if err != nil {
		t.Fatalf("CompareDirectories(global tolerance) error = %v", err)
	}
	if repGlobal.Summary.Unchanged != 3 {
		t.Errorf("CompareDirectories(global tolerance) Unchanged = %d, want 3 (changed-0-lo 25%% <= 50%%)", repGlobal.Summary.Unchanged)
	}

	if got := resolveCompareWorkers(8, 2); got != 2 {
		t.Errorf("resolveCompareWorkers(8, 2) = %d, want 2", got)
	}
	if got := resolveCompareWorkers(4, 0); got != 1 {
		t.Errorf("resolveCompareWorkers(4, 0) = %d, want 1", got)
	}
	if hasUsableSnapshot(capture.SnapshotEntry{Skipped: true, ImageRelPath: "a.png"}) {
		t.Error("hasUsableSnapshot(skipped) = true, want false")
	}
	if hasUsableSnapshot(capture.SnapshotEntry{Error: "err", ImageRelPath: "a.png"}) {
		t.Error("hasUsableSnapshot(error) = true, want false")
	}
	if hasUsableSnapshot(capture.SnapshotEntry{}) {
		t.Error("hasUsableSnapshot(empty path) = true, want false")
	}
	if statusPriority(Status("unknown")) != 6 {
		t.Errorf("statusPriority(unknown) = %d, want 6", statusPriority(Status("unknown")))
	}
}

func TestComparisonReport_HasVisualDiffs(t *testing.T) {
	testCases := []struct {
		name string
		sum  Summary
		want bool
	}{
		{name: "no diffs", sum: Summary{Unchanged: 5}, want: false},
		{name: "changed only", sum: Summary{Changed: 1}, want: true},
		{name: "added only", sum: Summary{Added: 1}, want: true},
		{name: "removed only", sum: Summary{Removed: 1}, want: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rep := ComparisonReport{Summary: tc.sum}
			if got := rep.HasVisualDiffs(); got != tc.want {
				t.Errorf("HasVisualDiffs() = %v, want %v", got, tc.want)
			}
		})
	}
}
