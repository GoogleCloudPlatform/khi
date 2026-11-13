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

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func TestFormatPRCommentBody(t *testing.T) {
	t.Run("empty reports with commit and build metadata", testFormatCommentEmptyReports)
	t.Run("multiple language reports exact formatting and metadata omission", testFormatCommentMultiReports)
	t.Run("truncates body cleanly at UTF-8 boundary and preserves short body at exact cap", testFormatCommentTruncation)
}

func testFormatCommentEmptyReports(t *testing.T) {
	t.Helper()
	got := FormatPRCommentBody("origin/main", nil, PRCommentMeta{
		CommitSHA: "964c03d6e66d24abcdef",
		BuildURL:  "https://console.cloud.google.com/cloud-build/builds/b1",
	})
	want := PRCommentMarker + "\n" +
		"## Code Quality Report (`origin/main`) — ✅ PASS\n\n" +
		"No Go or TypeScript files were modified in this change.\n" +
		"\n---\n_Commit: `964c03d6e66d` · [Cloud Build Logs](https://console.cloud.google.com/cloud-build/builds/b1)_\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("FormatPRCommentBody(empty) mismatch (-want +got):\n%s", diff)
	}
}

func testFormatCommentMultiReports(t *testing.T) {
	t.Helper()
	reports := []*rule.Report{
		{Language: "Go", BaseLabel: "origin/main", ProdFilesChanged: 1},
		{Language: "TypeScript", BaseLabel: "origin/main", Advisory: true, ProdFilesChanged: 2},
	}
	got := FormatPRCommentBody("origin/main", reports, PRCommentMeta{CommitSHA: "short123"})
	want := PRCommentMarker + "\n" +
		"## Go Quality Report (`origin/main`) — ✅ PASS\n\n" +
		"| Metric | Result |\n" +
		"| :--- | :--- |\n" +
		"| Changed Scope | 1 prod file(s), 0 test file(s), 0 unit(s) |\n\n\n" +
		"## TypeScript Quality Report (`origin/main`) — ✅ PASS\n\n" +
		"| Metric | Result |\n" +
		"| :--- | :--- |\n" +
		"| Changed Scope | 2 prod file(s), 0 test file(s), 0 unit(s) |\n\n" +
		"\n---\n_Commit: `short123`_\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("FormatPRCommentBody(multi) mismatch (-want +got):\n%s", diff)
	}

	noMeta := FormatPRCommentBody("origin/main", reports[:1], PRCommentMeta{})
	if strings.Contains(noMeta, "\n---\n") {
		t.Errorf("expected no footer separator when metadata is empty:\n%s", noMeta)
	}
}

func testFormatCommentTruncation(t *testing.T) {
	t.Helper()
	const suffix = "\n\n_(Report truncated to fit GitHub Check Run payload — see Cloud Build logs for full output.)_\n"

	sample := "hello world"
	if got := truncateCommentBody(sample, 200); got != sample {
		t.Errorf("truncateCommentBody(short) = %q, want %q", got, sample)
	}
	exactBody := strings.Repeat("x", len(suffix)+10)
	if got := truncateCommentBody(exactBody, len(exactBody)); got != exactBody {
		t.Errorf("truncateCommentBody(exact cap) truncated unexpectedly")
	}
	if got := truncateCommentBody(exactBody, len(suffix)); got != exactBody {
		t.Errorf("truncateCommentBody(cap == suffix len) truncated unexpectedly")
	}

	// Multibyte UTF-8 rune ("あ", 3 bytes) straddling cut index drops incomplete trailing byte cleanly.
	utf8Body := strings.Repeat("a", 20) + "あ" + strings.Repeat("b", 100)
	truncated := truncateCommentBody(utf8Body, len(suffix)+21)
	wantTrunc := strings.Repeat("a", 20) + suffix
	if diff := cmp.Diff(wantTrunc, truncated); diff != "" {
		t.Errorf("truncateCommentBody(utf8) mismatch (-want +got):\n%s", diff)
	}

	capped := FormatPRCommentBody("origin/main", nil, PRCommentMeta{
		CommitSHA: "abcdef123456",
		MaxBytes:  len(suffix) + 50,
	})
	if !strings.Contains(capped, "Report truncated") || !strings.Contains(capped, "Commit: `abcdef123456`") {
		t.Errorf("unexpected capped comment:\n%s", capped)
	}
}

func TestCommentEnvelopeRoundTripAndErrors(t *testing.T) {
	t.Run("round-trips multi-line body embedded in Cloud Build logs", testEnvelopeRoundTrip)
	t.Run("handles adjacent markers absent markers and malformed payloads", testEnvelopeBoundariesAndErrors)
}

func testEnvelopeRoundTrip(t *testing.T) {
	t.Helper()
	original := FormatPRCommentBody("origin/main", []*rule.Report{
		{Language: "Go", BaseLabel: "origin/main", ProdFilesChanged: 3, FuncsEvaluated: 9},
	}, PRCommentMeta{
		CommitSHA: "964c03d6e66d24",
		BuildURL:  "https://console.cloud.google.com/cloud-build/builds/123",
	})

	var logBuf bytes.Buffer
	logBuf.WriteString("Step #0: some build output\n")
	WriteCommentEnvelope(&logBuf, original)
	logBuf.WriteString("PUSH\nDONE\n```\n")

	decoded, found, err := ExtractCommentEnvelope(logBuf.String())
	if err != nil || !found {
		t.Fatalf("ExtractCommentEnvelope() = (found=%v, err=%v), want (true, nil)", found, err)
	}
	if diff := cmp.Diff(original, decoded); diff != "" {
		t.Errorf("decoded envelope mismatch (-want +got):\n%s", diff)
	}
}

func testEnvelopeBoundariesAndErrors(t *testing.T) {
	t.Helper()
	// Adjacent markers without newline (endIdx == 0) decode to empty string (kills endIdx < 0 -> <= 0 mutant).
	gotAdj, okAdj, errAdj := ExtractCommentEnvelope(EnvelopeBeginMarker + EnvelopeEndMarker)
	if errAdj != nil || !okAdj || gotAdj != "" {
		t.Errorf("adjacent markers = (%q, %v, %v), want (\"\", true, nil)", gotAdj, okAdj, errAdj)
	}

	// Begin marker at byte 0 (beginIdx == 0) decodes cleanly (kills beginIdx < 0 -> <= 0 mutant).
	gotZero, okZero, errZero := ExtractCommentEnvelope(EnvelopeBeginMarker + "\nYWJj\n" + EnvelopeEndMarker)
	if errZero != nil || !okZero || gotZero != "abc" {
		t.Errorf("beginIdx 0 = (%q, %v, %v), want (\"abc\", true, nil)", gotZero, okZero, errZero)
	}

	// Absent marker returns (empty, false, nil).
	if _, found, err := ExtractCommentEnvelope("plain build log without marker"); err != nil || found {
		t.Errorf("absent marker = (found=%v, err=%v), want (false, nil)", found, err)
	}

	// Unterminated begin marker returns error.
	if _, found, err := ExtractCommentEnvelope(EnvelopeBeginMarker + "\nYWJj"); err == nil || found {
		t.Errorf("unterminated marker = (found=%v, err=%v), want (false, error)", found, err)
	}

	// Invalid base64 payload returns decode error.
	bad := EnvelopeBeginMarker + "\n!@#$not-base64\n" + EnvelopeEndMarker
	if _, found, err := ExtractCommentEnvelope(bad); err == nil || found {
		t.Errorf("invalid base64 = (found=%v, err=%v), want (false, error)", found, err)
	}
}
