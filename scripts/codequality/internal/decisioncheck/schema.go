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

package decisioncheck

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

const (
	// EnforcementAdvisory marks non-blocking findings that report warnings without failing CI.
	EnforcementAdvisory EnforcementMode = "advisory"
	// EnforcementBlocking marks findings that fail the quality gate with exit code 1.
	EnforcementBlocking EnforcementMode = "blocking"

	// TargetGoLogicalPackage groups a Go feature package and its impl/internal subpackages into one bundle.
	TargetGoLogicalPackage TargetKind = "go_logical_package"
	// TargetChangedFile evaluates each matching modified file as an individual bundle.
	TargetChangedFile TargetKind = "changed_file"

	// ExtractorBundle evaluates one static question across the entire target bundle.
	ExtractorBundle ExtractorKind = "bundle"
	// ExtractorMarkdownSections expands one question per substantive Markdown heading section.
	ExtractorMarkdownSections ExtractorKind = "markdown_sections"
	// ExtractorGoSubstantiveUnits expands one question per substantive Go task or source-file subsystem in the bundle.
	ExtractorGoSubstantiveUnits ExtractorKind = "go_substantive_units"
	// ExtractorChangedGoDecls expands one question per top-level production Go declaration touched by the VCS diff.
	ExtractorChangedGoDecls ExtractorKind = "changed_go_decls"
	// ExtractorChangedGoFuncs expands one question per production Go function or method touched by the VCS diff.
	ExtractorChangedGoFuncs ExtractorKind = "changed_go_funcs"
	// ExtractorChangedGoTestFuncs expands one question per Go unit test function (Test*) touched by the VCS diff.
	ExtractorChangedGoTestFuncs ExtractorKind = "changed_go_test_funcs"
	// ExtractorChangedGoTestCases expands one question per table-driven test case or standalone Test* function touched by the VCS diff.
	ExtractorChangedGoTestCases ExtractorKind = "changed_go_test_cases"
	// ExtractorChangedTSDecls expands one question per production TypeScript declaration touched by the VCS diff.
	ExtractorChangedTSDecls ExtractorKind = "changed_ts_decls"
	// ExtractorChangedTSFuncs expands one question per production TypeScript function or class method touched by the VCS diff.
	ExtractorChangedTSFuncs ExtractorKind = "changed_ts_funcs"
	// ExtractorChangedTSSpecs expands one question per TypeScript Jasmine spec block (it/fit/xit) touched by the VCS diff.
	ExtractorChangedTSSpecs ExtractorKind = "changed_ts_specs"

	// QuestionTypeNoul evaluates a binary true/false proposition returning calibrated P(true).
	QuestionTypeNoul QuestionType = "noul"
	// QuestionTypeChoice selects among 2..255 mutually exclusive options with per-option probabilities.
	QuestionTypeChoice QuestionType = "choice"
	// QuestionTypeScore grades the input against 2..255 ordered rubric levels.
	QuestionTypeScore QuestionType = "score"
)

const maxChoiceOrScoreCardinality = 255

// EnforcementMode controls whether findings are advisory warnings or blocking violations.
type EnforcementMode string

// TargetKind selects how repository files are grouped into shared-context bundles.
type TargetKind string

// ExtractorKind selects the deterministic item extractor used for for_each question expansion.
type ExtractorKind string

// QuestionType identifies one of the three closed-set Gemini Decisions question primitives.
type QuestionType string

// RuleSpec defines a declarative YAML semantic verification rule evaluated via the Gemini Decisions API.
type RuleSpec struct {
	// ID is the unique snake_case rule identifier.
	ID string `yaml:"id"`
	// Name is the short human-readable title of the rule.
	Name string `yaml:"name"`
	// Description summarizes the invariant verified by this rule.
	Description string `yaml:"description"`
	// Enabled controls whether the rule is active by default.
	Enabled bool `yaml:"enabled"`
	// Enforcement specifies "advisory" or "blocking" gate behavior.
	Enforcement EnforcementMode `yaml:"enforcement"`
	// Target configures how target bundles are discovered and filtered.
	Target TargetSpec `yaml:"target"`
	// Context configures how the shared input text is assembled for each target bundle.
	Context ContextSpec `yaml:"context"`
	// Questions lists static or item-expanded structured questions evaluated per bundle.
	Questions []QuestionSpec `yaml:"questions"`
}

