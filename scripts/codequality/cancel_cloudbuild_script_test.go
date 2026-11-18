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
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type fakeCloudBuild struct {
	ID             string `json:"id"`
	BuildTriggerID string `json:"buildTriggerId"`
	CreateTime     string `json:"createTime"`
}

type cancelScriptCase struct {
	name            string
	prNumber        string
	commitSHA       string
	remoteHeadSHA   string
	prAPIState      string
	prAPIFail       bool
	currentBuild    fakeCloudBuild
	ongoingBuilds   []fakeCloudBuild
	wantErr         bool
	wantCancelledID []string
}

func TestCancelSupersededCloudBuildScript(t *testing.T) {
	repoRoot, err := gotestutil.ResolveRepoRoot("")
	if err != nil {
		t.Fatalf("ResolveRepoRoot() error: %v", err)
	}
	scriptPath := filepath.Join(repoRoot, "scripts", "cancel-superseded-cloudbuild.sh")

	testCases := []cancelScriptCase{
		{
			name:            "skips cleanly for non-PR builds",
			prNumber:        "",
			commitSHA:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			wantErr:         false,
			wantCancelledID: nil,
		},
		{
			name:            "cancels self when PR head moved to newer commit",
			prNumber:        "1290",
			commitSHA:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			remoteHeadSHA:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			currentBuild:    fakeCloudBuild{ID: "build-old"},
			wantErr:         true,
			wantCancelledID: []string{"build-old"},
		},
		{
			name:            "cancels self when PR is already closed",
			prNumber:        "1279",
			commitSHA:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			remoteHeadSHA:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			prAPIState:      "closed",
			currentBuild:    fakeCloudBuild{ID: "build-merged"},
			wantErr:         true,
			wantCancelledID: []string{"build-merged"},
		},
		{
			name:          "cancels self when newer ongoing build exists for same trigger and PR",
			prNumber:      "1290",
			commitSHA:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			remoteHeadSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			prAPIState:    "open",
			currentBuild: fakeCloudBuild{
				ID:             "build-1",
				BuildTriggerID: "trig-1",
				CreateTime:     "2026-11-18T04:00:00.100000Z",
			},
			ongoingBuilds: []fakeCloudBuild{
				{ID: "build-1", BuildTriggerID: "trig-1", CreateTime: "2026-11-18T04:00:00.100000Z"},
				{ID: "build-2", BuildTriggerID: "trig-1", CreateTime: "2026-11-18T04:00:05.000000Z"},
			},
			wantErr:         true,
			wantCancelledID: []string{"build-1"},
		},
		{
			name:          "cancels older ongoing builds when current build is newest even if GitHub API fails",
			prNumber:      "1290",
			commitSHA:     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			remoteHeadSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			prAPIFail:     true,
			currentBuild: fakeCloudBuild{
				ID:             "build-3",
				BuildTriggerID: "trig-1",
				CreateTime:     "2026-11-18T04:05:00.000000Z",
			},
			ongoingBuilds: []fakeCloudBuild{
				{ID: "build-1", BuildTriggerID: "trig-1", CreateTime: "2026-11-18T04:00:00.000000Z"},
				{ID: "build-2", BuildTriggerID: "trig-1", CreateTime: "2026-11-18T04:02:00.000000Z"},
				{ID: "build-3", BuildTriggerID: "trig-1", CreateTime: "2026-11-18T04:05:00.000000Z"},
			},
			wantErr:         false,
			wantCancelledID: []string{"build-1", "build-2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotCancelled, out, runErr := executeCancelScriptCase(t, scriptPath, tc)
			if (runErr != nil) != tc.wantErr {
				t.Fatalf("script error = %v, wantErr %v\noutput:\n%s", runErr, tc.wantErr, out)
			}
			if diff := cmp.Diff(tc.wantCancelledID, gotCancelled); diff != "" {
				t.Errorf("cancelled build IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func executeCancelScriptCase(t *testing.T, scriptPath string, tc cancelScriptCase) ([]string, string, error) {
	t.Helper()
	binDir := t.TempDir()
	cancelLog := filepath.Join(binDir, "cancelled.txt")
	describePath := filepath.Join(binDir, "describe.json")
	listPath := filepath.Join(binDir, "list.json")

	writeJSONFixture(t, describePath, tc.currentBuild)
	writeJSONFixture(t, listPath, tc.ongoingBuilds)
	writeFakeCancelToolchain(t, binDir, cancelLog, describePath, listPath, tc)

	buildID := tc.currentBuild.ID
	if buildID == "" {
		buildID = "build-default"
	}
	cmd := exec.CommandContext(t.Context(), "bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"PROJECT_ID=kubernetes-history-inspector",
		"BUILD_ID="+buildID,
		"LOCATION=asia-northeast1",
		"PR_NUMBER="+tc.prNumber,
		"COMMIT_SHA="+tc.commitSHA,
	)
	outBytes, runErr := cmd.CombinedOutput()
	return readCancelLines(t, cancelLog), string(outBytes), runErr
}

func writeJSONFixture(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error: %v", path, err)
	}
}

func writeFakeCancelToolchain(t *testing.T, binDir string, cancelLog string, describePath string, listPath string, tc cancelScriptCase) {
	t.Helper()
	fakeGit := `#!/bin/bash
set -euo pipefail
if [[ "${1:-}" == "ls-remote" && -n "` + tc.remoteHeadSHA + `" ]]; then
  printf '%s\trefs/pull/%s/head\n' "` + tc.remoteHeadSHA + `" "` + tc.prNumber + `"
fi
`
	curlBody := `printf '{"state":"%s"}\n' "` + tc.prAPIState + `"`
	if tc.prAPIFail {
		curlBody = `exit 22`
	}
	fakeCurl := `#!/bin/bash
set -euo pipefail
` + curlBody + `
`
	fakeGcloud := `#!/bin/bash
set -euo pipefail
sub="${2:-}"
case "${sub}" in
  describe)
    cat "` + describePath + `"
    ;;
  list)
    cat "` + listPath + `"
    ;;
  cancel)
    echo "${3}" >> "` + cancelLog + `"
    ;;
esac
`
	writeExecFile(t, filepath.Join(binDir, "git"), fakeGit)
	writeExecFile(t, filepath.Join(binDir, "curl"), fakeCurl)
	writeExecFile(t, filepath.Join(binDir, "gcloud"), fakeGcloud)
}

func writeExecFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%s) error: %v", path, err)
	}
}

func readCancelLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("Open(%s) error: %v", path, err)
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			out = append(out, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}
	return out
}
