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

package progress

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
)

const defaultUpdateInterval = 500 * time.Millisecond

// TrackerOption configures a Tracker or RatioTracker.
type TrackerOption func(*trackerConfig)

// Tracker manages thread-safe progress counting, throughput calculation, and ETA estimation for discrete items.
type Tracker struct {
	tp        *inspectionmetadata.TaskProgressMetadata
	total     int64
	current   atomic.Int64
	startTime time.Time
	cfg       trackerConfig
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	doneOnce  sync.Once
}

// RatioUpdateOption provides additional context when updating a RatioTracker.
type RatioUpdateOption func(*ratioUpdateState)

// RatioTracker tracks continuous ratio progress combined with cumulative item counts, used for streaming log queries.
type RatioTracker struct {
	tp         *inspectionmetadata.TaskProgressMetadata
	startTime  time.Time
	cfg        trackerConfig
	hasUpdated atomic.Bool
}

type trackerConfig struct {
	unit string
}

type ratioUpdateState struct {
	currentStep int
	totalSteps  int
}

// WithUnit specifies the unit name for processed items, such as "logs", "chunks", or "manifests".
func WithUnit(unit string) TrackerOption {
	return func(c *trackerConfig) {
		c.unit = unit
	}
}

// NewTracker creates and starts a background progress tracker for a known total number of discrete items.
// If total <= 0, the tracker automatically reports indeterminate progress until Done is called.
func NewTracker(ctx context.Context, total int, opts ...TrackerOption) *Tracker {
	cfg := trackerConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	t := &Tracker{
		tp:        FromContext(ctx),
		total:     int64(total),
		startTime: time.Now(),
		cfg:       cfg,
	}

	if total <= 0 {
		t.tp.UpdateIndeterminate("")
		return t
	}

	t.tick()

	tickerCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		ticker := time.NewTicker(defaultUpdateInterval)
		defer ticker.Stop()
		for {
			select {
			case <-tickerCtx.Done():
				return
			case <-ticker.C:
				t.tick()
			}
		}
	}()

	return t
}

// Inc increments the completed item count by 1 in a thread-safe manner.
func (t *Tracker) Inc() {
	t.current.Add(1)
}

// Add adds delta to the completed item count in a thread-safe manner.
func (t *Tracker) Add(delta int) {
	t.current.Add(int64(delta))
}

// Done stops the background reporting ticker and flushes the latest progress state.
func (t *Tracker) Done() {
	t.doneOnce.Do(func() {
		if t.cancel != nil {
			t.cancel()
			t.wg.Wait()
		}
		if t.total > 0 {
			t.tick()
		} else {
			t.tp.Update(1.0, "Complete")
		}
	})
}

func (t *Tracker) tick() {
	if t.total <= 0 {
		return
	}
	cur := t.current.Load()
	if cur > t.total {
		cur = t.total
	}
	ratio := float32(cur) / float32(t.total)
	elapsed := time.Since(t.startTime)

	var countStr string
	if t.cfg.unit != "" {
		countStr = fmt.Sprintf("%d/%d %s", cur, t.total, t.cfg.unit)
	} else {
		countStr = fmt.Sprintf("%d/%d", cur, t.total)
	}

	var msg string
	if elapsed >= time.Second {
		rate := float64(cur) / elapsed.Seconds()
		eta := calculateETA(elapsed, ratio)
		rateUnit := rateUnitLabel(t.cfg.unit)
		msg = fmt.Sprintf("%s (%.1f %s, ETA %s)", countStr, rate, rateUnit, eta)
	} else {
		msg = countStr
	}

	t.tp.Update(ratio, msg)
}

// ForEach iterates over a slice synchronously while automatically tracking progress, speed, and ETA.
func ForEach[T any](ctx context.Context, items []T, fn func(index int, item T) error, opts ...TrackerOption) error {
	tracker := NewTracker(ctx, len(items), opts...)
	defer tracker.Done()

	for i, item := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(i, item); err != nil {
			return err
		}
		tracker.Inc()
	}
	return nil
}

// WithStep annotates the progress message with multi-step call indices, such as "[2/5]".
func WithStep(currentStep, totalSteps int) RatioUpdateOption {
	return func(s *ratioUpdateState) {
		s.currentStep = currentStep
		s.totalSteps = totalSteps
	}
}

// NewRatioTracker creates a tracker tailored for streaming operations where completion ratio and item counts arrive asynchronously.
func NewRatioTracker(ctx context.Context, opts ...TrackerOption) *RatioTracker {
	cfg := trackerConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &RatioTracker{
		tp:        FromContext(ctx),
		startTime: time.Now(),
		cfg:       cfg,
	}
}

// Update records the latest completion ratio (0.0 to 1.0) and cumulative item count, computing speed and ETA.
func (r *RatioTracker) Update(ratio float32, count int, opts ...RatioUpdateOption) {
	r.hasUpdated.Store(true)
	state := ratioUpdateState{}
	for _, opt := range opts {
		opt(&state)
	}
	elapsed := time.Since(r.startTime)

	var countStr string
	if r.cfg.unit != "" {
		countStr = fmt.Sprintf("%d %s", count, r.cfg.unit)
	} else {
		countStr = fmt.Sprintf("%d", count)
	}

	var msg string
	if elapsed >= time.Second {
		rate := float64(count) / elapsed.Seconds()
		eta := calculateETA(elapsed, ratio)
		rateUnit := rateUnitLabel(r.cfg.unit)
		msg = fmt.Sprintf("%s (%.2f %s, ETA %s)", countStr, rate, rateUnit, eta)
	} else {
		msg = countStr
	}

	if state.totalSteps > 0 {
		msg = fmt.Sprintf("[%d/%d] %s", state.currentStep, state.totalSteps, msg)
	}

	r.tp.Update(ratio, msg)
}

// Done marks the tracker complete if no streaming updates were reported.
func (r *RatioTracker) Done() {
	if !r.hasUpdated.Load() {
		r.tp.Update(1.0, "Complete")
	}
}

func rateUnitLabel(unit string) string {
	if unit != "" {
		return unit + "/s"
	}
	return "items/s"
}

// calculateETA estimates remaining duration based on elapsed time and completion ratio.
func calculateETA(elapsed time.Duration, ratio float32) string {
	if elapsed < time.Second || ratio <= 0.005 {
		return "--"
	}
	if ratio >= 1.0 {
		return "0s"
	}
	remainingSeconds := elapsed.Seconds() * float64(1.0-ratio) / float64(ratio)
	return formatETA(time.Duration(remainingSeconds * float64(time.Second)))
}

// formatETA formats a duration into a human-readable ETA string.
func formatETA(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	if s < 3600 {
		return fmt.Sprintf("%dm%02ds", s/60, s%60)
	}
	return fmt.Sprintf("%dh%02dm", s/3600, (s%3600)/60)
}
