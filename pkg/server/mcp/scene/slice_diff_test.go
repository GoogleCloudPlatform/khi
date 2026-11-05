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

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/google/go-cmp/cmp"
)

func TestSliceUnifiedDiff(t *testing.T) {
	oldNodeManifest := `apiVersion: v1
kind: Node
metadata:
  name: gke-node-1
spec:
  podCIDR: 10.0.0.0/24
  providerID: gce://proj/zone/node-1
  unschedulable: false
status:
  conditions:
  - reason: NoDiskPressure
    status: "False"
    type: DiskPressure
  - lastTransitionTime: "09:00:00Z"
    message: sufficient memory
    reason: KubeletHasSufficientMem
    status: "False"
    type: MemoryPressure
  - sep01: v
  - sep02: v
  - sep03: v
  - sep04: v
  - sep05: v
  - sep06: v
  - sep07: v
  - sep08: v
  - readyHeader: node-ready
    reason: KubeletReady
    status: "True"
    type: Ready
  tail01: a
  tail02: a
  tail03: a
  tail04: a
  tail05: a
  tail06: a
  tail07: a
  allocatable:
    memory: 8Gi
`

	newNodeManifest := `apiVersion: v1
kind: Node
metadata:
  name: gke-node-1
spec:
  podCIDR: 10.0.0.0/24
  providerID: gce://proj/zone/node-1
  unschedulable: false
status:
  conditions:
  - reason: NoDiskPressure
    status: "False"
    type: DiskPressure
  - lastTransitionTime: "09:00:00Z"
    message: sufficient memory
    reason: KubeletHasSufficientMem
    status: "True"
    type: MemoryPressure
  - sep01: v
  - sep02: v
  - sep03: v
  - sep04: v
  - sep05: v
  - sep06: v
  - sep07: v
  - sep08: v
  - readyHeader: node-ready
    reason: KubeletNotReady
    status: "False"
    type: Ready
  tail01: a
  tail02: a
  tail03: a
  tail04: a
  tail05: a
  tail06: a
  tail07: a
  allocatable:
    memory: 4Gi
`

	initialManifest := `apiVersion: v1
kind: Endpoints
metadata:
  name: frontend-svc
subsets:
- addresses:
  - ip: 10.0.1.5
    nodeName: gke-node-1
    ready: true
  - ip: 10.0.1.6
  targetRef:
    kind: Pod
    name: frontend-pod-2
  ports:
  - port: 80
`

	testCases := []struct {
		name              string
		oldText           string
		newText           string
		isInitialRevision bool
		opts              DiffSliceOptions
		want              *DiffSliceResult
	}{
		{
			name:    "slices two disjoint condition hunks preserving paired deletions and distinguishing unchanged vs diff gaps",
			oldText: oldNodeManifest,
			newText: newNodeManifest,
			opts: DiffSliceOptions{
				MaxLines: 14,
				Windows: []SliceWindow{
					{Query: "MemoryPressure", BeforeLines: 1, AfterLines: 0},
					{Query: "KubeletNotReady", BeforeLines: 1, AfterLines: 1},
				},
			},
			want: &DiffSliceResult{
				Changes:        workbench.LineChangeCount{Added: 4, Deleted: 4},
				TotalDiffLines: 23,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindDeleted, OldLineNum: 17, Text: "    status: \"False\""},
					{Kind: ExcerptLineKindAdded, NewLineNum: 17, Text: "    status: \"True\""},
					{
						Kind:       ExcerptLineKindContext,
						OldLineNum: 18,
						NewLineNum: 18,
						Text:       "    type: MemoryPressure",
						Parts: []LinePart{
							{Text: "    type: "},
							{Text: "MemoryPressure", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 8, Text: "⋮ 8 unchanged lines omitted"},
					{Kind: ExcerptLineKindContext, OldLineNum: 27, NewLineNum: 27, Text: "  - readyHeader: node-ready"},
					{Kind: ExcerptLineKindDeleted, OldLineNum: 28, Text: "    reason: KubeletReady"},
					{Kind: ExcerptLineKindDeleted, OldLineNum: 29, Text: "    status: \"True\""},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 28,
						Text:       "    reason: KubeletNotReady",
						Parts: []LinePart{
							{Text: "    reason: "},
							{Text: "KubeletNotReady", Matched: true},
						},
					},
					{Kind: ExcerptLineKindAdded, NewLineNum: 29, Text: "    status: \"False\""},
					{Kind: ExcerptLineKindContext, OldLineNum: 30, NewLineNum: 30, Text: "    type: Ready"},
					{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 diff lines omitted"},
				},
			},
		},
		{
			name:    "preserves added/deleted change pair on single-line match",
			oldText: oldNodeManifest,
			newText: newNodeManifest,
			opts: DiffSliceOptions{
				Windows: []SliceWindow{
					{Query: "KubeletNotReady", BeforeLines: 0, AfterLines: 0},
				},
			},
			want: &DiffSliceResult{
				Changes:        workbench.LineChangeCount{Added: 4, Deleted: 4},
				TotalDiffLines: 23,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 8, Text: "⋮ 8 diff lines omitted"},
					{Kind: ExcerptLineKindDeleted, OldLineNum: 28, Text: "    reason: KubeletReady"},
					{Kind: ExcerptLineKindDeleted, OldLineNum: 29, Text: `    status: "True"`},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 28,
						Text:       "    reason: KubeletNotReady",
						Parts: []LinePart{
							{Text: "    reason: "},
							{Text: "KubeletNotReady", Matched: true},
						},
					},
					{Kind: ExcerptLineKindAdded, NewLineNum: 29, Text: `    status: "False"`},
					{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 diff lines omitted"},
				},
			},
		},
		{
			name:              "initial revision slices multiple spots across full added manifest",
			oldText:           "ignored",
			newText:           initialManifest,
			isInitialRevision: true,
			opts: DiffSliceOptions{
				Windows: []SliceWindow{
					{Query: "ready: true", BeforeLines: 0, AfterLines: 0},
					{Query: "targetRef:", BeforeLines: 0, AfterLines: 2},
				},
			},
			want: &DiffSliceResult{
				Changes:        workbench.LineChangeCount{Added: 15},
				TotalDiffLines: 15,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 8, Text: "⋮ 8 diff lines omitted"},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 9,
						Text:       "    ready: true",
						Parts: []LinePart{
							{Text: "    "},
							{Text: "ready: true", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 1, Text: "⋮ 1 diff line omitted"},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 11,
						Text:       "  targetRef:",
						Parts: []LinePart{
							{Text: "  "},
							{Text: "targetRef:", Matched: true},
						},
					},
					{Kind: ExcerptLineKindAdded, NewLineNum: 12, Text: "    kind: Pod"},
					{Kind: ExcerptLineKindAdded, NewLineNum: 13, Text: "    name: frontend-pod-2"},
					{Kind: ExcerptLineKindOmitted, Omitted: 2, Text: "⋮ 2 diff lines omitted"},
				},
			},
		},
		{
			name: "omitted windows returns leading hunk lines up to maxLines",
			oldText: `a: 1
b: 2
`,
			newText: `a: 1
b: 3
c: 4
d: 5
`,
			opts: DiffSliceOptions{
				MaxLines: 3,
			},
			want: &DiffSliceResult{
				Changes:        workbench.LineChangeCount{Added: 3, Deleted: 1},
				TotalDiffLines: 5,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindContext, OldLineNum: 1, NewLineNum: 1, Text: "a: 1"},
					{Kind: ExcerptLineKindDeleted, OldLineNum: 2, Text: "b: 2"},
					{Kind: ExcerptLineKindAdded, NewLineNum: 2, Text: "b: 3"},
					{Kind: ExcerptLineKindOmitted, Omitted: 2, Text: "⋮ 2 diff lines omitted"},
				},
			},
		},
		{
			name: "renders both unchanged and diff omitted placeholders when gap skips an entire middle hunk",
			oldText: `a: 1
b: 2
c: 3
d: 4
e: 5
f: 6
g: 7
h: 8
i: 9
j: 10
k: 11
l: 12
m: 13
n: 14
o: 15
p: 16
q: 17
r: 18
s: 19
`,
			newText: `a: 1_mod
b: 2
c: 3
d: 4
e: 5
f: 6
g: 7
h: 8
i: 9
j: 10_mod
k: 11
l: 12
m: 13
n: 14
o: 15
p: 16
q: 17
r: 18
s: 19_mod
`,
			opts: DiffSliceOptions{
				Windows: []SliceWindow{
					{Query: "1_mod"},
					{Query: "19_mod"},
				},
			},
			want: &DiffSliceResult{
				Changes:        workbench.LineChangeCount{Added: 3, Deleted: 3},
				TotalDiffLines: 18,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindDeleted, OldLineNum: 1, Text: "a: 1"},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 1,
						Text:       "a: 1_mod",
						Parts: []LinePart{
							{Text: "a: "},
							{Text: "1_mod", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 unchanged lines omitted"},
					{Kind: ExcerptLineKindOmitted, Omitted: 8, Text: "⋮ 8 diff lines omitted"},
					{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 unchanged lines omitted"},
					{Kind: ExcerptLineKindDeleted, OldLineNum: 19, Text: "s: 19"},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 19,
						Text:       "s: 19_mod",
						Parts: []LinePart{
							{Text: "s: "},
							{Text: "19_mod", Matched: true},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SliceUnifiedDiff(tc.oldText, tc.newText, tc.isInitialRevision, tc.opts)
			if err != nil {
				t.Fatalf("SliceUnifiedDiff() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SliceUnifiedDiff() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSliceUnifiedDiff_Errors(t *testing.T) {
	oldYAML := `apiVersion: v1
kind: Pod
metadata:
  name: untouched-pod
  l1: a
  l2: a
  l3: a
  l4: a
status:
  phase: Pending
`
	newYAML := `apiVersion: v1
kind: Pod
metadata:
  name: untouched-pod
  l1: a
  l2: a
  l3: a
  l4: a
status:
  phase: Running
`

	t.Run("rejects empty query", func(t *testing.T) {
		_, err := SliceUnifiedDiff(oldYAML, newYAML, false, DiffSliceOptions{
			Windows: []SliceWindow{{Query: ""}},
		})
		var gotErr *InvalidSliceWindowError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceUnifiedDiff() err = %T (%v), want *InvalidSliceWindowError", err, err)
		}
		want := &InvalidSliceWindowError{WindowIndex: 0, Reason: "query must not be empty"}
		if diff := cmp.Diff(want, gotErr); diff != "" {
			t.Errorf("InvalidSliceWindowError mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("does not match unchanged lines outside diff hunks and returns diagnostic preview", func(t *testing.T) {
		_, err := SliceUnifiedDiff(oldYAML, newYAML, false, DiffSliceOptions{
			Windows: []SliceWindow{{Query: "untouched-pod"}},
		})
		var gotErr *SliceQueryNotFoundError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceUnifiedDiff() err = %T (%v), want *SliceQueryNotFoundError", err, err)
		}
		want := &SliceQueryNotFoundError{
			WindowIndex:  0,
			Query:        "untouched-pod",
			TopLevelKeys: []string{"status"},
			Preview: `-  phase: Pending
+  phase: Running`,
		}
		if diff := cmp.Diff(want, gotErr); diff != "" {
			t.Errorf("SliceQueryNotFoundError mismatch (-want +got):\n%s", diff)
		}
	})
}
