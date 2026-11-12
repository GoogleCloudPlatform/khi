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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

func TestEmitOutputsMultiLanguageAndEmpty(t *testing.T) {
	var out bytes.Buffer
	err := emitOutputs(emitRequest{
		flags:     cliFlags{format: "text"},
		baseLabel: "trunk()",
		reports: []*rule.Report{
			{Language: "Go", BaseLabel: "trunk()"},
			{Language: "TypeScript", BaseLabel: "trunk()"},
		},
		colorMode: gotestutil.ColorNever,
		env:       cliEnv{stdout: &out, getenv: func(string) string { return "" }},
	})
	if err != nil {
		t.Fatalf("emitOutputs() error: %v", err)
	}
	if !strings.Contains(out.String(), "Go Quality Report") || !strings.Contains(out.String(), "TypeScript Quality Report") {
		t.Errorf("expected both Go and TypeScript reports in stdout:\n%s", out.String())
	}
}
