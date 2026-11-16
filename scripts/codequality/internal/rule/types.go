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

const (
	// StageAST identifies fast static AST structural and naming rules.
	StageAST RuleStage = "AST"
	// StageDynamic identifies test-execution coverage, mutation, and redundancy rules.
	StageDynamic RuleStage = "DYNAMIC"
)

const (
	// RuleFuncCognitive bounds production function cognitive complexity.
	RuleFuncCognitive RuleID = "func_cognitive"
	// RuleFuncNesting bounds maximum control-flow nesting depth in a production function.
	RuleFuncNesting RuleID = "func_nesting"
	// RuleFuncCodeLines bounds non-blank, non-comment code lines in a production function.
	RuleFuncCodeLines RuleID = "func_code_lines"
	// RuleFuncStatements bounds AST statements in a production function.
	RuleFuncStatements RuleID = "func_statements"
	// RuleFuncParams bounds input parameter count on a production function.
	RuleFuncParams RuleID = "func_params"
	// RuleFuncResults bounds return value count on a production function.
	RuleFuncResults RuleID = "func_results"
	// RuleControlFlagParam forbids boolean control-flow flag parameters on multi-parameter functions.
	RuleControlFlagParam RuleID = "control_flag_param"
	// RuleFileCodeLines bounds non-blank, non-comment, non-import code lines in a production Go file.
	RuleFileCodeLines RuleID = "file_code_lines"
	// RuleCatchAllFilename forbids low-cohesion catch-all filenames such as util.go or helpers.go.
	RuleCatchAllFilename RuleID = "catchall_filename"
	// RuleCodeClone bounds contiguous normalized AST statements duplicated across production functions.
	RuleCodeClone RuleID = "code_clone"
	// RuleTRunCognitive bounds cognitive complexity inside a t.Run or b.Run subtest closure body.
	RuleTRunCognitive RuleID = "trun_cognitive"
	// RuleTestHelperCognitive bounds cognitive complexity inside non-entry helper functions in *_test.go.
	RuleTestHelperCognitive RuleID = "test_helper_cognitive"
	// RuleTestFileCodeLines bounds non-blank, non-comment, non-import code lines in a *_test.go file.
	RuleTestFileCodeLines RuleID = "test_file_code_lines"
	// RuleFuncUncoveredLines bounds uncovered substantive executable lines in touched production functions.
	RuleFuncUncoveredLines RuleID = "func_uncovered_lines"
	// RuleFuncSurvivingMutants bounds surviving mutants in touched production functions.
	RuleFuncSurvivingMutants RuleID = "func_surviving_mutants"
	// RuleTestRedundancy bounds subsumed duplicate or assertion-free subtests in touched Test* functions.
	RuleTestRedundancy RuleID = "test_redundancy"
)

const (
	// CategoryDispatch marks flat command/token/event dispatchers whose branch count tracks an enum or schema.
	CategoryDispatch AllowCategory = "dispatch"
	// CategoryProtocol marks signatures or structures dictated by an external interface, RPC, or Go stdlib contract.
	CategoryProtocol AllowCategory = "protocol"
	// CategoryEquivalent marks mutations that yield identical observable outputs for all valid inputs.
	CategoryEquivalent AllowCategory = "equivalent"
	// CategoryPerf marks fast-path, caching, or pre-sizing logic that does not alter observable domain output.
	CategoryPerf AllowCategory = "perf"
	// CategoryObservability marks telemetry, progress, or diagnostic paths outside functional assertions.
	CategoryObservability AllowCategory = "observability"
	// CategoryExternal marks OS, subprocess, or external I/O boundaries verified outside unit tests.
	CategoryExternal AllowCategory = "external"
)

const (
	// OperatorROR represents Relational Operator Replacement (<, <=, >, >=, ==, !=).
	OperatorROR MutantOperator = "ROR"
	// OperatorLCR represents Logical Connector Replacement (&&, ||).
	OperatorLCR MutantOperator = "LCR"
	// OperatorBCR represents Branch Condition Replacement (forcing non-comparison if conditions to true/false).
	OperatorBCR MutantOperator = "BCR"
	// OperatorAOR represents Arithmetic and Concatenation Operator Replacement (+, -, *, /).
	OperatorAOR MutantOperator = "AOR"
	// OperatorUOI represents Unary Operator Removal (!x -> x).
	OperatorUOI MutantOperator = "UOI"
	// OperatorBLR represents Boolean Literal Replacement in return statements (true <-> false).
	OperatorBLR MutantOperator = "BLR"
	// OperatorSBR represents Statement and Side-Effect Removal (=, +=, -=, ++, --).
	OperatorSBR MutantOperator = "SBR"
)

