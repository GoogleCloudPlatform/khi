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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/report"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type emitRequest struct {
	flags     cliFlags
	baseLabel string
	reports   []*rule.Report
	qaReport  *rule.QAReport
	colorMode gotestutil.ColorMode
	env       cliEnv
}

func emitOutputs(_ context.Context, req emitRequest) error {
	if req.flags.format == "json" {
		return report.RenderQAJSON(req.env.stdout, req.qaReport)
	}
	if req.flags.subcommand == subcmdQA {
		emitQAOutputs(req)
	} else {
		emitCheckOutputs(req)
	}
	if req.flags.commentEnvelope {
		body := report.FormatPRCommentBody(req.baseLabel, req.reports, report.PRCommentMeta{
			CommitSHA: req.flags.commitSHA,
			BuildURL:  req.flags.buildURL,
		})
		report.WriteCommentEnvelope(req.env.stdout, body)
	}
	if req.flags.summaryFile != "" {
		if req.flags.filter == string(rule.FilterAdvisory) && len(req.reports) == 0 {
			return nil
		}
		return writeSummaryFile(req.flags.summaryFile, req.baseLabel, req.reports)
	}
	return nil
}

func emitQAOutputs(req emitRequest) {
	if req.flags.format == "github" {
		report.RenderQAGitHub(req.env.stdout, req.qaReport)
		return
	}
	style := gotestutil.TermStyle{Enabled: gotestutil.ResolveColorEnabled(req.colorMode, req.env.stdout, req.env.getenv)}
	report.RenderQAText(req.env.stdout, req.qaReport, style)
}

func emitCheckOutputs(req emitRequest) {
	if req.flags.format == "github" {
		for _, rep := range req.reports {
			report.RenderGitHubAnnotations(req.env.stdout, rep, req.flags.maxAnnotations)
		}
		return
	}
	emitTextOutputs(req)
}

func emitTextOutputs(req emitRequest) {
	style := gotestutil.TermStyle{Enabled: gotestutil.ResolveColorEnabled(req.colorMode, req.env.stdout, req.env.getenv)}
	if len(req.reports) == 0 {
		msg := fmt.Sprintf("Code Quality Report (base: %s): no Go or TypeScript files changed.", req.baseLabel)
		if req.flags.filter == string(rule.FilterAdvisory) {
			msg = fmt.Sprintf("Decision Rules Report (base: %s): no matching targets changed.", req.baseLabel)
		}
		fmt.Fprintf(req.env.stdout, "%s\n", style.Dim(msg))
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
