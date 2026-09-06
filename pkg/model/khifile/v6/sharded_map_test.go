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

package khifilev6

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestShardedStringUint32Map_BasicOperations(t *testing.T) {
	testCases := []struct {
		name       string
		operations func(m *shardedStringUint32Map) error
	}{
		{
			name: "load non-existent key returns false",
			operations: func(m *shardedStringUint32Map) error {
				if _, ok := m.Load("missing"); ok {
					return fmt.Errorf("expected missing key not found")
				}
				if _, ok := m.LoadBytes([]byte("missing")); ok {
					return fmt.Errorf("expected missing key bytes not found")
				}
				return nil
			},
		},
		{
			name: "store and load string and bytes",
			operations: func(m *shardedStringUint32Map) error {
				val, loaded := m.LoadOrStore("key1", 42)
				if loaded || val != 42 {
					return fmt.Errorf("LoadOrStore returned (%v, %v), want (42, false)", val, loaded)
				}
				gotVal, ok := m.Load("key1")
				if !ok || gotVal != 42 {
					return fmt.Errorf("Load returned (%v, %v), want (42, true)", gotVal, ok)
				}
				gotValBytes, ok := m.LoadBytes([]byte("key1"))
				if !ok || gotValBytes != 42 {
					return fmt.Errorf("LoadBytes returned (%v, %v), want (42, true)", gotValBytes, ok)
				}
				return nil
			},
		},
		{
			name: "load or store bytes when not present and when present",
			operations: func(m *shardedStringUint32Map) error {
				val, loaded := m.LoadOrStoreBytes([]byte("key2"), 100)
				if loaded || val != 100 {
					return fmt.Errorf("LoadOrStoreBytes first call returned (%v, %v), want (100, false)", val, loaded)
				}
				val2, loaded2 := m.LoadOrStoreBytes([]byte("key2"), 200)
				if !loaded2 || val2 != 100 {
					return fmt.Errorf("LoadOrStoreBytes second call returned (%v, %v), want (100, true)", val2, loaded2)
				}
				return nil
			},
		},
		{
			name: "range iterates all entries",
			operations: func(m *shardedStringUint32Map) error {
				m.LoadOrStore("a", 1)
				m.LoadOrStore("b", 2)
				m.LoadOrStore("c", 3)

				collected := make(map[string]uint32)
				m.Range(func(k string, v uint32) bool {
					collected[k] = v
					return true
				})

				want := map[string]uint32{"a": 1, "b": 2, "c": 3}
				if diff := cmp.Diff(want, collected); diff != "" {
					return fmt.Errorf("Range mismatch (-want +got):\n%s", diff)
				}
				if m.Len() != 3 {
					return fmt.Errorf("Len() returned %d, want 3", m.Len())
				}
				return nil
			},
		},
		{
			name: "dispose clears all shards",
			operations: func(m *shardedStringUint32Map) error {
				m.LoadOrStore("x", 10)
				m.Dispose()
				if _, ok := m.Load("x"); ok {
					return fmt.Errorf("Load returned true after Dispose")
				}
				if m.Len() != 0 {
					return fmt.Errorf("Len() returned %d after Dispose, want 0", m.Len())
				}
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := newShardedStringUint32Map()
			if err := tc.operations(m); err != nil {
				t.Errorf("operation failed: %v", err)
			}
		})
	}
}

func TestShardedStringUint32Map_ConcurrentAccess(t *testing.T) {
	m := newShardedStringUint32Map()
	var wg sync.WaitGroup
	workers := 16
	itemsPerWorker := 1000

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < itemsPerWorker; i++ {
				key := fmt.Sprintf("worker-%d-item-%d", workerID, i)
				keyBytes := []byte(key)
				m.LoadOrStoreBytes(keyBytes, uint32(i))

				val, ok := m.LoadBytes(keyBytes)
				if !ok || val != uint32(i) {
					t.Errorf("concurrent LoadBytes failed for key %s", key)
				}
			}
		}(w)
	}

	wg.Wait()
	if m.Len() != workers*itemsPerWorker {
		t.Errorf("expected %d items, got %d", workers*itemsPerWorker, m.Len())
	}
}

func BenchmarkShardedStringUint32Map_LoadBytes(b *testing.B) {
	m := newShardedStringUint32Map()
	key := []byte("example.kubernetes.io/metadata.name")
	m.LoadOrStoreBytes(key, 12345)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		val, ok := m.LoadBytes(key)
		if !ok || val != 12345 {
			b.Fatalf("LoadBytes failed")
		}
	}
}
