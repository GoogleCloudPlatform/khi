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
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
)

var ErrNotFoundInStorageErr = errors.New("not found in storage")

type Cache[T any] interface {
	// Get returns the associated data with converting to T
	Get(key string) (T, error)
	// SetAsync store the given cache data with the associated key in async.
	SetAsync(key string, value []byte)
}

type BasicCacheImpl[T any] struct {
	storage             CacheItemStorageProvider
	conveter            CacheItemConverter[T]
	itemReleaseStrategy CacheItemReleaseStrategy
	cache               map[string]T
	lock                sync.Mutex
}

type CacheItemReleaseStrategy interface {
	TouchAndGetRemovedKey(key string) (removedKey string)
}

type CacheItemConverter[T any] interface {
	Deserialize(source []byte) (T, error)
	Serialize(item T) ([]byte, error)
	Default() T
}

type CacheItemStorageProvider interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}

func NewCache[T any](storage CacheItemStorageProvider, resolver CacheItemConverter[T], itemReleaseStrategy CacheItemReleaseStrategy) Cache[T] {
	return &BasicCacheImpl[T]{
		storage:             storage,
		conveter:            resolver,
		itemReleaseStrategy: itemReleaseStrategy,
		cache:               map[string]T{},
	}
}

// Get the instance. It retrives the data from the storage interface when it's not available in the cache.
func (c *BasicCacheImpl[T]) Get(key string) (T, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if value, contained := c.cache[key]; contained {
		// ignore the returned value.
		// The cache won't purge any existing key because the item is already in the cache.
		c.itemReleaseStrategy.TouchAndGetRemovedKey(key)
		return value, nil
	} else {
		serializedData, err := c.storage.Get(key)
		if err != nil {
			return c.conveter.Default(), err
		}
		releasedKey := c.itemReleaseStrategy.TouchAndGetRemovedKey(key)
		data, err := c.conveter.Deserialize(serializedData)
		if err != nil {
			return c.conveter.Default(), err
		}
		// Release the purged value in cache
		if releasedKey != "" {
			delete(c.cache, releasedKey)
		}
		c.cache[key] = data
		return data, nil
	}
}

func (c *BasicCacheImpl[T]) SetAsync(key string, value []byte) {
	c.lock.Lock()
	go func() {
		defer c.lock.Unlock()
		data, err := c.conveter.Deserialize(value)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to store cache data: %v", err))
			return
		}
		if _, contained := c.cache[key]; contained {
			// ignore the returned value.
			// The cache won't purge any existing key because the item is already in the cache.
			c.itemReleaseStrategy.TouchAndGetRemovedKey(key)
		} else {
			releasedKey := c.itemReleaseStrategy.TouchAndGetRemovedKey(key)
			if releasedKey != "" {
				delete(c.cache, releasedKey)
			}
		}
		c.cache[key] = data
		err = c.storage.Set(key, value)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to store cache data: %v", err))
		}
	}()
}

type ShardingMapStorageProvider struct {
	shardingMap *common.ShardingMap[[]byte]
}

func NewShardingMapStorageProvider(shardingProvider common.MapShardingProvider) *ShardingMapStorageProvider {
	return &ShardingMapStorageProvider{
		shardingMap: common.NewShardingMap[[]byte](shardingProvider),
	}
}

// Get implements CacheItemStorageProvider.
func (s *ShardingMapStorageProvider) Get(key string) ([]byte, error) {
	shard := s.shardingMap.AcquireShardReadonly(key)
	defer s.shardingMap.ReleaseShardReadonly(key)
	if value, contained := shard[key]; contained {
		return value, nil
	} else {
		return nil, ErrNotFoundInStorageErr
	}
}

// Set implements CacheItemStorageProvider.
func (s *ShardingMapStorageProvider) Set(key string, value []byte) error {
	shard := s.shardingMap.AcquireShard(key)
	defer s.shardingMap.ReleaseShard(key)
	shard[key] = value
	return nil
}

var _ CacheItemStorageProvider = (*ShardingMapStorageProvider)(nil)
