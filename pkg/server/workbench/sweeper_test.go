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

package workbench

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type mockSweeperTarget struct {
	mu          sync.Mutex
	leases      map[string]time.Time
	removedIDs  []string
	removeCalls int
}

func (m *mockSweeperTarget) Leases() map[string]time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make(map[string]time.Time, len(m.leases))
	for k, v := range m.leases {
		res[k] = v
	}
	return res
}

func (m *mockSweeperTarget) Remove(workbenchID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.leases, workbenchID)
	m.removedIDs = append(m.removedIDs, workbenchID)
	m.removeCalls++
}

func TestSweeper_Sweep(t *testing.T) {
	baseTime := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name        string
		leases      map[string]time.Time
		now         time.Time
		wantRemoved []string
	}{
		{
			name: "evicts expired leases only",
			leases: map[string]time.Time{
				"wb-expired-1": baseTime.Add(-10 * time.Second),
				"wb-expired-2": baseTime.Add(-1 * time.Second),
				"wb-active-1":  baseTime.Add(10 * time.Second),
			},
			now:         baseTime,
			wantRemoved: []string{"wb-expired-1", "wb-expired-2"},
		},
		{
			name: "does nothing when all active",
			leases: map[string]time.Time{
				"wb-active-1": baseTime.Add(10 * time.Second),
			},
			now:         baseTime,
			wantRemoved: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sweeper := NewSweeper(10 * time.Millisecond)
			target := &mockSweeperTarget{
				leases: tc.leases,
			}

			count := sweeper.Sweep(target, tc.now)
			if count != len(tc.wantRemoved) {
				t.Errorf("Sweep() = %d, want %d", count, len(tc.wantRemoved))
			}

			target.mu.Lock()
			removed := slices.Clone(target.removedIDs)
			target.mu.Unlock()

			slices.Sort(removed)
			want := slices.Clone(tc.wantRemoved)
			slices.Sort(want)

			if diff := cmp.Diff(want, removed); diff != "" {
				t.Errorf("removed IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSweeper_RunAndStop(t *testing.T) {
	sweeper := NewSweeper(10 * time.Millisecond)
	target := &mockSweeperTarget{
		leases: map[string]time.Time{
			"session-auto-1": time.Now().Add(-1 * time.Minute),
		},
	}

	sweeper.Run(target)

	// Wait for background tick
	time.Sleep(30 * time.Millisecond)
	sweeper.Stop()

	target.mu.Lock()
	calls := target.removeCalls
	target.mu.Unlock()

	if calls == 0 {
		t.Errorf("expected sweeper to have called Remove at least once during background run")
	}
}
