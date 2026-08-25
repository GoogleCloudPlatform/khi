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

package asynctask

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var errTaskFailure = errors.New("task failure")

func TestAsyncTaskManager_DoAsyncOrGet(t *testing.T) {
	testCases := []struct {
		name         string
		worker       func(ctx context.Context) (string, error)
		waitDuration time.Duration
		wantInitial  AsyncTaskResult[string]
		wantFinal    AsyncTaskResult[string]
	}{
		{
			name: "successful async task execution and memoization",
			worker: func(ctx context.Context) (string, error) {
				time.Sleep(20 * time.Millisecond)
				return "success-result", nil
			},
			waitDuration: 50 * time.Millisecond,
			wantInitial: AsyncTaskResult[string]{
				Status: StatusRunning,
			},
			wantFinal: AsyncTaskResult[string]{
				Status: StatusCompleted,
				Value:  "success-result",
			},
		},
		{
			name: "failed async task execution and error memoization",
			worker: func(ctx context.Context) (string, error) {
				time.Sleep(20 * time.Millisecond)
				return "", errTaskFailure
			},
			waitDuration: 50 * time.Millisecond,
			wantInitial: AsyncTaskResult[string]{
				Status: StatusRunning,
			},
			wantFinal: AsyncTaskResult[string]{
				Status: StatusFailed,
				Error:  errTaskFailure,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := NewAsyncTaskManager[string, string]()
			defer mgr.Close()

			initial := mgr.DoAsyncOrGet("slot-1", "input-1", tc.worker)
			if diff := cmp.Diff(tc.wantInitial, initial, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("initial result mismatch (-want +got):\n%s", diff)
			}

			// While running, second call should still return StatusRunning
			second := mgr.DoAsyncOrGet("slot-1", "input-1", tc.worker)
			if diff := cmp.Diff(tc.wantInitial, second, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("in-flight result mismatch (-want +got):\n%s", diff)
			}

			time.Sleep(tc.waitDuration)

			final := mgr.DoAsyncOrGet("slot-1", "input-1", tc.worker)
			if diff := cmp.Diff(tc.wantFinal, final, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("final result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAsyncTaskManager_CancelOnInputKeyChange(t *testing.T) {
	mgr := NewAsyncTaskManager[string, string]()
	defer mgr.Close()

	var task1Canceled atomic.Bool
	var task2Ran atomic.Bool

	// Start task 1
	mgr.DoAsyncOrGet("slot-1", "input-A", func(ctx context.Context) (string, error) {
		select {
		case <-ctx.Done():
			task1Canceled.Store(true)
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return "result-A", nil
		}
	})

	time.Sleep(10 * time.Millisecond)

	// Switch to input-B while task 1 is in-flight
	mgr.DoAsyncOrGet("slot-1", "input-B", func(ctx context.Context) (string, error) {
		task2Ran.Store(true)
		return "result-B", nil
	})

	time.Sleep(50 * time.Millisecond)

	if !task1Canceled.Load() {
		t.Errorf("expected task 1 to be canceled upon input change, but it was not")
	}
	if !task2Ran.Load() {
		t.Errorf("expected task 2 to run for new input key, but it did not")
	}

	resultB := mgr.DoAsyncOrGet("slot-1", "input-B", nil)
	wantResultB := AsyncTaskResult[string]{
		Status: StatusCompleted,
		Value:  "result-B",
	}
	if diff := cmp.Diff(wantResultB, resultB, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("result B mismatch (-want +got):\n%s", diff)
	}
}

func TestAsyncTaskManager_ResetAndCancel(t *testing.T) {
	mgr := NewAsyncTaskManager[string, string]()
	defer mgr.Close()

	mgr.DoAsyncOrGet("slot-1", "input-1", func(ctx context.Context) (string, error) {
		return "value-1", nil
	})
	time.Sleep(20 * time.Millisecond)

	// Verify cached
	cached := mgr.DoAsyncOrGet("slot-1", "input-1", nil)
	if cached.Status != StatusCompleted {
		t.Fatalf("expected StatusCompleted, got %v", cached.Status)
	}

	// Reset slot
	mgr.Reset("slot-1")

	// After reset, calling DoAsyncOrGet with new worker should run again
	ranAgain := false
	newResult := mgr.DoAsyncOrGet("slot-1", "input-1", func(ctx context.Context) (string, error) {
		ranAgain = true
		return "value-2", nil
	})
	if newResult.Status != StatusRunning {
		t.Errorf("expected StatusRunning after reset, got %v", newResult.Status)
	}

	time.Sleep(20 * time.Millisecond)
	if !ranAgain {
		t.Errorf("expected worker to run again after reset")
	}
}
