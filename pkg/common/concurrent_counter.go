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

// A thread-safe counter data structure
type ConcurrentCounter struct {
	counts *ShardingMap[int]
}

func NewConcurrentCounter(shardingMap *ShardingMap[int]) *ConcurrentCounter {
	return &ConcurrentCounter{
		counts: shardingMap,
	}
}

func NewDefaultConcurrentCounter(shardingProvider MapShardingProvider) *ConcurrentCounter {
	return NewConcurrentCounter(NewShardingMap[int](shardingProvider))
}

func (c *ConcurrentCounter) Get(key string) int {
	shard := c.counts.AcquireShardReadonly(key)
	defer c.counts.ReleaseShardReadonly(key)
	if count, found := shard[key]; found {
		return count
	} else {
		return 0
	}
}

func (c *ConcurrentCounter) Incr(key string) int {
	shard := c.counts.AcquireShard(key)
	defer c.counts.ReleaseShard(key)
	if count, found := shard[key]; found {
		shard[key] = count + 1
	} else {
		shard[key] = 1
	}
	return shard[key]
}
