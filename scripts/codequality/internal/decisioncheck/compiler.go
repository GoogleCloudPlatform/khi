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
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// MaxQuestionsPerRequest is the maximum number of structured questions per Gemini Decisions API request.
const MaxQuestionsPerRequest = 1000

// DecisionRequest is the wire request body sent to POST /v1beta/decisions.
type DecisionRequest struct {
	// Model identifies the Gemini Decision model (for example "gemini-decision-1").
	Model string `json:"model"`
	// Input holds the shared text context evaluated across all Questions.
	Input string `json:"input"`
	// Questions maps caller-assigned question keys to structured WireQuestions.
	Questions map[string]WireQuestion `json:"questions"`
}

// WireQuestion represents one typed closed-set question in a DecisionRequest.
type WireQuestion struct {
	// Type is "noul", "choice", or "score".
	Type QuestionType `json:"type"`
	// Instructions holds the rendered prompt text for this question.
	Instructions string `json:"instructions"`
	// Criteria holds type-specific anchor descriptions (map for noul/choice, []string for score).
	Criteria any `json:"criteria,omitempty"`
}

// DecisionResponse is the wire response body returned by POST /v1beta/decisions.
type DecisionResponse struct {
	// Answers maps requested question keys to calibrated model WireAnswers.
	Answers map[string]WireAnswer `json:"answers"`
}

// WireAnswer holds the calibrated outcome probabilities and score for a single question.
type WireAnswer struct {
	// Noul holds calibrated P(true) in [0.01, 0.99] for noul questions.
	Noul *float64 `json:"noul,omitempty"`
	// Choice holds the top-selected option key for choice questions.
	Choice string `json:"choice,omitempty"`
	// Score holds the probability-weighted expected level index for score questions.
	Score *float64 `json:"score,omitempty"`
	// Probabilities maps option keys or level indices to normalized probabilities.
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	// Confidence holds the model's prediction confidence in [0, 1].
	Confidence float64 `json:"confidence"`
}

// CompiledBatch pairs a wire DecisionRequest (<= 1,000 questions) with local evaluation metadata.
type CompiledBatch struct {
	// Request is the wire payload sent to the Decisions API.
	Request DecisionRequest
	// Order lists question keys in deterministic evaluation order.
	Order []string
	// Meta maps each wire question key to its source location, pass spec, and outcome templates.
	Meta map[string]compiledQuestionMeta
}

type compiledQuestionMeta struct {
	spec QuestionSpec
	item ExtractedItem
}

type findingSummary struct {
	blocking bool
	outcome  string
	why      string
	fix      string
}

// CompileQuestions expands all questions in r over bundle and partitions them into <=1,000-question batches.
func CompileQuestions(model string, sharedInput string, r RuleSpec, bundle TargetBundle) ([]CompiledBatch, error) {
	type keyedEntry struct {
		key  string
		wire WireQuestion
		meta compiledQuestionMeta
	}
	var all []keyedEntry
	for _, q := range r.Questions {
		items, err := extractItemsForQuestion(q.ForEach, bundle)
		if err != nil {
			return nil, err
		}
		for idx, item := range items {
			wireQ, buildErr := renderWireQuestion(q, item)
			if buildErr != nil {
				return nil, fmt.Errorf("rule %q question %q: %w", r.ID, q.ID, buildErr)
			}
			key := formatExpandedKey(q.ID, q.ForEach, idx)
			all = append(all, keyedEntry{
				key:  key,
				wire: wireQ,
				meta: compiledQuestionMeta{spec: q, item: item},
			})
		}
	}

	var batches []CompiledBatch
	for chunk := range slices.Chunk(all, MaxQuestionsPerRequest) {
		qMap := make(map[string]WireQuestion, len(chunk))
		mMap := make(map[string]compiledQuestionMeta, len(chunk))
		order := make([]string, 0, len(chunk))
		for _, entry := range chunk {
			order = append(order, entry.key)
			qMap[entry.key] = entry.wire
			mMap[entry.key] = entry.meta
		}
		batches = append(batches, CompiledBatch{
			Request: DecisionRequest{
				Model:     model,
				Input:     sharedInput,
				Questions: qMap,
			},
			Order: order,
			Meta:  mMap,
		})
	}
	return batches, nil
}

