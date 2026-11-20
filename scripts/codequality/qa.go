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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/qarunner"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func evaluateQASubcommand(ctx context.Context, req checkStagesRequest) ([]*rule.Report, *rule.QAReport, error) {
	trustedRoot, err := resolveTrustedRoot(req.repoRoot, req.flags.trustedRoot)
	if err != nil {
		return nil, nil, err
	}

	var collectedReports []*rule.Report
	qaRep, err := qarunner.Run(ctx, qarunner.RunConfig{
		RepoRoot:           req.repoRoot,
		TrustedRoot:        trustedRoot,
		Speed:              req.speed,
		Filter:             req.filter,
		Lang:               req.flags.lang,
		Paths:              parseCSVList(req.flags.paths),
		Diff:               req.diffRes,
		SkipSources:        parseCSVSet(req.flags.skipSources),
		UseContainer:       req.flags.useContainer,
		UnusedExportsCheck: evaluateUnusedExportsForQA,
		CodequalityCheck: func(ctx context.Context, phaseSpeed rule.Speed, _ rule.Filter) (*rule.QAReport, error) {
			phaseReq := req
			phaseReq.speed = phaseSpeed
			reports, subQARep, subErr := evaluateCheckPhase(ctx, phaseReq)
			if subErr != nil {
				return nil, subErr
			}
			collectedReports = append(collectedReports, reports...)
			return subQARep, nil
		},
	})
	if err != nil {
		return nil, nil, err
	}
	return collectedReports, qaRep, nil
}

func evaluateUnusedExportsForQA(repoRoot string, paths []string) ([]rule.QAItem, error) {
	violations, err := astcheck.CheckUnusedExports(repoRoot, "./...")
	if err != nil {
		return nil, err
	}
	patterns, matchAll := gotestutil.NormalizePathPatterns(paths)
	filterPaths := len(paths) > 0 && !matchAll

	items := make([]rule.QAItem, 0, len(violations))
	for _, v := range violations {
		if filterPaths && !gotestutil.MatchesAnyPathPattern(v.File, patterns) {
			continue
		}
		items = append(items, rule.QAItem{
			ID:      fmt.Sprintf("unused_exports:%s:%s:%d:%d", v.Rule, v.File, v.Line, len(items)+1),
			Source:  "unused_exports",
			Check:   v.Rule,
			File:    v.File,
			Line:    v.Line,
			Summary: v.Message,
			Detail:  v.Message,
		})
	}
	return items, nil
}
