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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typeddict"
)

// naivePatternFinder is a simple implementation of PatternFinder.
// It iterates through all registered patterns for every Match call.
type naivePatternFinder[T any] struct {
	patterns *typeddict.TypedDict[T]
	keys     []string
	mu       sync.RWMutex
}

// NewNaivePatternFinder creates a new instance of naivePatternFinder.
func NewNaivePatternFinder[T any]() PatternFinder[T] {
	return &naivePatternFinder[T]{
		patterns: typeddict.NewTypedDict[T](),
		keys:     []string{},
	}
}

// AddPattern adds a new pattern and its outcome to the finder.
func (f *naivePatternFinder[T]) AddPattern(pattern string, outcome T) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	_, found := typeddict.Get[T](f.patterns, pattern)
	if found {
		return ErrPatternAlreadyExists
	}

	typeddict.Set(f.patterns, pattern, outcome)
	f.keys = append(f.keys, pattern)
	return nil
}

// GetPattern retrieves the outcome for a given pattern.
func (f *naivePatternFinder[T]) GetPattern(pattern string) (T, error) {
	value, ok := typeddict.Get[T](f.patterns, pattern)
	if !ok {
		return *new(T), ErrPatternNotFound
	}
	return value, nil
}

// DeletePattern removes a pattern from the finder.
func (f *naivePatternFinder[T]) DeletePattern(pattern string) (T, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	value, ok := typeddict.Get[T](f.patterns, pattern)
	if !ok {
		return *new(T), ErrPatternNotFound
	}

	typeddict.Delete[T](f.patterns, pattern)

	for i, key := range f.keys {
		if key == pattern {
			f.keys = append(f.keys[:i], f.keys[i+1:]...)
			break
		}
	}

	return value, nil
}

// Match checks for the longest registered pattern that is a prefix of the searchTarget.
func (f *naivePatternFinder[T]) Match(searchTarget []rune) *PatternMatchResult[T] {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var bestMatch *PatternMatchResult[T]
	targetStr := string(searchTarget)

	for _, pattern := range f.keys {
		if strings.HasPrefix(targetStr, pattern) {
			if bestMatch == nil || len(pattern) > bestMatch.End {
				value, _ := typeddict.Get(f.patterns, pattern)
				bestMatch = &PatternMatchResult[T]{
					Value: value,
					Start: 0, // Start is always 0 for a prefix match on a given slice
					End:   len(pattern),
				}
			}
		}
	}

	return bestMatch
}
