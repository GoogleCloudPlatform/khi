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

package task

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type debugRunnable struct {
	waitFor     int
	shouldError bool
	resolves    []string
}

var errFoo = fmt.Errorf("an error for test")

func (r *debugRunnable) Run(ctx context.Context, v *VariableSet) error {
	select {
	case <-time.After(time.Millisecond * time.Duration(r.waitFor)):
		break
	case <-ctx.Done():
		return context.Canceled
	}
	if r.shouldError {
		return errFoo
	}
	for _, resolve := range r.resolves {
		v.Set(resolve, struct{}{})
	}
	return nil
}

func newDebugRunnable(waitFor int, resolve ...string) *debugRunnable {
	return &debugRunnable{
		waitFor:     waitFor,
		shouldError: false,
		resolves:    resolve,
	}
}

func newErrorDebugRunnable(waitFor int) *debugRunnable {
	return &debugRunnable{
		waitFor:     waitFor,
		shouldError: true,
	}
}

var _ Runnable = (*debugRunnable)(nil)

func TestLocalRunnerToBeCompleted(t *testing.T) {
	testCases := []struct {
		definitions    []Definition
		expectError    bool
		expectedStatus []*LocalRunnerTaskStat
	}{
		{
			definitions: []Definition{
				newDebugDefinition("foo1", []string{}).WithRunnable(newDebugRunnable(100, "foo1")),
				newDebugDefinition("foo2", []string{"foo1"}).WithRunnable(newDebugRunnable(100, "foo2")),
				newDebugDefinition("foo3", []string{"foo1"}).WithRunnable(newDebugRunnable(100, "foo3")),
			},
			expectError: false,
			expectedStatus: []*LocalRunnerTaskStat{
				{
					Phase: LocalRunnerTaskStatPhaseStopped,
					Error: nil,
				},
				{
					Phase: LocalRunnerTaskStatPhaseStopped,
					Error: nil,
				},
				{
					Phase: LocalRunnerTaskStatPhaseStopped,
					Error: nil,
				},
			},
		},
		{
			definitions: []Definition{
				newDebugDefinition("foo1", []string{}).WithRunnable(newDebugRunnable(100, "foo1")).WithThreadUnsafeLabel(),
				newDebugDefinition("foo2", []string{"foo1"}).WithRunnable(newDebugRunnable(100, "foo2")).WithThreadUnsafeLabel(),
				newDebugDefinition("foo3", []string{"foo2"}).WithRunnable(newDebugRunnable(100, "foo3")).WithThreadUnsafeLabel(),
			},
			expectError: false,
			expectedStatus: []*LocalRunnerTaskStat{
				{
					Phase: LocalRunnerTaskStatPhaseStopped,
					Error: nil,
				},
				{
					Phase: LocalRunnerTaskStatPhaseStopped,
					Error: nil,
				},
				{
					Phase: LocalRunnerTaskStatPhaseStopped,
					Error: nil,
				},
			},
		},
		{
			definitions: []Definition{
				newDebugDefinition("foo1", []string{}).WithRunnable(newErrorDebugRunnable(100)),
				newDebugDefinition("foo2", []string{"foo1"}).WithRunnable(newDebugRunnable(100, "foo2")),
				newDebugDefinition("foo3", []string{"foo1"}).WithRunnable(newDebugRunnable(100, "foo3")),
			},
			expectError: true,
			expectedStatus: []*LocalRunnerTaskStat{
				{
					Phase: LocalRunnerTaskStatPhaseStopped,
					Error: errFoo,
				},
				{
					Phase: LocalRunnerTaskStatPhaseWaiting,
					Error: nil,
				},
				{
					Phase: LocalRunnerTaskStatPhaseWaiting,
					Error: nil,
				},
			},
		},
		{
			definitions: []Definition{
				newDebugDefinition("foo1", []string{}).WithRunnable(newErrorDebugRunnable(100)).WithThreadUnsafeLabel(),
				newDebugDefinition("foo2", []string{"foo1"}).WithRunnable(newDebugRunnable(100, "foo2")).WithThreadUnsafeLabel(),
				newDebugDefinition("foo3", []string{"foo2"}).WithRunnable(newDebugRunnable(100, "foo3")).WithThreadUnsafeLabel(),
			},
			expectError: true,
			expectedStatus: []*LocalRunnerTaskStat{
				{
					Phase: LocalRunnerTaskStatPhaseStopped,
					Error: errFoo,
				},
				{
					Phase: LocalRunnerTaskStatPhaseWaiting,
					Error: nil,
				},
				{
					Phase: LocalRunnerTaskStatPhaseWaiting,
					Error: nil,
				},
			},
		},
		{
			definitions: []Definition{
				newDebugDefinition("foo1", []string{}).WithRunnable(newErrorDebugRunnable(100)),
				newDebugDefinition("foo2", []string{}).WithRunnable(newDebugRunnable(1000, "foo2")),
				newDebugDefinition("foo3", []string{"foo1", "foo2"}).WithRunnable(newDebugRunnable(100, "foo3")).WithThreadUnsafeLabel(),
			},
			expectError: true,
			expectedStatus: []*LocalRunnerTaskStat{
				{
					Phase: LocalRunnerTaskStatPhaseStopped,
					Error: errFoo,
				},
				{
					Phase: LocalRunnerTaskStatPhaseStopped,
					Error: context.Canceled,
				},
				{
					Phase: LocalRunnerTaskStatPhaseWaiting,
					Error: nil,
				},
			},
		},
	}
	for testIndex, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%d", testIndex), func(t *testing.T) {
			defSet, err := NewSet(testCase.definitions)
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
			defSet, err = defSet.ResolveTask(defSet)
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
			runner, err := NewLocalRunner(defSet)
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
			err = runner.Run(context.Background(), 0, map[string]any{})
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
			<-runner.Wait()
			v, err := runner.Result()
			if !testCase.expectError {
				if err != nil {
					t.Errorf("unexpected error %v", err)
				}
				if v == nil {
					t.Errorf("the result variable is empty")
				}
			} else {
				if err == nil {
					t.Errorf("error hasn't been thrown")
				}
				if v != nil {
					t.Errorf("the result variable contains a value, but it was expected to be nil\n%v", v)
				}
			}
			status := runner.TaskStatuses()
			if diff := cmp.Diff(status, testCase.expectedStatus, cmpopts.IgnoreUnexported(), cmpopts.IgnoreFields(LocalRunnerTaskStat{}, "StartTime", "EndTime", "Error")); diff != "" {
				t.Errorf("task status is not matching with the expected status\n%s", diff)
			}
			for taskIndex, stat := range status {
				if testCase.expectedStatus[taskIndex].Error == nil {
					if stat.Error != nil {
						t.Errorf("task index:%d must end without an error", taskIndex)
					}
				} else {
					if stat.Error == nil {
						t.Errorf("task index:%d must end with an error. But got err=nil", taskIndex)
					} else {
						actualErr := testCase.expectedStatus[taskIndex].Error.Error()
						expectedErr := stat.Error.Error()
						if actualErr != expectedErr {
							t.Errorf("task index:%d must end with an error `%s`. But got err=%s", taskIndex, expectedErr, actualErr)
						}
					}
				}
			}
		})
	}
}
