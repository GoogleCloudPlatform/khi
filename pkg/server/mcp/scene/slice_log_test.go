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
}
