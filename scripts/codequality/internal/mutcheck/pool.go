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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/redundancy"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type pkgRedundancyTracker struct {
	mu               sync.Mutex
	pkgPlans         []*FilePlan
	changedFuncs     []redundancy.ChangedTestFunc
	leafOrder        []string
	polarityByLeaf   map[string]map[string]struct{}
	targetedRunRegex string
	mutantRuns       map[int]redundancy.VerboseTestResult
}

type mutantJob struct {
	mutant           *Mutant
	binPath          string
	pkgAbsDir        string
	perMutantTimeout time.Duration
	tracker          *pkgRedundancyTracker
}

type trackerSetup struct {
	binPath       string
	pkgLabel      string
	pkgAbsDir     string
	pkgWorkDir    string
	baselineDur   time.Duration
	perMutTimeout time.Duration
	workers       int
	baselineOut   []byte
	changedFuncs  []redundancy.ChangedTestFunc
	pkgPlans      []*FilePlan
	progress      ProgressHooks
}

type leafTask struct {
	idx  int
	leaf string
}

type leafTraceCollector struct {
	mu        sync.Mutex
	cfg       trackerSetup
	total     int
	doneCount int
	baseEnv   []string
	result    map[string]map[string]struct{}
}

func (t *pkgRedundancyTracker) recordMutantRun(mutantID int, res redundancy.VerboseTestResult) {
	t.mu.Lock()
	t.mutantRuns[mutantID] = res
	t.mu.Unlock()
}

func prepareRedundancyTracker(ctx context.Context, cfg trackerSetup) *pkgRedundancyTracker {
	parsed := redundancy.ParseVerboseTestOutput(cfg.baselineOut, false)
	fnSet := make(map[string]struct{}, len(cfg.changedFuncs))
	for _, fn := range cfg.changedFuncs {
		fnSet[fn.FuncName] = struct{}{}
	}
	var relevant []string
	for _, leaf := range parsed.LeafOrder {
		topFn, _ := redundancy.SplitTopAndSubtest(leaf)
		if _, ok := fnSet[topFn]; ok {
			relevant = append(relevant, leaf)
			if len(relevant) >= maxTracedSubtestsPerPkg {
				break
			}
		}
	}
	if len(relevant) < 2 {
		return nil
	}
	polarities := traceLeafPolarities(ctx, cfg, relevant)
	targetedRegex := ""
	if cfg.baselineDur >= fastVerboseBaselineCap {
		targetedRegex = redundancy.BuildTestFuncsRunRegex(cfg.changedFuncs)
	}
	return &pkgRedundancyTracker{
		pkgPlans:         cfg.pkgPlans,
		changedFuncs:     cfg.changedFuncs,
		leafOrder:        relevant,
		polarityByLeaf:   polarities,
		targetedRunRegex: targetedRegex,
		mutantRuns:       make(map[int]redundancy.VerboseTestResult),
	}
}

func (c *leafTraceCollector) runWorker(ctx context.Context, taskCh <-chan leafTask) {
	for task := range taskCh {
		if ctx.Err() != nil {
			return
		}
		tokens := executeLeafPolarityTrace(ctx, c.baseEnv, c.cfg, task.idx, task.leaf)
		c.record(task.leaf, tokens)
	}
}

func (c *leafTraceCollector) record(leaf string, tokens map[string]struct{}) {
	c.mu.Lock()
	if len(tokens) > 0 {
		c.result[leaf] = tokens
	}
	c.doneCount++
	curDone := c.doneCount
	c.mu.Unlock()
	if c.cfg.progress != nil {
		c.cfg.progress.SubtestProgress(c.cfg.pkgLabel, curDone, c.total)
	}
}

func traceLeafPolarities(ctx context.Context, cfg trackerSetup, leaves []string) map[string]map[string]struct{} {
	taskCh := make(chan leafTask, len(leaves))
	for i, leaf := range leaves {
		taskCh <- leafTask{idx: i, leaf: leaf}
	}
	close(taskCh)

	collector := &leafTraceCollector{
		cfg:     cfg,
		total:   len(leaves),
		baseEnv: os.Environ(),
		result:  make(map[string]map[string]struct{}, len(leaves)),
	}

	var wg sync.WaitGroup
	for range min(cfg.workers, len(leaves)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			collector.runWorker(ctx, taskCh)
		}()
	}
	wg.Wait()
	return collector.result
}

