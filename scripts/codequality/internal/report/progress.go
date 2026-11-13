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
	"cmp"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const (
	progressBarWidth    = 16
	defaultTickInterval = 120 * time.Millisecond
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ProgressConfig configures terminal progress output.
type ProgressConfig struct {
	// Writer receives formatted progress lines or spinner updates.
	Writer io.Writer
	// Interactive enables ANSI spinner rendering when true.
	Interactive bool
	// Style controls ANSI color formatting.
	Style gotestutil.TermStyle
	// TickInterval sets the interactive spinner refresh interval.
	TickInterval time.Duration
}

// ProgressReporter emits live TTY spinner bars or non-TTY milestone lines.
type ProgressReporter struct {
	mu               sync.Mutex
	w                io.Writer
	interactive      bool
	style            gotestutil.TermStyle
	tickInterval     time.Duration
	startTime        time.Time
	statusText       string
	frameIdx         int
	stopCh           chan struct{}
	doneCh           chan struct{}
	stopped          bool
	mutTotal         int
	mutDone          int
	mutKilled        int
	mutSurvived      int
	nextMilestonePct int
}

// NewProgressReporter creates and starts a ProgressReporter from cfg.
func NewProgressReporter(cfg ProgressConfig) *ProgressReporter {
	p := &ProgressReporter{
		w:            cfg.Writer,
		interactive:  cfg.Interactive,
		style:        cfg.Style,
		tickInterval: cfg.TickInterval,
		startTime:    time.Now(),
	}
	if p.w != nil && p.interactive {
		p.stopCh = make(chan struct{})
		p.doneCh = make(chan struct{})
		go p.spinLoop()
	}
	return p
}

// Phase updates the high-level pipeline step status.
func (p *ProgressReporter) Phase(step int, totalSteps int, message string) {
	if p == nil || p.w == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	prefix := fmt.Sprintf("[%d/%d]", step, totalSteps)
	if p.interactive {
		p.statusText = fmt.Sprintf("%s %s", p.style.BoldCyan(prefix), message)
		p.renderInteractiveLocked()
		return
	}
	fmt.Fprintf(p.w, "%s %s\n", prefix, message)
}

// SubtestProgress updates per-subtest polarity trace progress.
func (p *ProgressReporter) SubtestProgress(pkgLabel string, done int, total int) {
	if p == nil || p.w == nil || total <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	if p.interactive {
		bar := renderBar(done, total, progressBarWidth)
		p.statusText = fmt.Sprintf("%s Tracing subtests in %s %s %d/%d",
			p.style.BoldCyan("[3/4]"), p.style.Bold(pkgLabel), p.style.Cyan(bar), done, total)
		p.renderInteractiveLocked()
		return
	}
	if done == 1 {
		fmt.Fprintf(p.w, "[3/4] Tracing %d changed subtest(s) in %s...\n", total, pkgLabel)
	}
}

// StartMutantPool initializes worker-pool progress tracking for total mutants.
func (p *ProgressReporter) StartMutantPool(total int, workers int) {
	if p == nil || p.w == nil || total <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	p.mutTotal = total
	p.mutDone = 0
	p.mutKilled = 0
	p.mutSurvived = 0
	p.nextMilestonePct = 25
	if p.interactive {
		p.updateMutantStatusLocked()
		p.renderInteractiveLocked()
		return
	}
	fmt.Fprintf(p.w, "[4/4] Evaluating %d reached Go mutant(s) across %d worker(s)...\n", total, workers)
}

// RecordMutantResult records one completed mutant evaluation.
func (p *ProgressReporter) RecordMutantResult(status rule.MutantStatus) {
	if p == nil || p.w == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped || p.mutTotal <= 0 {
		return
	}
	p.mutDone++
	switch status {
	case rule.MutantKilled, rule.MutantTimedOut:
		p.mutKilled++
	case rule.MutantSurvived:
		p.mutSurvived++
	}
	if p.interactive {
		p.updateMutantStatusLocked()
		p.renderInteractiveLocked()
		return
	}
	p.emitNonTTYMilestoneLocked()
}

// Finish stops any active spinner and clears transient status output.
func (p *ProgressReporter) Finish() {
	if p == nil || p.w == nil {
		return
	}
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.stopped = true
	stopCh := p.stopCh
	doneCh := p.doneCh
	interactive := p.interactive
	elapsed := time.Since(p.startTime).Seconds()
	p.mu.Unlock()

	if doneCh != nil {
		close(stopCh)
		<-doneCh
	}
	if interactive {
		fmt.Fprintf(p.w, "\r\x1b[K%s\n", p.style.Dim(fmt.Sprintf("✓ Completed in %.1fs", elapsed)))
	}
}

func (p *ProgressReporter) spinLoop() {
	defer close(p.doneCh)
	ticker := time.NewTicker(cmp.Or(p.tickInterval, defaultTickInterval))
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.tickSpinner()
		}
	}
}

func (p *ProgressReporter) tickSpinner() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.stopped && p.statusText != "" {
		p.frameIdx = (p.frameIdx + 1) % len(spinnerFrames)
		p.renderInteractiveLocked()
	}
}

func (p *ProgressReporter) emitNonTTYMilestoneLocked() {
	if p.mutTotal < 8 {
		return
	}
	pct := (p.mutDone * 100) / p.mutTotal
	if pct < p.nextMilestonePct {
		return
	}
	elapsed := time.Since(p.startTime).Seconds()
	fmt.Fprintf(p.w, "  ... %d/%d (%d%%) — %d killed, %d survived (%.1fs)\n",
		p.mutDone, p.mutTotal, pct, p.mutKilled, p.mutSurvived, elapsed)
	for p.nextMilestonePct <= pct {
		p.nextMilestonePct += 25
	}
}

func (p *ProgressReporter) updateMutantStatusLocked() {
	bar := renderBar(p.mutDone, p.mutTotal, progressBarWidth)
	pct := (p.mutDone * 100) / p.mutTotal
	killedStr := p.style.Green(fmt.Sprintf("killed:%d", p.mutKilled))
	survStr := p.style.Dim("survived:0")
	if p.mutSurvived > 0 {
		survStr = p.style.BoldRed(fmt.Sprintf("survived:%d", p.mutSurvived))
	}
	p.statusText = fmt.Sprintf("%s Mutants %s %d/%d (%d%%)  %s  %s",
		p.style.BoldCyan("[4/4]"), p.style.Cyan(bar), p.mutDone, p.mutTotal, pct, killedStr, survStr)
}

func (p *ProgressReporter) renderInteractiveLocked() {
	frame := p.style.BoldCyan(spinnerFrames[p.frameIdx])
	elapsed := p.style.Dim(fmt.Sprintf("(%.1fs)", time.Since(p.startTime).Seconds()))
	fmt.Fprintf(p.w, "\r\x1b[K%s %s %s", frame, p.statusText, elapsed)
}

func renderBar(done int, total int, width int) string {
	if total <= 0 || width <= 0 {
		return "[]"
	}
	filled := min(width, max(0, (done*width)/total))
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}
