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

import "errors"

var (
	// ErrPatternAlreadyExists is returned when trying to add a pattern that already exists.
	ErrPatternAlreadyExists = errors.New("pattern already exists")
	// ErrPatternNotFound is returned when a specified pattern cannot be found.
	ErrPatternNotFound = errors.New("pattern not found")
)

// PatternFinder provides a way to find the longest registered pattern that is a prefix of a given text.
type PatternFinder[T any] interface {
	// AddPattern adds a pattern to the finder.
	// It returns ErrPatternAlreadyExists if the pattern has already been added.
	AddPattern(pattern string, outcome T) error
	GetPattern(pattern string) (T, error)
	DeletePattern(pattern string) (T, error)

	// Match checks if any registered pattern is a prefix of the searchTarget.
	// It returns the longest valid match. If no patterns match, it returns nil.
	// The result's Start field will always be 0.
	Match(searchTarget []rune) *PatternMatchResult[T]
}