// EvaluateBatch checks resp against batch pass predicates and returns deterministic findings.
func EvaluateBatch(r RuleSpec, forceBlocking bool, batch CompiledBatch, resp DecisionResponse) ([]rule.DecisionItemFinding, error) {
	blocking := forceBlocking || r.Enforcement == EnforcementBlocking
	var findings []rule.DecisionItemFinding
	for _, key := range batch.Order {
		meta := batch.Meta[key]
		ans, ok := resp.Answers[key]
		if !ok {
			return nil, fmt.Errorf("missing answer for question %q in rule %q", key, r.ID)
		}
		finding, failed, err := evaluateSingleAnswer(r, blocking, meta, ans)
		if err != nil {
			return nil, err
		}
		if failed {
			findings = append(findings, finding)
		}
	}
	return findings, nil
}

// CanonicalQuestionsJSON encodes questionsByKey as JSON for content-addressed batch caching.
func CanonicalQuestionsJSON(questionsByKey map[string]WireQuestion) ([]byte, error) {
	return json.Marshal(questionsByKey)
}

func extractItemsForQuestion(kind ExtractorKind, bundle TargetBundle) ([]ExtractedItem, error) {
	switch kind {
	case "", ExtractorBundle:
		return []ExtractedItem{{
			Target:    bundle.ID,
			File:      bundle.PrimaryFile,
			Line:      1,
			StartLine: 1,
			EndLine:   1,
			Name:      bundle.ID,
			Kind:      "bundle",
		}}, nil
	case ExtractorMarkdownSections:
		var out []ExtractedItem
		for _, bf := range bundle.Files {
			if strings.HasSuffix(strings.ToLower(bf.RelPath), ".md") {
				out = append(out, ExtractMarkdownSections(bundle.ID, bf.RelPath, string(bf.Content))...)
			}
		}
		return out, nil
	case ExtractorGoSubstantiveUnits:
		return bundle.SubstantiveGoUnits, nil
	case ExtractorChangedGoDecls:
		return ExtractChangedGoDecls(bundle.ID, bundle.Files, bundle.ChangedLines)
	case ExtractorChangedGoFuncs:
		return ExtractChangedGoFuncs(bundle.ID, bundle.Files, bundle.ChangedLines)
	case ExtractorChangedGoTestFuncs:
		return ExtractChangedGoTestFuncs(bundle.ID, bundle.Files, bundle.ChangedLines)
	case ExtractorChangedGoTestCases:
		return ExtractChangedGoTestCases(bundle.ID, bundle.Files, bundle.ChangedLines)
	case ExtractorChangedTSDecls:
		return ExtractChangedTSDecls(bundle.ID, bundle.WebDir, bundle.Files, bundle.ChangedLines)
	case ExtractorChangedTSFuncs:
		return ExtractChangedTSFuncs(bundle.ID, bundle.WebDir, bundle.Files, bundle.ChangedLines)
	case ExtractorChangedTSSpecs:
		return ExtractChangedTSSpecs(bundle.ID, bundle.WebDir, bundle.Files, bundle.ChangedLines)
	default:
		return nil, fmt.Errorf("unsupported extractor %q", kind)
	}
}

func renderWireQuestion(q QuestionSpec, item ExtractedItem) (WireQuestion, error) {
	inst, err := renderTemplate(q.Instructions, item)
	if err != nil {
		return WireQuestion{}, err
	}
	criteria, err := renderWireCriteria(q, item)
	if err != nil {
		return WireQuestion{}, err
	}
	return WireQuestion{
		Type:         q.Type,
		Instructions: inst,
		Criteria:     criteria,
	}, nil
}

func renderWireCriteria(q QuestionSpec, item ExtractedItem) (any, error) {
	switch q.Type {
	case QuestionTypeNoul:
		return renderNoulCriteria(q.Criteria, item)
	case QuestionTypeChoice:
		return renderChoiceCriteria(q.Options, item)
	case QuestionTypeScore:
		return renderScoreCriteria(q.Levels, item)
	default:
		return nil, fmt.Errorf("unsupported question type %q", q.Type)
	}
}

func renderNoulCriteria(spec NoulCriteriaSpec, item ExtractedItem) (any, error) {
	trueDesc, err := renderTemplate(spec.TrueDescription, item)
	if err != nil {
		return nil, err
	}
	falseDesc, err := renderTemplate(spec.FalseDescription, item)
	if err != nil {
		return nil, err
	}
	criteriaByKey := make(map[string]string, 2)
	if trueDesc != "" {
		criteriaByKey["true_description"] = trueDesc
	}
	if falseDesc != "" {
		criteriaByKey["false_description"] = falseDesc
	}
	if len(criteriaByKey) == 0 {
		return nil, nil
	}
	return criteriaByKey, nil
}

