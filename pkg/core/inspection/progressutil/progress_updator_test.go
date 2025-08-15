// Copyright 2024 Google LLC
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

package progressutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
)

func TestNewProgressUpdator(t *testing.T) {
	p := &progress.TaskProgress{}
	interval := 100 * time.Millisecond
	onTick := func(tp *progress.TaskProgress) {}

	updator := NewProgressUpdator(p, interval, onTick)

	if updator.Progress != p {
		t.Errorf("Progress field not set correctly")
	}
	if updator.Interval != interval {
		t.Errorf("Interval field not set correctly")
	}
	if updator.OnTick == nil {
		t.Errorf("OnTick field not set")
	}
}

func TestProgressUpdator_StartAndDone(t *testing.T) {
	var mu sync.Mutex
	var tickCount int
	onTick := func(tp *progress.TaskProgress) {
		mu.Lock()
		defer mu.Unlock()
		tickCount++
	}

	p := &progress.TaskProgress{}
	interval := 50 * time.Millisecond
	updator := NewProgressUpdator(p, interval, onTick)

	if err := updator.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned an error: %v", err)
	}

	// Check for immediate tick
	mu.Lock()
	if tickCount != 1 {
		t.Errorf("OnTick should be called immediately, got count %d", tickCount)
	}
	mu.Unlock()

	// Check for subsequent tick
	time.Sleep(interval * 2)
	mu.Lock()
	if tickCount < 2 {
		t.Errorf("OnTick should be called again after interval, got count %d", tickCount)
	}
	initialTickCount := tickCount
	mu.Unlock()

	// Check that it stops
	if err := updator.Done(); err != nil {
		t.Fatalf("Done() returned an error: %v", err)
	}
	time.Sleep(interval * 2)

	mu.Lock()
	if tickCount > initialTickCount {
		t.Errorf("OnTick should not be called after Done, got count %d, want %d", tickCount, initialTickCount)
	}
	mu.Unlock()
}

func TestProgressUpdator_DoneWithoutStart(t *testing.T) {
	updator := NewProgressUpdator(&progress.TaskProgress{}, 1*time.Second, func(tp *progress.TaskProgress) {})
	err := updator.Done()
	if err == nil {
		t.Errorf("Done() should return an error if Start() was not called")
	}
}

func TestProgressUpdator_ParentContextCancellation(t *testing.T) {
	var mu sync.Mutex
	var tickCount int
	onTick := func(tp *progress.TaskProgress) {
		mu.Lock()
		defer mu.Unlock()
		tickCount++
	}

	p := &progress.TaskProgress{}
	interval := 50 * time.Millisecond
	updator := NewProgressUpdator(p, interval, onTick)

	ctx, cancel := context.WithCancel(context.Background())
	if err := updator.Start(ctx); err != nil {
		t.Fatalf("Start() returned an error: %v", err)
	}

	// Check for immediate tick
	mu.Lock()
	if tickCount != 1 {
		t.Errorf("OnTick should be called immediately, got count %d", tickCount)
	}
	mu.Unlock()

	cancel()
	time.Sleep(interval * 2)

	mu.Lock()
	if tickCount > 1 {
		t.Errorf("OnTick should not be called after context is canceled, got count %d", tickCount)
	}
	mu.Unlock()
}
