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

package ttlcleaner

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type mockExpirableTarget[K comparable] struct {
	mu          sync.Mutex
	expirations map[K]time.Time
	evicted     []K
	evictError  error
}

func (m *mockExpirableTarget[K]) Expirations() map[K]time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make(map[K]time.Time, len(m.expirations))
	for k, v := range m.expirations {
		copied[k] = v
	}
	return copied
}

func (m *mockExpirableTarget[K]) Evict(key K) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.evictError != nil {
		return m.evictError
	}
	m.evicted = append(m.evicted, key)
	delete(m.expirations, key)
	return nil
}

func TestTTLCleaner_Sweep(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name        string
		expirations map[string]time.Time
		evictError  error
		wantEvicted []string
	}{
		{
			name: "evicts expired entries and keeps active ones",
			expirations: map[string]time.Time{
				"expired-1": now.Add(-10 * time.Minute),
				"expired-2": now.Add(-1 * time.Minute),
				"active-1":  now.Add(5 * time.Minute),
				"active-2":  now.Add(1 * time.Hour),
			},
			wantEvicted: []string{"expired-1", "expired-2"},
		},
		{
			name: "no entries expired",
			expirations: map[string]time.Time{
				"active-1": now.Add(1 * time.Minute),
				"active-2": now.Add(10 * time.Minute),
			},
			wantEvicted: nil,
		},
		{
			name: "handles eviction error gracefully",
			expirations: map[string]time.Time{
				"expired-1": now.Add(-5 * time.Minute),
			},
			evictError:  errors.New("eviction failed"),
			wantEvicted: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target := &mockExpirableTarget[string]{
				expirations: tc.expirations,
				evictError:  tc.evictError,
			}
			cleaner := NewTTLCleaner[string](target, time.Minute)

			gotEvicted := cleaner.Sweep(now)
			slices.Sort(gotEvicted)
			slices.Sort(tc.wantEvicted)

			if diff := cmp.Diff(tc.wantEvicted, gotEvicted); diff != "" {
				t.Errorf("Sweep() evicted keys mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTTLCleaner_StartStop(t *testing.T) {
	target := &mockExpirableTarget[string]{
		expirations: map[string]time.Time{
			"expired": time.Now().Add(-1 * time.Minute),
		},
	}
	cleaner := NewTTLCleaner[string](target, 10*time.Millisecond)

	cleaner.Start()
	// Calling Start again should be a safe no-op.
	cleaner.Start()

	time.Sleep(50 * time.Millisecond)

	cleaner.Stop()
	// Calling Stop again should be safe.
	cleaner.Stop()

	target.mu.Lock()
	evictedCount := len(target.evicted)
	target.mu.Unlock()

	if evictedCount == 0 {
		t.Errorf("expected at least 1 eviction after running cleaner, got 0")
	}
}
