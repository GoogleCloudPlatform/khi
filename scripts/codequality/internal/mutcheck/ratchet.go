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

package mutcheck

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func collectSurvivingMutantFindings(
	opts Options,
	allPlans []*FilePlan,
	res *Result,
) {
	survivedByKey := groupSurvivedByFunc(allPlans)
	limit := opts.Thresholds.Limit(rule.RuleFuncSurvivingMutants)

	for _, relFile := range sortedMapKeys(opts.ASTResult.TouchedFuncs) {
		fm := opts.ASTResult.Files[relFile]
		if fm == nil {
			continue
		}
		for _, fn := range opts.ASTResult.TouchedFuncs[relFile] {
			survived := survivedByKey[relFile+"|"+fn.Key]
			if len(survived) <= limit {
				continue
			}
			recordMutOutcome(res, fm, fn, limit, survived)
		}
	}
}

func recordMutOutcome(
	res *Result,
	fm *astcheck.FileMetrics,
	fn astcheck.FuncMetrics,
	limit int,
	survived []rule.MutantDetail,
) {
	reportLine := fn.StartLine
	if len(survived) > 0 {
		reportLine = survived[0].Line
	}
	res.Findings = append(res.Findings, rule.Finding{
		Rule:             rule.RuleFuncSurvivingMutants,
		Kind:             rule.ViolationAdvisory,
		File:             fm.File,
		Line:             reportLine,
		EndLine:          fn.EndLine,
		Subject:          fn.DisplayName,
		After:            len(survived),
		Limit:            limit,
		Detail:           fmt.Sprintf("%d mutant(s) survived unit tests", len(survived)),
		SurvivingMutants: survived,
	})
}
