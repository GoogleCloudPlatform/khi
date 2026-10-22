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
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/md"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
)

const (
	// RuleIDLogExistence identifies the validation rule that checks whether inline code spans exist in the inspection logs.
	RuleIDLogExistence = "log-existence"
)

// LogExistenceRule verifies that inline code spans in a Markdown report exist in the inspection logs.
type LogExistenceRule struct{}

var _ ValidationRule = (*LogExistenceRule)(nil)

// NewLogExistenceRule creates a new LogExistenceRule.
func NewLogExistenceRule() *LogExistenceRule {
	return &LogExistenceRule{}
}

// ID returns the rule identifier.
func (r *LogExistenceRule) ID() string {
	return RuleIDLogExistence
}

// Description returns a short summary of the rule.
func (r *LogExistenceRule) Description() string {
	return "Inline code spans (`...`) must exist in the inspection logs."
}

type spanValidationOutcome struct {
	grounded bool
	message  string
}

// Validate checks every inline code span in report against the workbench's search index and trigram index.
func (r *LogExistenceRule) Validate(ctx context.Context, report string, wb *Workbench) ([]ValidationFinding, error) {
	index, _, err := wb.readyIndex()
	if err != nil {
		return nil, err
	}
	spans := md.ExtractInlineCodeSpans(report)
	if len(spans) == 0 {
		return nil, nil
	}

	matcher := newLogTermMatcher(index)
	spanCache := make(map[string]spanValidationOutcome)
	var findings []ValidationFinding

	for _, span := range spans {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		outcome, ok := spanCache[span.Text]
		if !ok {
			outcome = matcher.evaluateSpan(span.Text)
			spanCache[span.Text] = outcome
		}
		if !outcome.grounded {
			findings = append(findings, ValidationFinding{
				RuleID:  r.ID(),
				Line:    span.Line,
				Target:  span.Text,
				Message: outcome.message,
			})
		}
	}

	return findings, nil
}

type logTermMatcher struct {
	index      *SearchIndex
	serializer *khifilev6model.DirectYAMLSerializer
	termCache  map[string]bool
	yamlCache  map[uint32]string
}

func newLogTermMatcher(index *SearchIndex) *logTermMatcher {
	return &logTermMatcher{
		index:      index,
		serializer: khifilev6model.NewDirectYAMLSerializer(),
		termCache:  make(map[string]bool),
		yamlCache:  make(map[uint32]string),
	}
}

func (m *logTermMatcher) evaluateSpan(text string) spanValidationOutcome {
	if m.isTermGrounded(text) {
		return spanValidationOutcome{grounded: true}
	}

	if !strings.ContainsAny(text, " \t\r\n") {
		if strings.Contains(text, "/") {
			if grounded, missing, ok := m.checkCompositeParts(splitNonEmpty(text, "/")); ok {
				if grounded {
					return spanValidationOutcome{grounded: true}
				}
				return spanValidationOutcome{
					grounded: false,
					message:  formatCompositeMissingMessage(missing),
				}
			}
		} else if strings.Count(text, "=") == 1 && !strings.ContainsAny(text, "<>!~") {
			k, v, _ := strings.Cut(text, "=")
			if grounded, missing, ok := m.checkCompositeParts([]string{k, v}); ok {
				if grounded {
					return spanValidationOutcome{grounded: true}
				}
				return spanValidationOutcome{
					grounded: false,
					message:  formatCompositeMissingMessage(missing),
				}
			}
		}
	}

	return spanValidationOutcome{
		grounded: false,
		message:  "Not found in inspection logs.",
	}
}

// checkCompositeParts checks whether parts form a valid multi-segment expression (at least 2 parts, each >= 2 runes) and verifies each part against the logs.
func (m *logTermMatcher) checkCompositeParts(parts []string) (grounded bool, missing []string, validComposite bool) {
	if len(parts) < 2 {
		return false, nil, false
	}
	for _, p := range parts {
		if utf8.RuneCountInString(p) < 2 {
			return false, nil, false
		}
	}
	for _, p := range parts {
		if !m.isTermGrounded(p) {
			missing = append(missing, p)
		}
	}
	return len(missing) == 0, missing, true
}

func splitNonEmpty(s, sep string) []string {
	raw := strings.Split(s, sep)
	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func formatCompositeMissingMessage(missing []string) string {
	if len(missing) == 1 {
		return fmt.Sprintf("Not found in inspection logs (unmatched component: `%s`).", missing[0])
	}
	quoted := make([]string, len(missing))
	for i, item := range missing {
		quoted[i] = "`" + item + "`"
	}
	return fmt.Sprintf("Not found in inspection logs (unmatched components: %s).", strings.Join(quoted, ", "))
}

// isTermGrounded checks whether term appears in any log summary or serialized log YAML body using the trigram index for candidate filtering.
func (m *logTermMatcher) isTermGrounded(term string) bool {
	if term == "" {
		return true
	}
	if cached, ok := m.termCache[term]; ok {
		return cached
	}

	found := m.searchLogsForTerm(term)
	m.termCache[term] = found
	return found
}

func (m *logTermMatcher) searchLogsForTerm(term string) bool {
	// TrigramIndex is nil when only the base SearchIndex has been built without async indexing;
	// FindCandidateLogs also returns nil when the query is shorter than 3 bytes (no trigrams).
	// In both cases we fall back to scanning all logs.
	if m.index.TrigramIndex != nil {
		if candidates := m.index.TrigramIndex.FindCandidateLogs(regexp.QuoteMeta(term)); candidates != nil {
			it := candidates.Iterator()
			for it.HasNext() {
				logID := it.Next()
				if m.logContainsTerm(m.index.GetLog(logID), term) {
					return true
				}
			}
			return false
		}
	}

	for i := range m.index.Logs {
		if m.logContainsTerm(&m.index.Logs[i], term) {
			return true
		}
	}
	return false
}

func (m *logTermMatcher) logContainsTerm(l *cel.LogData, term string) bool {
	if l == nil || l.ID == 0 {
		return false
	}
	if l.SummaryStringID != 0 {
		summary := m.index.InternPool.ResolveStringFromID(l.SummaryStringID)
		if strings.Contains(summary, term) {
			return true
		}
	}
	if l.BodyStructID != 0 {
		yamlStr, ok := m.resolveStructYAML(l.BodyStructID)
		if ok && strings.Contains(yamlStr, term) {
			return true
		}
	}
	return false
}

func (m *logTermMatcher) resolveStructYAML(structID uint32) (string, bool) {
	if cached, ok := m.yamlCache[structID]; ok {
		return cached, cached != ""
	}
	yamlStr, err := m.serializer.SerializeFlatStruct(structID, m.index.InternPool)
	if err != nil {
		m.yamlCache[structID] = ""
		return "", false
	}
	m.yamlCache[structID] = yamlStr
	return yamlStr, true
}
