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
	"strings"
	"testing"
)

func TestFormatPRCommentBody(t *testing.T) {
	changedReport := ComparisonReport{
		BaseSHA: "base1234567890",
		HeadSHA: "head9876543210",
		Summary: Summary{Total: 10, Changed: 1, Added: 1, Removed: 1, Unchanged: 7},
		Items: []StoryComparison{
			{
				ID:               "story/header--default",
				Title:            "Header",
				Name:             "Default",
				Status:           StatusChanged,
				DiffPixels:       420,
				TotalPixels:      10000,
				DiffRatio:        0.042,
				BeforeWidth:      1280,
				BeforeHeight:     720,
				AfterWidth:       1280,
				AfterHeight:      720,
				BeforeRelPath:    "before/story/header--default.png",
				AfterRelPath:     "after/story/header--default.png",
				DiffRelPath:      "diff/story/header--default.png",
				CompositeRelPath: "composite/story/header--default.png",
			},
			{
				ID:           "story/new--one",
				Title:        "New",
				Name:         "One",
				Status:       StatusAdded,
				AfterRelPath: "after/story/new--one.png",
			},
			{
				ID:            "story/old--one",
				Title:         "Old",
				Name:          "One",
				Status:        StatusRemoved,
				BeforeRelPath: "before/story/old--one.png",
			},
		},
	}

	testCases := []struct {
		name            string
		rep             ComparisonReport
		meta            PRCommentMeta
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "console mode links to storage.cloud.google.com without inline markdown images",
			rep:  changedReport,
			meta: PRCommentMeta{
				CommitSHA:     "head9876543210",
				BaseSHA:       "base1234567890",
				BaseBranch:    "main",
				BuildURL:      "https://console.cloud.google.com/cloud-build/builds/1",
				ReportBaseURL: "https://storage.cloud.google.com/khi-vrt-artifacts/reports/head/1",
				InlineImages:  false,
			},
			wantContains: []string{
				PRCommentMarker,
				"VISUAL DIFF — 1 changed, 1 added, 1 removed",
				"[Open HTML Report (GCS)](https://storage.cloud.google.com/khi-vrt-artifacts/reports/head/1/index.html)",
				"requires authenticated GCP project access",
				"| 1280x720 |",
				"[Composite](https://storage.cloud.google.com/khi-vrt-artifacts/reports/head/1/composite/story/header--default.png)",
				"[Before](https://storage.cloud.google.com/khi-vrt-artifacts/reports/head/1/before/story/header--default.png)",
				"[Diff](https://storage.cloud.google.com/khi-vrt-artifacts/reports/head/1/diff/story/header--default.png)",
				"[After](https://storage.cloud.google.com/khi-vrt-artifacts/reports/head/1/after/story/header--default.png)",
				"- [`New / One`](https://storage.cloud.google.com/khi-vrt-artifacts/reports/head/1/after/story/new--one.png)",
				"- [`Old / One`](https://storage.cloud.google.com/khi-vrt-artifacts/reports/head/1/before/story/old--one.png)",
				"Base (`main`): `base12345678`",
				"[Cloud Build Logs](https://console.cloud.google.com/cloud-build/builds/1)_\n",
			},
			wantNotContains: []string{
				"![Header / Default]",
				"1280x720 → 1280x720",
			},
		},
		{
			name: "public mode embeds inline composite preview images",
			rep:  changedReport,
			meta: PRCommentMeta{
				CommitSHA:     "head9876543210",
				ReportBaseURL: "https://storage.googleapis.com/khi-vrt-artifacts/reports/head/1",
				InlineImages:  true,
			},
			wantContains: []string{
				"![Header / Default](https://storage.googleapis.com/khi-vrt-artifacts/reports/head/1/composite/story/header--default.png)",
			},
			wantNotContains: []string{
				"requires authenticated GCP project access",
			},
		},
		{
			name: "errored run highlights runtime errors",
			rep: ComparisonReport{
				Summary: Summary{Total: 1, Errored: 1},
				Items: []StoryComparison{
					{
						ID:     "story/broken--one",
						Title:  "Broken",
						Name:   "One",
						Status: StatusErrored,
						Error:  "uncaught | error\nline2",
					},
				},
			},
			meta: PRCommentMeta{CommitSHA: "abc123456789"},
			wantContains: []string{
				"❌ ERROR — 1 story error(s)",
				`uncaught \| error line2`,
			},
		},
		{
			name: "skipped run emits concise skipped summary",
			rep:  ComparisonReport{},
			meta: PRCommentMeta{
				CommitSHA:     "abc123456789",
				SkippedReason: "No frontend-affecting files changed compared to `origin/main`.",
			},
			wantContains: []string{
				"⏭️ SKIPPED",
				"No frontend-affecting files changed",
			},
		},
		{
			name: "truncates oversized body to MaxBytes while preserving footer",
			rep:  changedReport,
			meta: PRCommentMeta{
				CommitSHA:     "head9876543210",
				ReportBaseURL: "https://storage.cloud.google.com/khi-vrt-artifacts/reports/head/1",
				MaxBytes:      320,
			},
			wantContains: []string{
				"Report truncated to fit GitHub Check Run payload",
				"Commit: `head98765432`",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatPRCommentBody(tc.rep, tc.meta)
			for _, sub := range tc.wantContains {
				if !strings.Contains(got, sub) {
					t.Errorf("FormatPRCommentBody() missing %q in:\n%s", sub, got)
				}
			}
			for _, sub := range tc.wantNotContains {
				if strings.Contains(got, sub) {
					t.Errorf("FormatPRCommentBody() unexpectedly contained %q in:\n%s", sub, got)
				}
			}
			if tc.meta.MaxBytes > 0 && len(got) > tc.meta.MaxBytes {
				t.Errorf("FormatPRCommentBody() length = %d, exceeds MaxBytes %d", len(got), tc.meta.MaxBytes)
			}
		})
	}
}

