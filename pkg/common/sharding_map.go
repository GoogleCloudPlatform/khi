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

package common

import (
	"sync"
)

// ShardingMap provides locks by shards in a large map.
// This prevents lock entire map to access single element in a map.
type ShardingMap[T any] struct {
	shardingProvider MapShardingProvider
	shardedMaps      []map[string]T
	locks            []sync.RWMutex
}

func NewShardingMap[T any](shardingProvider MapShardingProvider) *ShardingMap[T] {
	shardCount := shardingProvider.GetMaxShardCount()
	shardedMaps := make([]map[string]T, shardCount)
	for i := 0; i < shardCount; i++ {
		shardedMaps[i] = map[string]T{}
	}
	return &ShardingMap[T]{
		shardingProvider: shardingProvider,
		shardedMaps:      shardedMaps,
		locks:            make([]sync.RWMutex, shardCount),
	}
}

func (m *ShardingMap[T]) AcquireShard(key string) map[string]T {
	shard := m.shardingProvider.GetShard(key)
	m.locks[shard].Lock()
	return m.shardedMaps[shard]
}

func (m *ShardingMap[T]) AcquireShardReadonly(key string) map[string]T {
	shard := m.shardingProvider.GetShard(key)
	m.locks[shard].RLock()
	return m.shardedMaps[shard]
}

func (m *ShardingMap[T]) ReleaseShard(key string) {
	shard := m.shardingProvider.GetShard(key)
	m.locks[shard].Unlock()
}

func (m *ShardingMap[T]) ReleaseShardReadonly(key string) {
	shard := m.shardingProvider.GetShard(key)
	m.locks[shard].RUnlock()
}

func (m *ShardingMap[T]) AllKeys() (result []string) {
	for i := 0; i < len(m.locks); i++ {
		m.locks[i].RLock()
		shard := m.shardedMaps[i]
		for key := range shard {
			result = append(result, key)
		}
		m.locks[i].RUnlock()
	}
	return result
}

// MapShardingProvider interface defines how keys are mapped to specific shard.
type MapShardingProvider interface {
	// GetMaxShardCount returns the maximum count of shards. This must be constant value.
	GetMaxShardCount() int
	// GetShard returns the index of shard mapping to the given key. The index must be less than the returned value of GetMaxShardCount()
	GetShard(key string) int
}

type SuffixShardingProvider struct {
	ShardCount   int
	suffixLength int
}

func NewSuffixShardingProvider(shardCount int, suffixLength int) *SuffixShardingProvider {
	return &SuffixShardingProvider{
		ShardCount:   shardCount,
		suffixLength: suffixLength,
	}
}

// GetMaxShardCount implements MapShardingProvider.
func (h *SuffixShardingProvider) GetMaxShardCount() int {
	return h.ShardCount
}

// GetShard implements MapShardingProvider.
func (h *SuffixShardingProvider) GetShard(key string) int {
	shard := 0
	for i := 0; i < h.suffixLength; i++ {
		if len(key) <= i {
			break
		}
		shard = (shard + int(key[len(key)-1-i])) % h.ShardCount
	}
	return shard
}

var _ (MapShardingProvider) = (*SuffixShardingProvider)(nil)
