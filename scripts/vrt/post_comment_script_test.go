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

	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/report"
)

type postVRTScriptScenario struct {
	mode       string
	conclusion string
	wantSubstr []string
	wantStdout string
}

// TestPostVRTPRCommentScript verifies that post-vrt-pr-comment.sh extracts envelopes and updates PR comments.
func TestPostVRTPRCommentScript(t *testing.T) {
	repoRoot := resolveVRTRepoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "post-vrt-pr-comment.sh")

	t.Run("extracts envelope and creates sticky comment when none exists", func(t *testing.T) {
		runPostVRTScriptScenario(t, scriptPath, postVRTScriptScenario{
			mode:       "create_with_envelope",
			conclusion: "success",
			wantSubstr: []string{
				"POST repos/GoogleCloudPlatform/khi/issues/1280/comments",
				report.PRCommentMarker,
				"Frontend Visual Regression Report",
			},
		})
	})

	t.Run("extracts envelope from Cloud Build step-prefixed logs", func(t *testing.T) {
		runPostVRTScriptScenario(t, scriptPath, postVRTScriptScenario{
			mode:       "create_with_step_prefixed_envelope",
			conclusion: "success",
			wantSubstr: []string{
				"POST repos/GoogleCloudPlatform/khi/issues/1280/comments",
				report.PRCommentMarker,
				"Frontend Visual Regression Report",
			},
		})
	})

	t.Run("updates existing sticky comment with fallback body when envelope absent", func(t *testing.T) {
		runPostVRTScriptScenario(t, scriptPath, postVRTScriptScenario{
			mode:       "update_fallback",
			conclusion: "failure",
			wantSubstr: []string{
				"PATCH repos/GoogleCloudPlatform/khi/issues/comments/5151",
				"Cloud Build `failure`",
				"https://example.com/cb/vrt-1",
			},
		})
	})

	t.Run("skips cleanly when no open PR matches HEAD_SHA", func(t *testing.T) {
		runPostVRTScriptScenario(t, scriptPath, postVRTScriptScenario{
			mode:       "no_open_pr",
			conclusion: "success",
			wantStdout: "No open pull request currently points at HEAD",
		})
	})
}

func runPostVRTScriptScenario(t *testing.T, scriptPath string, sc postVRTScriptScenario) {
	t.Helper()
	tmpDir := t.TempDir()
	recFile := filepath.Join(tmpDir, "gh_calls.txt")
	checkLogFile := filepath.Join(tmpDir, "check_log.txt")

	var logBuf bytes.Buffer
	switch sc.mode {
	case "create_with_envelope":
		body := report.FormatPRCommentBody(report.ComparisonReport{
			Summary: report.Summary{Total: 10, Unchanged: 10},
		}, report.PRCommentMeta{CommitSHA: "abcdef123456"})
		report.WriteCommentEnvelope(&logBuf, body)
	case "create_with_step_prefixed_envelope":
		var raw bytes.Buffer
		body := report.FormatPRCommentBody(report.ComparisonReport{
			Summary: report.Summary{Total: 10, Unchanged: 10},
		}, report.PRCommentMeta{CommitSHA: "abcdef123456"})
		report.WriteCommentEnvelope(&raw, body)
		for _, line := range strings.Split(strings.TrimSuffix(raw.String(), "\n"), "\n") {
			logBuf.WriteString(`Step #1 - "vrt": ` + line + "\n")
		}
	default:
		logBuf.WriteString("Step #0: storybook build failed\n")
	}
	if err := os.WriteFile(checkLogFile, logBuf.Bytes(), 0o600); err != nil {
		t.Fatalf("WriteFile(checkLogFile) error: %v", err)
	}

	writeFakeVRTGHCLI(t, tmpDir, recFile, checkLogFile, sc.mode)

	cmd := exec.CommandContext(context.Background(), "/bin/bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"PATH="+tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GITHUB_REPO=GoogleCloudPlatform/khi",
		"CHECK_RUN_ID=77777",
		"HEAD_SHA=abcdef1234567890",
		"CONCLUSION="+sc.conclusion,
		"DETAILS_URL=https://example.com/cb/vrt-1",
	)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("post-vrt-pr-comment.sh failed: %v\n%s", err, string(outBytes))
	}
	if sc.wantStdout != "" && !strings.Contains(string(outBytes), sc.wantStdout) {
		t.Errorf("stdout = %q, want substring %q", string(outBytes), sc.wantStdout)
	}
	if len(sc.wantSubstr) > 0 {
		recorded, readErr := os.ReadFile(recFile)
		if readErr != nil {
			t.Fatalf("ReadFile(recFile) error: %v", readErr)
		}
		for _, sub := range sc.wantSubstr {
			if !strings.Contains(string(recorded), sub) {
				t.Errorf("recorded gh call missing %q:\n%s", sub, string(recorded))
			}
		}
	}
}

func writeFakeVRTGHCLI(t *testing.T, dir string, recFile string, checkLogFile string, mode string) {
	t.Helper()
	fakeGH := `#!/bin/bash
set -euo pipefail
case "$*" in
  *"commits/abcdef1234567890/pulls"*)
    if [[ "` + mode + `" == "no_open_pr" ]]; then
      exit 0
    fi
    echo "1280"
    ;;
  *"check-runs/77777"*)
    cat "` + checkLogFile + `"
    ;;
  *"--paginate repos/GoogleCloudPlatform/khi/issues/1280/comments"*)
    if [[ "` + mode + `" == "update_fallback" ]]; then
      echo "5151"
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

func resolveVRTRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("failed to locate repository go.mod from %s", dir)
		}
		dir = parent
	}
}
