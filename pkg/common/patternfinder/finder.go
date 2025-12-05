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

import "fmt"

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
	runes := []rune(searchText)
	starters := make(map[rune]struct{}, len(starterRunes))
	for _, r := range starterRunes {
		starters[r] = struct{}{}
	}

	var results []PatternMatchResult[T]
	i := 0

	// Handle the case where a match can start at the very beginning
	if includeFirst {
		if match := finder.Match(runes); match != nil {
			results = append(results, PatternMatchResult[T]{
				Value: match.Value,
				Start: 0,
				End:   match.End,
			})
			i = match.End // Advance past this match
		}
	}

	for i < len(runes) {
		// Find the next starter rune
		_, isStarter := starters[runes[i]]
		if !isStarter {
			i++
			continue
		}

		// Starter rune found, attempt to match from the next position
		searchPosition := i + 1
		if searchPosition >= len(runes) {
			break // Reached the end of the string
		}

		searchSlice := runes[searchPosition:]
		if match := finder.Match(searchSlice); match != nil {
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
			// No match, just advance to the next character
			i++
		}
	}

	return results
}
