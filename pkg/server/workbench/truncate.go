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

package workbench

import (
	"strings"
	"unicode/utf8"
)

// bodyByteLimit is the maximum byte size per chunk (64 KB) for log bodies, manifests, and diffs.
const bodyByteLimit = 65536

// BodyChunk holds a line-boundary-truncated slice of a body string along with continuation metadata.
type BodyChunk struct {
	// Content is the sliced body text for the current chunk.
	Content string
	// TotalBytes is the total byte length of the full body before truncation.
	TotalBytes int
	// NextByteOffset is the byte offset to pass in the next call to continue reading, or 0 when no bytes remain.
	NextByteOffset int
}

// Truncated reports whether additional bytes remain after this chunk.
func (c BodyChunk) Truncated() bool {
	return c.NextByteOffset > 0
}

// truncateBody slices body starting at byteOffset up to byteLimit bytes, cutting at the last newline boundary within the limit.
// If a single line exceeds byteLimit with no newline, it cuts at the largest valid UTF-8 rune boundary within byteLimit.
func truncateBody(body string, byteOffset, byteLimit int) BodyChunk {
	totalBytes := len(body)
	if byteOffset >= totalBytes {
		return BodyChunk{
			TotalBytes: totalBytes,
		}
	}

	remaining := body[byteOffset:]
	if len(remaining) <= byteLimit {
		return BodyChunk{
			Content:    remaining,
			TotalBytes: totalBytes,
		}
	}

	cutLen := lastLineOrRuneBoundary(remaining, byteLimit)
	return BodyChunk{
		Content:        remaining[:cutLen],
		TotalBytes:     totalBytes,
		NextByteOffset: byteOffset + cutLen,
	}
}

// lastLineOrRuneBoundary returns the cut length of s within limit bytes, preferring the last newline and then the last rune start.
// It returns limit when no rune start exists in s[1:limit+1], which only happens for invalid UTF-8, so that reading always advances.
func lastLineOrRuneBoundary(s string, limit int) int {
	if idx := strings.LastIndexByte(s[:limit], '\n'); idx >= 0 {
		return idx + 1
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	if cut == 0 {
		return limit
	}
	return cut
}
