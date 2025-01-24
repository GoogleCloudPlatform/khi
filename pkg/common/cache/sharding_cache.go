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

import "github.com/GoogleCloudPlatform/khi/pkg/common"

type StorageProviderGenerator = func() CacheItemStorageProvider

type CacheItemReleaseStrategyGenerator = func() CacheItemReleaseStrategy

type ShardingCache[T any] struct {
	caches           []Cache[T]
	shardingProvider common.MapShardingProvider
}

// Get implements Cache.
func (s *ShardingCache[T]) Get(key string) (T, error) {
	shard := s.shardingProvider.GetShard(key)
	return s.caches[shard].Get(key)
}

// Set implements Cache.
func (s *ShardingCache[T]) SetAsync(key string, value []byte) {
	shard := s.shardingProvider.GetShard(key)
	s.caches[shard].SetAsync(key, value)
}

var _ Cache[any] = (*ShardingCache[any])(nil)

func NewShardingCache[T any](storageGenerator StorageProviderGenerator, resolver CacheItemConverter[T], itemReleaseStrategyGenerator CacheItemReleaseStrategyGenerator, shardingProvider common.MapShardingProvider) *ShardingCache[T] {
	caches := make([]Cache[T], shardingProvider.GetMaxShardCount())
	for i := 0; i < shardingProvider.GetMaxShardCount(); i++ {
		caches[i] = NewCache(storageGenerator(), resolver, itemReleaseStrategyGenerator())
	}
	return &ShardingCache[T]{
		caches:           caches,
		shardingProvider: shardingProvider,
	}
}
