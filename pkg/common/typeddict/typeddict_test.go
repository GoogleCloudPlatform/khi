// Copyright 2025 Google LLC
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

package typeddict

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestTypedDict_Set_Get_Delete(t *testing.T) {
	t.Parallel()
	d := NewTypedDict[int]()
	key := "test_key"
	value := 123

	// Test Set and Get
	Set(d, key, value)
	got, ok := Get(d, key)
	if !ok {
		t.Fatalf("expected to find key %q, but it was not found", key)
	}
	if got != value {
		t.Errorf("expected value %d, but got %d", value, got)
	}

	// Test Delete
	Delete(d, key)
	_, ok = Get(d, key)
	if ok {
		t.Errorf("expected key %q to be deleted, but it was found", key)
	}
}

func TestTypedDict_GetOrDefault(t *testing.T) {
	t.Parallel()
	d := NewTypedDict[string]()
	key := "existing_key"
	value := "hello"
	defaultValue := "default"

	// Test with existing key
	Set(d, key, value)
	got := GetOrDefault(d, key, defaultValue)
	if got != value {
		t.Errorf("expected value %q for existing key, but got %q", value, got)
	}

	// Test with non-existing key
	got = GetOrDefault(d, "non_existing_key", defaultValue)
	if got != defaultValue {
		t.Errorf("expected default value %q for non-existing key, but got %q", defaultValue, got)
	}
}

func TestTypedDict_GetOrSetFunc(t *testing.T) {
	t.Parallel()
	d := NewTypedDict[int]()
	key := "get_or_set_key"
	initialValue := 42
	newValue := 99

	// Test the "set" path
	var setFuncCalled bool
	got := GetOrSetFunc(d, key, func() int {
		setFuncCalled = true
		return initialValue
	})

	if !setFuncCalled {
		t.Error("expected set function to be called, but it was not")
	}
	if got != initialValue {
		t.Errorf("expected value %d from set function, but got %d", initialValue, got)
	}

	// Verify it was stored
	stored, _ := Get(d, key)
	if stored != initialValue {
		t.Errorf("expected value %d to be stored, but got %d", initialValue, stored)
	}

	// Test the "get" path
	setFuncCalled = false // reset flag
	got = GetOrSetFunc(d, key, func() int {
		setFuncCalled = true
		return newValue // This should not be called or set
	})

	if setFuncCalled {
		t.Error("expected set function not to be called for existing key, but it was")
	}
	if got != initialValue {
		t.Errorf("expected to get existing value %d, but got %d", initialValue, got)
	}
}

func TestTypedDict_Concurrent(t *testing.T) {
	t.Parallel()
	d := NewTypedDict[int]()
	numGoroutines := 100
	numOperations := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", goroutineID, j%10)
				value := goroutineID*1000 + j

				// Mix of operations
				switch j % 4 {
				case 0:
					Set(d, key, value)
				case 1:
					Get(d, key)
				case 2:
					GetOrSetFunc(d, key, func() int { return value })
				case 3:
					if j%20 == 0 { // Delete less frequently
						Delete(d, key)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	// No explicit checks for values, just that it completes without race conditions.
	// Run with -race flag to be sure.
}

// --- Benchmarks ---

func BenchmarkTypedDict_Get(b *testing.B) {
	d := NewTypedDict[int]()
	key := "bench_key"
	Set(d, key, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Get(d, key)
	}
}

func BenchmarkTypedDict_Set(b *testing.B) {
	d := NewTypedDict[int]()
	key := "bench_key"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Set(d, key, i)
	}
}

func BenchmarkTypedDict_GetOrSetFunc_GetPath(b *testing.B) {
	d := NewTypedDict[int]()
	key := "bench_key"
	Set(d, key, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetOrSetFunc(d, key, func() int { return 2 })
	}
}

func BenchmarkTypedDict_GetOrSetFunc_SetPath(b *testing.B) {
	d := NewTypedDict[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench_key_" + strconv.Itoa(i)
		GetOrSetFunc(d, key, func() int { return 1 })
	}
}

func BenchmarkTypedDict_ConcurrentReadWrite(b *testing.B) {
	d := NewTypedDict[int]()
	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		Set(d, "key_"+strconv.Itoa(i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "key_" + strconv.Itoa(i%numKeys)
			if i%10 == 0 {
				Set(d, key, i)
			} else {
				Get(d, key)
			}
			i++
		}
	})
}
