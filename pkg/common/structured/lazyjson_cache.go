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
	"os"
	"sync"
)

var lazyJSONCacheDisabled = os.Getenv("KHI_DISABLE_LAZYJSON_CACHE") == "1"

// lazyJSONCacheKey identifies a specific field lookup within a LazyJSONNode.
type lazyJSONCacheKey struct {
	ptr   uintptr
	index int
	key   string
}

// lazyJSONCacheEntry represents a single cached child index in the LRU doubly linked list.
type lazyJSONCacheEntry struct {
	key      lazyJSONCacheKey
	dataRef  []byte
	valIndex int
	prev     *lazyJSONCacheEntry
	next     *lazyJSONCacheEntry
}

// lazyJSONCacheShard represents an independent LRU cache partition protected by its own mutex.
type lazyJSONCacheShard struct {
	mu       sync.Mutex
	capacity int
	items    map[lazyJSONCacheKey]*lazyJSONCacheEntry
	head     *lazyJSONCacheEntry
	tail     *lazyJSONCacheEntry
}

func newLazyJSONCacheShard(capacity int) *lazyJSONCacheShard {
	return &lazyJSONCacheShard{
		capacity: capacity,
		items:    make(map[lazyJSONCacheKey]*lazyJSONCacheEntry, capacity),
	}
}

func (s *lazyJSONCacheShard) get(k lazyJSONCacheKey) (int, bool) {
	s.mu.Lock()
	entry, ok := s.items[k]
	if !ok {
		s.mu.Unlock()
		return 0, false
	}
	s.moveToHead(entry)
	valIndex := entry.valIndex
	s.mu.Unlock()
	return valIndex, true
}

func (s *lazyJSONCacheShard) put(k lazyJSONCacheKey, valIndex int, dataRef []byte) {
	s.mu.Lock()
	if entry, ok := s.items[k]; ok {
		entry.valIndex = valIndex
		entry.dataRef = dataRef
		s.moveToHead(entry)
		s.mu.Unlock()
		return
	}

	entry := &lazyJSONCacheEntry{
		key:      k,
		dataRef:  dataRef,
		valIndex: valIndex,
	}
	s.items[k] = entry
	s.addToHead(entry)

	if len(s.items) > s.capacity {
		s.removeTail()
	}
	s.mu.Unlock()
}

func (s *lazyJSONCacheShard) putIfAbsent(k lazyJSONCacheKey, valIndex int, dataRef []byte) {
	s.mu.Lock()
	if entry, ok := s.items[k]; ok {
		s.moveToHead(entry)
		s.mu.Unlock()
		return
	}

	entry := &lazyJSONCacheEntry{
		key:      k,
		dataRef:  dataRef,
		valIndex: valIndex,
	}
	s.items[k] = entry
	s.addToHead(entry)

	if len(s.items) > s.capacity {
		s.removeTail()
	}
	s.mu.Unlock()
}

func (s *lazyJSONCacheShard) clear() {
	s.mu.Lock()
	s.items = make(map[lazyJSONCacheKey]*lazyJSONCacheEntry, s.capacity)
	s.head = nil
	s.tail = nil
	s.mu.Unlock()
}

func (s *lazyJSONCacheShard) addToHead(entry *lazyJSONCacheEntry) {
	entry.prev = nil
	entry.next = s.head
	if s.head != nil {
		s.head.prev = entry
	}
	s.head = entry
	if s.tail == nil {
		s.tail = entry
	}
}

func (s *lazyJSONCacheShard) moveToHead(entry *lazyJSONCacheEntry) {
	if s.head == entry {
		return
	}
	if entry.prev != nil {
		entry.prev.next = entry.next
	}
	if entry.next != nil {
		entry.next.prev = entry.prev
	}
	if s.tail == entry {
		s.tail = entry.prev
	}

	entry.prev = nil
	entry.next = s.head
	if s.head != nil {
		s.head.prev = entry
	}
	s.head = entry
}

func (s *lazyJSONCacheShard) removeTail() {
	if s.tail == nil {
		return
	}
	delete(s.items, s.tail.key)
	s.tail.dataRef = nil
	if s.tail.prev != nil {
		s.tail.prev.next = nil
		s.tail = s.tail.prev
	} else {
		s.head = nil
		s.tail = nil
	}
}

// lazyJSONCache provides a sharded, thread-safe LRU cache for LazyJSONNode child key lookups.
type lazyJSONCache struct {
	shards    []*lazyJSONCacheShard
	shardMask uint64
}

func newLazyJSONCache(shardCount, shardCap int) *lazyJSONCache {
	shards := make([]*lazyJSONCacheShard, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = newLazyJSONCacheShard(shardCap)
	}
	return &lazyJSONCache{
		shards:    shards,
		shardMask: uint64(shardCount - 1),
	}
}

func hashLazyJSONKey(ptr uintptr, index int, key string) uint64 {
	var h uint64 = 14695981039346656037
	h ^= uint64(ptr)
	h *= 1099511628211
	h ^= uint64(index)
	h *= 1099511628211
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= 1099511628211
	}
	return h
}

func (c *lazyJSONCache) get(ptr uintptr, index int, key string) (int, bool) {
	k := lazyJSONCacheKey{ptr: ptr, index: index, key: key}
	shardIdx := hashLazyJSONKey(ptr, index, key) & c.shardMask
	return c.shards[shardIdx].get(k)
}

func (c *lazyJSONCache) put(ptr uintptr, index int, key string, valIndex int, dataRef []byte) {
	k := lazyJSONCacheKey{ptr: ptr, index: index, key: key}
	shardIdx := hashLazyJSONKey(ptr, index, key) & c.shardMask
	c.shards[shardIdx].put(k, valIndex, dataRef)
}

func (c *lazyJSONCache) putIfAbsent(ptr uintptr, index int, key string, valIndex int, dataRef []byte) {
	k := lazyJSONCacheKey{ptr: ptr, index: index, key: key}
	shardIdx := hashLazyJSONKey(ptr, index, key) & c.shardMask
	c.shards[shardIdx].putIfAbsent(k, valIndex, dataRef)
}

func (c *lazyJSONCache) clear() {
	for _, shard := range c.shards {
		shard.clear()
	}
}

const (
	lazyJSONCacheShardCount = 64
	lazyJSONCacheShardCap   = 512
)

var globalLazyJSONCache = newLazyJSONCache(lazyJSONCacheShardCount, lazyJSONCacheShardCap)

// ResetGlobalLazyJSONCache clears all entries in the global LazyJSONNode LRU cache.
func ResetGlobalLazyJSONCache() {
	globalLazyJSONCache.clear()
}
