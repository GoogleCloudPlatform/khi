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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/report"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type postScriptScenario struct {
	name             string
	mode             string
	conclusion       string
	decisionsSummary string
	wantSubstr       []string
	wantStdout       string
}

func TestPostCodequalityPRCommentScript(t *testing.T) {
	repoRoot, err := gotestutil.ResolveRepoRoot("")
	if err != nil {
		t.Fatalf("ResolveRepoRoot() error: %v", err)
	}
	scriptPath := filepath.Join(repoRoot, "scripts", "post-codequality-pr-comment.sh")

	testCases := []postScriptScenario{
		// Decodes the Base64 envelope, splices DECISIONS_SUMMARY_FILE before the footer, and creates a new sticky comment.
		{
			name:             "extracts envelope and creates sticky comment when none exists",
			mode:             "create_with_envelope",
			conclusion:       "success",
			decisionsSummary: "## Decision Rules Report (`origin/main`) — ⚠️ ADVISORY (1 finding(s))\n",
			wantSubstr:       []string{"POST repos/GoogleCloudPlatform/khi/issues/1273/comments", report.PRCommentMarker, "Go Quality Report", "Decision Rules Report", "Commit: `abcdef123456`"},
		},
		// Strips Cloud Build step prefixes from each envelope line before decoding and creating the sticky comment.
		{
			name:       "extracts envelope from Cloud Build step-prefixed logs",
			mode:       "create_with_step_prefixed_envelope",
			conclusion: "success",
			wantSubstr: []string{"POST repos/GoogleCloudPlatform/khi/issues/1273/comments", report.PRCommentMarker, "Go Quality Report"},
		},
		// Falls back to the missing-envelope summary when the Base64 payload inside the envelope markers is corrupted.
		{
			name:       "falls back cleanly when envelope payload is corrupted",
			mode:       "corrupted_envelope",
			conclusion: "success",
			wantSubstr: []string{"POST repos/GoogleCloudPlatform/khi/issues/1273/comments", report.PRCommentMarker, "Cloud Build completed without an embedded report envelope."},
		},
		// Updates an existing sticky comment with a Cloud Build failure fallback message when no envelope is present.
		{
			name:       "updates existing sticky comment with fallback body when envelope absent",
			mode:       "update_fallback",
			conclusion: "failure",
			wantSubstr: []string{"PATCH repos/GoogleCloudPlatform/khi/issues/comments/4242", "Cloud Build `failure`", "https://example.com/cb/1"},
		},
		// Exits cleanly without posting a comment when no open pull request matches HEAD_SHA.
		{
			name:       "skips cleanly when no open PR matches HEAD_SHA",
			mode:       "no_open_pr",
			conclusion: "success",
			wantStdout: "No open pull request currently points at HEAD",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runPostCommentScriptScenario(t, scriptPath, tc)
		})
	}
}

func runPostCommentScriptScenario(t *testing.T, scriptPath string, sc postScriptScenario) {
	t.Helper()
	tmpDir := t.TempDir()
	recFile := filepath.Join(tmpDir, "gh_calls.txt")
	checkLogFile := writeCheckRunLogFixture(t, tmpDir, sc.mode)
	writeFakeGHCLI(t, tmpDir, recFile, checkLogFile, sc.mode)
	decFile := writeOptionalDecisionsSummary(t, tmpDir, sc.decisionsSummary)

	cmd := exec.CommandContext(context.Background(), "/bin/bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"PATH="+tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GITHUB_REPO=GoogleCloudPlatform/khi",
		"CHECK_RUN_ID=98765",
		"HEAD_SHA=abcdef1234567890",
		"CONCLUSION="+sc.conclusion,
		"DETAILS_URL=https://example.com/cb/1",
		"DECISIONS_SUMMARY_FILE="+decFile,
	)
	outBytes, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("post-codequality-pr-comment.sh failed: %v\n%s", err, string(outBytes))
	}
	assertPostCommentScriptResult(t, string(outBytes), recFile, sc)
}

