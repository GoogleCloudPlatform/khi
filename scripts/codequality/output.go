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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/report"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

type emitRequest struct {
	flags     cliFlags
	baseLabel string
	reports   []*rule.Report
	colorMode gotestutil.ColorMode
	env       cliEnv
}

func emitOutputs(req emitRequest) error {
	if req.flags.format == "github" {
		for _, rep := range req.reports {
			report.RenderGitHubAnnotations(req.env.stdout, rep, req.flags.maxAnnotations)
		}
	} else {
		emitTextOutputs(req)
	}
	if req.flags.summaryFile != "" {
		return writeSummaryFile(req.flags.summaryFile, req.baseLabel, req.reports)
	}
	return nil
}

func emitTextOutputs(req emitRequest) {
	style := gotestutil.TermStyle{Enabled: gotestutil.ResolveColorEnabled(req.colorMode, req.env.stdout, req.env.getenv)}
	if len(req.reports) == 0 {
		fmt.Fprintf(req.env.stdout, "%s\n", style.Dim(fmt.Sprintf("Code Quality Report (base: %s): no Go or TypeScript files changed.", req.baseLabel)))
		return
	}
	for i, rep := range req.reports {
		if i > 0 {
			fmt.Fprintln(req.env.stdout)
		}
		report.RenderText(req.env.stdout, rep, style)
	}
}

func writeSummaryFile(summaryPath string, baseLabel string, reports []*rule.Report) error {
	var buf bytes.Buffer
	if len(reports) == 0 {
		fmt.Fprintf(&buf, "## Code Quality Report (`%s`) — ✅ PASS\n\nNo Go or TypeScript files were modified in this change.\n", baseLabel)
	} else {
		for i, rep := range reports {
			if i > 0 {
				fmt.Fprintln(&buf)
			}
			report.RenderGitHubSummary(&buf, rep)
		}
	}
	return gotestutil.AppendSummaryFile(summaryPath, buf.Bytes())
}
