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
	"sync"
)

const (
	fnvOffset64 = 14695981039346656037
	fnvPrime64  = 1099511628211

	shardedMapNumShards = 256
	shardedMapMask      = shardedMapNumShards - 1
)

func hashBytes(b []byte) uint64 {
	var h uint64 = fnvOffset64
	for _, c := range b {
		h ^= uint64(c)
		h *= fnvPrime64
	}
	return h
}

func hashString(s string) uint64 {
	var h uint64 = fnvOffset64
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= fnvPrime64
	}
	return h
}

func shardIndexFromHash(h uint64) uint8 {
	return uint8((h ^ (h >> 32) ^ (h >> 16) ^ (h >> 8)) & shardedMapMask)
}

type stringUint32Shard struct {
	sync.RWMutex
	m map[string]uint32
}

// shardedStringUint32Map provides a thread-safe, sharded map from string to uint32.
// It avoids sync.Map interface boxing overhead and supports zero-allocation lookup from byte slices.
type shardedStringUint32Map struct {
	shards [shardedMapNumShards]stringUint32Shard
}

func newShardedStringUint32Map() *shardedStringUint32Map {
	sm := &shardedStringUint32Map{}
	for i := range sm.shards {
		sm.shards[i].m = make(map[string]uint32, 16)
	}
	return sm
}

// Load retrieves the uint32 value associated with key.
func (sm *shardedStringUint32Map) Load(key string) (uint32, bool) {
	shard := &sm.shards[shardIndexFromHash(hashString(key))]
	shard.RLock()
	val, ok := shard.m[key]
	shard.RUnlock()
	return val, ok
}

// LoadBytes retrieves the uint32 value associated with byte slice key without heap allocation.
func (sm *shardedStringUint32Map) LoadBytes(key []byte) (uint32, bool) {
	shard := &sm.shards[shardIndexFromHash(hashBytes(key))]
	shard.RLock()
	val, ok := shard.m[string(key)]
	shard.RUnlock()
	return val, ok
}

// LoadOrStore returns the existing uint32 value if present. Otherwise, it stores and returns the given value.
func (sm *shardedStringUint32Map) LoadOrStore(key string, val uint32) (uint32, bool) {
	shard := &sm.shards[shardIndexFromHash(hashString(key))]
	shard.Lock()
	if existing, ok := shard.m[key]; ok {
		shard.Unlock()
		return existing, true
	}
	shard.m[key] = val
	shard.Unlock()
	return val, false
}

// LoadOrStoreBytes returns the existing value if present. Otherwise, it allocates the string key, stores, and returns val.
func (sm *shardedStringUint32Map) LoadOrStoreBytes(key []byte, val uint32) (uint32, bool) {
	shard := &sm.shards[shardIndexFromHash(hashBytes(key))]
	shard.Lock()
	if existing, ok := shard.m[string(key)]; ok {
		shard.Unlock()
		return existing, true
	}
	keyStr := string(key)
	shard.m[keyStr] = val
	shard.Unlock()
	return val, false
}

// Range calls f sequentially for every key and value present in the map.
// If f returns false, Range stops the iteration.
func (sm *shardedStringUint32Map) Range(f func(key string, val uint32) bool) {
	for i := range sm.shards {
		shard := &sm.shards[i]
		shard.RLock()
		for k, v := range shard.m {
			if !f(k, v) {
				shard.RUnlock()
				return
			}
		}
		shard.RUnlock()
	}
}

// Len returns the total number of entries across all shards.
func (sm *shardedStringUint32Map) Len() int {
	total := 0
	for i := range sm.shards {
		shard := &sm.shards[i]
		shard.RLock()
		total += len(shard.m)
		shard.RUnlock()
	}
	return total
}

// Dispose clears all internal shard maps to allow GC reclamation.
func (sm *shardedStringUint32Map) Dispose() {
	for i := range sm.shards {
		shard := &sm.shards[i]
		shard.Lock()
		shard.m = nil
		shard.Unlock()
	}
}