// TargetSpec configures bundle grouping and path filtering for a RuleSpec.
type TargetSpec struct {
	// Kind selects "go_logical_package" or "changed_file".
	Kind TargetKind `yaml:"kind"`
	// Include lists repository-relative path globs eligible for this rule.
	Include []string `yaml:"include"`
	// Exclude lists repository-relative path globs excluded from this rule.
	Exclude []string `yaml:"exclude"`
	// RequireFiles lists relative filenames that must exist inside a package directory for evaluation.
	RequireFiles []string `yaml:"require_files"`
}

// ContextSpec configures what files and metadata are rendered into the shared input context.
type ContextSpec struct {
	// IncludeFiles lists target-relative file globs bundled for go_logical_package targets.
	IncludeFiles []string `yaml:"include_files"`
	// ExcludeFiles lists target-relative file globs excluded from bundle rendering.
	ExcludeFiles []string `yaml:"exclude_files"`
	// ReferenceFiles lists trusted repository-relative reference documents prepended to the context.
	ReferenceFiles []string `yaml:"reference_files"`
	// IncludeCompanionGo bundles sibling <base>.go production files alongside *_test.go changed_file targets when true.
	IncludeCompanionGo bool `yaml:"include_companion_go"`
	// IncludeCompanionTS bundles sibling <base>.ts production files alongside *.spec.ts changed_file targets when true.
	IncludeCompanionTS bool `yaml:"include_companion_ts"`
	// IncludeDiff appends unified VCS diff hunks for the target when true.
	IncludeDiff bool `yaml:"include_diff"`
	// LineNumbers prefixes rendered file lines with 1-based line numbers when true.
	LineNumbers bool `yaml:"line_numbers"`
	// IncludeGoASTSummary appends a compact Go AST declaration index when true.
	IncludeGoASTSummary bool `yaml:"include_go_ast_summary"`
}

// QuestionSpec defines one structured question template within a RuleSpec.
type QuestionSpec struct {
	// ID is the unique question key prefix within the rule.
	ID string `yaml:"id"`
	// ForEach optionally names an ExtractorKind to fan out one question per extracted item.
	ForEach ExtractorKind `yaml:"for_each"`
	// Type specifies "noul", "choice", or "score".
	Type QuestionType `yaml:"type"`
	// Instructions is a Go text/template string specifying the question prompt.
	Instructions string `yaml:"instructions"`
	// Criteria configures optional true/false descriptions for noul questions.
	Criteria NoulCriteriaSpec `yaml:"criteria"`
	// Options maps option keys to descriptions and outcome-specific fix hints for choice questions.
	Options map[string]ChoiceOptionSpec `yaml:"options"`
	// Levels lists ordered rubric levels (0..N-1) and outcome-specific fix hints for score questions.
	Levels []ScoreLevelSpec `yaml:"levels"`
	// FixHint is a Go text/template remediation instruction (mandatory for noul; fallback for choice and score).
	FixHint string `yaml:"fix_hint"`
	// Pass defines the probabilistic pass threshold for this question.
	Pass PassSpec `yaml:"pass"`
}

// NoulCriteriaSpec holds optional true/false anchor descriptions for a noul question.
type NoulCriteriaSpec struct {
	// TrueDescription describes what constitutes a "yes" outcome (supports text/template).
	TrueDescription string `yaml:"true_description"`
	// FalseDescription describes what constitutes a "no" outcome (supports text/template).
	FalseDescription string `yaml:"false_description"`
}

// ChoiceOptionSpec defines one mutually exclusive option in a choice question.
type ChoiceOptionSpec struct {
	// Description defines when the model should select this option (supports text/template).
	Description string `yaml:"description"`
	// FixHint provides the actionable remediation template when this non-passing option is selected.
	FixHint string `yaml:"fix_hint"`
}

