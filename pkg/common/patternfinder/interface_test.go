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
	"strings"
	"testing"
)

func TestPatternFinderImplementations(t *testing.T) {
	finders := []struct {
		name        string
		constructor func() PatternFinder[int]
	}{
		{
			name: "naive",
			constructor: func() PatternFinder[int] {
				return NewNaivePatternFinder[int]()
			},
		},
		{
			name: "trie",
			constructor: func() PatternFinder[int] {
				return NewTriePatternFinder[int]()
			},
		},
	}

	for _, f := range finders {
		t.Run(f.name, func(t *testing.T) {
			// The Add_Get_Delete and ErrorConditions tests are still valid
			// as those parts of the interface have not changed.
			t.Run("Add_Get_Delete", func(t *testing.T) {
				finder := f.constructor()
				pattern := "hello"
				outcome := 123

				err := finder.AddPattern(pattern, outcome)
				if err != nil {
					t.Fatalf("AddPattern failed: %v", err)
				}

				gotOutcome, err := finder.GetPattern(pattern)
				if err != nil {
					t.Fatalf("GetPattern failed: %v", err)
				}
				if gotOutcome != outcome {
					t.Errorf("got outcome %d, want %d", gotOutcome, outcome)
				}

				deletedOutcome, err := finder.DeletePattern(pattern)
				if err != nil {
					t.Fatalf("DeletePattern failed: %v", err)
				}
				if deletedOutcome != outcome {
					t.Errorf("deleted outcome %d, want %d", deletedOutcome, outcome)
				}

				_, err = finder.GetPattern(pattern)
				if err != ErrPatternNotFound {
					t.Errorf("expected ErrPatternNotFound after delete, but got %v", err)
				}
			})

			t.Run("ErrorConditions", func(t *testing.T) {
				finder := f.constructor()
				pattern := "test_pattern"

				_, err := finder.GetPattern("non_existent")
				if err != ErrPatternNotFound {
					t.Errorf("expected ErrPatternNotFound, but got %v", err)
				}

				err = finder.AddPattern(pattern, 1)
				if err != nil {
					t.Fatalf("first AddPattern failed: %v", err)
				}
				err = finder.AddPattern(pattern, 2)
				if err != ErrPatternAlreadyExists {
					t.Errorf("expected ErrPatternAlreadyExists, but got %v", err)
				}
			})

			t.Run("Match", func(t *testing.T) {
				finder := f.constructor()
				finder.AddPattern("a", 1)
				finder.AddPattern("ab", 2)
				finder.AddPattern("abc", 3)
				finder.AddPattern(" unrelated", 99) // Should not match

				testCases := []struct {
					name      string
					text      string
					wantMatch bool
					wantValue int
					wantStart int
					wantEnd   int
				}{
					{"no match", "xyz", false, 0, 0, 0},
					{"exact match", "abc", true, 3, 0, 3},
					{"longer text", "abcd", true, 3, 0, 3},
					{"intermediate match", "ab", true, 2, 0, 2},
					{"shortest match", "a", true, 1, 0, 1},
					{"empty text", "", false, 0, 0, 0},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						result := finder.Match([]rune(tc.text))

						if !tc.wantMatch {
							if result != nil {
								t.Errorf("expected no match, but got one: %+v", result)
							}
							return
						}

						if result == nil {
							t.Fatal("expected a match, but got nil")
						}
						if result.Value != tc.wantValue {
							t.Errorf("got value %d, want %d", result.Value, tc.wantValue)
						}
						if result.Start != tc.wantStart {
							t.Errorf("got start %d, want %d", result.Start, tc.wantStart)
						}
						if result.End != tc.wantEnd {
							t.Errorf("got end %d, want %d", result.End, tc.wantEnd)
						}
					})
				}
			})
		})
	}
}

func BenchmarkPatternFinder(b *testing.B) {
	finders := []struct {
		name        string
		constructor func() PatternFinder[int]
	}{
		{
			name: "naive",
			constructor: func() PatternFinder[int] {
				return NewNaivePatternFinder[int]()
			},
		},
		{
			name: "trie",
			constructor: func() PatternFinder[int] {
				return NewTriePatternFinder[int]()
			},
		},
	}

	scenarios := []struct {
		name        string
		numPatterns int
	}{
		{"100_patterns", 100},
		{"1000_patterns", 1000},
		{"10000_patterns", 10000},
	}

	for _, f := range finders {
		b.Run(f.name, func(b *testing.B) {
			// AddPattern benchmark remains the same
			b.Run("AddPattern", func(b *testing.B) {
				patterns := generatePatterns(1000)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					finder := f.constructor()
					for j, p := range patterns {
						finder.AddPattern(p, j)
					}
				}
			})

			for _, s := range scenarios {
				b.Run(fmt.Sprintf("Match/%s", s.name), func(b *testing.B) {
					finder := f.constructor()
					patterns := generatePatterns(s.numPatterns)
					for i, p := range patterns {
						finder.AddPattern(p, i)
					}

					// Text that will match the last and longest pattern
					matchText := []rune(patterns[s.numPatterns-1] + "_extra_suffix")
					// Text that will not match any pattern
					noMatchText := []rune("zzzz_no_match_here")

					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						// Alternate between matching and not matching to get a mixed workload
						if i%2 == 0 {
							finder.Match(matchText)
						} else {
							finder.Match(noMatchText)
						}
					}
				})
			}
		})
	}
}

// generatePatterns creates a slice of unique string patterns.
func generatePatterns(count int) []string {
	patterns := make([]string, count)
	for i := 0; i < count; i++ {
		// Create incrementally longer patterns for prefix testing
		patterns[i] = fmt.Sprintf("p%d_", i) + strings.Repeat("a", i%10)
	}
	return patterns
}
