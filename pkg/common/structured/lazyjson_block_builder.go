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
)

// DefaultLazyJSONBlockMaxEntries is the default maximum number of entries per block.
const DefaultLazyJSONBlockMaxEntries = 100

// DefaultLazyJSONBlockMaxBytes is the default maximum uncompressed byte size per block.
const DefaultLazyJSONBlockMaxBytes = 256 * 1024

// LazyJSONBlockBuilder batches multiple JSON entries into blocks, registers them into LazyJSONBlockStore,
// and compresses them when size thresholds are reached.
type LazyJSONBlockBuilder struct {
	store          *LazyJSONBlockStore
	maxEntries     int
	maxBytes       int
	currentBlockID uint32
	currentBuffer  []byte
	entryCount     int
	hasActiveBlock bool
	mu             sync.Mutex
}

func newLazyJSONBlockBuilder(store *LazyJSONBlockStore, maxEntries, maxBytes int) *LazyJSONBlockBuilder {
	if maxEntries <= 0 {
		maxEntries = DefaultLazyJSONBlockMaxEntries
	}
	if maxBytes <= 0 {
		maxBytes = DefaultLazyJSONBlockMaxBytes
	}
	return &LazyJSONBlockBuilder{
		store:      store,
		maxEntries: maxEntries,
		maxBytes:   maxBytes,
	}
}

// Add appends a JSON byte slice to the current block buffer and returns a LazyJSONNode pointing to its location.
func (b *LazyJSONBlockBuilder) Add(data []byte) *LazyJSONNode {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.hasActiveBlock {
		b.startNewBlockLocked()
	}

	offset := uint32(len(b.currentBuffer))
	length := uint32(len(data))
	b.currentBuffer = append(b.currentBuffer, data...)
	b.entryCount++

	b.store.mu.RLock()
	block := b.store.blocks[b.currentBlockID]
	b.store.mu.RUnlock()

	block.mu.Lock()
	block.uncompressed = b.currentBuffer
	block.mu.Unlock()

	node := &LazyJSONNode{
		store:   b.store,
		blockID: b.currentBlockID,
		offset:  offset,
		length:  length,
		index:   0,
	}

	if b.entryCount >= b.maxEntries || len(b.currentBuffer) >= b.maxBytes {
		b.flushCurrentBlockLocked()
	}

	return node
}

// Flush compresses and commits any remaining entries in the current block buffer.
func (b *LazyJSONBlockBuilder) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.hasActiveBlock && len(b.currentBuffer) > 0 {
		b.flushCurrentBlockLocked()
	}
}

func (b *LazyJSONBlockBuilder) startNewBlockLocked() {
	b.currentBuffer = make([]byte, 0, b.maxBytes)
	b.currentBlockID = b.store.ReserveBlock(b.currentBuffer)
	b.entryCount = 0
	b.hasActiveBlock = true
}

func (b *LazyJSONBlockBuilder) flushCurrentBlockLocked() {
	if !b.hasActiveBlock {
		return
	}
	compressed, err := compressBlockData(b.currentBuffer)
	if err != nil {
		panic(fmt.Sprintf("failed to compress block %d: %v", b.currentBlockID, err))
	}
	b.store.SetCompressed(b.currentBlockID, compressed)
	b.hasActiveBlock = false
	b.currentBuffer = nil
	b.entryCount = 0
}
