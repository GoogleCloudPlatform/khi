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
	"fmt"
	"unicode/utf8"
)

// PatternMatchResult represents a single match found within a larger text.
// It includes the start and end positions of the match.
type PatternMatchResult[T any] struct {
	Value T
	Start int
	End   int
}

// GetMatchedString extracts the matched string from the original string.
func (p *PatternMatchResult[T]) GetMatchedString(original string) (string, error) {
	if p.Start < 0 || p.End > len(original) {
		return "", fmt.Errorf("invalid match range: start=%d, end=%d", p.Start, p.End)
	}
	return original[p.Start:p.End], nil
}

// FindAllWithStarterRunes finds all occurrences of patterns within a search text.
// The search for a pattern only begins after encountering one of the specified starterRunes.
//
// Parameters:
//   - searchText: The string to search within.
//   - finder: The PatternFinder implementation to use for matching prefixes.
//   - includeFirst: If true, a match is attempted from the very beginning of the searchText,
//     without waiting for a starterRune. Useful for cases where the entire
//     string itself could be a valid pattern.
//   - starterRunes: A set of runes that act as triggers. When one of these runes is encountered,
//     a pattern search is attempted on the text immediately following the rune.
//
// Returns:
//
//	A slice of PatternMatchResult for every non-overlapping match found.
func FindAllWithStarterRunes[T any](searchText string, finder PatternFinder[T], includeFirst bool, starterRunes ...rune) []PatternMatchResult[T] {
	if len(searchText) == 0 {
		return nil
	}

	var results []PatternMatchResult[T]
	i := 0

	// Handle the case where a match can start at the very beginning
	if includeFirst {
		if match, ok := finder.Match(searchText); ok {
			results = append(results, PatternMatchResult[T]{
				Value: match.Value,
				Start: 0,
				End:   match.End,
			})
			i = match.End // Advance past this match
		}
	}

	for i < len(searchText) {
		var r rune
		var size int
		if searchText[i] < utf8.RuneSelf {
			r = rune(searchText[i])
			size = 1
		} else {
			r, size = utf8.DecodeRuneInString(searchText[i:])
		}

		isStarter := false
		for _, s := range starterRunes {
			if r == s {
				isStarter = true
				break
			}
		}

		if !isStarter {
			i += size
			continue
		}

		// Starter rune found, attempt to match from the next position
		searchPosition := i + size
		if searchPosition >= len(searchText) {
			break // Reached the end of the string
		}

		searchSlice := searchText[searchPosition:]
		if match, ok := finder.Match(searchSlice); ok {
			// A match was found, calculate absolute positions
			matchStart := searchPosition
			matchEnd := matchStart + match.End
			results = append(results, PatternMatchResult[T]{
				Value: match.Value,
				Start: matchStart,
				End:   matchEnd,
			})
			// Advance the main loop cursor past the found match
			i = matchEnd
		} else {
			// No match, advance to the next character
			i += size
		}
	}

	return results
}
