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

package structured

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLazyJSONBlockStore_BasicAndEviction(t *testing.T) {
	testCases := []struct {
		name       string
		shardCount int
		shardCap   int
		items      []string
	}{
		{
			name:       "store with small cache capacity triggering eviction and re-decompression",
			shardCount: 2,
			shardCap:   2,
			items: []string{
				`{"msg":"block-0"}`,
				`{"msg":"block-1"}`,
				`{"msg":"block-2"}`,
				`{"msg":"block-3"}`,
				`{"msg":"block-4"}`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewLazyJSONBlockStore(tc.shardCount, tc.shardCap)
			blockIDs := make([]uint32, len(tc.items))

			for i, item := range tc.items {
				data := []byte(item)
				blockID := store.ReserveBlock(data)
				compressed, err := compressBlockData(data)
				if err != nil {
					t.Fatalf("compressBlockData() failed: %v", err)
				}
				store.SetCompressed(blockID, compressed)
				blockIDs[i] = blockID
			}

			// Read back in reverse order to ensure evicted blocks are re-decompressed correctly.
			for i := len(tc.items) - 1; i >= 0; i-- {
				expected := tc.items[i]
				gotBuf := store.GetBuffer(blockIDs[i], 0, uint32(len(expected)))
				if diff := cmp.Diff(expected, string(gotBuf)); diff != "" {
					t.Errorf("GetBuffer() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestLazyJSONBlockBuilder_BatchingAndNodeAccess(t *testing.T) {
	testCases := []struct {
		name       string
		maxEntries int
		maxBytes   int
		entries    []string
	}{
		{
			name:       "batching by max entries",
			maxEntries: 3,
			maxBytes:   1024 * 1024,
			entries: []string{
				`{"id":1,"name":"first"}`,
				`{"id":2,"name":"second"}`,
				`{"id":3,"name":"third"}`,
				`{"id":4,"name":"fourth"}`,
				`{"id":5,"name":"fifth"}`,
			},
		},
		{
			name:       "batching by byte size",
			maxEntries: 100,
			maxBytes:   30,
			entries: []string{
				`{"k":"val1"}`,
				`{"k":"val2"}`,
				`{"k":"val3"}`,
				`{"k":"val4"}`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewLazyJSONBlockStore(2, 2)
			builder := store.NewBuilder(tc.maxEntries, tc.maxBytes)

			nodes := make([]*LazyJSONNode, len(tc.entries))
			for i, entry := range tc.entries {
				nodes[i] = builder.Add([]byte(entry))
			}
			builder.Flush()

			for i, node := range nodes {
				expected := tc.entries[i]
				gotBuf := node.getBuffer()
				if diff := cmp.Diff(expected, string(gotBuf)); diff != "" {
					t.Errorf("node.getBuffer() mismatch (-want +got):\n%s", diff)
				}

				// Verify JSON node field access.
				val, ok := node.GetChildByKey("id")
				if ok {
					id, err := val.NodeScalarValue()
					if err != nil {
						t.Errorf("NodeScalarValue() failed: %v", err)
					}
					expectedID := fmt.Sprintf("%d", i+1)
					if diff := cmp.Diff(expectedID, fmt.Sprintf("%v", id)); diff != "" {
						t.Errorf("node field 'id' mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestLazyJSONBlockStore_ConcurrentAccess(t *testing.T) {
	store := NewLazyJSONBlockStore(4, 2)
	builder := store.NewBuilder(10, 1024)

	const totalEntries = 100
	nodes := make([]*LazyJSONNode, totalEntries)
	for i := 0; i < totalEntries; i++ {
		entry := fmt.Sprintf(`{"index":%d,"data":"concurrent-test-%d"}`, i, i)
		nodes[i] = builder.Add([]byte(entry))
	}
	builder.Flush()

	var wg sync.WaitGroup
	const concurrency = 8
	for c := 0; c < concurrency; c++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < totalEntries; i++ {
				node := nodes[(i*7+workerID)%totalEntries]
				valNode, ok := node.GetChildByKey("index")
				if !ok {
					t.Errorf("worker %d failed to GetChildByKey", workerID)
					return
				}
				idxVal, err := valNode.NodeScalarValue()
				if err != nil {
					t.Errorf("worker %d failed to get scalar: %v", workerID, err)
					return
				}
				expectedIdx := (i*7 + workerID) % totalEntries
				if diff := cmp.Diff(fmt.Sprintf("%d", expectedIdx), fmt.Sprintf("%v", idxVal)); diff != "" {
					t.Errorf("concurrent scalar mismatch (-want +got):\n%s", diff)
				}
			}
		}(c)
	}
	wg.Wait()
}
