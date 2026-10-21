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
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
)

const (
	// RuleIDUngroundedInlineCode identifies the validation rule that checks inline code spans against inspection logs.
	RuleIDUngroundedInlineCode = "ungrounded-inline-code"
)

// InlineCodeSpan represents a single inline code span extracted from a Markdown report.
type InlineCodeSpan struct {
	// Text is the unescaped content inside the inline backticks.
	Text string
	// Line is the 1-based line number where the inline code span starts.
	Line int
}

// ReportDocument holds the parsed structure of a Markdown report for validation rules.
type ReportDocument struct {
	// Raw is the normalized Markdown text.
	Raw string
	// Lines contains the individual lines of the Markdown report.
	Lines []string
	// InlineCodeSpans lists all non-empty inline code spans outside fenced code blocks.
	InlineCodeSpans []InlineCodeSpan
}

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

// ValidationRule defines a modular validation check executed against a ReportDocument using Workbench inspection data.
type ValidationRule interface {
	// ID returns the unique identifier of the validation rule.
	ID() string
	// Description returns a short summary of what the rule checks.
	Description() string
	// Validate inspects the parsed ReportDocument against the Workbench and returns any findings.
	Validate(ctx context.Context, doc *ReportDocument, wb *Workbench) ([]ValidationFinding, error)
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
	// InlineCodeSpanCount is the total number of inline code spans inspected in the report.
	InlineCodeSpanCount int
	// Findings lists all validation issues sorted by line number and rule ID.
	Findings []ValidationFinding
}

// DefaultValidationRules returns the standard set of report validation rules.
func DefaultValidationRules() []ValidationRule {
	return []ValidationRule{
		NewInlineCodeGroundingRule(),
	}
}

// ValidateReport parses the Markdown report and executes the specified validation rules against the workbench.
// When no rules are passed, DefaultValidationRules() is used.
func (w *Workbench) ValidateReport(ctx context.Context, report string, rules ...ValidationRule) (*ReportValidationResult, error) {
	if _, _, err := w.readyIndex(); err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		rules = DefaultValidationRules()
	}

	doc := ParseReportDocument(report)
	summaries := make([]ValidationRuleSummary, 0, len(rules))
	var allFindings []ValidationFinding

	for _, rule := range rules {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		findings, err := rule.Validate(ctx, doc, w)
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
		Rules:               summaries,
		InlineCodeSpanCount: len(doc.InlineCodeSpans),
		Findings:            allFindings,
	}, nil
}

