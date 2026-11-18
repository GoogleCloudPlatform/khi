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
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type fakeCheckRun struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion,omitempty"`
	DetailsURL string `json:"details_url,omitempty"`
	App        struct {
		Slug string `json:"slug"`
	} `json:"app"`
}

type gateScriptCase struct {
	name           string
	runs           []fakeCheckRun
	wantSkip       bool
	wantMethodLine string
	wantStatus     string
	wantConclusion string
}

func TestSyncCloudBuildGateScript(t *testing.T) {
	repoRoot, err := gotestutil.ResolveRepoRoot("")
	if err != nil {
		t.Fatalf("ResolveRepoRoot() error: %v", err)
	}
	scriptPath := filepath.Join(repoRoot, "scripts", "sync-cloudbuild-gate.sh")

	testCases := []gateScriptCase{
		{
			name: "creates in_progress gate when deploy-ondemand is still running",
			runs: []fakeCheckRun{
				newCBRun(101, "github-codequality (kubernetes-history-inspector)", "completed", "success"),
				newCBRun(102, "github-deploy-ondemand (kubernetes-history-inspector)", "in_progress", ""),
			},
			wantMethodLine: "ARGS: api --method POST repos/GoogleCloudPlatform/khi/check-runs --input",
			wantStatus:     "in_progress",
			wantConclusion: "",
		},
		{
			name: "patches existing in_progress gate to success when all checks pass",
			runs: []fakeCheckRun{
				newCBRun(101, "github-codequality (kubernetes-history-inspector)", "completed", "success"),
				newCBRun(102, "github-deploy-ondemand (kubernetes-history-inspector)", "completed", "success"),
				newActionsGateRun(200, "in_progress", ""),
			},
			wantMethodLine: "ARGS: api --method PATCH repos/GoogleCloudPlatform/khi/check-runs/200 --input",
			wantStatus:     "completed",
			wantConclusion: "success",
		},
		{
			name: "fails fast when codequality fails even if deploy-ondemand has not started",
			runs: []fakeCheckRun{
				newCBRun(101, "github-codequality (kubernetes-history-inspector)", "completed", "failure"),
			},
			wantMethodLine: "ARGS: api --method POST repos/GoogleCloudPlatform/khi/check-runs --input",
			wantStatus:     "completed",
			wantConclusion: "failure",
		},
		{
			name: "creates fresh gate run when rerun succeeds after prior completed failure",
			runs: []fakeCheckRun{
				newCBRun(101, "github-codequality (kubernetes-history-inspector)", "completed", "failure"),
				newCBRun(102, "github-deploy-ondemand (kubernetes-history-inspector)", "completed", "success"),
				newActionsGateRun(200, "completed", "failure"),
				newCBRun(301, "github-codequality (kubernetes-history-inspector)", "completed", "success"),
			},
			wantMethodLine: "ARGS: api --method POST repos/GoogleCloudPlatform/khi/check-runs --input",
			wantStatus:     "completed",
			wantConclusion: "success",
		},
		{
			name: "skips API write when completed gate conclusion already matches",
			runs: []fakeCheckRun{
				newCBRun(101, "github-codequality (kubernetes-history-inspector)", "completed", "success"),
				newCBRun(102, "github-deploy-ondemand (kubernetes-history-inspector)", "completed", "success"),
				newActionsGateRun(200, "completed", "success"),
			},
			wantSkip: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runGateScriptCase(t, scriptPath, tc)
		})
	}
}

func newCBRun(id int, name string, status string, conclusion string) fakeCheckRun {
	var r fakeCheckRun
	r.ID = id
	r.Name = name
	r.Status = status
	r.Conclusion = conclusion
	r.DetailsURL = "https://console.cloud.google.com/cloud-build/builds/1"
	r.App.Slug = "google-cloud-build"
	return r
}

func newActionsGateRun(id int, status string, conclusion string) fakeCheckRun {
	var r fakeCheckRun
	r.ID = id
	r.Name = "cloudbuild-gate"
	r.Status = status
	r.Conclusion = conclusion
	r.App.Slug = "github-actions"
	return r
}

func runGateScriptCase(t *testing.T, scriptPath string, tc gateScriptCase) {
	t.Helper()
	tmpDir := t.TempDir()
	runsFile := filepath.Join(tmpDir, "runs.jsonl")
	recArgsFile := filepath.Join(tmpDir, "gh_args.txt")
	recBodyFile := filepath.Join(tmpDir, "gh_body.json")

	writeRunsJSONL(t, runsFile, tc.runs)
	writeFakeGateGH(t, tmpDir, runsFile, recArgsFile, recBodyFile)

	cmd := exec.CommandContext(context.Background(), "bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"PATH="+tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GITHUB_REPO=GoogleCloudPlatform/khi",
		"HEAD_SHA=abcdef1234567890",
	)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sync-cloudbuild-gate.sh failed: %v\n%s", err, string(outBytes))
	}
	if tc.wantSkip {
		if _, statErr := os.Stat(recArgsFile); !os.IsNotExist(statErr) {
			t.Fatalf("expected no write API call, but gh_args.txt exists")
		}
		return
	}
	verifyRecordedGateCall(t, recArgsFile, recBodyFile, tc)
}

func writeRunsJSONL(t *testing.T, path string, runs []fakeCheckRun) {
	t.Helper()
	var sb strings.Builder
	for _, r := range runs {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("json.Marshal() error: %v", err)
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error: %v", path, err)
	}
}

func writeFakeGateGH(t *testing.T, dir string, runsFile string, recArgsFile string, recBodyFile string) {
	t.Helper()
	script := `#!/bin/bash
set -euo pipefail
if [[ "$*" == *"--paginate repos/GoogleCloudPlatform/khi/commits/abcdef1234567890/check-runs"* ]]; then
  cat "` + runsFile + `"
  exit 0
fi
if [[ "$*" == *"--method "* ]]; then
  echo "ARGS: $*" > "` + recArgsFile + `"
  prev=""
  for arg in "$@"; do
    if [[ "${prev}" == "--input" ]]; then
      cp "${arg}" "` + recBodyFile + `"
    fi
    prev="${arg}"
  done
  exit 0
fi
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(fake gh) error: %v", err)
	}
}

func verifyRecordedGateCall(t *testing.T, recArgsFile string, recBodyFile string, tc gateScriptCase) {
	t.Helper()
	argsBytes, err := os.ReadFile(recArgsFile)
	if err != nil {
		t.Fatalf("ReadFile(recArgsFile) error: %v", err)
	}
	if !strings.Contains(string(argsBytes), tc.wantMethodLine) {
		t.Errorf("recorded args = %q, want substring %q", string(argsBytes), tc.wantMethodLine)
	}
	bodyBytes, err := os.ReadFile(recBodyFile)
	if err != nil {
		t.Fatalf("ReadFile(recBodyFile) error: %v", err)
	}
	var payload struct {
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		t.Fatalf("Unmarshal(payload) error: %v", err)
	}
	if payload.Status != tc.wantStatus {
		t.Errorf("payload.Status = %q, want %q", payload.Status, tc.wantStatus)
	}
	if payload.Conclusion != tc.wantConclusion {
		t.Errorf("payload.Conclusion = %q, want %q", payload.Conclusion, tc.wantConclusion)
	}
}
