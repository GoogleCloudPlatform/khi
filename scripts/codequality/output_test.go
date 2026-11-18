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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/report"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

// TestEmitOutputsMultiLanguageAndEnvelope verifies that emitOutputs renders multi-language text reports and optional Base64 PR comment envelopes.
func TestEmitOutputsMultiLanguageAndEnvelope(t *testing.T) {
	t.Run("renders multi-language text reports without envelope when disabled", testEmitMultiLanguageText)
	t.Run("appends decodable Base64 comment envelope when enabled", testEmitCommentEnvelope)
}

func TestEmitOutputsEmptyReportsText(t *testing.T) {
	testCases := []struct {
		name          string
		decisionsOnly bool
		want          string
	}{
		// Prints the decision-specific no-targets message when decisions-only produces zero reports.
		{
			name:          "decisions-only empty reports",
			decisionsOnly: true,
			want:          "Decision Rules Report (base: origin/main): no matching targets changed.\n",
		},
		// Prints the Go/TypeScript no-changes message when standard mode produces zero reports.
		{
			name:          "standard empty reports",
			decisionsOnly: false,
			want:          "Code Quality Report (base: origin/main): no Go or TypeScript files changed.\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer

			err := emitOutputs(context.Background(), emitRequest{
				flags:     cliFlags{format: "text", decisionsOnly: tc.decisionsOnly},
				baseLabel: "origin/main",
				colorMode: gotestutil.ColorNever,
				env:       cliEnv{stdout: &out, stderr: &out, getenv: func(string) string { return "" }},
			})

			if err != nil {
				t.Fatalf("emitOutputs() error: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("stdout = %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestEmitOutputsDecisionsOnlyEmptySkipsSummaryFile verifies that emitOutputs does not create a summary file when decisions-only produces zero reports.
func TestEmitOutputsDecisionsOnlyEmptySkipsSummaryFile(t *testing.T) {
	summaryPath := filepath.Join(t.TempDir(), "summary.md")
	var out bytes.Buffer

	err := emitOutputs(context.Background(), emitRequest{
		flags:     cliFlags{format: "text", decisionsOnly: true, summaryFile: summaryPath},
		baseLabel: "origin/main",
		reports:   nil,
		colorMode: gotestutil.ColorNever,
		env:       cliEnv{stdout: &out, stderr: &out, getenv: func(string) string { return "" }},
	})

	if err != nil {
		t.Fatalf("emitOutputs() error: %v", err)
	}
	if _, statErr := os.Stat(summaryPath); !os.IsNotExist(statErr) {
		t.Errorf("os.Stat(%s) error = %v, want not-exist", summaryPath, statErr)
	}
}

func TestEmitOutputsSummaryFile(t *testing.T) {
	testCases := []struct {
		name          string
		decisionsOnly bool
		reports       []*rule.Report
		wantSubstr    string
	}{
		// Writes the decision summary file when decisions-only produces at least one report.
		{
			name:          "decisions-only non-empty reports writes summary file",
			decisionsOnly: true,
			reports: []*rule.Report{
				{
					Language:  "Decisions",
					BaseLabel: "origin/main",
					Decisions: rule.DecisionSummary{Enabled: true, RulesEvaluated: 1, TargetsEvaluated: 1},
				},
			},
			wantSubstr: "## Decision Rules Report",
		},
		// Writes the empty-changes summary message when standard mode produces zero reports.
		{
			name:          "standard empty reports writes no-changes summary",
			decisionsOnly: false,
			reports:       nil,
			wantSubstr:    "No Go or TypeScript files were modified",
		},
		// Separates multiple summary sections with a blank line when multiple reports are present.
		{
			name:          "multi-report summary separates sections",
			decisionsOnly: false,
			reports: []*rule.Report{
				{Language: "Go", BaseLabel: "origin/main"},
				{Language: "TypeScript", BaseLabel: "origin/main"},
			},
			wantSubstr: "\n\n## TypeScript Quality Report",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			summaryPath := filepath.Join(t.TempDir(), "summary.md")
			var out bytes.Buffer

			err := emitOutputs(context.Background(), emitRequest{
				flags:     cliFlags{format: "text", decisionsOnly: tc.decisionsOnly, summaryFile: summaryPath},
				baseLabel: "origin/main",
				reports:   tc.reports,
				colorMode: gotestutil.ColorNever,
				env:       cliEnv{stdout: &out, stderr: &out, getenv: func(string) string { return "" }},
			})

			if err != nil {
				t.Fatalf("emitOutputs() error: %v", err)
			}
			content, readErr := os.ReadFile(summaryPath)
			if readErr != nil {
				t.Fatalf("ReadFile(%s) error: %v", summaryPath, readErr)
			}
			if !strings.Contains(string(content), tc.wantSubstr) {
				t.Errorf("summary missing %q:\n%s", tc.wantSubstr, string(content))
			}
		})
	}
}

// testEmitMultiLanguageText verifies that emitOutputs separates multiple text reports with a blank line and no leading newline.
func testEmitMultiLanguageText(t *testing.T) {
	t.Helper()
	var out bytes.Buffer
	err := emitOutputs(context.Background(), emitRequest{
		flags:     cliFlags{format: "text"},
		baseLabel: "trunk()",
		reports: []*rule.Report{
			{Language: "Go", BaseLabel: "trunk()"},
			{Language: "TypeScript", BaseLabel: "trunk()"},
		},
		colorMode: gotestutil.ColorNever,
		env:       cliEnv{stdout: &out, stderr: &out, getenv: func(string) string { return "" }},
	})
	if err != nil {
		t.Fatalf("emitOutputs() error: %v", err)
	}
	got := out.String()
	if strings.HasPrefix(got, "\n") {
		t.Errorf("unexpected leading blank line in multi-report text output:\n%q", got)
	}
	if !strings.Contains(got, "Go Quality Report") || !strings.Contains(got, "\n\n══ TypeScript Quality Report") {
		t.Errorf("expected both Go and TypeScript reports separated by blank line in stdout:\n%s", got)
	}
	if strings.Contains(got, report.EnvelopeBeginMarker) {
		t.Errorf("unexpected comment envelope when commentEnvelope=false:\n%s", got)
	}
}

func testEmitCommentEnvelope(t *testing.T) {
	t.Helper()
	var out bytes.Buffer
	err := emitOutputs(context.Background(), emitRequest{
		flags: cliFlags{
			format:          "text",
			commentEnvelope: true,
			commitSHA:       "964c03d6e66d24",
			buildURL:        "https://console.cloud.google.com/cloud-build/builds/b99",
		},
		baseLabel: "origin/main",
		reports:   []*rule.Report{{Language: "Go", BaseLabel: "origin/main", ProdFilesChanged: 2}},
		colorMode: gotestutil.ColorNever,
		env:       cliEnv{stdout: &out, stderr: &out, getenv: func(string) string { return "" }},
	})
	if err != nil {
		t.Fatalf("emitOutputs(commentEnvelope) error: %v", err)
	}
	decoded, found, decErr := report.ExtractCommentEnvelope(out.String())
	if decErr != nil || !found {
		t.Fatalf("ExtractCommentEnvelope() = (found=%v, err=%v)", found, decErr)
	}
	if !strings.Contains(decoded, report.PRCommentMarker) || !strings.Contains(decoded, "964c03d6e66d") || !strings.Contains(decoded, "builds/b99") {
		t.Errorf("unexpected decoded envelope:\n%s", decoded)
	}
}
