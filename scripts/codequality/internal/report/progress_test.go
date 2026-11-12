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

package report

import (
	"bytes"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
	"github.com/google/go-cmp/cmp"
)

var elapsedPattern = regexp.MustCompile(`\d+\.\ds`)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func normalizeElapsed(s string) string {
	return elapsedPattern.ReplaceAllString(s, "0.0s")
}

func waitForSubstring(t *testing.T, sb *syncBuffer, sub string) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if strings.Contains(sb.String(), sub) {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in interactive output:\n%s", sub, sb.String())
}

func TestProgressReporterNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressReporter(ProgressConfig{
		Writer:      &buf,
		Interactive: false,
		Style:       gotestutil.TermStyle{},
	})
	if p.doneCh != nil {
		close(p.stopCh)
		t.Fatalf("expected no spinner goroutine in non-interactive mode")
	}

	p.Phase(1, 4, "Scanning diff")
	p.SubtestProgress("./pkg/a", 1, 2)
	p.SubtestProgress("./pkg/a", 2, 2)
	p.SubtestProgress("./pkg/a", 1, 0)
	p.RecordMutantResult(rule.MutantKilled)
	p.StartMutantPool(0, 1)

	p.StartMutantPool(7, 1)
	for range 7 {
		p.RecordMutantResult(rule.MutantKilled)
	}

	p.StartMutantPool(8, 2)
	statuses := []rule.MutantStatus{
		rule.MutantSurvived,
		rule.MutantTimedOut,
		rule.MutantKilled,
		rule.MutantKilled,
		rule.MutantKilled,
		rule.MutantKilled,
		rule.MutantKilled,
		rule.MutantKilled,
	}
	for _, st := range statuses {
		p.RecordMutantResult(st)
	}

	p.Finish()
	p.Phase(2, 4, "After finish")
	p.SubtestProgress("./pkg/a", 1, 2)
	p.StartMutantPool(8, 1)
	p.RecordMutantResult(rule.MutantKilled)
	p.Finish()

	want := `[1/4] Scanning diff
[3/4] Tracing 2 changed subtest(s) in ./pkg/a...
[4/4] Evaluating 7 reached Go mutant(s) across 1 worker(s)...
[4/4] Evaluating 8 reached Go mutant(s) across 2 worker(s)...
  ... 2/8 (25%) — 1 killed, 1 survived (0.0s)
  ... 4/8 (50%) — 3 killed, 1 survived (0.0s)
  ... 6/8 (75%) — 5 killed, 1 survived (0.0s)
  ... 8/8 (100%) — 7 killed, 1 survived (0.0s)
`
	if diff := cmp.Diff(want, normalizeElapsed(buf.String())); diff != "" {
		t.Errorf("non-interactive progress mismatch (-want +got):\n%s", diff)
	}
}

func TestProgressReporterInteractive(t *testing.T) {
	defaultReporter := NewProgressReporter(ProgressConfig{
		Writer:      &syncBuffer{},
		Interactive: true,
	})
	defaultReporter.Finish()

	spinBuf := &syncBuffer{}
	spinProg := NewProgressReporter(ProgressConfig{
		Writer:       spinBuf,
		Interactive:  true,
		Style:        gotestutil.TermStyle{},
		TickInterval: 2 * time.Millisecond,
	})
	spinProg.tickSpinner()
	if got := spinBuf.String(); got != "" {
		spinProg.Finish()
		t.Fatalf("expected no spinner output before status text is set, got %q", got)
	}
	spinProg.Phase(1, 4, "Scanning diff")
	waitForSubstring(t, spinBuf, "\r\x1b[K⠙ [1/4] Scanning diff ")
	spinProg.Finish()

	afterSpin := spinBuf.String()
	spinProg.tickSpinner()
	if diff := cmp.Diff(afterSpin, spinBuf.String()); diff != "" {
		t.Errorf("spinner output changed after Finish() (-want +got):\n%s", diff)
	}

	barBuf := &syncBuffer{}
	barProg := NewProgressReporter(ProgressConfig{
		Writer:       barBuf,
		Interactive:  true,
		Style:        gotestutil.TermStyle{},
		TickInterval: time.Hour,
	})
	barProg.Phase(1, 4, "Scanning diff")
	barProg.SubtestProgress("./pkg/a", 1, 2)
	barProg.StartMutantPool(2, 1)
	barProg.RecordMutantResult(rule.MutantKilled)
	barProg.RecordMutantResult(rule.MutantSurvived)
	barProg.Finish()
	barProg.Phase(2, 4, "Ignored")
	barProg.SubtestProgress("./pkg/a", 2, 2)
	barProg.StartMutantPool(2, 1)
	barProg.RecordMutantResult(rule.MutantKilled)
	barProg.Finish()

	wantBar := "\r\x1b[K⠋ [1/4] Scanning diff (0.0s)" +
		"\r\x1b[K⠋ [3/4] Tracing subtests in ./pkg/a [████████░░░░░░░░] 1/2 (0.0s)" +
		"\r\x1b[K⠋ [4/4] Mutants [░░░░░░░░░░░░░░░░] 0/2 (0%)  killed:0  survived:0 (0.0s)" +
		"\r\x1b[K⠋ [4/4] Mutants [████████░░░░░░░░] 1/2 (50%)  killed:1  survived:0 (0.0s)" +
		"\r\x1b[K⠋ [4/4] Mutants [████████████████] 2/2 (100%)  killed:1  survived:1 (0.0s)" +
		"\r\x1b[K✓ Completed in 0.0s\n"
	if diff := cmp.Diff(wantBar, normalizeElapsed(barBuf.String())); diff != "" {
		t.Errorf("interactive bar transcript mismatch (-want +got):\n%s", diff)
	}
}

func TestProgressReporterNilAndInvalidInputs(t *testing.T) {
	var nilPtr *ProgressReporter
	nilPtr.Phase(1, 4, "noop")
	nilPtr.SubtestProgress("./pkg/a", 1, 2)
	nilPtr.StartMutantPool(4, 1)
	nilPtr.RecordMutantResult(rule.MutantKilled)
	nilPtr.Finish()

	nilWriter := NewProgressReporter(ProgressConfig{Writer: nil, Interactive: true})
	if nilWriter.doneCh != nil {
		close(nilWriter.stopCh)
		t.Fatalf("expected no spinner goroutine when Writer is nil")
	}
	nilWriter.Phase(1, 4, "noop")
	nilWriter.SubtestProgress("./pkg/a", 1, 2)
	nilWriter.StartMutantPool(4, 1)
	nilWriter.RecordMutantResult(rule.MutantKilled)
	nilWriter.Finish()

	if got := renderBar(1, 0, 16); got != "[]" {
		t.Errorf("renderBar(1, 0, 16) = %q, want []", got)
	}
	if got := renderBar(1, 2, 0); got != "[]" {
		t.Errorf("renderBar(1, 2, 0) = %q, want []", got)
	}
}
