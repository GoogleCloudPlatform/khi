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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/redundancy"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type spyProgress struct {
	phases   atomic.Int32
	subtests atomic.Int32
	pools    atomic.Int32
	results  atomic.Int32
}

func (s *spyProgress) Phase(_ int, _ int, _ string)           { s.phases.Add(1) }
func (s *spyProgress) SubtestProgress(_ string, _ int, _ int) { s.subtests.Add(1) }
func (s *spyProgress) StartMutantPool(_ int, _ int)           { s.pools.Add(1) }
func (s *spyProgress) RecordMutantResult(_ rule.MutantStatus) { s.results.Add(1) }

func TestApplyMutantBudgetAndClassifyRunError(t *testing.T) {
	testCases := []struct {
		name        string
		maxMutants  int
		wantLen     int
		wantM2State rule.MutantStatus
	}{
		{
			name:        "round-robins across files hitting inner break and marks overflow skipped",
			maxMutants:  3,
			wantLen:     3,
			wantM2State: rule.MutantPending,
		},
		{
			name:        "unlimited budget keeps all jobs",
			maxMutants:  0,
			wantLen:     4,
			wantM2State: rule.MutantPending,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m1 := &Mutant{File: "a.go", Status: rule.MutantPending}
			m2 := &Mutant{File: "a.go", Status: rule.MutantPending}
			m3 := &Mutant{File: "b.go", Status: rule.MutantPending}
			m4 := &Mutant{File: "b.go", Status: rule.MutantPending}
			sel := applyMutantBudget([]mutantJob{{mutant: m1}, {mutant: m2}, {mutant: m3}, {mutant: m4}}, tc.maxMutants)
			if len(sel) != tc.wantLen || m2.Status != tc.wantM2State {
				t.Errorf("got len=%d m2=%s, want len=%d m2=%s", len(sel), m2.Status, tc.wantLen, tc.wantM2State)
			}
		})
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := classifyRunError(canceledCtx, context.Background(), errors.New("boom")); got != rule.MutantSkippedBudget {
		t.Errorf("classifyRunError(canceled) = %q, want %q", got, rule.MutantSkippedBudget)
	}

	expiredCtx, cancelTimeout := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancelTimeout()
	time.Sleep(time.Millisecond)
	if got := classifyRunError(context.Background(), expiredCtx, errors.New("timeout")); got != rule.MutantTimedOut {
		t.Errorf("classifyRunError(deadline) = %q, want %q", got, rule.MutantTimedOut)
	}
}

func TestTrackerSetupAndTrackedExecution(t *testing.T) {
	// Untouched TestOther has 2 leaves while changed TestFocus has only 1 -> must return nil.
	singleLeafOut := `=== RUN   TestOther
=== RUN   TestOther/a
=== RUN   TestOther/b
--- PASS: TestOther/a (0.00s)
--- PASS: TestOther/b (0.00s)
=== RUN   TestFocus
--- PASS: TestFocus (0.00s)
`
	if tr := prepareRedundancyTracker(context.Background(), trackerSetup{
		baselineOut:  []byte(singleLeafOut),
		changedFuncs: []redundancy.ChangedTestFunc{{FuncName: "TestFocus"}},
	}); tr != nil {
		t.Fatalf("prepareRedundancyTracker(1 leaf in changed func) = %+v, want nil", tr)
	}

	// 42 subtests in TestFocus caps at maxTracedSubtestsPerPkg (40) and slow baseline sets targetedRunRegex.
	var sb strings.Builder
	sb.WriteString("=== RUN   TestFocus\n")
	for i := range 42 {
		fmt.Fprintf(&sb, "=== RUN   TestFocus/sub_%02d\n--- PASS: TestFocus/sub_%02d (0.00s)\n", i, i)
	}
	prog := &spyProgress{}
	workDir := t.TempDir()
	tr := prepareRedundancyTracker(context.Background(), trackerSetup{
		binPath:       "/bin/true",
		pkgLabel:      "./svc",
		pkgAbsDir:     workDir,
		pkgWorkDir:    workDir,
		baselineDur:   time.Second,
		perMutTimeout: time.Second,
		workers:       2,
		baselineOut:   []byte(sb.String()),
		changedFuncs:  []redundancy.ChangedTestFunc{{FuncName: "TestFocus"}},
		progress:      prog,
	})
	if tr == nil || len(tr.leafOrder) != maxTracedSubtestsPerPkg || tr.targetedRunRegex == "" {
		t.Fatalf("unexpected tracker setup: %+v", tr)
	}
	if prog.subtests.Load() != maxTracedSubtestsPerPkg {
		t.Errorf("SubtestProgress calls = %d, want %d", prog.subtests.Load(), maxTracedSubtestsPerPkg)
	}

	job := mutantJob{
		mutant:           &Mutant{ID: 1, Status: rule.MutantPending},
		binPath:          "/bin/true",
		pkgAbsDir:        workDir,
		perMutantTimeout: time.Second,
		tracker:          tr,
	}
	// Passing targeted run returns ("", false) so caller falls back to full binary run.
	if st, done := runTrackedMutant(context.Background(), context.Background(), nil, job); done || st != "" {
		t.Errorf("runTrackedMutant(targeted pass) = (%q, %v), want (\"\", false)", st, done)
	}
	// Canceled parent context returns (MutantSkippedBudget, true) immediately.
	canceledParent, cancel := context.WithCancel(context.Background())
	cancel()
	if st, done := runTrackedMutant(canceledParent, context.Background(), nil, job); !done || st != rule.MutantSkippedBudget {
		t.Errorf("runTrackedMutant(canceled) = (%q, %v), want (%q, true)", st, done, rule.MutantSkippedBudget)
	}
}

