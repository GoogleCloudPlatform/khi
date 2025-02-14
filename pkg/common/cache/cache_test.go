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

package cache

import (
	"crypto/rand"
	"fmt"
	"sync"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type testOnMemoryCacheItemStorageProvider struct {
	data map[string][]byte
}

func newTestOnMemoryCacheItemStorageProvider() *testOnMemoryCacheItemStorageProvider {
	return &testOnMemoryCacheItemStorageProvider{
		data: make(map[string][]byte),
	}
}

// Get implements CacheItemStorageProvider.
func (t *testOnMemoryCacheItemStorageProvider) Get(key string) ([]byte, error) {
	if v, ok := t.data[key]; ok {
		return v, nil
	}
	return nil, ErrNotFoundInStorageErr
}

// Set implements CacheItemStorageProvider.
func (t *testOnMemoryCacheItemStorageProvider) Set(key string, value []byte) error {
	t.data[key] = value
	return nil
}

var _ CacheItemStorageProvider = (*testOnMemoryCacheItemStorageProvider)(nil)

type testCacheItem struct {
	Value string
}

type testCacheItemConverter struct{}

// Default implements CacheItemConverter.
func (t *testCacheItemConverter) Default() *testCacheItem {
	return nil
}

// Deserialize implements CacheItemConverter.
func (t *testCacheItemConverter) Deserialize(source []byte) (*testCacheItem, error) {
	return &testCacheItem{Value: string(source)}, nil
}

// Serialize implements CacheItemConverter.
func (t *testCacheItemConverter) Serialize(item *testCacheItem) ([]byte, error) {
	return []byte(item.Value), nil
}

var _ CacheItemConverter[*testCacheItem] = (*testCacheItemConverter)(nil)

// Convert implements CacheItemConverter.

func TestCache(t *testing.T) {
	cache := NewCache(newTestOnMemoryCacheItemStorageProvider(), &testCacheItemConverter{}, NewLRUCacheItemReleaseStrategy(1))
	cache.SetAsync("key1", []byte("value1"))
	cache.SetAsync("key2", []byte("value2"))
	cache.SetAsync("key4", []byte("value4"))
	v, err := cache.Get("key1")
	if err != nil {
		t.Errorf(err.Error())
	}
	if v.Value != "value1" {
		t.Errorf("expected value1, got %s", v.Value)
	}
	v, err = cache.Get("key1")
	if err != nil {
		t.Errorf(err.Error())
	}
	if v.Value != "value1" {
		t.Errorf("expected value1, got %s", v.Value)
	}
	v, err = cache.Get("key2")
	if err != nil {
		t.Errorf(err.Error())
	}
	if v.Value != "value2" {
		t.Errorf("expected value1, got %s", v.Value)
	}
	v, err = cache.Get("key4")
	if err != nil {
		t.Errorf(err.Error())
	}
	if v.Value != "value4" {
		t.Errorf("expected value1, got %s", v.Value)
	}
	v, err = cache.Get("key5")
	if err != ErrNotFoundInStorageErr {
		t.Errorf("expected error, got %s", v.Value)
	}
}

func BenchmarkBasicCacheImpl(b *testing.B) {
	thread := 1000
	keyCount := 10
	keys := []string{}
	for i := 0; i < keyCount; i++ {
		keys = append(keys, fmt.Sprintf("key-%d", i))
	}
	wg := sync.WaitGroup{}
	wg.Add(thread)
	cache := NewCache(newTestOnMemoryCacheItemStorageProvider(), &testCacheItemConverter{}, NewLRUCacheItemReleaseStrategy(1))
	b.ResetTimer()
	for i := 0; i < thread; i++ {
		go func() {
			r := []byte{0}
			for j := 0; j < b.N; j++ {
				rand.Read(r)
				key := keys[int(r[0])%keyCount]
				cache.Get(key)
				cache.SetAsync(key, r)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkShardingCache(b *testing.B) {
	thread := 1000
	keyCount := 10
	keys := []string{}
	for i := 0; i < keyCount; i++ {
		keys = append(keys, fmt.Sprintf("key-%d", i))
	}
	wg := sync.WaitGroup{}
	wg.Add(thread)
	cache := NewShardingCache(func() CacheItemStorageProvider { return newTestOnMemoryCacheItemStorageProvider() }, &testCacheItemConverter{}, func() CacheItemReleaseStrategy { return NewLRUCacheItemReleaseStrategy(3) }, common.NewSuffixShardingProvider(16, 1))
	b.ResetTimer()
	for i := 0; i < thread; i++ {
		go func() {
			r := []byte{0}
			for j := 0; j < b.N; j++ {
				rand.Read(r)
				key := keys[int(r[0])%keyCount]
				cache.Get(key)
				cache.SetAsync(key, r)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
