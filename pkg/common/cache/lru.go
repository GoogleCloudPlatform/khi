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
	"sync"
)

var ErrEmptyChain = errors.New("empty chain")

type usageChainElem struct {
	Prev *usageChainElem
	Next *usageChainElem
	Key  string
}

// lruCacheItemReleaseStrategy is a data structure to find the least recent used key and updating the newly used key in O(1)
type lruCacheItemReleaseStrategy struct {
	// Least used element key
	head *usageChainElem
	// Newest element key
	tail *usageChainElem
	// References to the each chain elem
	refs     map[string]*usageChainElem
	size     int
	capacity int
	lock     sync.Mutex
}

var _ CacheItemReleaseStrategy = (*lruCacheItemReleaseStrategy)(nil)

func NewLRUCacheItemReleaseStrategy(capacity int) *lruCacheItemReleaseStrategy {
	return &lruCacheItemReleaseStrategy{
		head:     nil,
		tail:     nil,
		refs:     make(map[string]*usageChainElem),
		size:     0,
		capacity: capacity,
	}
}

// popHead removes the top element from this chain
func (c *lruCacheItemReleaseStrategy) popHead() (string, error) {
	if c.size == 0 {
		return "", ErrEmptyChain
	}
	prevHead := c.head
	c.head = c.head.Prev
	prevHead.Prev = nil
	prevHead.Next = nil
	c.head.Next = nil
	c.size--
	delete(c.refs, prevHead.Key)
	return prevHead.Key, nil
}

func (c *lruCacheItemReleaseStrategy) pushTail(chainElem *usageChainElem) {
	prevTail := c.tail
	if prevTail != nil {
		prevTail.Prev = chainElem
	}
	if c.head == nil {
		c.head = chainElem
	}
	chainElem.Next = prevTail
	c.tail = chainElem
	c.size++
}

func (c *lruCacheItemReleaseStrategy) unlink(chainElem *usageChainElem) {
	if chainElem.Prev != nil {
		chainElem.Prev.Next = chainElem.Next
	}
	if chainElem.Next != nil {
		chainElem.Next.Prev = chainElem.Prev
	}
	if chainElem == c.head {
		c.head = chainElem.Prev
	}
	if chainElem == c.tail {
		c.tail = chainElem.Next
	}
	chainElem.Prev = nil
	chainElem.Next = nil
	c.size--
}

// TouchAndGetRemovedKey update the key to be recently used.
// If it was newly added and chain has the size over capacity, it will return the key to be removed.
func (c *lruCacheItemReleaseStrategy) TouchAndGetRemovedKey(key string) (removedKey string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if elem, ok := c.refs[key]; ok {
		if elem == c.tail {
			return ""
		}
		c.unlink(elem)
		c.pushTail(elem)
	} else {
		chainElem := &usageChainElem{
			Prev: nil,
			Next: c.tail,
			Key:  key,
		}
		c.refs[key] = chainElem
		c.pushTail(chainElem)
		if c.size > c.capacity {
			removedKey, _ = c.popHead()
			return removedKey
		}
	}
	return ""
}
