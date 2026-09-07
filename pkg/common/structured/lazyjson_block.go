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

	"github.com/golang/snappy"
)

func compressBlockData(data []byte) ([]byte, error) {
	return snappy.Encode(nil, data), nil
}

type lazyJSONBlock struct {
	mu           sync.RWMutex
	compressed   []byte
	uncompressed []byte
}

func (b *lazyJSONBlock) getBuffer(store *LazyJSONBlockStore, blockID uint32) []byte {
	b.mu.RLock()
	if b.uncompressed != nil {
		data := b.uncompressed
		b.mu.RUnlock()
		store.touch(blockID)
		return data
	}
	b.mu.RUnlock()

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.uncompressed != nil {
		store.touch(blockID)
		return b.uncompressed
	}

	decompressed, err := store.decompress(b.compressed)
	if err != nil {
		panic(fmt.Sprintf("failed to decompress lazyJSON block %d: %v", blockID, err))
	}
	b.uncompressed = decompressed
	store.put(blockID, b)
	return decompressed
}

func (b *lazyJSONBlock) evict() {
	b.mu.Lock()
	if len(b.compressed) > 0 {
		b.uncompressed = nil
	}
	b.mu.Unlock()
}

type blockLRUEntry struct {
	blockID uint32
	block   *lazyJSONBlock
	prev    *blockLRUEntry
	next    *blockLRUEntry
}

type blockLRUShard struct {
	mu       sync.Mutex
	capacity int
	entries  map[uint32]*blockLRUEntry
	head     *blockLRUEntry
	tail     *blockLRUEntry
}

func newBlockLRUShard(capacity int) *blockLRUShard {
	return &blockLRUShard{
		capacity: capacity,
		entries:  make(map[uint32]*blockLRUEntry, capacity),
	}
}

func (s *blockLRUShard) touch(blockID uint32) {
	s.mu.Lock()
	if entry, ok := s.entries[blockID]; ok {
		s.moveToHead(entry)
	}
	s.mu.Unlock()
}

func (s *blockLRUShard) put(blockID uint32, b *lazyJSONBlock) {
	s.mu.Lock()
	if entry, ok := s.entries[blockID]; ok {
		s.moveToHead(entry)
		s.mu.Unlock()
		return
	}

	entry := &blockLRUEntry{blockID: blockID, block: b}
	s.entries[blockID] = entry
	s.addToHead(entry)

	var toEvict *lazyJSONBlock
	if len(s.entries) > s.capacity {
		toEvict = s.removeTail()
	}
	s.mu.Unlock()

	if toEvict != nil {
		toEvict.evict()
	}
}

func (s *blockLRUShard) addToHead(entry *blockLRUEntry) {
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

func (s *blockLRUShard) moveToHead(entry *blockLRUEntry) {
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

func (s *blockLRUShard) removeTail() *lazyJSONBlock {
	if s.tail == nil {
		return nil
	}
	evicted := s.tail
	delete(s.entries, evicted.blockID)
	if s.tail.prev != nil {
		s.tail.prev.next = nil
		s.tail = s.tail.prev
	} else {
		s.head = nil
		s.tail = nil
	}
	return evicted.block
}

// LazyJSONBlockStore manages blocks of compressed JSON data with a sharded LRU cache for decompressed blocks.
type LazyJSONBlockStore struct {
	mu        sync.RWMutex
	blocks    []*lazyJSONBlock
	shards    []*blockLRUShard
	shardMask uint32
	cache     *lazyJSONCache
}

// DefaultLazyJSONBlockStoreShardCount is the default shard count for LazyJSONBlockStore.
const DefaultLazyJSONBlockStoreShardCount = 64

// DefaultLazyJSONBlockStoreShardCap is the default capacity per shard, sizing total uncompressed cache to 1 GB (4,096 blocks * 256 KB).
const DefaultLazyJSONBlockStoreShardCap = 64

// NewDefaultLazyJSONBlockStore creates a new LazyJSONBlockStore with default 1 GB uncompressed cache capacity.
func NewDefaultLazyJSONBlockStore() *LazyJSONBlockStore {
	return NewLazyJSONBlockStore(DefaultLazyJSONBlockStoreShardCount, DefaultLazyJSONBlockStoreShardCap)
}

// NewLazyJSONBlockStore creates a new LazyJSONBlockStore with the specified shard count and shard capacity.
func NewLazyJSONBlockStore(shardCount, shardCap int) *LazyJSONBlockStore {
	if shardCount <= 0 || (shardCount&(shardCount-1)) != 0 {
		panic("shardCount must be a power of two")
	}
	shards := make([]*blockLRUShard, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = newBlockLRUShard(shardCap)
	}
	return &LazyJSONBlockStore{
		blocks:    make([]*lazyJSONBlock, 0, 64),
		shards:    shards,
		shardMask: uint32(shardCount - 1),
		cache:     newLazyJSONCache(lazyJSONCacheShardCount, lazyJSONCacheShardCap),
	}
}

// ReserveBlock registers a new block with initial uncompressed bytes and returns its unique block ID.
func (s *LazyJSONBlockStore) ReserveBlock(initialUncompressed []byte) uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()

	blockID := uint32(len(s.blocks))
	block := &lazyJSONBlock{
		uncompressed: initialUncompressed,
	}
	s.blocks = append(s.blocks, block)
	return blockID
}

// SetCompressed assigns compressed bytes to the designated block and inserts it into the LRU cache.
func (s *LazyJSONBlockStore) SetCompressed(blockID uint32, compressed []byte) {
	s.mu.RLock()
	block := s.blocks[blockID]
	s.mu.RUnlock()

	block.mu.Lock()
	block.compressed = compressed
	block.mu.Unlock()

	s.put(blockID, block)
}

// GetBuffer retrieves a byte slice from the uncompressed block at the given offset and length.
func (s *LazyJSONBlockStore) GetBuffer(blockID, offset, length uint32) []byte {
	s.mu.RLock()
	block := s.blocks[blockID]
	s.mu.RUnlock()

	decompressed := block.getBuffer(s, blockID)
	end := int(offset + length)
	if end > len(decompressed) {
		panic(fmt.Sprintf("lazyJSON block offset %d + length %d exceeds block size %d", offset, length, len(decompressed)))
	}
	return decompressed[offset:end]
}

func (s *LazyJSONBlockStore) touch(blockID uint32) {
	shardIdx := blockID & s.shardMask
	s.shards[shardIdx].touch(blockID)
}

func (s *LazyJSONBlockStore) put(blockID uint32, b *lazyJSONBlock) {
	shardIdx := blockID & s.shardMask
	s.shards[shardIdx].put(blockID, b)
}

func (s *LazyJSONBlockStore) decompress(compressed []byte) ([]byte, error) {
	decodedLen, err := snappy.DecodedLen(compressed)
	if err != nil {
		return nil, fmt.Errorf("failed to get snappy decoded length: %w", err)
	}
	decompressed := make([]byte, decodedLen)
	if _, err := snappy.Decode(decompressed, compressed); err != nil {
		return nil, fmt.Errorf("failed to decompress snappy block: %w", err)
	}
	return decompressed, nil
}

// NewBuilder creates a new LazyJSONBlockBuilder for appending log entries into this store.
func (s *LazyJSONBlockStore) NewBuilder(maxEntries, maxBytes int) *LazyJSONBlockBuilder {
	return newLazyJSONBlockBuilder(s, maxEntries, maxBytes)
}
