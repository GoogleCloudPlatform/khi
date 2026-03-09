// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-;
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logutil

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// TabulateLineType indicates the classified type of a line in a tabulate format.
type TabulateLineType int

const (
	// TabulateLineTypeUnknown represents an unrecognized or out-of-table line.
	TabulateLineTypeUnknown TabulateLineType = iota
	// TabulateLineTypeHeaderCandidate represents a line that looks like a table header.
	TabulateLineTypeHeaderCandidate
	// TabulateLineTypeSeparator represents a separator line (e.g., "--- ---").
	TabulateLineTypeSeparator
	// TabulateLineTypeBody represents a data row within a table.
	TabulateLineTypeBody
)

// TabulateParseResult holds the parsing result of a single line.
type TabulateParseResult struct {
	// Type classification of the processed line.
	Type TabulateLineType
	// Columns is populated for HeaderCandidate and Body types, representing the column names.
	Columns []string
	// Values contains the parsed key-value pairs for a Body line.
	Values map[string]string
}

// ColumnBoundary defines the character indices bounding a single column.
type ColumnBoundary struct {
	Name  string
	Left  int
	Right int
}

// TabulateReader is a stateful reader for space-padded tabulate tables.
// It tracks headers and vertical column boundaries dynamically.
type TabulateReader struct {
	ColumnBoundaries []ColumnBoundary
	Headers          []string
}

// NewTabulateReader creates a new TabulateReader instance.
func NewTabulateReader() *TabulateReader {
	return &TabulateReader{}
}

// Reset clears the internal state when a table ends or a format error occurs.
func (r *TabulateReader) Reset() {
	r.ColumnBoundaries = nil
	r.Headers = nil
}

// ParseLine processes a single line, returning the parsed TabulateParseResult.
// It sequentially checks if the line is a separator, a body row (if inside a table),
// or a header candidate.
// Returns an error if a format violation is found (which resets internal state).
func (r *TabulateReader) ParseLine(line string) (*TabulateParseResult, error) {
	// 1. Check if it's a separator line.
	if r.parseSeparator(line) {
		return &TabulateParseResult{
			Type:    TabulateLineTypeSeparator,
			Columns: r.Headers,
		}, nil
	}

	// 2. If boundaries are established, attempt to parse as a body row.
	if len(r.ColumnBoundaries) > 0 {
		values, err := r.parseBodyRow(line)
		if err != nil {
			// Format error implies the table has ended or is severely malformed.
			r.Reset()
			return nil, err
		}

		var cols []string
		for _, b := range r.ColumnBoundaries {
			cols = append(cols, b.Name)
		}

		return &TabulateParseResult{
			Type:    TabulateLineTypeBody,
			Columns: cols,
			Values:  values,
		}, nil
	}

	// 3. If not in a table and not a separator, check if it's a header candidate.
	if r.parseHeaderCandidate(line) {
		return &TabulateParseResult{
			Type:    TabulateLineTypeHeaderCandidate,
			Columns: r.Headers,
		}, nil
	}

	// Otherwise, it's an unknown line.
	return &TabulateParseResult{
		Type: TabulateLineTypeUnknown,
	}, nil
}

// parseSeparator checks if the line is a separator and establishes ColumnBoundaries.
func (r *TabulateReader) parseSeparator(line string) bool {
	if !r.isSeparatorLine(line) {
		return false
	}
	groups := r.extractSeparatorGroups(line)
	r.ColumnBoundaries = r.calculateBoundaries(groups)
	return true
}

// isSeparatorLine checks if the line consists purely of hyphen blocks ("---").
func (r *TabulateReader) isSeparatorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	hasDash := false
	for _, ch := range line {
		if ch != '-' && ch != ' ' {
			return false
		}
		if ch == '-' {
			hasDash = true
		}
	}
	return hasDash
}

type separatorGroup struct {
	start int
	end   int
}

// extractSeparatorGroups finds the start and end indices of each hyphen block.
func (r *TabulateReader) extractSeparatorGroups(line string) []separatorGroup {
	var groups []separatorGroup
	inGroup := false
	start := 0
	for i, ch := range line {
		if ch == '-' {
			if !inGroup {
				start = i
				inGroup = true
			}
		} else {
			if inGroup {
				groups = append(groups, separatorGroup{start: start, end: i})
				inGroup = false
			}
		}
	}
	if inGroup {
		groups = append(groups, separatorGroup{start: start, end: len(line)})
	}
	return groups
}

// calculateBoundaries converts hyphen block groups into column boundaries using the midpoint between groups.
func (r *TabulateReader) calculateBoundaries(groups []separatorGroup) []ColumnBoundary {
	var boundaries []ColumnBoundary
	for i := 0; i < len(groups); i++ {
		left := 0
		if i > 0 {
			left = (groups[i-1].end + groups[i].start) / 2
		}
		right := math.MaxInt32
		if i < len(groups)-1 {
			right = (groups[i].end + groups[i+1].start) / 2
		}

		name := fmt.Sprintf("Column_%d", i)
		if i < len(r.Headers) {
			name = r.Headers[i]
		}

		boundaries = append(boundaries, ColumnBoundary{
			Name:  name,
			Left:  left,
			Right: right,
		})
	}
	return boundaries
}

// match 2 or more consecutive whitespace characters.
var twoOrMoreSpaces = regexp.MustCompile(`\s{2,}`)

// parseHeaderCandidate attempts to extract headers split by significant whitespace.
func (r *TabulateReader) parseHeaderCandidate(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	parts := twoOrMoreSpaces.Split(trimmed, -1)
	if len(parts) > 0 {
		r.Headers = parts
		return true
	}
	return false
}

// parseBodyRow extracts maps of column-value pairs using active boundaries.
func (r *TabulateReader) parseBodyRow(line string) (map[string]string, error) {
	if len(r.ColumnBoundaries) == 0 {
		return nil, fmt.Errorf("column boundaries are not initialized")
	}

	// Verify that text does not cross gap boundaries
	for i := 0; i < len(r.ColumnBoundaries)-1; i++ {
		rightBound := r.ColumnBoundaries[i].Right
		if rightBound < len(line) && line[rightBound] != ' ' {
			return nil, fmt.Errorf("format error: non-space character at boundary index %d", rightBound)
		}
	}

	values := make(map[string]string)
	for _, b := range r.ColumnBoundaries {
		left := b.Left
		right := b.Right

		if left >= len(line) {
			values[b.Name] = ""
			continue
		}
		if right > len(line) {
			right = len(line)
		}

		val := strings.TrimSpace(line[left:right])
		values[b.Name] = val
	}

	return values, nil
}