func TestMutantExecutionPassesShortFlag(t *testing.T) {
	testCases := []struct {
		name     string
		run      func(workDir string, scriptPath string)
		wantArgs []string
	}{
		// Passes -test.short alongside -test.run when tracing a single leaf subtest polarity.
		{
			name: "leaf polarity trace",
			run: func(workDir string, scriptPath string) {
				executeLeafPolarityTrace(context.Background(), nil, trackerSetup{
					binPath:       scriptPath,
					pkgAbsDir:     workDir,
					pkgWorkDir:    workDir,
					perMutTimeout: time.Second,
				}, 0, "TestFocus/case_a")
			},
			wantArgs: []string{"-test.short", "-test.run=^TestFocus$/^case_a$"},
		},
		// Passes -test.short alongside -test.failfast when evaluating an untracked mutant.
		{
			name: "untracked mutant evaluation",
			run: func(workDir string, scriptPath string) {
				evaluateSingleMutant(context.Background(), nil, mutantJob{
					mutant:           &Mutant{ID: 1},
					binPath:          scriptPath,
					pkgAbsDir:        workDir,
					perMutantTimeout: time.Second,
				})
			},
			wantArgs: []string{"-test.short", "-test.failfast"},
		},
		// Passes -test.short, -test.v, and -test.parallel=1 when running a tracked mutant without a targeted regex.
		{
			name: "tracked mutant without targeted regex",
			run: func(workDir string, scriptPath string) {
				runTrackedMutant(context.Background(), context.Background(), nil, mutantJob{
					mutant:           &Mutant{ID: 2},
					binPath:          scriptPath,
					pkgAbsDir:        workDir,
					perMutantTimeout: time.Second,
					tracker:          &pkgRedundancyTracker{mutantRuns: make(map[int]redundancy.VerboseTestResult)},
				})
			},
			wantArgs: []string{"-test.short", "-test.v", "-test.parallel=1"},
		},
		// Passes -test.short, -test.v, -test.parallel=1, and -test.run when running a tracked mutant with a targeted regex.
		{
			name: "tracked mutant with targeted regex",
			run: func(workDir string, scriptPath string) {
				runTrackedMutant(context.Background(), context.Background(), nil, mutantJob{
					mutant:           &Mutant{ID: 3},
					binPath:          scriptPath,
					pkgAbsDir:        workDir,
					perMutantTimeout: time.Second,
					tracker: &pkgRedundancyTracker{
						targetedRunRegex: "^(TestFocus)$",
						mutantRuns:       make(map[int]redundancy.VerboseTestResult),
					},
				})
			},
			wantArgs: []string{"-test.short", "-test.v", "-test.parallel=1", "-test.run=^(TestFocus)$"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workDir := t.TempDir()
			scriptPath, argsLogPath := writeArgRecorderScript(t, workDir)

			tc.run(workDir, scriptPath)

			gotArgs := readRecordedArgs(t, argsLogPath)
			if diff := cmp.Diff(tc.wantArgs, gotArgs); diff != "" {
				t.Errorf("command args mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestRunBaselineTestBinaryFlags verifies that runBaselineTestBinary passes -test.v and -test.parallel=1 only when verbose is true.
func TestRunBaselineTestBinaryFlags(t *testing.T) {
	testCases := []struct {
		name     string
		verbose  bool
		wantArgs []string
	}{
		// Omits -test.v and -test.parallel=1 when verbose is false.
		{
			name:     "non-verbose baseline",
			verbose:  false,
			wantArgs: []string{"-test.short"},
		},
		// Appends -test.v and -test.parallel=1 when verbose is true.
		{
			name:     "verbose baseline",
			verbose:  true,
			wantArgs: []string{"-test.short", "-test.v", "-test.parallel=1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workDir := t.TempDir()
			scriptPath, argsLogPath := writeArgRecorderScript(t, workDir)

			_, _, err := runBaselineTestBinary(context.Background(), baselineRequest{
				binPath:   scriptPath,
				pkgAbsDir: workDir,
				verbose:   tc.verbose,
			})

			if err != nil {
				t.Fatalf("runBaselineTestBinary() unexpected error: %v", err)
			}
			gotArgs := readRecordedArgs(t, argsLogPath)
			if diff := cmp.Diff(tc.wantArgs, gotArgs); diff != "" {
				t.Errorf("command args mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func writeArgRecorderScript(t *testing.T, dir string) (string, string) {
	t.Helper()
	argsLogPath := filepath.Join(dir, "args.log")
	scriptPath := filepath.Join(dir, "record_args.sh")
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$@" > %q
`, argsLogPath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatalf("WriteFile(%s) failed: %v", scriptPath, err)
	}
	return scriptPath, argsLogPath
}

func readRecordedArgs(t *testing.T, argsLogPath string) []string {
	t.Helper()
	raw, err := os.ReadFile(argsLogPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) failed: %v", argsLogPath, err)
	}
	trimmed := strings.TrimRight(string(raw), "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}