func renderChoiceCriteria(options map[string]ChoiceOptionSpec, item ExtractedItem) (map[string]string, error) {
	criteriaByKey := make(map[string]string, len(options))
	for optionKey, optionSpec := range options {
		desc, err := renderTemplate(optionSpec.Description, item)
		if err != nil {
			return nil, err
		}
		criteriaByKey[optionKey] = desc
	}
	return criteriaByKey, nil
}

func renderScoreCriteria(levels []ScoreLevelSpec, item ExtractedItem) ([]string, error) {
	levelDescriptions := make([]string, len(levels))
	for levelIdx, levelSpec := range levels {
		desc, err := renderTemplate(levelSpec.Description, item)
		if err != nil {
			return nil, err
		}
		levelDescriptions[levelIdx] = desc
	}
	return levelDescriptions, nil
}

func evaluateSingleAnswer(r RuleSpec, blocking bool, meta compiledQuestionMeta, ans WireAnswer) (rule.DecisionItemFinding, bool, error) {
	q := meta.spec
	if ans.Confidence < q.Pass.MinConfidence {
		return rule.DecisionItemFinding{}, false, nil
	}
	switch q.Type {
	case QuestionTypeNoul:
		return evaluateNoulAnswer(r, blocking, meta, ans)
	case QuestionTypeChoice:
		return evaluateChoiceAnswer(r, blocking, meta, ans)
	case QuestionTypeScore:
		return evaluateScoreAnswer(r, blocking, meta, ans)
	default:
		return rule.DecisionItemFinding{}, false, fmt.Errorf("unsupported question type %q", q.Type)
	}
}

func evaluateNoulAnswer(r RuleSpec, blocking bool, meta compiledQuestionMeta, ans WireAnswer) (rule.DecisionItemFinding, bool, error) {
	if ans.Noul == nil {
		return rule.DecisionItemFinding{}, false, fmt.Errorf("noul answer missing probability for question %q", meta.spec.ID)
	}
	pTrue := *ans.Noul
	wantTrue := *meta.spec.Pass.Expect
	actualProb := pTrue
	whyTmpl := meta.spec.Criteria.FalseDescription
	if !wantTrue {
		actualProb = 1.0 - pTrue
		whyTmpl = meta.spec.Criteria.TrueDescription
	}
	if actualProb >= meta.spec.Pass.MinProbability {
		return rule.DecisionItemFinding{}, false, nil
	}
	why, err := renderTemplate(cmp.Or(whyTmpl, meta.spec.Instructions), meta.item)
	if err != nil {
		return rule.DecisionItemFinding{}, false, err
	}
	fix, err := renderTemplate(meta.spec.FixHint, meta.item)
	if err != nil {
		return rule.DecisionItemFinding{}, false, err
	}
	outcome := fmt.Sprintf("p_true=%.2f (want %t >= %.2f, conf=%.2f)", pTrue, wantTrue, meta.spec.Pass.MinProbability, ans.Confidence)
	return newItemFinding(r, meta, findingSummary{blocking: blocking, outcome: outcome, why: why, fix: fix}), true, nil
}

func evaluateChoiceAnswer(r RuleSpec, blocking bool, meta compiledQuestionMeta, ans WireAnswer) (rule.DecisionItemFinding, bool, error) {
	allowed := meta.spec.Pass.AllowedPassChoices()
	passProb := sumChoiceProbabilities(allowed, ans.Probabilities)
	_, selectedAllowed := allowed[ans.Choice]
	if selectedAllowed && passProb >= meta.spec.Pass.MinProbability {
		return rule.DecisionItemFinding{}, false, nil
	}
	diagKey := resolveFailureChoiceKey(meta.spec.Options, allowed, ans)
	opt := meta.spec.Options[diagKey]
	why, err := renderTemplate(cmp.Or(opt.Description, meta.spec.Instructions), meta.item)
	if err != nil {
		return rule.DecisionItemFinding{}, false, err
	}
	fix, err := renderTemplate(cmp.Or(opt.FixHint, meta.spec.FixHint), meta.item)
	if err != nil {
		return rule.DecisionItemFinding{}, false, err
	}
	selProb := ans.Probabilities[diagKey]
	outcome := fmt.Sprintf("%s (p=%.2f, pass_p=%.2f, conf=%.2f)", diagKey, selProb, passProb, ans.Confidence)
	return newItemFinding(r, meta, findingSummary{blocking: blocking, outcome: outcome, why: why, fix: fix}), true, nil
}

