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
	"strings"
	"sync"
)

// radixNode represents a node in the Radix Tree (compressed Patricia tree).
type radixNode[T any] struct {
	prefix   string
	indices  string
	children []*radixNode[T]
	hasValue bool
	value    T
}

// radixPatternFinder is an implementation of PatternFinder using a Radix Tree.
type radixPatternFinder[T any] struct {
	root *radixNode[T]
	mu   sync.RWMutex
}

var _ PatternFinder[any] = (*radixPatternFinder[any])(nil)

// NewRadixPatternFinder creates a new instance of radixPatternFinder backed by a Radix Tree.
func NewRadixPatternFinder[T any]() PatternFinder[T] {
	return &radixPatternFinder[T]{
		root: &radixNode[T]{},
	}
}

// longestCommonPrefix returns the length of the longest common prefix of s1 and s2.
func longestCommonPrefix(s1, s2 string) int {
	maxLen := min(len(s1), len(s2))
	i := 0
	for i < maxLen && s1[i] == s2[i] {
		i++
	}
	return i
}

// AddPattern adds a new pattern and its outcome to the finder.
func (f *radixPatternFinder[T]) AddPattern(pattern string, outcome T) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if pattern == "" {
		if f.root.hasValue {
			return ErrPatternAlreadyExists
		}
		f.root.hasValue = true
		f.root.value = outcome
		return nil
	}

	curr := f.root
	search := pattern

	for {
		idx := strings.IndexByte(curr.indices, search[0])
		if idx == -1 {
			child := &radixNode[T]{
				prefix:   search,
				hasValue: true,
				value:    outcome,
			}
			curr.indices += string([]byte{search[0]})
			curr.children = append(curr.children, child)
			return nil
		}

		child := curr.children[idx]
		commonPrefixLen := longestCommonPrefix(search, child.prefix)

		if commonPrefixLen == len(child.prefix) {
			if commonPrefixLen == len(search) {
				if child.hasValue {
					return ErrPatternAlreadyExists
				}
				child.hasValue = true
				child.value = outcome
				return nil
			}
			search = search[commonPrefixLen:]
			curr = child
			continue
		}

		splitNode := &radixNode[T]{
			prefix:   child.prefix[:commonPrefixLen],
			indices:  string([]byte{child.prefix[commonPrefixLen]}),
			children: []*radixNode[T]{child},
		}

		child.prefix = child.prefix[commonPrefixLen:]

		if commonPrefixLen == len(search) {
			splitNode.hasValue = true
			splitNode.value = outcome
		} else {
			newChild := &radixNode[T]{
				prefix:   search[commonPrefixLen:],
				hasValue: true,
				value:    outcome,
			}
			splitNode.indices += string([]byte{search[commonPrefixLen]})
			splitNode.children = append(splitNode.children, newChild)
		}

		curr.children[idx] = splitNode
		return nil
	}
}

// GetPattern retrieves the outcome for a given pattern.
func (f *radixPatternFinder[T]) GetPattern(pattern string) (T, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	curr := f.root
	search := pattern

	for {
		if len(search) == 0 {
			if curr.hasValue {
				return curr.value, nil
			}
			return *new(T), ErrPatternNotFound
		}

		idx := strings.IndexByte(curr.indices, search[0])
		if idx == -1 {
			return *new(T), ErrPatternNotFound
		}

		child := curr.children[idx]
		if !strings.HasPrefix(search, child.prefix) {
			return *new(T), ErrPatternNotFound
		}

		search = search[len(child.prefix):]
		curr = child
	}
}

// DeletePattern removes a pattern from the finder.
func (f *radixPatternFinder[T]) DeletePattern(pattern string) (T, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if pattern == "" {
		if !f.root.hasValue {
			return *new(T), ErrPatternNotFound
		}
		out := f.root.value
		f.root.hasValue = false
		f.root.value = *new(T)
		return out, nil
	}

	return f.delete(f.root, pattern)
}

// delete removes a pattern from the subtree rooted at curr.
func (f *radixPatternFinder[T]) delete(curr *radixNode[T], search string) (T, error) {
	idx := strings.IndexByte(curr.indices, search[0])
	if idx == -1 {
		return *new(T), ErrPatternNotFound
	}

	child := curr.children[idx]
	if !strings.HasPrefix(search, child.prefix) {
		return *new(T), ErrPatternNotFound
	}

	if len(search) == len(child.prefix) {
		if !child.hasValue {
			return *new(T), ErrPatternNotFound
		}
		out := child.value
		child.hasValue = false
		child.value = *new(T)

		if len(child.children) == 0 {
			curr.indices = curr.indices[:idx] + curr.indices[idx+1:]
			copy(curr.children[idx:], curr.children[idx+1:])
			curr.children[len(curr.children)-1] = nil
			curr.children = curr.children[:len(curr.children)-1]
		} else if len(child.children) == 1 {
			grandChild := child.children[0]
			child.prefix += grandChild.prefix
			child.indices = grandChild.indices
			child.children = grandChild.children
			child.hasValue = grandChild.hasValue
			child.value = grandChild.value
		}

		return out, nil
	}

	out, err := f.delete(child, search[len(child.prefix):])
	if err != nil {
		return *new(T), err
	}

	if len(child.children) == 1 && !child.hasValue {
		grandChild := child.children[0]
		child.prefix += grandChild.prefix
		child.indices = grandChild.indices
		child.children = grandChild.children
		child.hasValue = grandChild.hasValue
		child.value = grandChild.value
	}

	return out, nil
}

// Match checks for the longest registered pattern that is a prefix of the searchTarget.
func (f *radixPatternFinder[T]) Match(searchTarget string) (PatternMatchResult[T], bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	curr := f.root
	search := searchTarget
	currentOffset := 0

	var bestVal T
	var bestEnd int
	var found bool

	if curr.hasValue {
		bestVal = curr.value
		bestEnd = 0
		found = true
	}

	for len(search) > 0 {
		idx := strings.IndexByte(curr.indices, search[0])
		if idx == -1 {
			break
		}

		child := curr.children[idx]
		if !strings.HasPrefix(search, child.prefix) {
			break
		}

		currentOffset += len(child.prefix)
		search = search[len(child.prefix):]
		curr = child

		if curr.hasValue {
			bestVal = curr.value
			bestEnd = currentOffset
			found = true
		}
	}

	if found {
		return PatternMatchResult[T]{
			Value: bestVal,
			Start: 0,
			End:   bestEnd,
		}, true
	}
	return PatternMatchResult[T]{}, false
}
