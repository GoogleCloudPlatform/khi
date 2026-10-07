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

// TruncatedBody holds a line-boundary-truncated slice of a body string along with truncation metadata.
type TruncatedBody struct {
	// Content is the sliced body text for the current chunk.
	Content string
	// TotalBytes is the total byte length of the full body before truncation.
	TotalBytes int
	// NextByteOffset is the byte offset to pass in the next call to continue reading, or 0 when not truncated.
	NextByteOffset int
	// Truncated indicates whether additional bytes remain after this chunk.
	Truncated bool
}

// truncateBody slices body starting at byteOffset up to byteLimit bytes, cutting at the last newline boundary within the limit.
// If a single line exceeds byteLimit with no newline, it cuts at the largest valid UTF-8 rune boundary within byteLimit.
func truncateBody(body string, byteOffset, byteLimit int) TruncatedBody {
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
	return TruncatedBody{
		Content:        rem[:cutLen],
		TotalBytes:     totalBytes,
		NextByteOffset: byteOffset + cutLen,
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
