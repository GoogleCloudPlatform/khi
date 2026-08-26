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

package streamingutil

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestAsyncJobManager_Poll_FastPath(t *testing.T) {
	testCases := []struct {
		name       string
		runner     JobRunner[string, int]
		wantStatus *PollStatus[string, int]
	}{
		{
			name: "job completes immediately",
			runner: func(ctx context.Context, onProgress func(string) error) (int, error) {
				return 42, nil
			},
			wantStatus: &PollStatus[string, int]{
				IsDone: true,
				Result: 42,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewAsyncJobManager[string, int](time.Minute, time.Minute)
			defer m.Close()

			status, err := m.Poll(context.Background(), "", 100*time.Millisecond, tc.runner)
			if err != nil {
				t.Fatalf("Poll() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantStatus, status, cmpopts.IgnoreFields(PollStatus[string, int]{}, "JobID")); diff != "" {
				t.Errorf("Poll() status mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAsyncJobManager_Poll_MultiTurn(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "job returns progress and then result on subsequent poll",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewAsyncJobManager[string, int](time.Minute, time.Minute)
			defer m.Close()

			releaseCh := make(chan struct{})
			runner := func(ctx context.Context, onProgress func(string) error) (int, error) {
				_ = onProgress("phase-1")
				<-releaseCh
				return 100, nil
			}

			// First poll: fastWait=5ms, runner is blocked on releaseCh
			status1, err := m.Poll(context.Background(), "", 5*time.Millisecond, runner)
			if err != nil {
				t.Fatalf("first Poll() unexpected error: %v", err)
			}
			if status1.IsDone {
				t.Errorf("expected isDone=false on first poll, got true")
			}
			if status1.Progress != "phase-1" {
				t.Errorf("expected progress 'phase-1', got '%s'", status1.Progress)
			}

			// Release runner
			close(releaseCh)
			time.Sleep(20 * time.Millisecond)

			// Second poll with same JobID
			status2, err := m.Poll(context.Background(), status1.JobID, 5*time.Millisecond, runner)
			if err != nil {
				t.Fatalf("second Poll() unexpected error: %v", err)
			}
			if !status2.IsDone {
				t.Errorf("expected isDone=true on second poll, got false")
			}
			if status2.Result != 100 {
				t.Errorf("expected result 100, got %d", status2.Result)
			}
		})
	}
}

func TestAsyncJobManager_Cancel(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "cancel aborts job context",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewAsyncJobManager[string, int](time.Minute, time.Minute)
			defer m.Close()

			canceledCh := make(chan struct{})
			runner := func(ctx context.Context, onProgress func(string) error) (int, error) {
				select {
				case <-ctx.Done():
					close(canceledCh)
					return 0, ctx.Err()
				case <-time.After(5 * time.Second):
					return 1, nil
				}
			}

			status, err := m.Poll(context.Background(), "", 5*time.Millisecond, runner)
			if err != nil {
				t.Fatalf("Poll() unexpected error: %v", err)
			}

			ok := m.Cancel(status.JobID)
			if !ok {
				t.Fatalf("expected Cancel() to return true")
			}

			select {
			case <-canceledCh:
				// Success
			case <-time.After(500 * time.Millisecond):
				t.Errorf("expected runner context to be canceled within 500ms")
			}
		})
	}
}