func executeLeafPolarityTrace(
	ctx context.Context,
	baseEnv []string,
	cfg trackerSetup,
	idx int,
	leaf string,
) map[string]struct{} {
	traceFile := filepath.Join(cfg.pkgWorkDir, fmt.Sprintf("leaf_%d.trace", idx))
	runCtx, cancel := context.WithTimeout(ctx, cfg.perMutTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, cfg.binPath, "-test.run="+redundancy.BuildSingleLeafRunRegex(leaf))
	cmd.Dir = cfg.pkgAbsDir
	cmd.Env = buildMutantEnv(baseEnv, 0, traceFile)
	_ = cmd.Run()
	tokens, _, err := ParsePolarityTraceFile(traceFile)
	if err != nil {
		return nil
	}
	return tokens
}

func runMutantWorkerPool(ctx context.Context, jobs []mutantJob, workers int, progress ProgressHooks) {
	jobCh := make(chan mutantJob, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	var wg sync.WaitGroup
	for range min(workers, max(len(jobs), 1)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			consumeMutantJobs(ctx, jobCh, progress)
		}()
	}
	wg.Wait()
}

func consumeMutantJobs(ctx context.Context, jobCh <-chan mutantJob, progress ProgressHooks) {
	baseEnv := os.Environ()
	for job := range jobCh {
		status := rule.MutantSkippedBudget
		if ctx.Err() == nil {
			status = evaluateSingleMutant(ctx, baseEnv, job)
		}
		job.mutant.Status = status
		if progress != nil {
			progress.RecordMutantResult(status)
		}
	}
}

func evaluateSingleMutant(parentCtx context.Context, baseEnv []string, job mutantJob) rule.MutantStatus {
	mutCtx, cancel := context.WithTimeout(parentCtx, job.perMutantTimeout)
	defer cancel()

	if job.tracker != nil {
		if status, done := runTrackedMutant(parentCtx, mutCtx, baseEnv, job); done {
			return status
		}
	}
	cmd := exec.CommandContext(mutCtx, job.binPath, "-test.failfast")
	cmd.Dir = job.pkgAbsDir
	cmd.Env = buildMutantEnv(baseEnv, job.mutant.ID, "")
	err := cmd.Run()
	return classifyRunError(parentCtx, mutCtx, err)
}

func runTrackedMutant(
	parentCtx context.Context,
	mutCtx context.Context,
	baseEnv []string,
	job mutantJob,
) (rule.MutantStatus, bool) {
	args := []string{"-test.v"}
	if job.tracker.targetedRunRegex != "" {
		args = append(args, "-test.run="+job.tracker.targetedRunRegex)
	}
	cmd := exec.CommandContext(mutCtx, job.binPath, args...)
	cmd.Dir = job.pkgAbsDir
	cmd.Env = buildMutantEnv(baseEnv, job.mutant.ID, "")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	status := classifyRunError(parentCtx, mutCtx, err)
	if status == rule.MutantSkippedBudget || status == rule.MutantTimedOut {
		return status, true
	}
	job.tracker.recordMutantRun(job.mutant.ID, redundancy.ParseVerboseTestOutput(out.Bytes(), err != nil))
	if err != nil {
		return rule.MutantKilled, true
	}
	if job.tracker.targetedRunRegex == "" {
		return rule.MutantSurvived, true
	}
	return "", false
}

func classifyRunError(parentCtx context.Context, mutCtx context.Context, err error) rule.MutantStatus {
	if parentCtx.Err() != nil {
		return rule.MutantSkippedBudget
	}
	if errors.Is(mutCtx.Err(), context.DeadlineExceeded) {
		return rule.MutantTimedOut
	}
	if err != nil {
		return rule.MutantKilled
	}
	return rule.MutantSurvived
}

func applyMutantBudget(jobs []mutantJob, maxMutants int) []mutantJob {
	if maxMutants <= 0 || len(jobs) <= maxMutants {
		return jobs
	}
	var fileOrder []string
	byFile := make(map[string][]mutantJob)
	for _, j := range jobs {
		f := j.mutant.File
		if _, seen := byFile[f]; !seen {
			fileOrder = append(fileOrder, f)
		}
		byFile[f] = append(byFile[f], j)
	}
	return selectRoundRobinJobs(fileOrder, byFile, maxMutants)
}

func selectRoundRobinJobs(fileOrder []string, byFile map[string][]mutantJob, maxMutants int) []mutantJob {
	selected := make([]mutantJob, 0, maxMutants)
	cursors := make(map[string]int, len(fileOrder))
	for len(selected) < maxMutants {
		for _, f := range fileOrder {
			if len(selected) >= maxMutants {
				break
			}
			idx := cursors[f]
			if idx < len(byFile[f]) {
				selected = append(selected, byFile[f][idx])
				cursors[f] = idx + 1
			}
		}
	}
	for _, f := range fileOrder {
		for i := cursors[f]; i < len(byFile[f]); i++ {
			byFile[f][i].mutant.Status = rule.MutantSkippedBudget
		}
	}
	return selected
}
