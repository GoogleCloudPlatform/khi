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

package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParallelChunkMap(t *testing.T) {
	testCases := []struct {
		name        string
		items       []int
		numWorkers  int
		workerFunc  func(ctx context.Context, workerIdx int, chunk []int, onProcessed func(int)) ([]int, error)
		wantResults [][]int
		wantErr     bool
	}{
		{
			name:       "processes empty slice",
			items:      nil,
			numWorkers: 4,
			workerFunc: func(ctx context.Context, workerIdx int, chunk []int, onProcessed func(int)) ([]int, error) {
				return chunk, nil
			},
			wantResults: nil,
			wantErr:     false,
		},
		{
			name:       "maps integers across chunks in order",
			items:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			numWorkers: 3,
			workerFunc: func(ctx context.Context, workerIdx int, chunk []int, onProcessed func(int)) ([]int, error) {
				res := make([]int, len(chunk))
				for i, v := range chunk {
					res[i] = v * 2
					onProcessed(1)
				}
				return res, nil
			},
			wantResults: [][]int{
				{2, 4, 6, 8},
				{10, 12, 14, 16},
				{18, 20},
			},
			wantErr: false,
		},
		{
			name:       "adjusts worker count when chunkSize rounding yields fewer chunks",
			items:      []int{1, 2, 3, 4, 5},
			numWorkers: 4,
			workerFunc: func(ctx context.Context, workerIdx int, chunk []int, onProcessed func(int)) ([]int, error) {
				return chunk, nil
			},
			wantResults: [][]int{
				{1, 2},
				{3, 4},
				{5},
			},
			wantErr: false,
		},
		{
			name:       "propagates error from worker function",
			items:      []int{1, 2, 3, 4},
			numWorkers: 2,
			workerFunc: func(ctx context.Context, workerIdx int, chunk []int, onProcessed func(int)) ([]int, error) {
				if workerIdx == 1 {
					return nil, errors.New("test worker error")
				}
				return chunk, nil
			},
			wantResults: nil,
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParallelChunkMap(
				context.Background(),
				tc.items,
				tc.workerFunc,
				nil,
				ProgressOptions{NumWorkers: tc.numWorkers},
			)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParallelChunkMap() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.wantResults, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ParallelChunkMap() results mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParallelChunkMap_ProgressReporting(t *testing.T) {
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}

	var mu sync.Mutex
	var reportedPcts []float64
	var reportedMsgs []string

	onProgress := func(pct float64, msg string) error {
		mu.Lock()
		defer mu.Unlock()
		reportedPcts = append(reportedPcts, pct)
		reportedMsgs = append(reportedMsgs, msg)
		return nil
	}

	opts := ProgressOptions{
		Interval:    10 * time.Millisecond,
		MessageFmt:  "Items (%d/%d)",
		MinProgress: 0.10,
		MaxProgress: 0.80,
		NumWorkers:  4,
	}

	_, err := ParallelChunkMap(
		context.Background(),
		items,
		func(ctx context.Context, workerIdx int, chunk []int, onProcessed func(int)) (int, error) {
			for range chunk {
				time.Sleep(1 * time.Millisecond)
				onProcessed(1)
			}
			return len(chunk), nil
		},
		onProgress,
		opts,
	)
	if err != nil {
		t.Fatalf("ParallelChunkMap() unexpected error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(reportedPcts) == 0 {
		t.Fatalf("expected progress reports, got none")
	}

	lastPct := reportedPcts[len(reportedPcts)-1]
	if lastPct != 0.80 {
		t.Errorf("expected final progress to be 0.80, got %f", lastPct)
	}

	lastMsg := reportedMsgs[len(reportedMsgs)-1]
	expectedMsg := fmt.Sprintf("Items (%d/%d)", len(items), len(items))
	if lastMsg != expectedMsg {
		t.Errorf("expected final message %q, got %q", expectedMsg, lastMsg)
	}
}

func TestParallelChunkMap_Cancellation(t *testing.T) {
	testCases := []struct {
		name       string
		setupCtx   func() (context.Context, context.CancelFunc)
		workerFunc func(ctx context.Context, workerIdx int, chunk []int, onProcessed func(int)) (int, error)
		wantErr    error
	}{
		{
			name: "returns error immediately when context is already canceled",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			workerFunc: func(ctx context.Context, workerIdx int, chunk []int, onProcessed func(int)) (int, error) {
				return len(chunk), nil
			},
			wantErr: context.Canceled,
		},
		{
			name: "aborts worker execution when context is canceled mid-flight",
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			workerFunc: func(ctx context.Context, workerIdx int, chunk []int, onProcessed func(int)) (int, error) {
				select {
				case <-ctx.Done():
					return 0, ctx.Err()
				case <-time.After(1 * time.Second):
					return len(chunk), nil
				}
			},
			wantErr: context.Canceled,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.setupCtx()
			defer cancel()

			if tc.name == "aborts worker execution when context is canceled mid-flight" {
				time.AfterFunc(10*time.Millisecond, cancel)
			}

			items := []int{1, 2, 3, 4, 5, 6, 7, 8}
			_, err := ParallelChunkMap(
				ctx,
				items,
				tc.workerFunc,
				nil,
				ProgressOptions{NumWorkers: 2},
			)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("ParallelChunkMap() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
