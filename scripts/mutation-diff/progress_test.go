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

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

func TestProgressReporterNonInteractiveMilestones(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressReporter(&buf, false, gotestutil.TermStyle{Enabled: false})

	p.Phase(1, 4, "Collecting VCS diff...")
	p.SubtestProgress("./pkg/foo", 1, 3)
	p.SubtestProgress("./pkg/foo", 2, 3)
	p.StartMutantPool(8, 2)
	for i := range 8 {
		if i == 2 {
			p.RecordMutantResult(MutantSurvived)
		} else {
			p.RecordMutantResult(MutantKilled)
		}
	}
	p.Finish()
	// Calling Finish a second time or emitting updates after Finish must be a no-op.
	p.Finish()
	p.Phase(4, 4, "stale post-finish phase")

	got := buf.String()
	for _, wantSub := range []string{
		"[1/4] Collecting VCS diff...",
		"[3/4] Tracing 3 changed subtest(s) in ./pkg/foo...",
		"[4/4] Evaluating 8 reached Go mutant(s) across 2 worker(s)...",
		"2/8 (25%) — 2 killed, 0 survived",
		"4/8 (50%) — 3 killed, 1 survived",
		"6/8 (75%) — 5 killed, 1 survived",
		"8/8 (100%) — 7 killed, 1 survived",
	} {
		if !strings.Contains(got, wantSub) {
			t.Errorf("non-interactive progress missing %q in:\n%s", wantSub, got)
		}
	}
	if strings.Contains(got, "\r") {
		t.Errorf("non-interactive progress must not emit carriage returns, got:\n%q", got)
	}
	if strings.Contains(got, "stale post-finish") {
		t.Errorf("progress emitted output after Finish():\n%s", got)
	}
}

func TestProgressReporterInteractiveInPlaceBar(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressReporter(&buf, true, gotestutil.TermStyle{Enabled: true})

	p.Phase(2, 4, "Measuring line coverage...")
	p.SubtestProgress("./pkg/bar", 2, 4)
	p.StartMutantPool(4, 2)
	p.RecordMutantResult(MutantKilled)
	p.RecordMutantResult(MutantSurvived)
	p.Finish()

	got := buf.String()
	for _, wantSub := range []string{
		"\r\x1b[K",
		"[████████░░░░░░░░]",
		"survived:1",
		"✓ Completed in",
	} {
		if !strings.Contains(got, wantSub) {
			t.Errorf("interactive progress missing %q in:\n%q", wantSub, got)
		}
	}
	if p.doneCh == nil {
		t.Fatal("expected interactive reporter to initialize spinner doneCh")
	}
	select {
	case <-p.doneCh:
	default:
		t.Fatal("expected spinner goroutine to terminate during Finish()")
	}

	// A nil writer must never spawn a background spinner goroutine even if interactive is true.
	nilReporter := NewProgressReporter(nil, true, gotestutil.TermStyle{Enabled: true})
	if nilReporter.doneCh != nil {
		t.Fatal("expected nil writer not to spawn a spinner goroutine")
	}
	nilReporter.Phase(1, 4, "noop")
	nilReporter.Finish()
}
