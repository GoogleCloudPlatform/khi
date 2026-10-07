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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTruncateBody(t *testing.T) {
	testCases := []struct {
		name       string
		body       string
		byteOffset int
		byteLimit  int
		want       BodyChunk
	}{
		{
			name:       "body within limit is returned in full without truncation",
			body:       "apiVersion: v1\nkind: Pod\n",
			byteOffset: 0,
			byteLimit:  64,
			want: BodyChunk{
				Content:        "apiVersion: v1\nkind: Pod\n",
				TotalBytes:     25,
				NextByteOffset: 0,
			},
		},
		{
			name:       "body exceeding limit is cut at line boundary",
			body:       "line1: aaaa\nline2: bbbb\nline3: cccc\n",
			byteOffset: 0,
			byteLimit:  28,
			want: BodyChunk{
				Content:        "line1: aaaa\nline2: bbbb\n",
				TotalBytes:     36,
				NextByteOffset: 24,
			},
		},
		{
			name:       "continuation from NextByteOffset returns remaining lines",
			body:       "line1: aaaa\nline2: bbbb\nline3: cccc\n",
			byteOffset: 24,
			byteLimit:  28,
			want: BodyChunk{
				Content:        "line3: cccc\n",
				TotalBytes:     36,
				NextByteOffset: 0,
			},
		},
		{
			name:       "continuation that is truncated again reports an absolute next offset",
			body:       "line1: aaaa\nline2: bbbb\nline3: cccc\n",
			byteOffset: 12,
			byteLimit:  12,
			want: BodyChunk{
				Content:        "line2: bbbb\n",
				TotalBytes:     36,
				NextByteOffset: 24,
			},
		},
		{
			name:       "offset at or beyond total length returns empty non-truncated chunk",
			body:       "line1: aaaa\n",
			byteOffset: 12,
			byteLimit:  28,
			want: BodyChunk{
				Content:        "",
				TotalBytes:     12,
				NextByteOffset: 0,
			},
		},
		{
			name:       "single line exceeding limit without newline cuts at UTF-8 rune boundary",
			body:       "あいうえお\n",
			byteOffset: 0,
			byteLimit:  7,
			want: BodyChunk{
				Content:        "あい",
				TotalBytes:     16,
				NextByteOffset: 6,
			},
		},
		{
			name:       "invalid UTF-8 without any rune start within limit is cut at the limit to keep advancing",
			body:       "\x80\x80\x80\x80\x80",
			byteOffset: 0,
			byteLimit:  3,
			want: BodyChunk{
				Content:        "\x80\x80\x80",
				TotalBytes:     5,
				NextByteOffset: 3,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateBody(tc.body, tc.byteOffset, tc.byteLimit)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("truncateBody() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTruncateBody_RoundTripReconstruction(t *testing.T) {
	testCases := []struct {
		name      string
		body      string
		byteLimit int
	}{
		{
			name:      "multi-line YAML reconstructed across multiple chunks",
			body:      strings.Repeat("  key-entry: value-1234567890\n", 100),
			byteLimit: 250,
		},
		{
			name:      "mixed line lengths and final line without trailing newline",
			body:      "first: 1\n" + strings.Repeat("middle: abcdefghijklmnop\n", 20) + "last: true",
			byteLimit: 90,
		},
		{
			name:      "multi-byte body without newlines reconstructed across rune-boundary chunks",
			body:      strings.Repeat("あ", 30),
			byteLimit: 7,
		},
		{
			name:      "invalid UTF-8 body without newlines reconstructed across fixed-size chunks",
			body:      strings.Repeat("\x80", 10),
			byteLimit: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var reconstructed strings.Builder
			offset := 0
			chunks := 0
			for {
				res := truncateBody(tc.body, offset, tc.byteLimit)
				reconstructed.WriteString(res.Content)
				chunks++
				if !res.Truncated() {
					break
				}
				if res.NextByteOffset <= offset {
					t.Fatalf("NextByteOffset did not advance: got %d from offset %d", res.NextByteOffset, offset)
				}
				offset = res.NextByteOffset
			}
			if chunks < 2 {
				t.Errorf("expected multiple chunks, got %d", chunks)
			}
			if diff := cmp.Diff(tc.body, reconstructed.String()); diff != "" {
				t.Errorf("reconstructed body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