// ScoreLevelSpec defines one discrete ordinal level in a score rubric question.
type ScoreLevelSpec struct {
	// Description defines the rubric criteria for this 0-based level (supports text/template).
	Description string `yaml:"description"`
	// FixHint provides the actionable remediation template when this failing level is most likely.
	FixHint string `yaml:"fix_hint"`
}

// PassSpec defines pass/fail thresholds over calibrated Gemini Decisions responses.
type PassSpec struct {
	// Expect specifies the required boolean outcome for noul questions.
	Expect *bool `yaml:"expect"`
	// ExpectedChoice specifies a single passing option key for choice questions.
	ExpectedChoice string `yaml:"expected_choice"`
	// AllowedChoices lists multiple acceptable option keys for choice questions.
	AllowedChoices []string `yaml:"allowed_choices"`
	// MinProbability sets the minimum calibrated probability required on the expected noul/choice outcome.
	MinProbability float64 `yaml:"min_probability"`
	// MinScore sets the minimum probability-weighted expected score required for score questions.
	MinScore *float64 `yaml:"min_score"`
	// MaxScore sets the maximum probability-weighted expected score allowed for score questions.
	MaxScore *float64 `yaml:"max_score"`
	// MinConfidence optionally requires confidence >= MinConfidence when evaluating findings.
	MinConfidence float64 `yaml:"min_confidence"`
}

// ExtractedItem holds deterministic metadata bound to Go text/template variables during question expansion.
type ExtractedItem struct {
	// Target is the bundle identifier (logical package directory or relative file path).
	Target string
	// File is the repository-relative file path of the item.
	File string
	// Line is the 1-based primary line number of the item.
	Line int
	// StartLine is the 1-based starting line number of the item span.
	StartLine int
	// EndLine is the 1-based ending line number of the item span.
	EndLine int
	// Heading is the Markdown section title (populated by markdown_sections).
	Heading string
	// Name is the Go identifier, test case label, or task variable name (populated by Go AST extractors).
	Name string
	// Kind describes the extracted item category (for example "registered DAG task", "func", or "test case").
	Kind string
	// Doc holds the extracted GoDoc or preceding test-case explanation comment text when present.
	Doc string
}

// ParseRuleYAML decodes and validates a single YAML rule specification with strict field checking.
func ParseRuleYAML(data []byte) (RuleSpec, error) {
	spec := RuleSpec{
		Enabled:     true,
		Enforcement: EnforcementAdvisory,
		Context: ContextSpec{
			LineNumbers: true,
		},
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&spec); err != nil {
		return RuleSpec{}, fmt.Errorf("failed to decode rule YAML: %w", err)
	}
	if err := ValidateRule(spec); err != nil {
		return RuleSpec{}, err
	}
	return spec, nil
}

// LoadRulesDir loads, validates, and returns all *.yaml / *.yml RuleSpecs in dir sorted by ID.
func LoadRulesDir(dir string) ([]RuleSpec, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read decision rules directory %s: %w", dir, err)
	}
	var specs []RuleSpec
	seenIDs := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFilename(entry.Name()) {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		raw, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			return nil, fmt.Errorf("failed reading %s: %w", fullPath, readErr)
		}
		spec, parseErr := ParseRuleYAML(raw)
		if parseErr != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), parseErr)
		}
		if prev, dup := seenIDs[spec.ID]; dup {
			return nil, fmt.Errorf("duplicate rule id %q in %s and %s", spec.ID, prev, entry.Name())
		}
		seenIDs[spec.ID] = entry.Name()
		specs = append(specs, spec)
	}
	slices.SortFunc(specs, func(a, b RuleSpec) int {
		return strings.Compare(a.ID, b.ID)
	})
	return specs, nil
}

