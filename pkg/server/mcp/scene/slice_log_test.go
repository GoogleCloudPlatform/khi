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

package scene

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSliceLogBody(t *testing.T) {
	const serializedLogYAML = `insertId: abc-123
protoPayload:
  authenticationInfo:
    principalEmail: kubelet@node-1
  methodName: io.k8s.core.v1.pods.status.patch
resource:
  labels:
    pod_name: web-1
textPayload: "panic: runtime error: out of memory\ngoroutine 1 [running]:\nmain.handleRequest(0x1)\n\t/app/server.go:42\nmain.main()\n\t/app/main.go:10"
timestamp: 2026-10-01T09:25:00Z
`

	testCases := []struct {
		name    string
		yaml    string
		summary string
		opts    LogSliceOptions
		want    *LogSliceResult
	}{
		{
			name:    "expands multiline quoted string and slices multiple spots with prepended summary",
			yaml:    serializedLogYAML,
			summary: "Container web-1 terminated with OOMKilled",
			opts: LogSliceOptions{
				IncludeSummary: boolRef(true),
				MaxLines:       10,
				Windows: []SliceWindow{
					{Query: "principalEmail", BeforeLines: 0, AfterLines: 0},
					{Query: "handleRequest", BeforeLines: 0, AfterLines: 1},
				},
			},
			want: &LogSliceResult{
				TotalLines: 16,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindSummary, Text: "Container web-1 terminated with OOMKilled"},
					{Kind: ExcerptLineKindOmitted, Omitted: 3, Text: "⋮ 3 lines omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 4,
						Text:       "    principalEmail: kubelet@node-1",
						Parts: []LinePart{
							{Text: "    "},
							{Text: "principalEmail", Matched: true},
							{Text: ": kubelet@node-1"},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 7, Text: "⋮ 7 lines omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 12,
						Text:       "  main.handleRequest(0x1)",
						Parts: []LinePart{
							{Text: "  main."},
							{Text: "handleRequest", Matched: true},
							{Text: "(0x1)"},
						},
					},
					{Kind: ExcerptLineKindContext, NewLineNum: 13, Text: "  \t/app/server.go:42"},
					{Kind: ExcerptLineKindOmitted, Omitted: 3, Text: "⋮ 3 lines omitted"},
				},
			},
		},
		{
			name:    "combines JSONPath YAML selector with path-scoped multiline substring query",
			yaml:    serializedLogYAML,
			summary: "Container web-1 terminated with OOMKilled",
			opts: LogSliceOptions{
				IncludeSummary: boolRef(true),
				MaxLines:       10,
				Windows: []SliceWindow{
					{Path: "$.protoPayload.authenticationInfo.principalEmail"},
					{Path: "$.textPayload", Query: "handleRequest", AfterLines: 1},
				},
			},
			want: &LogSliceResult{
				TotalLines: 16,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindSummary, Text: "Container web-1 terminated with OOMKilled"},
					{Kind: ExcerptLineKindOmitted, Omitted: 3, Text: "⋮ 3 lines omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 4,
						Text:       "    principalEmail: kubelet@node-1",
						Parts: []LinePart{
							{Text: "    "},
							{Text: "principalEmail: kubelet@node-1", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 7, Text: "⋮ 7 lines omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 12,
						Text:       "  main.handleRequest(0x1)",
						Parts: []LinePart{
							{Text: "  main."},
							{Text: "handleRequest", Matched: true},
							{Text: "(0x1)"},
						},
					},
					{Kind: ExcerptLineKindContext, NewLineNum: 13, Text: "  \t/app/server.go:42"},
					{Kind: ExcerptLineKindOmitted, Omitted: 3, Text: "⋮ 3 lines omitted"},
				},
			},
		},
		{
			name:    "omitted windows falls back to parser-distilled summary",
			yaml:    serializedLogYAML,
			summary: "Container web-1 terminated with OOMKilled",
			opts:    LogSliceOptions{},
			want: &LogSliceResult{
				TotalLines: 16,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindSummary, Text: "Container web-1 terminated with OOMKilled"},
				},
			},
		},
		{
			name:    "omits summary when includeSummary is false and retains beforeLines context",
			yaml:    serializedLogYAML,
			summary: "Container web-1 terminated with OOMKilled",
			opts: LogSliceOptions{
				IncludeSummary: boolRef(false),
				Windows: []SliceWindow{
					{Query: "principalEmail", BeforeLines: 1, AfterLines: 0},
				},
			},
			want: &LogSliceResult{
				TotalLines: 16,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 2, Text: "⋮ 2 lines omitted"},
					{Kind: ExcerptLineKindContext, NewLineNum: 3, Text: "  authenticationInfo:"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 4,
						Text:       "    principalEmail: kubelet@node-1",
						Parts: []LinePart{
							{Text: "    "},
							{Text: "principalEmail", Matched: true},
							{Text: ": kubelet@node-1"},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 12, Text: "⋮ 12 lines omitted"},
				},
			},
		},
		{
			name:    "omits empty summary even when includeSummary is true and slices multiline block header",
			yaml:    serializedLogYAML,
			summary: "",
			opts: LogSliceOptions{
				IncludeSummary: boolRef(true),
				MaxLines:       2,
				Windows: []SliceWindow{
					{Path: "$.textPayload"},
				},
			},
			want: &LogSliceResult{
				TotalLines: 16,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 8, Text: "⋮ 8 lines omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 9,
						Text:       "textPayload: |",
						Parts: []LinePart{
							{Text: "textPayload: |", Matched: true},
						},
					},
					{Kind: ExcerptLineKindContext, NewLineNum: 10, Text: "  panic: runtime error: out of memory"},
					{Kind: ExcerptLineKindOmitted, Omitted: 6, Text: "⋮ 6 lines omitted"},
				},
			},
		},
		{
			name: "respects path scope boundary, beforeLines, maxMatches across scopes, and query-over-node highlight",
			yaml: `items:
- note: match-1
- note: match-2
afterItems: match-1
`,
			summary: "Two items",
			opts: LogSliceOptions{
				IncludeSummary: boolRef(false),
				Windows: []SliceWindow{
					{Path: "$.items[0]"},
					{Path: "$.items[*].note", Query: "match", BeforeLines: 1, MaxMatches: 2},
				},
			},
			want: &LogSliceResult{
				TotalLines: 4,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 1, Text: "⋮ 1 line omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 2,
						Text:       "- note: match-1",
						Parts: []LinePart{
							{Text: "- note: "},
							{Text: "match", Matched: true},
							{Text: "-1"},
						},
					},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 3,
						Text:       "- note: match-2",
						Parts: []LinePart{
							{Text: "- note: "},
							{Text: "match", Matched: true},
							{Text: "-2"},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 1, Text: "⋮ 1 line omitted"},
				},
			},
		},
		{
			name: "keeps quoted lines containing literal backslash-n unexpanded and stops at maxMatches=1",
			yaml: `literal: "keep \\n literal"
hits:
  first: target
  second: target
outside: target
`,
			summary: "Escapes",
			opts: LogSliceOptions{
				IncludeSummary: boolRef(false),
				Windows: []SliceWindow{
					{Path: "$.literal"},
					{Path: "$.hits", Query: "target", BeforeLines: 1, MaxMatches: 1},
				},
			},
			want: &LogSliceResult{
				TotalLines: 5,
				Lines: []ExcerptLine{
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 1,
						Text:       `literal: "keep \\n literal"`,
						Parts:      []LinePart{{Text: `literal: "keep \\n literal"`, Matched: true}},
					},
					{Kind: ExcerptLineKindContext, NewLineNum: 2, Text: "hits:"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 3,
						Text:       "  first: target",
						Parts: []LinePart{
							{Text: "  first: "},
							{Text: "target", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 2, Text: "⋮ 2 lines omitted"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SliceLogBody(tc.yaml, tc.summary, tc.opts)
			if err != nil {
				t.Fatalf("SliceLogBody() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SliceLogBody() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSliceLogBody_Errors(t *testing.T) {
	const logYAML = `insertId: evt-1
message: OOMKilled
`

	t.Run("unmatched substring query returns SliceQueryNotFoundError", func(t *testing.T) {
		_, err := SliceLogBody(logYAML, "Container app OOMKilled", LogSliceOptions{
			Windows: []SliceWindow{{Query: "NoSuchField"}},
		})
		var gotErr *SliceQueryNotFoundError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceLogBody() err = %T (%v), want *SliceQueryNotFoundError", err, err)
		}
		want := &SliceQueryNotFoundError{
			WindowIndex:  0,
			Query:        "NoSuchField",
			TopLevelKeys: []string{"insertId", "message"},
			Summary:      "Container app OOMKilled",
		}
		if diff := cmp.Diff(want, gotErr); diff != "" {
			t.Errorf("SliceQueryNotFoundError mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty path and query returns InvalidSliceWindowError", func(t *testing.T) {
		_, err := SliceLogBody(logYAML, "Container app OOMKilled", LogSliceOptions{
			Windows: []SliceWindow{{}},
		})
		var gotErr *InvalidSliceWindowError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceLogBody() err = %T (%v), want *InvalidSliceWindowError", err, err)
		}
	})

	t.Run("omitted windows with empty summary rejects implicit raw-line dump", func(t *testing.T) {
		_, err := SliceLogBody(logYAML, "", LogSliceOptions{})
		var gotErr *InvalidSliceWindowError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceLogBody() err = %T (%v), want *InvalidSliceWindowError", err, err)
		}
	})

	t.Run("unmatched JSONPath returns SliceQueryNotFoundError with available keys", func(t *testing.T) {
		_, err := SliceLogBody(logYAML, "Container app OOMKilled", LogSliceOptions{
			Windows: []SliceWindow{{Path: "$.missing"}},
		})
		var gotErr *SliceQueryNotFoundError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceLogBody() err = %T (%v), want *SliceQueryNotFoundError", err, err)
		}
		want := &SliceQueryNotFoundError{
			WindowIndex:   0,
			Path:          "$.missing",
			AvailableKeys: []string{"insertId", "message"},
			TopLevelKeys:  []string{"insertId", "message"},
			Summary:       "Container app OOMKilled",
		}
		if diff := cmp.Diff(want, gotErr); diff != "" {
			t.Errorf("SliceQueryNotFoundError mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("matched JSONPath with unmatched query returns SliceQueryNotFoundError", func(t *testing.T) {
		_, err := SliceLogBody(logYAML, "Container app OOMKilled", LogSliceOptions{
			Windows: []SliceWindow{{Path: "$.message", Query: "NoMatch"}},
		})
		var gotErr *SliceQueryNotFoundError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceLogBody() err = %T (%v), want *SliceQueryNotFoundError", err, err)
		}
		want := &SliceQueryNotFoundError{
			WindowIndex:  0,
			Path:         "$.message",
			Query:        "NoMatch",
			TopLevelKeys: []string{"insertId", "message"},
			Summary:      "Container app OOMKilled",
		}
		if diff := cmp.Diff(want, gotErr); diff != "" {
			t.Errorf("SliceQueryNotFoundError mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty body returns nil expansion slices and invalid escape line remains unexpanded", func(t *testing.T) {
		exp, starts, ends := expandMultilineYAMLLinesWithMap("")
		if exp != nil || starts != nil || ends != nil {
			t.Errorf("expandMultilineYAMLLinesWithMap(\"\") = (%v, %v, %v), want (nil, nil, nil)", exp, starts, ends)
		}
		expInvalid, _, _ := expandMultilineYAMLLinesWithMap(`invalid: "bad \q\nkeep"`)
		if diff := cmp.Diff([]string{`invalid: "bad \q\nkeep"`}, expInvalid); diff != "" {
			t.Errorf("expandMultilineYAMLLinesWithMap(invalid) mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestLogSlicer_DocCachesParsedDocument(t *testing.T) {
	s := &logSlicer{yamlBody: "message: ok\n"}
	first, err := s.doc()
	if err != nil {
		t.Fatalf("first doc() unexpected error = %v", err)
	}
	second, err := s.doc()
	if err != nil {
		t.Fatalf("second doc() unexpected error = %v", err)
	}
	if first == nil || first != second {
		t.Errorf("second doc() pointer = %p, want cached %p", second, first)
	}
}
