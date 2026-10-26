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

package workbench

import (
	"cmp"
	"context"
	"fmt"
	"slices"
)

// ValidationFinding represents a single issue detected by a ValidationRule.
type ValidationFinding struct {
	// RuleID identifies the ValidationRule that produced this finding.
	RuleID string
	// Line is the 1-based line number in the Markdown report where the issue was found.
	Line int
	// Target is the specific token or expression that triggered the finding.
	Target string
	// Message explains why the target was flagged.
	Message string
}

// ValidationRule defines a modular validation check executed against a Markdown report using Workbench inspection data.
type ValidationRule interface {
	// ID returns the unique identifier of the validation rule.
	ID() string
	// Description returns a short summary of what the rule checks.
	Description() string
	// Validate inspects the Markdown report against the Workbench and returns any findings.
	Validate(ctx context.Context, report string, wb *Workbench) ([]ValidationFinding, error)
}

// ValidationRuleSummary describes a validation rule executed during report validation.
type ValidationRuleSummary struct {
	// ID is the unique identifier of the validation rule.
	ID string
	// Description is a short summary of what the rule checks.
	Description string
	// FindingCount is the number of findings produced by this rule.
	FindingCount int
}

// ReportValidationResult contains the aggregated results of running validation rules on a Markdown report.
type ReportValidationResult struct {
	// Rules holds the summaries of all validation rules that were executed.
	Rules []ValidationRuleSummary
	// Findings lists all validation issues sorted by line number and rule ID.
	Findings []ValidationFinding
}

// DefaultValidationRules returns the standard set of report validation rules.
func DefaultValidationRules() []ValidationRule {
	return []ValidationRule{
		NewLogExistenceRule(),
	}
}

// ValidateReport executes the specified validation rules on the Markdown report against the workbench.
// When no rules are passed, DefaultValidationRules() is used.
func (w *Workbench) ValidateReport(ctx context.Context, report string, rules ...ValidationRule) (*ReportValidationResult, error) {
	if _, _, err := w.readyIndex(); err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		rules = DefaultValidationRules()
	}

	summaries := make([]ValidationRuleSummary, 0, len(rules))
	var allFindings []ValidationFinding

	for _, rule := range rules {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		findings, err := rule.Validate(ctx, report, w)
		if err != nil {
			return nil, fmt.Errorf("validation rule %s failed: %w", rule.ID(), err)
		}
		summaries = append(summaries, ValidationRuleSummary{
			ID:           rule.ID(),
			Description:  rule.Description(),
			FindingCount: len(findings),
		})
		allFindings = append(allFindings, findings...)
	}

	slices.SortFunc(allFindings, func(a, b ValidationFinding) int {
		if c := cmp.Compare(a.Line, b.Line); c != 0 {
			return c
		}
		if c := cmp.Compare(a.RuleID, b.RuleID); c != 0 {
			return c
		}
		return cmp.Compare(a.Target, b.Target)
	})

	return &ReportValidationResult{
		Rules:    summaries,
		Findings: allFindings,
	}, nil
}