func TestCommentEnvelopeRoundTrip(t *testing.T) {
	wantBody := PRCommentMarker + "\n## Frontend Visual Regression Report — ✅ PASS\n"
	var logBuf bytes.Buffer
	logBuf.WriteString("some build output\n")
	WriteCommentEnvelope(&logBuf, wantBody)
	logBuf.WriteString("trailing line\n")

	got, ok, err := ExtractCommentEnvelope(logBuf.String())
	if err != nil {
		t.Fatalf("ExtractCommentEnvelope() unexpected error = %v", err)
	}
	if !ok {
		t.Fatal("ExtractCommentEnvelope() found = false, want true")
	}
	if got != wantBody {
		t.Errorf("ExtractCommentEnvelope() = %q, want %q", got, wantBody)
	}

	_, ok, err = ExtractCommentEnvelope("no envelope here")
	if err != nil || ok {
		t.Errorf("ExtractCommentEnvelope(missing) = (_, %v, %v), want (_, false, nil)", ok, err)
	}

	if _, ok, err := ExtractCommentEnvelope(EnvelopeBeginMarker + "\nunterminated"); err == nil || ok {
		t.Errorf("ExtractCommentEnvelope(unterminated) = (ok=%v, err=%v), want (false, non-nil)", ok, err)
	}

	gotEmpty, ok, err := ExtractCommentEnvelope(EnvelopeBeginMarker + EnvelopeEndMarker)
	if err != nil || !ok || gotEmpty != "" {
		t.Errorf("ExtractCommentEnvelope(empty) = (%q, %v, %v), want (\"\", true, nil)", gotEmpty, ok, err)
	}
}

func TestFormatPRCommentBody_OverflowAndFallbacks(t *testing.T) {
	items := make([]StoryComparison, 0, 50)
	for range 17 {
		items = append(items,
			StoryComparison{
				ID:           "story/changed-many",
				Status:       StatusChanged,
				SizeMismatch: true,
				BeforeWidth:  100,
				BeforeHeight: 100,
				AfterWidth:   200,
				AfterHeight:  150,
				DiffRatio:    0.25,
				DiffPixels:   100,
			},
			StoryComparison{
				ID:     "story/errored-many",
				Status: StatusErrored,
				Error:  strings.Repeat("e", 250),
			},
			StoryComparison{
				ID:     "story/added-many",
				Status: StatusAdded,
			},
		)
	}
	rep := ComparisonReport{
		Summary: Summary{Total: len(items), Changed: 17, Errored: 17, Added: 17},
		Items:   items,
	}

	// Empty ReportBaseURL & empty footer metadata exercise fallback paths.
	body := FormatPRCommentBody(rep, PRCommentMeta{})
	for _, want := range []string{
		"100x100 → 200x150",
		"and 2 more changed stories",
		"and 2 more errored stories._",
		"and 2 more._",
		"- `story/added-many`\n",
		"| — |",
		strings.Repeat("e", 197) + "...",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("FormatPRCommentBody(overflow) missing %q", want)
		}
	}
	if strings.Contains(body, "\n---\n") {
		t.Error("expected empty footer when CommitSHA/BaseSHA/BuildURL are all empty")
	}

	// Inline images enabled on a changed item without CompositeRelPath, and an added item without AfterRelPath.
	noLinksBody := FormatPRCommentBody(ComparisonReport{
		Summary: Summary{Total: 2, Changed: 1, Added: 1},
		Items: []StoryComparison{
			{ID: "story/no-links", Status: StatusChanged},
			{ID: "story/added-nopath", Status: StatusAdded},
		},
	}, PRCommentMeta{
		ReportBaseURL: "https://storage.googleapis.com/b/r",
		InlineImages:  true,
	})
	if !strings.Contains(noLinksBody, "| — |") {
		t.Errorf("expected em-dash links when story has no rel paths:\n%s", noLinksBody)
	}
	if !strings.Contains(noLinksBody, "- `story/added-nopath`\n") {
		t.Errorf("expected plain bullet when added item has empty AfterRelPath:\n%s", noLinksBody)
	}

	// Verify helper boundary conditions directly.
	if got := displayName(StoryComparison{ID: "story/id-only", Title: "TitleOnly"}); got != "story/id-only" {
		t.Errorf("displayName(title only) = %q, want story/id-only", got)
	}
	if got := displayName(StoryComparison{ID: "story/id-only", Name: "NameOnly"}); got != "story/id-only" {
		t.Errorf("displayName(name only) = %q, want story/id-only", got)
	}
	exact200 := strings.Repeat("a", 200)
	if got := escapeMarkdownCell(exact200); got != exact200 {
		t.Errorf("escapeMarkdownCell(200 chars) truncated unexpectedly: len=%d", len(got))
	}
	exactBody := strings.Repeat("x", 150)
	if got := truncateCommentBody(exactBody, 150); got != exactBody {
		t.Errorf("truncateCommentBody(len==maxBytes) modified body")
	}
	const truncSuffix = "\n\n_(Report truncated to fit GitHub Check Run payload — open the GCS HTML report for full details.)_\n"
	if got := truncateCommentBody(exactBody, len(truncSuffix)); got != exactBody {
		t.Errorf("truncateCommentBody(maxBytes==len(suffix)) modified body")
	}
	truncated := truncateCommentBody(strings.Repeat("x", 300), 200)
	if len(truncated) > 200 || !strings.HasSuffix(truncated, truncSuffix) {
		t.Errorf("truncateCommentBody(300, 200) len=%d, want <= 200 with suffix", len(truncated))
	}
}
