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

package rule

// DecisionItemFinding represents a single failed question from a declarative YAML Gemini Decisions rule.
type DecisionItemFinding struct {
	// Index is the 1-based display index within the DecisionSummary.
	Index int
	// RuleID is the unique identifier of the YAML rule that produced this finding.
	RuleID string
	// RuleName is the human-readable title of the YAML rule.
	RuleName string
	// QuestionID is the question template identifier within the YAML rule.
	QuestionID string
	// Blocking reports whether this finding fails the build gate (exit code 1).
	Blocking bool
	// Target identifies the logical package or file bundle evaluated.
	Target string
	// File is the slash-separated repository-relative file path associated with the finding.
	File string
	// Line is the 1-based starting line number of the finding location.
	Line int
	// EndLine is the 1-based ending line number of the finding location.
	EndLine int
	// Subject names the section heading, Mermaid edge, or code declaration evaluated.
	Subject string
	// Outcome summarizes the model's probabilistic verdict (selected option, probability, or rubric score).
	Outcome string
	// Why explains which criterion or failure option was triggered.
	Why string
	// Fix provides the template-expanded remediation instruction for the triggered outcome.
	Fix string
}

// DecisionSummary aggregates execution metadata and findings across all declarative YAML Gemini Decisions rules.
type DecisionSummary struct {
	// Enabled reports whether Gemini Decisions evaluation was active for this run.
	Enabled bool
	// Skipped reports whether evaluation was skipped (for example due to missing credentials or zero matching targets).
	Skipped bool
	// SkipReason explains why evaluation was skipped when Skipped is true.
	SkipReason string
	// RulesEvaluated is the count of enabled YAML rules matched against the target scope.
	RulesEvaluated int
	// TargetsEvaluated is the total number of target bundles evaluated across matched rules.
	TargetsEvaluated int
	// QuestionsEvaluated is the total number of structured questions evaluated across all target bundles.
	QuestionsEvaluated int
	// CacheHits is the number of question batches served from the local SQLite cache without API calls.
	CacheHits int
	// APICalls is the number of live HTTP requests sent to the Gemini Decisions API.
	APICalls int
	// Findings lists all non-passing question outcomes in deterministic order.
	Findings []DecisionItemFinding
}

// BlockingCount returns the number of blocking findings in d.
func (d DecisionSummary) BlockingCount() int {
	count := 0
	for _, f := range d.Findings {
		if f.Blocking {
			count++
		}
	}
	return count
}

// AdvisoryCount returns the number of non-blocking advisory findings in d.
func (d DecisionSummary) AdvisoryCount() int {
	return len(d.Findings) - d.BlockingCount()
}

// HasBlockingFindings reports whether d contains at least one blocking finding.
func (d DecisionSummary) HasBlockingFindings() bool {
	return d.BlockingCount() > 0
}
