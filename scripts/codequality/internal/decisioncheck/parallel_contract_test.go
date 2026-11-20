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
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

var _ Decider = (*concurrentBarrierDecider)(nil)

type concurrentBarrierDecider struct {
	barrierTarget     int32
	barrierTimeout    time.Duration
	completionTimeout time.Duration
	targetOrder       []string

	mu            sync.Mutex
	arrived       int32
	barrierClosed bool
	barrierCh     chan struct{}
	doneByTarget  map[string]chan struct{}
	closedTargets map[string]bool

	inFlight    atomic.Int32
	maxInFlight atomic.Int32
	calls       atomic.Int32
}

func TestEvaluateParallelWorkers(t *testing.T) {
	testCases := []struct {
		name             string
		workers          int
		changedFiles     []string
		seedFiles        []string
		barrierTarget    int32
		reverseOrder     []string
		wantMinInFlight  int32
		wantDeciderCalls int32
		wantSummary      rule.DecisionSummary
	}{
		// Executes 4 uncached bundles concurrently under Workers=4 with reverse completion order and deterministic sorted findings.
		{
			name:             "concurrent 4 workers cold cache reverse completion",
			workers:          4,
			changedFiles:     []string{"pkg/d/delta.go", "pkg/b/beta.go", "pkg/a/alpha.go", "pkg/c/gamma.go"},
			barrierTarget:    2,
			reverseOrder:     []string{"pkg/d/delta.go", "pkg/c/gamma.go", "pkg/b/beta.go", "pkg/a/alpha.go"},
			wantMinInFlight:  2,
			wantDeciderCalls: 4,
			wantSummary:      buildFourBundleSummary(0, 4),
		},
		// Serves cached bundles without API calls while concurrently evaluating uncached bundles and aggregating totals.
		{
			name:             "concurrent 4 workers mixed cache hits and misses",
			workers:          4,
			changedFiles:     []string{"pkg/a/alpha.go", "pkg/b/beta.go", "pkg/c/gamma.go", "pkg/d/delta.go"},
			seedFiles:        []string{"pkg/a/alpha.go", "pkg/c/gamma.go"},
			barrierTarget:    2,
			reverseOrder:     []string{"pkg/d/delta.go", "pkg/b/beta.go"},
			wantMinInFlight:  2,
			wantDeciderCalls: 2,
			wantSummary:      buildFourBundleSummary(2, 2),
		},
		// Executes bundles sequentially when Workers is 1 and produces the same deterministic sorted findings.
		{
			name:             "sequential 1 worker baseline",
			workers:          1,
			changedFiles:     []string{"pkg/d/delta.go", "pkg/a/alpha.go", "pkg/c/gamma.go", "pkg/b/beta.go"},
			barrierTarget:    1,
			wantMinInFlight:  1,
			wantDeciderCalls: 4,
			wantSummary:      buildFourBundleSummary(0, 4),
		},
	}

	for _, tc := range testCases {
		ctx := context.Background()
		repo, spec := setupParallelTargetsFixture(t)
		cache := openParallelMemoryCache(t, ctx)
		seedDecisionCacheEntries(t, ctx, repo, spec, cache, tc.seedFiles)
		dec := newConcurrentBarrierDecider(tc.barrierTarget, tc.reverseOrder)
		opts := Options{
			Rules:         []RuleSpec{spec},
			ForceBlocking: true,
			Workers:       tc.workers,
			TargetOpts: ResolveOptions{
				RepoRoot:     repo,
				ChangedFiles: tc.changedFiles,
			},
			Cache: cache,
			ResolveDecider: func(context.Context) (Decider, string, error) {
				return dec, "", nil
			},
		}

		gotSummary, evalErr := Evaluate(ctx, opts)

		if evalErr != nil {
			t.Fatalf("%s: Evaluate() unexpected error: %v", tc.name, evalErr)
		}
		if diff := cmp.Diff(tc.wantSummary, gotSummary); diff != "" {
			t.Errorf("%s: Evaluate() mismatch (-want +got):\n%s", tc.name, diff)
		}
		if gotMax := dec.maxInFlight.Load(); gotMax < tc.wantMinInFlight {
			t.Errorf("%s: maxInFlight = %d, want >= %d", tc.name, gotMax, tc.wantMinInFlight)
		}
		if gotCalls := dec.calls.Load(); gotCalls != tc.wantDeciderCalls {
			t.Errorf("%s: decider calls = %d, want %d", tc.name, gotCalls, tc.wantDeciderCalls)
		}
	}
}

func newConcurrentBarrierDecider(barrierTarget int32, targetOrder []string) *concurrentBarrierDecider {
	doneByTarget := make(map[string]chan struct{}, len(targetOrder))
	for _, target := range targetOrder {
		doneByTarget[target] = make(chan struct{})
	}
	return &concurrentBarrierDecider{
		barrierTarget:     barrierTarget,
		barrierTimeout:    2 * time.Second,
		completionTimeout: 500 * time.Millisecond,
		targetOrder:       targetOrder,
		barrierCh:         make(chan struct{}),
		doneByTarget:      doneByTarget,
		closedTargets:     make(map[string]bool, len(targetOrder)),
	}
}

func (d *concurrentBarrierDecider) Decide(ctx context.Context, req DecisionRequest) (DecisionResponse, error) {
	current := d.inFlight.Add(1)
	defer d.inFlight.Add(-1)
	d.recordMaxInFlight(current)
	d.calls.Add(1)

	d.waitForBarrier(ctx)
	d.waitAndSignalReverseOrder(ctx, req.Input)

	pLow := 0.10
	return DecisionResponse{
		Answers: map[string]WireAnswer{
			"unit_0": {Noul: &pLow, Confidence: 0.90},
		},
	}, nil
}

