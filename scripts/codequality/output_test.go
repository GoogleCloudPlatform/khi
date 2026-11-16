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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/report"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

func TestEmitOutputsMultiLanguageAndEnvelope(t *testing.T) {
	t.Run("renders multi-language text reports without envelope when disabled", testEmitMultiLanguageText)
	t.Run("appends decodable Base64 comment envelope when enabled", testEmitCommentEnvelope)
}

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
	if !strings.Contains(got, "Go Quality Report") || !strings.Contains(got, "TypeScript Quality Report") {
		t.Errorf("expected both Go and TypeScript reports in stdout:\n%s", got)
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