// ParseReportDocument parses a Markdown report and extracts inline code spans while skipping fenced code blocks.
func ParseReportDocument(report string) *ReportDocument {
	normalized := strings.ReplaceAll(report, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")

	var spans []InlineCodeSpan
	inFence := false
	var fenceChar byte
	fenceLen := 0

	for i, line := range lines {
		lineNum := i + 1
		stripped := stripBlockquoteAndIndent(line)

		if !inFence {
			if ch, count, ok := parseOpeningFence(stripped); ok {
				inFence = true
				fenceChar = ch
				fenceLen = count
				continue
			}
			spans = append(spans, extractLineInlineCodeSpans(line, lineNum)...)
		} else if isClosingFence(stripped, fenceChar, fenceLen) {
			inFence = false
			fenceChar = 0
			fenceLen = 0
		}
	}

	return &ReportDocument{
		Raw:             normalized,
		Lines:           lines,
		InlineCodeSpans: spans,
	}
}

// stripBlockquoteAndIndent removes leading blockquote markers ('>') and whitespace from a line so fence markers inside blockquotes or lists are recognized.
func stripBlockquoteAndIndent(line string) string {
	s := line
	for {
		s = strings.TrimLeft(s, " \t")
		if strings.HasPrefix(s, ">") {
			s = s[1:]
			continue
		}
		return s
	}
}

// parseOpeningFence checks whether a line begins a CommonMark fenced code block of at least 3 backticks or tildes.
func parseOpeningFence(stripped string) (byte, int, bool) {
	if len(stripped) < 3 {
		return 0, 0, false
	}
	ch := stripped[0]
	if ch != '`' && ch != '~' {
		return 0, 0, false
	}
	count := countLeadingByte(stripped, ch)
	if count < 3 {
		return 0, 0, false
	}
	info := stripped[count:]
	// Per CommonMark spec, a backtick fence info string cannot contain backtick characters.
	if ch == '`' && strings.IndexByte(info, '`') >= 0 {
		return 0, 0, false
	}
	return ch, count, true
}

// isClosingFence checks whether a line closes an active fenced code block.
func isClosingFence(stripped string, fenceChar byte, minLen int) bool {
	if len(stripped) < minLen || stripped[0] != fenceChar {
		return false
	}
	count := countLeadingByte(stripped, fenceChar)
	if count < minLen {
		return false
	}
	return strings.TrimSpace(stripped[count:]) == ""
}

func countLeadingByte(s string, b byte) int {
	i := 0
	for i < len(s) && s[i] == b {
		i++
	}
	return i
}

// extractLineInlineCodeSpans extracts non-empty CommonMark inline code spans from a single line.
func extractLineInlineCodeSpans(line string, lineNum int) []InlineCodeSpan {
	var spans []InlineCodeSpan
	i := 0
	n := len(line)

	for i < n {
		if line[i] != '`' {
			i++
			continue
		}
		openLen := countLeadingByte(line[i:], '`')
		contentStart := i + openLen

		closeStart := findClosingBacktickRun(line, contentStart, openLen)
		if closeStart < 0 {
			i = contentStart
			continue
		}

		content := normalizeInlineCodeContent(line[contentStart:closeStart])
		if strings.TrimSpace(content) != "" {
			spans = append(spans, InlineCodeSpan{
				Text: content,
				Line: lineNum,
			})
		}
		i = closeStart + openLen
	}

	return spans
}

// findClosingBacktickRun finds the index of the next backtick run of exact length targetLen starting at or after start.
func findClosingBacktickRun(line string, start, targetLen int) int {
	j := start
	n := len(line)
	for j < n {
		if line[j] != '`' {
			j++
			continue
		}
		runLen := countLeadingByte(line[j:], '`')
		if runLen == targetLen {
			return j
		}
		j += runLen
	}
	return -1
}

// normalizeInlineCodeContent strips a single leading and trailing space if both are present and the span is not all spaces, matching CommonMark code span normalization.
func normalizeInlineCodeContent(s string) string {
	if len(s) >= 2 && s[0] == ' ' && s[len(s)-1] == ' ' && strings.Trim(s, " ") != "" {
		return s[1 : len(s)-1]
	}
	return s
}

// InlineCodeGroundingRule verifies that inline code spans in a Markdown report exist in the inspection logs.
type InlineCodeGroundingRule struct{}

var _ ValidationRule = (*InlineCodeGroundingRule)(nil)

// NewInlineCodeGroundingRule creates a new InlineCodeGroundingRule.
func NewInlineCodeGroundingRule() *InlineCodeGroundingRule {
	return &InlineCodeGroundingRule{}
}

// ID returns the rule identifier.
func (r *InlineCodeGroundingRule) ID() string {
	return RuleIDUngroundedInlineCode
}

// Description returns a short summary of the rule.
func (r *InlineCodeGroundingRule) Description() string {
	return "Inline code spans (`...`) must exist in the inspection logs."
}

type spanValidationOutcome struct {
	grounded bool
	message  string
}

// Validate checks every inline code span in doc against the workbench's search index and trigram index.
func (r *InlineCodeGroundingRule) Validate(ctx context.Context, doc *ReportDocument, wb *Workbench) ([]ValidationFinding, error) {
	index, _, err := wb.readyIndex()
	if err != nil {
		return nil, err
	}
	if len(doc.InlineCodeSpans) == 0 {
		return nil, nil
	}

	matcher := newLogTermMatcher(index)
	spanCache := make(map[string]spanValidationOutcome)
	var findings []ValidationFinding

	for _, span := range doc.InlineCodeSpans {
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
	// FindCandidateLogs returns nil when the query is shorter than 3 bytes (no trigrams),
	// in which case we fall back to scanning all logs.
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
