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
	"unsafe"

	"github.com/google/go-cmp/cmp"
)

func TestLazyJSONCache_BasicGetPut(t *testing.T) {
	cache := newLazyJSONCache(4, 2)
	data := []byte(`{"foo":"bar"}`)
	ptr := uintptr(unsafe.Pointer(&data[0]))

	testCases := []struct {
		name      string
		ptr       uintptr
		index     int
		key       string
		valIndex  int
		wantFound bool
		wantVal   int
	}{
		{
			name:      "miss before put",
			ptr:       ptr,
			index:     0,
			key:       "foo",
			wantFound: false,
		},
		{
			name:      "hit after put",
			ptr:       ptr,
			index:     0,
			key:       "foo",
			valIndex:  7,
			wantFound: true,
			wantVal:   7,
		},
		{
			name:      "negative cache hit",
			ptr:       ptr,
			index:     0,
			key:       "nonexistent",
			valIndex:  -1,
			wantFound: true,
			wantVal:   -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantFound && tc.name != "miss before put" {
				cache.put(tc.ptr, tc.index, tc.key, tc.valIndex, data)
			}
			val, found := cache.get(tc.ptr, tc.index, tc.key)
			if diff := cmp.Diff(tc.wantFound, found); diff != "" {
				t.Fatalf("get() found mismatch (-want +got):\n%s", diff)
			}
			if tc.wantFound {
				if diff := cmp.Diff(tc.wantVal, val); diff != "" {
					t.Errorf("get() valIndex mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestLazyJSONCache_LRUEviction(t *testing.T) {
	// 1 shard with capacity 2
	cache := newLazyJSONCache(1, 2)
	data := []byte(`{"a":1,"b":2,"c":3}`)
	ptr := uintptr(unsafe.Pointer(&data[0]))

	cache.put(ptr, 0, "a", 10, data)
	cache.put(ptr, 0, "b", 20, data)

	// Access "a" to make it more recently used than "b"
	val, found := cache.get(ptr, 0, "a")
	if !found || val != 10 {
		t.Fatalf("expected 'a' to be found with value 10, got found=%v, val=%d", found, val)
	}

	// Insert "c", which should evict "b" (least recently used)
	cache.put(ptr, 0, "c", 30, data)

	testCases := []struct {
		key       string
		wantFound bool
		wantVal   int
	}{
		{key: "a", wantFound: true, wantVal: 10},
		{key: "b", wantFound: false, wantVal: 0},
		{key: "c", wantFound: true, wantVal: 30},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			gotVal, found := cache.get(ptr, 0, tc.key)
			if diff := cmp.Diff(tc.wantFound, found); diff != "" {
				t.Fatalf("found mismatch (-want +got):\n%s", diff)
			}
			if tc.wantFound {
				if diff := cmp.Diff(tc.wantVal, gotVal); diff != "" {
					t.Errorf("valIndex mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestLazyJSONCache_Concurrency(t *testing.T) {
	cache := newLazyJSONCache(16, 64)
	data := []byte(`{"key":"value"}`)
	ptr := uintptr(unsafe.Pointer(&data[0]))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key-%d-%d", goroutineID, j%10)
				cache.put(ptr, 0, key, j, data)
				val, found := cache.get(ptr, 0, key)
				if !found || val < 0 {
					t.Errorf("concurrent cache get failed for %s", key)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestLazyJSONCache_Clear(t *testing.T) {
	cache := newLazyJSONCache(4, 8)
	data := []byte(`{"foo":"bar"}`)
	ptr := uintptr(unsafe.Pointer(&data[0]))

	cache.put(ptr, 0, "foo", 42, data)
	if _, found := cache.get(ptr, 0, "foo"); !found {
		t.Fatal("expected 'foo' to be found before clear")
	}

	cache.clear()

	if _, found := cache.get(ptr, 0, "foo"); found {
		t.Error("expected 'foo' to be cleared after clear()")
	}
}