func writeCheckRunLogFixture(t *testing.T, tmpDir string, mode string) string {
	t.Helper()
	checkLogFile := filepath.Join(tmpDir, "check_log.txt")
	var logBuf bytes.Buffer
	switch mode {
	case "create_with_envelope":
		body := report.FormatPRCommentBody("origin/main", []*rule.Report{
			{Language: "Go", BaseLabel: "origin/main", ProdFilesChanged: 1},
		}, report.PRCommentMeta{CommitSHA: "abcdef123456"})
		report.WriteCommentEnvelope(&logBuf, body)
	case "create_with_step_prefixed_envelope":
		var raw bytes.Buffer
		body := report.FormatPRCommentBody("origin/main", []*rule.Report{
			{Language: "Go", BaseLabel: "origin/main", ProdFilesChanged: 1},
		}, report.PRCommentMeta{CommitSHA: "abcdef123456"})
		report.WriteCommentEnvelope(&raw, body)
		for _, line := range strings.Split(strings.TrimSuffix(raw.String(), "\n"), "\n") {
			logBuf.WriteString(`Step #1 - "codequality": ` + line + "\n")
		}
	case "corrupted_envelope":
		logBuf.WriteString("KHI_CODEQUALITY_B64_BEGIN\nnot!valid!base64\nKHI_CODEQUALITY_B64_END\n")
	default:
		logBuf.WriteString("Step #0: compilation failed\n")
	}
	if err := os.WriteFile(checkLogFile, logBuf.Bytes(), 0o600); err != nil {
		t.Fatalf("WriteFile(checkLogFile) error: %v", err)
	}
	return checkLogFile
}

func writeOptionalDecisionsSummary(t *testing.T, tmpDir string, summary string) string {
	t.Helper()
	if summary == "" {
		return ""
	}
	decFile := filepath.Join(tmpDir, "decisions.md")
	if err := os.WriteFile(decFile, []byte(summary), 0o600); err != nil {
		t.Fatalf("WriteFile(decFile) error: %v", err)
	}
	return decFile
}

func assertPostCommentScriptResult(t *testing.T, stdout string, recFile string, sc postScriptScenario) {
	t.Helper()
	if sc.wantStdout != "" && !strings.Contains(stdout, sc.wantStdout) {
		t.Errorf("stdout = %q, want substring %q", stdout, sc.wantStdout)
	}
	if len(sc.wantSubstr) == 0 {
		return
	}
	recorded, err := os.ReadFile(recFile)
	if err != nil {
		t.Fatalf("ReadFile(recFile) error: %v", err)
	}
	for _, sub := range sc.wantSubstr {
		if !strings.Contains(string(recorded), sub) {
			t.Errorf("recorded gh call missing %q:\n%s", sub, string(recorded))
		}
	}
}

func writeFakeGHCLI(t *testing.T, dir string, recFile string, checkLogFile string, mode string) {
	t.Helper()
	fakeGH := `#!/bin/bash
set -euo pipefail
case "$*" in
  *"commits/abcdef1234567890/pulls"*)
    if [[ "` + mode + `" == "no_open_pr" ]]; then
      exit 0
    fi
    echo "1273"
    ;;
  *"check-runs/98765"*)
    cat "` + checkLogFile + `"
    ;;
  *"--paginate repos/GoogleCloudPlatform/khi/issues/1273/comments"*)
    if [[ "` + mode + `" == "update_fallback" ]]; then
      echo "4242"
    fi
    ;;
  *"--method "*)
    {
      echo "ARGS: $*"
      cat
      echo ""
    } >> "` + recFile + `"
    ;;
esac
`
	ghPath := filepath.Join(dir, "gh")
	if err := os.WriteFile(ghPath, []byte(fakeGH), 0o755); err != nil {
		t.Fatalf("WriteFile(fake gh) error: %v", err)
	}
}