func (d *concurrentBarrierDecider) recordMaxInFlight(current int32) {
	for {
		prev := d.maxInFlight.Load()
		if current <= prev || d.maxInFlight.CompareAndSwap(prev, current) {
			return
		}
	}
}

func (d *concurrentBarrierDecider) waitForBarrier(ctx context.Context) {
	if d.barrierTarget <= 1 {
		return
	}
	d.mu.Lock()
	d.arrived++
	if d.arrived >= d.barrierTarget && !d.barrierClosed {
		d.barrierClosed = true
		close(d.barrierCh)
	}
	d.mu.Unlock()

	timer := time.NewTimer(d.barrierTimeout)
	defer timer.Stop()
	select {
	case <-d.barrierCh:
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (d *concurrentBarrierDecider) waitAndSignalReverseOrder(ctx context.Context, sharedInput string) {
	targetIdx := -1
	for i, target := range d.targetOrder {
		if strings.Contains(sharedInput, "=== TARGET: "+target+" ===") {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return
	}
	if targetIdx > 0 {
		prevCh := d.doneByTarget[d.targetOrder[targetIdx-1]]
		timer := time.NewTimer(d.completionTimeout)
		select {
		case <-prevCh:
		case <-ctx.Done():
		case <-timer.C:
		}
		timer.Stop()
	}

	target := d.targetOrder[targetIdx]
	d.mu.Lock()
	if !d.closedTargets[target] {
		d.closedTargets[target] = true
		close(d.doneByTarget[target])
	}
	d.mu.Unlock()
}

func setupParallelTargetsFixture(t *testing.T) (string, RuleSpec) {
	t.Helper()
	repo := t.TempDir()
	files := map[string]string{
		"pkg/a/alpha.go": "package a\n\ntype Alpha struct{}\n",
		"pkg/b/beta.go":  "package b\n\ntype Beta struct{}\n",
		"pkg/c/gamma.go": "package c\n\ntype Gamma struct{}\n",
		"pkg/d/delta.go": "package d\n\ntype Delta struct{}\n",
	}
	for rel, content := range files {
		full := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("os.MkdirAll error: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatalf("os.WriteFile error: %v", err)
		}
	}
	spec, err := ParseRuleYAML([]byte(`
id: file_doc_alignment
name: File Doc Alignment
target:
  kind: changed_file
  include: ["pkg/**/*.go"]
questions:
  - id: unit
    for_each: go_substantive_units
    type: noul
    instructions: "Is {{.Name}} valid?"
    criteria:
      false_description: "{{.Name}} failed check."
    fix_hint: "Fix {{.Name}}."
    pass:
      expect: true
      min_probability: 0.50
`))
	if err != nil {
		t.Fatalf("ParseRuleYAML() unexpected error: %v", err)
	}
	return repo, spec
}

func openParallelMemoryCache(t *testing.T, ctx context.Context) *DecisionCache {
	t.Helper()
	t.Chdir(t.TempDir())
	cache, err := OpenDecisionCache(ctx, "")
	if err != nil {
		t.Fatalf("OpenDecisionCache() unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = cache.Close() })
	return cache
}

func seedDecisionCacheEntries(t *testing.T, ctx context.Context, repo string, spec RuleSpec, cache *DecisionCache, seedFiles []string) {
	t.Helper()
	if len(seedFiles) == 0 {
		return
	}
	seedDecider := newConcurrentBarrierDecider(1, nil)
	_, err := Evaluate(ctx, Options{
		Rules:         []RuleSpec{spec},
		ForceBlocking: true,
		Workers:       1,
		TargetOpts: ResolveOptions{
			RepoRoot:     repo,
			ChangedFiles: seedFiles,
		},
		Cache: cache,
		ResolveDecider: func(context.Context) (Decider, string, error) {
			return seedDecider, "", nil
		},
	})
	if err != nil {
		t.Fatalf("seed Evaluate() unexpected error: %v", err)
	}
}

func buildFourBundleSummary(cacheHits int, apiCalls int) rule.DecisionSummary {
	type targetEntry struct {
		file    string
		subject string
	}
	entries := []targetEntry{
		{file: "pkg/a/alpha.go", subject: "alpha.go (Alpha)"},
		{file: "pkg/b/beta.go", subject: "beta.go (Beta)"},
		{file: "pkg/c/gamma.go", subject: "gamma.go (Gamma)"},
		{file: "pkg/d/delta.go", subject: "delta.go (Delta)"},
	}
	findings := make([]rule.DecisionItemFinding, len(entries))
	for i, entry := range entries {
		findings[i] = rule.DecisionItemFinding{
			Index:      i + 1,
			RuleID:     "file_doc_alignment",
			RuleName:   "File Doc Alignment",
			QuestionID: "unit",
			Blocking:   true,
			Target:     entry.file,
			File:       entry.file,
			Line:       1,
			EndLine:    3,
			Subject:    entry.subject,
			Outcome:    "p_true=0.10 (want true >= 0.50, conf=0.90)",
			Why:        entry.subject + " failed check.",
			Fix:        "Fix " + entry.subject + ".",
		}
	}
	return rule.DecisionSummary{
		Enabled:            true,
		RulesEvaluated:     1,
		TargetsEvaluated:   4,
		QuestionsEvaluated: 4,
		CacheHits:          cacheHits,
		APICalls:           apiCalls,
		Findings:           findings,
	}
}