// ValidateRule checks static schema invariants, template syntax, and outcome fix_hint completeness on r.
func ValidateRule(r RuleSpec) error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("rule id and name are required")
	}
	if r.Enforcement != EnforcementAdvisory && r.Enforcement != EnforcementBlocking {
		return fmt.Errorf("rule %q: invalid enforcement %q (want advisory or blocking)", r.ID, r.Enforcement)
	}
	if r.Target.Kind != TargetGoLogicalPackage && r.Target.Kind != TargetChangedFile {
		return fmt.Errorf("rule %q: invalid target.kind %q", r.ID, r.Target.Kind)
	}
	if len(r.Questions) == 0 {
		return fmt.Errorf("rule %q: at least one question is required", r.ID)
	}
	seenQ := make(map[string]struct{}, len(r.Questions))
	for _, q := range r.Questions {
		if _, dup := seenQ[q.ID]; dup {
			return fmt.Errorf("rule %q: duplicate question id %q", r.ID, q.ID)
		}
		seenQ[q.ID] = struct{}{}
		if err := validateQuestion(r.ID, q); err != nil {
			return err
		}
	}
	return nil
}

// AllowedPassChoices returns the deduplicated set of passing choice keys configured on p.
func (p PassSpec) AllowedPassChoices() map[string]struct{} {
	out := make(map[string]struct{})
	if p.ExpectedChoice != "" {
		out[p.ExpectedChoice] = struct{}{}
	}
	for _, c := range p.AllowedChoices {
		if c != "" {
			out[c] = struct{}{}
		}
	}
	return out
}

func validateQuestion(ruleID string, q QuestionSpec) error {
	if strings.TrimSpace(q.ID) == "" || strings.TrimSpace(q.Instructions) == "" {
		return fmt.Errorf("rule %q: question id and instructions are required", ruleID)
	}
	if !isValidExtractor(q.ForEach) {
		return fmt.Errorf("rule %q question %q: unknown for_each extractor %q", ruleID, q.ID, q.ForEach)
	}
	if err := validateTemplateStrings(q.Instructions, q.FixHint, q.Criteria.TrueDescription, q.Criteria.FalseDescription); err != nil {
		return fmt.Errorf("rule %q question %q: %w", ruleID, q.ID, err)
	}
	switch q.Type {
	case QuestionTypeNoul:
		return validateNoulQuestion(ruleID, q)
	case QuestionTypeChoice:
		return validateChoiceQuestion(ruleID, q)
	case QuestionTypeScore:
		return validateScoreQuestion(ruleID, q)
	default:
		return fmt.Errorf("rule %q question %q: invalid question type %q", ruleID, q.ID, q.Type)
	}
}

func validateNoulQuestion(ruleID string, q QuestionSpec) error {
	if q.Pass.Expect == nil {
		return fmt.Errorf("rule %q question %q: noul question requires pass.expect (true or false)", ruleID, q.ID)
	}
	if err := validateMinProbability(ruleID, q.ID, q.Pass.MinProbability); err != nil {
		return err
	}
	if strings.TrimSpace(q.FixHint) == "" {
		return fmt.Errorf("rule %q question %q: noul question requires a non-empty fix_hint", ruleID, q.ID)
	}
	return nil
}

func validateChoiceQuestion(ruleID string, q QuestionSpec) error {
	if len(q.Options) < 2 || len(q.Options) > maxChoiceOrScoreCardinality {
		return fmt.Errorf("rule %q question %q: choice requires 2..255 options (got %d)", ruleID, q.ID, len(q.Options))
	}
	allowed := q.Pass.AllowedPassChoices()
	if len(allowed) == 0 {
		return fmt.Errorf("rule %q question %q: choice requires pass.expected_choice or pass.allowed_choices", ruleID, q.ID)
	}
	for k := range allowed {
		if _, ok := q.Options[k]; !ok {
			return fmt.Errorf("rule %q question %q: passing choice %q not defined in options", ruleID, q.ID, k)
		}
	}
	if err := validateMinProbability(ruleID, q.ID, q.Pass.MinProbability); err != nil {
		return err
	}
	return validateChoiceOptions(ruleID, q, allowed)
}

func validateChoiceOptions(ruleID string, q QuestionSpec, allowed map[string]struct{}) error {
	for optKey, opt := range q.Options {
		if strings.TrimSpace(opt.Description) == "" {
			return fmt.Errorf("rule %q question %q option %q: description is required", ruleID, q.ID, optKey)
		}
		if err := validateTemplateStrings(opt.Description, opt.FixHint); err != nil {
			return fmt.Errorf("rule %q question %q option %q: %w", ruleID, q.ID, optKey, err)
		}
		_, isPass := allowed[optKey]
		if !isPass && strings.TrimSpace(opt.FixHint) == "" && strings.TrimSpace(q.FixHint) == "" {
			return fmt.Errorf("rule %q question %q option %q: non-passing option requires fix_hint", ruleID, q.ID, optKey)
		}
	}
	return nil
}

