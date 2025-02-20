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

package queryutil

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestDivideTimeSegments(t *testing.T) {
	testCases := []struct {
		name       string
		startTime  time.Time
		endTime    time.Time
		splitCount int
		expected   []time.Time
	}{
		{
			name:       "No split",
			startTime:  time.Date(2023, 8, 18, 10, 0, 0, 0, time.UTC),
			endTime:    time.Date(2023, 8, 18, 14, 0, 0, 0, time.UTC),
			splitCount: 1,
			expected: []time.Time{
				time.Date(2023, 8, 18, 10, 0, 0, 0, time.UTC),
				time.Date(2023, 8, 18, 14, 0, 0, 0, time.UTC),
			},
		},
		{
			name:       "Simple split in two",
			startTime:  time.Date(2023, 8, 18, 10, 0, 0, 0, time.UTC),
			endTime:    time.Date(2023, 8, 18, 14, 0, 0, 0, time.UTC),
			splitCount: 2,
			expected: []time.Time{
				time.Date(2023, 8, 18, 10, 0, 0, 0, time.UTC),
				time.Date(2023, 8, 18, 12, 0, 0, 0, time.UTC),
				time.Date(2023, 8, 18, 14, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := divideTimeSegments(tc.startTime, tc.endTime, tc.splitCount)

			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("Unexpected sub-intervals. Expected: %v, Actual: %v", tc.expected, actual)
			}
		})
	}
}

func TestParallelQueryWorkerThreadPool(t *testing.T) {

	testCases := []struct {
		name              string
		functions         []func()
		maxParallelCount  int
		expectedMaxActive int // Expected maximum number of active functions at a time
	}{
		{
			name: "Run 3 functions with parallelism 2",
			functions: []func(){
				func() { time.Sleep(100 * time.Millisecond) },
				func() { time.Sleep(50 * time.Millisecond) },
				func() { time.Sleep(150 * time.Millisecond) },
			},
			maxParallelCount:  2,
			expectedMaxActive: 2,
		},
		{
			name: "Run 5 functions with parallelism 1 (sequential)",
			functions: []func(){
				func() { time.Sleep(100 * time.Millisecond) },
				func() { time.Sleep(100 * time.Millisecond) },
				func() { time.Sleep(100 * time.Millisecond) },
				func() { time.Sleep(100 * time.Millisecond) },
				func() { time.Sleep(100 * time.Millisecond) },
			},
			maxParallelCount:  1,
			expectedMaxActive: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var maxActiveCount int
			var activeMutex sync.Mutex
			activeCount := 0
			pool := worker.NewPool(tc.maxParallelCount)
			for i := range tc.functions {
				originalFunc := tc.functions[i]
				pool.Run(func() { // Wrap the function
					activeMutex.Lock()
					maxActiveCount = max(maxActiveCount, activeCount+1) // Get and update maximum
					activeCount += 1
					activeMutex.Unlock()

					originalFunc() // Call the original function

					activeMutex.Lock()
					activeCount -= 1 // Function finished
					activeMutex.Unlock()
				})
			}

			if maxActiveCount != tc.expectedMaxActive {
				t.Errorf("Unexpected maximum active count. Expected: %d, Actual: %d", tc.expectedMaxActive, maxActiveCount)
			}
		})
	}
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