const (
	// MutantPending indicates the mutant is awaiting test execution.
	MutantPending MutantStatus = "PENDING"
	// MutantKilled indicates at least one unit test failed when the mutant was active.
	MutantKilled MutantStatus = "KILLED"
	// MutantTimedOut indicates test execution timed out when the mutant was active (counted as detected).
	MutantTimedOut MutantStatus = "TIMED_OUT"
	// MutantSurvived indicates all unit tests passed despite the active mutation.
	MutantSurvived MutantStatus = "SURVIVED"
	// MutantUncovered indicates unit tests never reached the mutated expression or statement.
	MutantUncovered MutantStatus = "UNCOVERED"
	// MutantSuppressed indicates the mutant was suppressed by a valid CODEQUALITY_ALLOW directive.
	MutantSuppressed MutantStatus = "SUPPRESSED"
	// MutantSkippedBudget indicates evaluation was skipped due to --max-mutants or --timeout limits.
	MutantSkippedBudget MutantStatus = "SKIPPED_BUDGET"
	// MutantCompileSkipped indicates the mutation site was dropped by compile bisect fallback.
	MutantCompileSkipped MutantStatus = "COMPILE_SKIPPED"
)

// AllAllowCategories lists valid AllowCategory values in deterministic reporting order.
var AllAllowCategories = []AllowCategory{
	CategoryDispatch,
	CategoryProtocol,
	CategoryEquivalent,
	CategoryPerf,
	CategoryObservability,
	CategoryExternal,
}

