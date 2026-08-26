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

package server

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResourceMonitorImpl(t *testing.T) {
	monitor := NewResourceMonitorImpl()
	defer monitor.Close()

	t.Run("GetUsedMemory", func(t *testing.T) {
		got := monitor.GetUsedMemory()
		if got == 0 {
			t.Errorf("GetUsedMemory() = %d; want > 0", got)
		}
	})

	t.Run("GetTotalMemory", func(t *testing.T) {
		got := monitor.GetTotalMemory()
		// Depending on the environment, gopsutil might fail or return 0.
		// But in a normal test environment it should return something > 0.
		if got == 0 {
			t.Logf("Warning: GetTotalMemory() returned %d", got)
		}

		// Test caching
		got2 := monitor.GetTotalMemory()
		if got != got2 {
			t.Errorf("GetTotalMemory() cached value mismatch: got %d, want %d", got2, got)
		}
	})

	t.Run("GetCPUUsage", func(t *testing.T) {
		got := monitor.GetCPUUsage()
		if got < 0 || got > 100 {
			t.Errorf("GetCPUUsage() = %f; want 0 <= got <= 100", got)
		}
	})

	t.Run("PanicsWhenDirectlyInitialized", func(t *testing.T) {
		uninit := &ResourceMonitorImpl{}
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("GetCPUUsage() did not panic on uninitialized struct")
			}
		}()
		uninit.GetCPUUsage()
	})
}

func TestResourceMonitorMock(t *testing.T) {
	testCases := []struct {
		name      string
		mock      ResourceMonitorMock
		wantUsed  uint64
		wantTotal uint64
		wantCPU   float64
	}{
		{
			name: "mock returns set values",
			mock: ResourceMonitorMock{
				UsedMemory:  100,
				TotalMemory: 1000,
				CPUUsage:    25.5,
			},
			wantUsed:  100,
			wantTotal: 1000,
			wantCPU:   25.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.wantUsed, tc.mock.GetUsedMemory()); diff != "" {
				t.Errorf("GetUsedMemory() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantTotal, tc.mock.GetTotalMemory()); diff != "" {
				t.Errorf("GetTotalMemory() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantCPU, tc.mock.GetCPUUsage()); diff != "" {
				t.Errorf("GetCPUUsage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
