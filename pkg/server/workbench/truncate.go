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
	"fmt"
	"strings"
	"unicode/utf8"
)

// DefaultBodyByteLimit is the default maximum byte size per chunk (64 KB) for log bodies, manifests, and diffs.
const DefaultBodyByteLimit = 65536

// TruncatedBody holds a line-boundary-truncated slice of a body string along with truncation metadata.
type TruncatedBody struct {
	// Content is the sliced body text for the current chunk, without the trailing truncation comment marker.
	Content string
	// TotalBytes is the total byte length of the full body before truncation.
	TotalBytes int
	// NextByteOffset is the byte offset to pass in the next call to continue reading, or 0 when not truncated.
	NextByteOffset int
	// Truncated indicates whether additional bytes remain after this chunk.
	Truncated bool
}

// FormatWithMarker returns Content with the KHI YAML/diff truncation comment appended when Truncated is true.
func (b TruncatedBody) FormatWithMarker() string {
	if !b.Truncated {
		return b.Content
	}
	marker := fmt.Sprintf(
		"# [KHI] truncated: showing %d of %d bytes. Call again with byteOffset=%d to read the rest.",
		b.NextByteOffset,
		b.TotalBytes,
		b.NextByteOffset,
	)
	if b.Content == "" || strings.HasSuffix(b.Content, "\n") {
		return b.Content + marker
	}
	return b.Content + "\n" + marker
}

// TruncateBody slices body starting at byteOffset up to byteLimit bytes, cutting at the last newline boundary within the limit.
// If a single line exceeds byteLimit with no newline, it cuts at the largest valid UTF-8 rune boundary within byteLimit.
// If byteLimit <= 0, DefaultBodyByteLimit is used.
func TruncateBody(body string, byteOffset, byteLimit int) TruncatedBody {
	if byteLimit <= 0 {
		byteLimit = DefaultBodyByteLimit
	}
	if byteOffset < 0 {
		byteOffset = 0
	}

	totalBytes := len(body)
	if byteOffset >= totalBytes {
		return TruncatedBody{
			TotalBytes: totalBytes,
		}
	}

	rem := body[byteOffset:]
	if len(rem) <= byteLimit {
		return TruncatedBody{
			Content:    rem,
			TotalBytes: totalBytes,
		}
	}

	cutLen := lastLineOrRuneBoundary(rem, byteLimit)
	nextOffset := byteOffset + cutLen
	return TruncatedBody{
		Content:        rem[:cutLen],
		TotalBytes:     totalBytes,
		NextByteOffset: nextOffset,
		Truncated:      true,
	}
}

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
