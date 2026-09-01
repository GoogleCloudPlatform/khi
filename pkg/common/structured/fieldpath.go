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
	"strings"
	"unique"
)

// EmptyFieldPath represents an empty field path pointing to the root node itself.
var EmptyFieldPath = FieldPath{}

// FieldPath represents a pre-parsed and interned field path for fast traversal on StandardMapNode.
type FieldPath struct {
	segments []string
	handles  []unique.Handle[string]
}

// CompileFieldPath parses and interns a field path string (e.g., "resource.labels.node_name").
func CompileFieldPath(s string) FieldPath {
	if s == "" {
		return EmptyFieldPath
	}
	segments := ParseFieldPath(s)
	handles := make([]unique.Handle[string], len(segments))
	for i, seg := range segments {
		handles[i] = unique.Make(seg)
	}
	return FieldPath{segments: segments, handles: handles}
}

// Segments returns the underlying path segments.
func (p FieldPath) Segments() []string {
	return p.segments
}

// IsEmpty returns true if the field path is empty.
func (p FieldPath) IsEmpty() bool {
	return len(p.segments) == 0
}

// ParseFieldPath splits a field path string according to specified rules.
// It uses '.' as a delimiter, but '\.' is treated as an escaped literal dot.
func ParseFieldPath(s string) []string {
	var result []string
	var currentSegment strings.Builder
	isEscaped := false

	for _, r := range s {
		if isEscaped {
			if r == '.' {
				currentSegment.WriteRune('.') // '\.' is treated as a literal '.' and added to the current segment
			} else {
				// If '\' is followed by something other than '.', treat '\' as a literal character too
				currentSegment.WriteRune('\\')
				currentSegment.WriteRune(r)
			}
			isEscaped = false
		} else {
			switch r {
			case '\\':
				isEscaped = true
			case '.':
				result = append(result, currentSegment.String())
				currentSegment.Reset() // Reset the current segment
			default:
				currentSegment.WriteRune(r)
			}
		}
	}

	if isEscaped {
		// If the string ends with '\', treat it as a literal '\'
		currentSegment.WriteRune('\\')
	}

	result = append(result, currentSegment.String())
	return result
}