var ruleSpecs = []RuleSpec{
	{
		ID:                RuleFuncCognitive,
		Stage:             StageAST,
		Label:             "Cognitive complexity",
		DefaultLimit:      15,
		AllowedCategories: []AllowCategory{CategoryDispatch, CategoryProtocol},
		Why:               "Deeply nested branching and mixed boolean chains exceed human and agent reasoning capacity.",
		Fix:               "Use early-return guard clauses, flatten switch dispatch, or extract cohesive domain methods (keep params <= 5).",
	},
	{
		ID:                RuleFuncNesting,
		Stage:             StageAST,
		Label:             "Control nesting depth",
		DefaultLimit:      4,
		AllowedCategories: []AllowCategory{CategoryDispatch, CategoryProtocol},
		Why:               "Control-flow nesting deeper than 4 levels hides state transitions and invites unhandled edge cases.",
		Fix:               "Invert preconditions into early-return/continue guards or extract inner loop/branch bodies.",
	},
	{
		ID:                RuleFuncCodeLines,
		Stage:             StageAST,
		Label:             "Function code lines",
		DefaultLimit:      60,
		AllowedCategories: []AllowCategory{CategoryDispatch, CategoryProtocol},
		Why:               "Long functions mix multiple levels of abstraction (comments and blank lines are already excluded).",
		Fix:               "Split distinct pipeline phases into focused functions or receiver methods; never delete GoDoc comments.",
	},
	{
		ID:                RuleFuncStatements,
		Stage:             StageAST,
		Label:             "Function statements",
		DefaultLimit:      40,
		AllowedCategories: []AllowCategory{CategoryDispatch, CategoryProtocol},
		Why:               "High statement density signals an oversized procedure even if formatted compactly.",
		Fix:               "Decompose sequential steps into named domain transformations or declarative table lookups.",
	},
	{
		ID:                RuleFuncParams,
		Stage:             StageAST,
		Label:             "Function parameters",
		DefaultLimit:      5,
		AllowedCategories: []AllowCategory{CategoryProtocol},
		Why:               "Functions taking >5 parameters indicate pass-through helper shredding or missing domain structs.",
		Fix:               "Group related inputs into a cohesive domain struct or attach behavior as a receiver method.",
	},
	{
		ID:                RuleFuncResults,
		Stage:             StageAST,
		Label:             "Function return values",
		DefaultLimit:      3,
		AllowedCategories: []AllowCategory{CategoryProtocol},
		Why:               "Returning >3 positional values makes call sites brittle and ambiguous.",
		Fix:               "Return a named result struct + error instead of positional tuples.",
	},
	{
		ID:                RuleControlFlagParam,
		Stage:             StageAST,
		Label:             "Control-flow bool parameter",
		DefaultLimit:      0,
		AllowedCategories: []AllowCategory{CategoryProtocol},
		Why:               "Boolean mode flags on multi-parameter functions couple two distinct behaviors inside one body.",
		Fix:               "Split into two explicit functions or pass a typed strategy/options struct.",
	},
	{
		ID:                RuleFileCodeLines,
		Stage:             StageAST,
		Label:             "Production file code lines",
		DefaultLimit:      450,
		AllowedCategories: []AllowCategory{CategoryDispatch, CategoryProtocol},
		Why:               "Monolithic files accumulate unrelated responsibilities and degrade code navigation.",
		Fix:               "Split types and functions into separate domain-named files within the same package (avoid util.go).",
	},
	{
		ID:                RuleCatchAllFilename,
		Stage:             StageAST,
		Label:             "Catch-all filename",
		DefaultLimit:      0,
		AllowedCategories: nil,
		Why:               "Generic filenames like util.go, helpers.go, or common.go become low-cohesion dumping grounds.",
		Fix:               "Rename the file after the specific domain concept, data structure, or pipeline stage it implements.",
	},
	{
		ID:                RuleCodeClone,
		Stage:             StageAST,
		Label:             "Duplicated AST block",
		DefaultLimit:      15,
		AllowedCategories: []AllowCategory{CategoryProtocol},
		Why:               "Copy-pasted statement blocks force parallel bug fixes and drift silently over time.",
		Fix:               "Extract the shared control flow into a parameterized helper, generic function, or shared type.",
	},
	{
		ID:                RuleTRunCognitive,
		Stage:             StageAST,
		Label:             "Subtest t.Run complexity",
		DefaultLimit:      8,
		AllowedCategories: []AllowCategory{CategoryProtocol},
		Why:               "Branching logic inside t.Run bodies re-implements production logic in tests and hides assertion bugs.",
		Fix:               "Declare expected outputs as explicit literals in testCases and compare with a linear cmp.Diff(tc.want, got).",
	},
	{
		ID:                RuleTestHelperCognitive,
		Stage:             StageAST,
		Label:             "Test helper complexity",
		DefaultLimit:      15,
		AllowedCategories: []AllowCategory{CategoryProtocol},
		Why:               "Complex test helpers become untested production code inside *_test.go.",
		Fix:               "Simplify test fixtures or move reusable builders into tested test packages.",
	},
	{
		ID:                RuleTestFileCodeLines,
		Stage:             StageAST,
		Label:             "Test file code lines",
		DefaultLimit:      800,
		AllowedCategories: []AllowCategory{CategoryDispatch, CategoryProtocol},
		Why:               "Oversized test files obscure coverage gaps and slow maintenance.",
		Fix:               "Split production responsibilities into smaller files with matching *_test.go files or trim redundant fixtures.",
	},
	{
		ID:                RuleFuncUncoveredLines,
		Stage:             StageDynamic,
		Label:             "Uncovered executable lines",
		DefaultLimit:      0,
		AllowedCategories: []AllowCategory{CategoryExternal, CategoryObservability, CategoryPerf},
		Why:               "Substantive branches in touched functions were never executed by unit tests (boilerplate error guards are already excluded).",
		Fix:               "Add focused table-driven test cases covering the unexecuted branch, or delete unreachable production code.",
	},
	{
		ID:                RuleFuncSurvivingMutants,
		Stage:             StageDynamic,
		Label:             "Surviving mutants",
		DefaultLimit:      0,
		AllowedCategories: []AllowCategory{CategoryEquivalent, CategoryPerf, CategoryObservability, CategoryExternal},
		Why:               "Mutating operators or branches in this touched function did not cause any unit test to fail.",
		Fix:               "Delete redundant guards first, or strengthen assertions (cmp.Diff on full outputs and boundary inputs).",
	},
	{
		ID:                RuleTestRedundancy,
		Stage:             StageDynamic,
		Label:             "Redundant test cases",
		DefaultLimit:      0,
		AllowedCategories: nil,
		Why:               "Subtest exercises an identical branch-polarity path with no unique mutant kills, or checks only err != nil on an already-covered path.",
		Fix:               "Remove duplicate equivalence-class cases, or assert concrete output values so the case kills a unique mutant.",
	},
}

