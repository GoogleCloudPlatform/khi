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
	"cmp"
	"context"
	"slices"
	"sync"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

const defaultEvaluationWorkers = 4

// Options configures a multi-rule Gemini Decisions evaluation run.
type Options struct {
	// Rules lists the pre-loaded YAML RuleSpecs to consider.
	Rules []RuleSpec
	// RuleFilter restricts execution to specific rule IDs when non-empty.
	RuleFilter map[string]struct{}
	// Model specifies the Gemini Decision model ID (defaults to DefaultDecisionsModel).
	Model string
	// ForceBlocking promotes all evaluated rules to blocking violations when true.
	ForceBlocking bool
	// Workers bounds concurrent target bundle evaluations (defaults to 4 when <= 0).
	Workers int
	// TargetOpts configures target bundle discovery and context rendering.
	TargetOpts ResolveOptions
	// Cache optionally persists and serves cached DecisionResponses.
	Cache *DecisionCache
	// ResolveDecider lazily resolves the Decider backend only when a cache miss requires a live API call.
	ResolveDecider func(ctx context.Context) (Decider, string, error)
}

type batchLookupResult struct {
	resp       DecisionResponse
	cacheHit   bool
	skipReason string
}

type ruleEvalOutcome struct {
	targets   int
	questions int
	cacheHits int
	apiCalls  int
	skipped   bool
	skipMsg   string
	findings  []rule.DecisionItemFinding
}

// Evaluate executes all active Rules in opts against matching targets and returns a deterministic DecisionSummary.
func Evaluate(ctx context.Context, opts Options) (rule.DecisionSummary, error) {
	activeRules := selectActiveRules(opts.Rules, opts.RuleFilter)
	if len(activeRules) == 0 {
		return rule.DecisionSummary{
			Enabled:    true,
			Skipped:    true,
			SkipReason: "no enabled decision rules matched",
		}, nil
	}

	model := cmp.Or(opts.Model, DefaultDecisionsModel)
	sum := rule.DecisionSummary{
		Enabled:        true,
		RulesEvaluated: len(activeRules),
	}

	for _, r := range activeRules {
		out, err := evaluateRule(ctx, model, r, opts)
		if err != nil {
			return rule.DecisionSummary{}, err
		}
		if out.skipped {
			return rule.DecisionSummary{
				Enabled:    true,
				Skipped:    true,
				SkipReason: out.skipMsg,
			}, nil
		}
		sum.TargetsEvaluated += out.targets
		sum.QuestionsEvaluated += out.questions
		sum.CacheHits += out.cacheHits
		sum.APICalls += out.apiCalls
		sum.Findings = append(sum.Findings, out.findings...)
	}

	sortAndNumberFindings(sum.Findings)
	return sum, nil
}

func selectActiveRules(rules []RuleSpec, filter map[string]struct{}) []RuleSpec {
	var out []RuleSpec
	for _, r := range rules {
		if len(filter) > 0 {
			if _, ok := filter[r.ID]; ok {
				out = append(out, r)
			}
			continue
		}
		if r.Enabled {
			out = append(out, r)
		}
	}
	return out
}

func evaluateRule(ctx context.Context, model string, r RuleSpec, opts Options) (ruleEvalOutcome, error) {
	bundles, err := ResolveTargets(r, opts.TargetOpts)
	if err != nil {
		return ruleEvalOutcome{}, err
	}
	out := ruleEvalOutcome{targets: len(bundles)}
	if len(bundles) == 0 {
		return out, nil
	}
	results, err := evaluateBundlesConcurrently(ctx, model, r, bundles, opts)
	if err != nil {
		return ruleEvalOutcome{}, err
	}
	for _, batchOut := range results {
		if batchOut.skipped {
			return batchOut, nil
		}
		out.questions += batchOut.questions
		out.cacheHits += batchOut.cacheHits
		out.apiCalls += batchOut.apiCalls
		out.findings = append(out.findings, batchOut.findings...)
	}
	return out, nil
}

func evaluateBundlesConcurrently(ctx context.Context, model string, r RuleSpec, bundles []TargetBundle, opts Options) ([]ruleEvalOutcome, error) {
	workers := opts.Workers
	if workers <= 0 {
		workers = defaultEvaluationWorkers
	}
	workerCount := min(workers, len(bundles))
	results := make([]ruleEvalOutcome, len(bundles))
	errs := make([]error, len(bundles))
	jobs := make(chan int, len(bundles))
	for i := range bundles {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				results[idx], errs[idx] = evaluateSingleBundle(ctx, model, r, bundles[idx], opts)
			}
		}()
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

func evaluateSingleBundle(ctx context.Context, model string, r RuleSpec, bundle TargetBundle, opts Options) (ruleEvalOutcome, error) {
	sharedInput, err := RenderSharedInput(r, bundle, opts.TargetOpts)
	if err != nil {
		return ruleEvalOutcome{}, err
	}
	batches, err := CompileQuestions(ctx, model, sharedInput, r, bundle)
	if err != nil {
		return ruleEvalOutcome{}, err
	}
	return executeCompiledBatches(ctx, r, batches, opts)
}

func executeCompiledBatches(ctx context.Context, r RuleSpec, batches []CompiledBatch, opts Options) (ruleEvalOutcome, error) {
	var out ruleEvalOutcome
	for _, batch := range batches {
		out.questions += len(batch.Order)
		lookup, err := fetchOrLookupBatch(ctx, batch.Request, opts)
		if err != nil {
			return ruleEvalOutcome{}, err
		}
		if lookup.skipReason != "" {
			return ruleEvalOutcome{skipped: true, skipMsg: lookup.skipReason}, nil
		}
		if lookup.cacheHit {
			out.cacheHits++
		} else {
			out.apiCalls++
		}
		findings, evalErr := EvaluateBatch(r, opts.ForceBlocking, batch, lookup.resp)
		if evalErr != nil {
			return ruleEvalOutcome{}, evalErr
		}
		out.findings = append(out.findings, findings...)
	}
	return out, nil
}

func fetchOrLookupBatch(ctx context.Context, req DecisionRequest, opts Options) (batchLookupResult, error) {
	key, err := ComputeBatchCacheKey(req)
	if err != nil {
		return batchLookupResult{}, err
	}
	if cached, ok, lookupErr := opts.Cache.Lookup(ctx, key); lookupErr != nil {
		return batchLookupResult{}, lookupErr
	} else if ok {
		return batchLookupResult{resp: cached, cacheHit: true}, nil
	}

	if opts.ResolveDecider == nil {
		return batchLookupResult{skipReason: "decision client not configured"}, nil
	}
	decider, skipReason, err := opts.ResolveDecider(ctx)
	if err != nil || decider == nil {
		return batchLookupResult{skipReason: cmp.Or(skipReason, "decision client unavailable")}, err
	}

	resp, err := decider.Decide(ctx, req)
	if err != nil {
		return batchLookupResult{}, err
	}
	if err := opts.Cache.Store(ctx, key, resp); err != nil {
		return batchLookupResult{}, err
	}
	return batchLookupResult{resp: resp}, nil
}

func sortAndNumberFindings(findings []rule.DecisionItemFinding) {
	slices.SortFunc(findings, func(a, b rule.DecisionItemFinding) int {
		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.RuleID, b.RuleID),
			cmp.Compare(a.QuestionID, b.QuestionID),
			cmp.Compare(a.Subject, b.Subject),
		)
	})
	for i := range findings {
		findings[i].Index = i + 1
	}
}