func evaluateScoreAnswer(r RuleSpec, blocking bool, meta compiledQuestionMeta, ans WireAnswer) (rule.DecisionItemFinding, bool, error) {
	if ans.Score == nil {
		return rule.DecisionItemFinding{}, false, fmt.Errorf("score answer missing numeric score for question %q", meta.spec.ID)
	}
	scoreVal := *ans.Score
	p := meta.spec.Pass
	belowMin := p.MinScore != nil && scoreVal < *p.MinScore
	aboveMax := p.MaxScore != nil && scoreVal > *p.MaxScore
	if !belowMin && !aboveMax {
		return rule.DecisionItemFinding{}, false, nil
	}
	lvlIdx := mostLikelyFailingLevel(meta.spec, ans.Probabilities, scoreVal)
	lvl := meta.spec.Levels[lvlIdx]
	why, err := renderTemplate(lvl.Description, meta.item)
	if err != nil {
		return rule.DecisionItemFinding{}, false, err
	}
	fix, err := renderTemplate(cmp.Or(lvl.FixHint, meta.spec.FixHint), meta.item)
	if err != nil {
		return rule.DecisionItemFinding{}, false, err
	}
	outcome := formatScoreOutcome(scoreVal, len(meta.spec.Levels)-1, p, ans.Confidence)
	return newItemFinding(r, meta, findingSummary{blocking: blocking, outcome: outcome, why: why, fix: fix}), true, nil
}

func formatScoreOutcome(scoreVal float64, maxIdx int, p PassSpec, conf float64) string {
	if p.MinScore != nil {
		return fmt.Sprintf("score=%.2f/%d (< %.2f, conf=%.2f)", scoreVal, maxIdx, *p.MinScore, conf)
	}
	return fmt.Sprintf("score=%.2f/%d (> %.2f, conf=%.2f)", scoreVal, maxIdx, *p.MaxScore, conf)
}

func sumChoiceProbabilities(allowed map[string]struct{}, probs map[string]float64) float64 {
	total := 0.0
	for k := range allowed {
		total += probs[k]
	}
	return total
}

func resolveFailureChoiceKey(options map[string]ChoiceOptionSpec, allowed map[string]struct{}, ans WireAnswer) string {
	if _, isPass := allowed[ans.Choice]; !isPass {
		if _, exists := options[ans.Choice]; exists {
			return ans.Choice
		}
	}
	bestKey := ""
	bestProb := -1.0
	for _, k := range slices.Sorted(maps.Keys(options)) {
		if _, isPass := allowed[k]; isPass {
			continue
		}
		if p := ans.Probabilities[k]; p > bestProb {
			bestProb = p
			bestKey = k
		}
	}
	return bestKey
}

func mostLikelyFailingLevel(q QuestionSpec, probs map[string]float64, scoreVal float64) int {
	bestIdx := -1
	bestProb := -1.0
	for i := range q.Levels {
		if !isFailingScoreLevel(i, q.Pass) {
			continue
		}
		p := probs[strconv.Itoa(i)]
		if p > bestProb {
			bestProb = p
			bestIdx = i
		}
	}
	if bestIdx >= 0 {
		return bestIdx
	}
	clamped := int(math.Round(scoreVal))
	return max(0, min(clamped, len(q.Levels)-1))
}

func newItemFinding(r RuleSpec, meta compiledQuestionMeta, sum findingSummary) rule.DecisionItemFinding {
	it := meta.item
	subject := firstNonEmpty(it.Heading, it.Name, it.Target)
	return rule.DecisionItemFinding{
		RuleID:     r.ID,
		RuleName:   r.Name,
		QuestionID: meta.spec.ID,
		Blocking:   sum.blocking,
		Target:     it.Target,
		File:       it.File,
		Line:       max(it.Line, 1),
		EndLine:    max(it.EndLine, max(it.Line, 1)),
		Subject:    subject,
		Outcome:    sum.outcome,
		Why:        sum.why,
		Fix:        sum.fix,
	}
}

func formatExpandedKey(qID string, kind ExtractorKind, idx int) string {
	if kind == "" || kind == ExtractorBundle {
		return qID
	}
	return fmt.Sprintf("%s_%d", qID, idx)
}

func renderTemplate(raw string, item ExtractedItem) (string, error) {
	if raw == "" {
		return "", nil
	}
	tmpl, err := template.New("q").Option("missingkey=error").Parse(raw)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, item); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}
