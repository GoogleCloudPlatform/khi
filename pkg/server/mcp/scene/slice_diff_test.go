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
			name:    "slices two disjoint condition subtrees via JSONPath filter preserving paired deletions and unchanged vs diff gaps",
			oldText: oldNodeManifest,
			newText: newNodeManifest,
			opts: DiffSliceOptions{
				MaxLines: 16,
				Paths: []string{
					"$.status.conditions[?(@.type=='MemoryPressure')]",
					"$.status.conditions[?(@.type=='Ready')]",
				},
			},
			want: &DiffSliceResult{
				Changes:        workbench.LineChangeCount{Added: 4, Deleted: 4},
				TotalDiffLines: 23,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindContext, OldLineNum: 14, NewLineNum: 14, Text: `  - lastTransitionTime: "09:00:00Z"`},
					{Kind: ExcerptLineKindContext, OldLineNum: 15, NewLineNum: 15, Text: "    message: sufficient memory"},
					{Kind: ExcerptLineKindContext, OldLineNum: 16, NewLineNum: 16, Text: "    reason: KubeletHasSufficientMem"},
					{Kind: ExcerptLineKindDeleted, OldLineNum: 17, Text: `    status: "False"`},
					{Kind: ExcerptLineKindAdded, NewLineNum: 17, Text: `    status: "True"`},
					{
						Kind:       ExcerptLineKindContext,
						OldLineNum: 18,
						NewLineNum: 18,
						Text:       "    type: MemoryPressure",
						Parts: []LinePart{
							{Text: "    "},
							{Text: "type: MemoryPressure", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 8, Text: "⋮ 8 unchanged lines omitted"},
					{Kind: ExcerptLineKindContext, OldLineNum: 27, NewLineNum: 27, Text: "  - readyHeader: node-ready"},
					{Kind: ExcerptLineKindDeleted, OldLineNum: 28, Text: "    reason: KubeletReady"},
					{Kind: ExcerptLineKindDeleted, OldLineNum: 29, Text: `    status: "True"`},
					{Kind: ExcerptLineKindAdded, NewLineNum: 28, Text: "    reason: KubeletNotReady"},
					{Kind: ExcerptLineKindAdded, NewLineNum: 29, Text: `    status: "False"`},
					{
						Kind:       ExcerptLineKindContext,
						OldLineNum: 30,
						NewLineNum: 30,
						Text:       "    type: Ready",
						Parts: []LinePart{
							{Text: "    "},
							{Text: "type: Ready", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 diff lines omitted"},
				},
			},
		},
		{
			name:    "combines unchanged spec field and changed status field in a single diff card",
			oldText: oldNodeManifest,
			newText: newNodeManifest,
			opts: DiffSliceOptions{
				Paths: []string{
					"$.spec.podCIDR",
					"$.status.conditions[?(@.type=='Ready')].reason",
				},
			},
			want: &DiffSliceResult{
				Changes:        workbench.LineChangeCount{Added: 4, Deleted: 4},
				TotalDiffLines: 23,
				Lines: []ExcerptLine{
					{
						Kind:       ExcerptLineKindContext,
						OldLineNum: 6,
						NewLineNum: 6,
						Text:       "  podCIDR: 10.0.0.0/24",
						Parts: []LinePart{
							{Text: "  "},
							{Text: "podCIDR: 10.0.0.0/24", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 7, Text: "⋮ 7 unchanged lines omitted"},
					{Kind: ExcerptLineKindOmitted, Omitted: 8, Text: "⋮ 8 diff lines omitted"},
					{Kind: ExcerptLineKindOmitted, Omitted: 7, Text: "⋮ 7 unchanged lines omitted"},
					{
						Kind:       ExcerptLineKindDeleted,
						OldLineNum: 28,
						Text:       "    reason: KubeletReady",
						Parts: []LinePart{
							{Text: "    "},
							{Text: "reason: KubeletReady", Matched: true},
						},
					},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 28,
						Text:       "    reason: KubeletNotReady",
						Parts: []LinePart{
							{Text: "    "},
							{Text: "reason: KubeletNotReady", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 10, Text: "⋮ 10 diff lines omitted"},
				},
			},
		},
		{
			name: "keeps paired deletion and addition adjacent inside a multi-line replacement block",
			oldText: `spec:
  minReplicas: 1
  replicas: 2
  maxReplicas: 5
`,
			newText: `spec:
  minReplicas: 2
  replicas: 4
  maxReplicas: 10
`,
			opts: DiffSliceOptions{
				Paths: []string{"$.spec.replicas"},
			},
			want: &DiffSliceResult{
				Changes:        workbench.LineChangeCount{Added: 3, Deleted: 3},
				TotalDiffLines: 7,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 3, Text: "⋮ 3 diff lines omitted"},
					{
						Kind:       ExcerptLineKindDeleted,
						OldLineNum: 3,
						Text:       "  replicas: 2",
						Parts: []LinePart{
							{Text: "  "},
							{Text: "replicas: 2", Matched: true},
						},
					},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 3,
						Text:       "  replicas: 4",
						Parts: []LinePart{
							{Text: "  "},
							{Text: "replicas: 4", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 2, Text: "⋮ 2 diff lines omitted"},
				},
			},
		},
		{
			name:              "initial revision slices multiple JSONPath subtrees across full added manifest",
			oldText:           "ignored",
			newText:           initialManifest,
			isInitialRevision: true,
			opts: DiffSliceOptions{
				Paths: []string{
					"$.subsets[0].addresses[0].ready",
					"$.subsets[0].targetRef",
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
				Paths: []string{"$.a", "$.s"},
			},
			want: &DiffSliceResult{
				Changes:        workbench.LineChangeCount{Added: 3, Deleted: 3},
				TotalDiffLines: 18,
				Lines: []ExcerptLine{
					{
						Kind:       ExcerptLineKindDeleted,
						OldLineNum: 1,
						Text:       "a: 1",
						Parts:      []LinePart{{Text: "a: 1", Matched: true}},
					},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 1,
						Text:       "a: 1_mod",
						Parts:      []LinePart{{Text: "a: 1_mod", Matched: true}},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 unchanged lines omitted"},
					{Kind: ExcerptLineKindOmitted, Omitted: 8, Text: "⋮ 8 diff lines omitted"},
					{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 unchanged lines omitted"},
					{
						Kind:       ExcerptLineKindDeleted,
						OldLineNum: 19,
						Text:       "s: 19",
						Parts:      []LinePart{{Text: "s: 19", Matched: true}},
					},
					{
						Kind:       ExcerptLineKindAdded,
						NewLineNum: 19,
						Text:       "s: 19_mod",
						Parts:      []LinePart{{Text: "s: 19_mod", Matched: true}},
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
status:
  phase: Pending
`
	newYAML := `apiVersion: v1
kind: Pod
metadata:
  name: untouched-pod
status:
  phase: Running
`

	t.Run("rejects omitted paths instead of falling back to implicit leading lines", func(t *testing.T) {
		_, err := SliceUnifiedDiff(oldYAML, newYAML, false, DiffSliceOptions{})
		var gotErr *InvalidJSONPathError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceUnifiedDiff() err = %T (%v), want *InvalidJSONPathError", err, err)
		}
		want := &InvalidJSONPathError{
			PathIndex: -1,
			Reason:    "paths must contain at least one JSONPath selector",
		}
		if diff := cmp.Diff(want, gotErr); diff != "" {
			t.Errorf("InvalidJSONPathError mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("rejects invalid JSONPath syntax", func(t *testing.T) {
		_, err := SliceUnifiedDiff(oldYAML, newYAML, false, DiffSliceOptions{
			Paths: []string{"$.status["},
		})
		var gotErr *InvalidJSONPathError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceUnifiedDiff() err = %T (%v), want *InvalidJSONPathError", err, err)
		}
		if gotErr.PathIndex != 0 {
			t.Errorf("PathIndex = %d, want 0", gotErr.PathIndex)
		}
	})

	t.Run("returns SliceQueryNotFoundError with deepest matched prefix keys and diff preview", func(t *testing.T) {
		_, err := SliceUnifiedDiff(oldYAML, newYAML, false, DiffSliceOptions{
			Paths: []string{"$.status.podIP"},
		})
		var gotErr *SliceQueryNotFoundError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceUnifiedDiff() err = %T (%v), want *SliceQueryNotFoundError", err, err)
		}
		want := &SliceQueryNotFoundError{
			WindowIndex:   0,
			Path:          "$.status.podIP",
			MatchedPrefix: "$.status",
			AvailableKeys: []string{"phase"},
			TopLevelKeys:  []string{"status"},
			Preview: `-  phase: Pending
+  phase: Running`,
		}
		if diff := cmp.Diff(want, gotErr); diff != "" {
			t.Errorf("SliceQueryNotFoundError mismatch (-want +got):\n%s", diff)
		}
	})
}