func validateScoreQuestion(ruleID string, q QuestionSpec) error {
	if len(q.Levels) < 2 || len(q.Levels) > maxChoiceOrScoreCardinality {
		return fmt.Errorf("rule %q question %q: score requires 2..255 levels (got %d)", ruleID, q.ID, len(q.Levels))
	}
	if q.Pass.MinScore == nil && q.Pass.MaxScore == nil {
		return fmt.Errorf("rule %q question %q: score requires pass.min_score or pass.max_score", ruleID, q.ID)
	}
	maxLevel := float64(len(q.Levels) - 1)
	if err := validateScoreBound(ruleID, q.ID, "min_score", q.Pass.MinScore, maxLevel); err != nil {
		return err
	}
	if err := validateScoreBound(ruleID, q.ID, "max_score", q.Pass.MaxScore, maxLevel); err != nil {
		return err
	}
	return validateScoreLevels(ruleID, q)
}

func validateScoreLevels(ruleID string, q QuestionSpec) error {
	for idx, lvl := range q.Levels {
		if strings.TrimSpace(lvl.Description) == "" {
			return fmt.Errorf("rule %q question %q level %d: description is required", ruleID, q.ID, idx)
		}
		if err := validateTemplateStrings(lvl.Description, lvl.FixHint); err != nil {
			return fmt.Errorf("rule %q question %q level %d: %w", ruleID, q.ID, idx, err)
		}
		if isFailingScoreLevel(idx, q.Pass) && strings.TrimSpace(lvl.FixHint) == "" && strings.TrimSpace(q.FixHint) == "" {
			return fmt.Errorf("rule %q question %q level %d: failing score level requires fix_hint", ruleID, q.ID, idx)
		}
	}
	return nil
}

func validateMinProbability(ruleID string, qID string, minProb float64) error {
	if minProb <= 0 || minProb >= 1 {
		return fmt.Errorf("rule %q question %q: pass.min_probability must be in (0, 1)", ruleID, qID)
	}
	return nil
}

func validateScoreBound(ruleID string, qID string, field string, bound *float64, maxLevel float64) error {
	if bound != nil && (*bound < 0 || *bound > maxLevel) {
		return fmt.Errorf("rule %q question %q: pass.%s out of range [0, %.0f]", ruleID, qID, field, maxLevel)
	}
	return nil
}

func isFailingScoreLevel(idx int, p PassSpec) bool {
	if p.MinScore != nil && idx < int(math.Ceil(*p.MinScore)) {
		return true
	}
	if p.MaxScore != nil && idx > int(math.Floor(*p.MaxScore)) {
		return true
	}
	return false
}

func isValidExtractor(e ExtractorKind) bool {
	switch e {
	case "",
		ExtractorBundle,
		ExtractorMarkdownSections,
		ExtractorGoSubstantiveUnits,
		ExtractorChangedGoDecls,
		ExtractorChangedGoFuncs,
		ExtractorChangedGoTestFuncs,
		ExtractorChangedGoTestCases,
		ExtractorChangedTSDecls,
		ExtractorChangedTSFuncs,
		ExtractorChangedTSSpecs:
		return true
	default:
		return false
	}
}

func validateTemplateStrings(texts ...string) error {
	sample := ExtractedItem{
		Target: "pkg/sample", File: "pkg/sample/README.md",
		Line: 1, StartLine: 1, EndLine: 2,
		Heading: "Overview",
		Name:    "Sample", Kind: "type", Doc: "Sample doc comment.",
	}
	for _, txt := range texts {
		if txt == "" {
			continue
		}
		tmpl, err := template.New("check").Option("missingkey=error").Parse(txt)
		if err != nil {
			return fmt.Errorf("invalid template syntax: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, sample); err != nil {
			return fmt.Errorf("template references unknown field: %w", err)
		}
	}
	return nil
}

func isYAMLFilename(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}
