// Copyright 2025 Google LLC
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

package patternfinder

import (
	"sync"
)

// trieNode represents a node in the Trie.
type trieNode[T any] struct {
	children       map[rune]*trieNode[T]
	isEndOfPattern bool
	outcome        T
}

// newTrieNode creates a new Trie node.
func newTrieNode[T any]() *trieNode[T] {
	return &trieNode[T]{
		children: make(map[rune]*trieNode[T]),
	}
}

// triePatternFinder is an implementation of PatternFinder using a Trie data structure.
type triePatternFinder[T any] struct {
	root *trieNode[T]
	mu   sync.RWMutex
}

// NewTriePatternFinder creates a new instance of triePatternFinder.
func NewTriePatternFinder[T any]() PatternFinder[T] {
	return &triePatternFinder[T]{
		root: newTrieNode[T](),
	}
}

// AddPattern adds a new pattern and its outcome to the finder.
func (f *triePatternFinder[T]) AddPattern(pattern string, outcome T) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	node := f.root
	for _, r := range pattern {
		if _, ok := node.children[r]; !ok {
			node.children[r] = newTrieNode[T]()
		}
		node = node.children[r]
	}

	if node.isEndOfPattern {
		return ErrPatternAlreadyExists
	}

	node.isEndOfPattern = true
	node.outcome = outcome
	return nil
}

// findNode traverses the Trie and returns the node corresponding to the pattern.
func (f *triePatternFinder[T]) findNode(pattern string) *trieNode[T] {
	node := f.root
	for _, r := range pattern {
		if n, ok := node.children[r]; ok {
			node = n
		} else {
			return nil
		}
	}
	return node
}

// GetPattern retrieves the outcome for a given pattern.
func (f *triePatternFinder[T]) GetPattern(pattern string) (T, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	node := f.findNode(pattern)
	if node == nil || !node.isEndOfPattern {
		return *new(T), ErrPatternNotFound
	}

	return node.outcome, nil
}

// DeletePattern removes a pattern from the finder.
func (f *triePatternFinder[T]) DeletePattern(pattern string) (T, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	node := f.findNode(pattern)
	if node == nil || !node.isEndOfPattern {
		return *new(T), ErrPatternNotFound
	}

	originalOutcome := node.outcome
	node.isEndOfPattern = false
	node.outcome = *new(T) // Clear the outcome

	return originalOutcome, nil
}

// Match checks for the longest registered pattern that is a prefix of the searchTarget.
func (f *triePatternFinder[T]) Match(searchTarget []rune) *PatternMatchResult[T] {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var bestMatch *PatternMatchResult[T]
	node := f.root

	for i, r := range searchTarget {
		if nextNode, ok := node.children[r]; ok {
			node = nextNode
			if node.isEndOfPattern {
				// Found a valid pattern, record it as the current best match
				bestMatch = &PatternMatchResult[T]{
					Value: node.outcome,
					Start: 0, // Start is always 0 for a prefix match on a given slice
					End:   i + 1,
				}
			}
		} else {
			// No further matches possible
			break
		}
	}

	return bestMatch
}