// RuleStage identifies whether a quality rule runs statically on ASTs or dynamically via test execution.
type RuleStage string

// RuleID uniquely identifies a quantitative Go quality rule enforced by codequality.
type RuleID string

// AllowCategory classifies why a HARD_LIMIT violation is legitimately suppressed via CODEQUALITY_ALLOW.
type AllowCategory string

// MutantOperator identifies the category of AST mutation applied to a Go expression or statement.
type MutantOperator string

// MutantStatus represents the evaluation outcome of a candidate mutant.
type MutantStatus string

// RuleSpec defines static metadata, default threshold, allowed suppression categories, and remediation hints for a RuleID.
type RuleSpec struct {
	// ID is the unique identifier for the rule.
	ID RuleID
	// Stage indicates whether the rule runs in AST or DYNAMIC stage.
	Stage RuleStage
	// Label is the short human-readable display name of the rule.
	Label string
	// DefaultLimit is the default upper defect or complexity bound.
	DefaultLimit int
	// AllowedCategories lists permitted CODEQUALITY_ALLOW categories for this rule.
	AllowedCategories []AllowCategory
	// Why explains the rationale for enforcing the rule.
	Why string
	// Fix provides actionable remediation guidance for violations.
	Fix string
}

// Thresholds configures upper defect/cost limits across all codequality rules.
type Thresholds struct {
	// Limits maps RuleIDs to configured upper bounds.
	Limits map[RuleID]int
	// MinParamsForFlagCheck is the minimum parameter count before checking for control-flow bool flags.
	MinParamsForFlagCheck int
}

// AllRuleSpecs returns a copy of all registered quality RuleSpecs in canonical display order.
func AllRuleSpecs() []RuleSpec {
	out := make([]RuleSpec, len(ruleSpecs))
	copy(out, ruleSpecs)
	return out
}

// LookupRuleSpec returns the RuleSpec registered for id.
func LookupRuleSpec(id RuleID) (RuleSpec, bool) {
	for _, spec := range ruleSpecs {
		if spec.ID == id {
			return spec, true
		}
	}
	return RuleSpec{}, false
}

// DefaultThresholds returns the standard KHI Go quality limits.
func DefaultThresholds() Thresholds {
	limits := make(map[RuleID]int, len(ruleSpecs))
	for _, spec := range ruleSpecs {
		limits[spec.ID] = spec.DefaultLimit
	}
	return Thresholds{
		Limits:                limits,
		MinParamsForFlagCheck: 3,
	}
}

// IsValid reports whether o is a recognized MutantOperator code.
func (o MutantOperator) IsValid() bool {
	switch o {
	case OperatorROR, OperatorLCR, OperatorBCR, OperatorAOR, OperatorUOI, OperatorBLR, OperatorSBR:
		return true
	default:
		return false
	}
}

// Label returns a concise human-readable description of what o mutates.
func (o MutantOperator) Label() string {
	switch o {
	case OperatorROR:
		return "boundary/comparison"
	case OperatorLCR:
		return "logical &&/||"
	case OperatorBCR:
		return "branch forced true/false"
	case OperatorAOR:
		return "arithmetic/concat"
	case OperatorUOI:
		return "negation/sign"
	case OperatorBLR:
		return "return bool flipped"
	case OperatorSBR:
		return "statement skipped"
	default:
		return string(o)
	}
}

// AllowsCategory reports whether cat is permitted in CODEQUALITY_ALLOW for spec.
func (s RuleSpec) AllowsCategory(cat AllowCategory) bool {
	for _, allowed := range s.AllowedCategories {
		if allowed == cat {
			return true
		}
	}
	return false
}

// Limit returns the configured upper limit for id, falling back to its RuleSpec default.
func (t Thresholds) Limit(id RuleID) int {
	if v, ok := t.Limits[id]; ok {
		return v
	}
	spec, _ := LookupRuleSpec(id)
	return spec.DefaultLimit
}
