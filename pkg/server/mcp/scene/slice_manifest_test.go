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

func TestSliceManifestBody(t *testing.T) {
	const hpaManifestYAML = `apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: frontend-hpa
  namespace: default
spec:
  maxReplicas: 10
  minReplicas: 2
  metrics:
  - resource:
      name: cpu
      target:
        averageUtilization: 40
        type: Utilization
    type: Resource
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: frontend-web
status:
  currentMetrics:
  - resource:
      current:
        averageUtilization: 85
        averageValue: 850m
      name: cpu
    type: Resource
  currentReplicas: 4
  desiredReplicas: 9
`

	testCases := []struct {
		name string
		yaml string
		opts ManifestSliceOptions
		want *ManifestSliceResult
	}{
		{
			name: "slices disjoint spec and status windows with leading, interior, and trailing omitted bands",
			yaml: hpaManifestYAML,
			opts: ManifestSliceOptions{
				MaxLines: 12,
				Windows: []SliceWindow{
					{Query: "averageUtilization: 40", BeforeLines: 1, AfterLines: 1},
					{Query: "averageUtilization: 85", BeforeLines: 1, AfterLines: 1},
				},
			},
			want: &ManifestSliceResult{
				TotalLines: 29,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 11, Text: "⋮ 11 lines omitted"},
					{Kind: ExcerptLineKindContext, NewLineNum: 12, Text: "      target:"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 13,
						Text:       "        averageUtilization: 40",
						Parts: []LinePart{
							{Text: "        "},
							{Text: "averageUtilization: 40", Matched: true},
						},
					},
					{Kind: ExcerptLineKindContext, NewLineNum: 14, Text: "        type: Utilization"},
					{Kind: ExcerptLineKindOmitted, Omitted: 8, Text: "⋮ 8 lines omitted"},
					{Kind: ExcerptLineKindContext, NewLineNum: 23, Text: "      current:"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 24,
						Text:       "        averageUtilization: 85",
						Parts: []LinePart{
							{Text: "        "},
							{Text: "averageUtilization: 85", Matched: true},
						},
					},
					{Kind: ExcerptLineKindContext, NewLineNum: 25, Text: "        averageValue: 850m"},
					{Kind: ExcerptLineKindOmitted, Omitted: 4, Text: "⋮ 4 lines omitted"},
				},
			},
		},
		{
			name: "merges overlapping context intervals cleanly without duplicate lines",
			yaml: hpaManifestYAML,
			opts: ManifestSliceOptions{
				Windows: []SliceWindow{
					{Query: "maxReplicas: 10", BeforeLines: 0, AfterLines: 1},
					{Query: "minReplicas: 2", BeforeLines: 1, AfterLines: 0},
				},
			},
			want: &ManifestSliceResult{
				TotalLines: 29,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 6, Text: "⋮ 6 lines omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 7,
						Text:       "  maxReplicas: 10",
						Parts: []LinePart{
							{Text: "  "},
							{Text: "maxReplicas: 10", Matched: true},
						},
					},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 8,
						Text:       "  minReplicas: 2",
						Parts: []LinePart{
							{Text: "  "},
							{Text: "minReplicas: 2", Matched: true},
						},
					},
					{Kind: ExcerptLineKindOmitted, Omitted: 21, Text: "⋮ 21 lines omitted"},
				},
			},
		},
		{
			name: "omitted windows returns leading lines up to maxLines with trailing omitted band",
			yaml: hpaManifestYAML,
			opts: ManifestSliceOptions{
				MaxLines: 4,
			},
			want: &ManifestSliceResult{
				TotalLines: 29,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindContext, NewLineNum: 1, Text: "apiVersion: autoscaling/v2"},
					{Kind: ExcerptLineKindContext, NewLineNum: 2, Text: "kind: HorizontalPodAutoscaler"},
					{Kind: ExcerptLineKindContext, NewLineNum: 3, Text: "metadata:"},
					{Kind: ExcerptLineKindContext, NewLineNum: 4, Text: "  name: frontend-hpa"},
					{Kind: ExcerptLineKindOmitted, Omitted: 25, Text: "⋮ 25 lines omitted"},
				},
			},
		},
		{
			name: "empty manifest with no windows returns empty result",
			yaml: "",
			opts: ManifestSliceOptions{},
			want: &ManifestSliceResult{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SliceManifestBody(tc.yaml, tc.opts)
			if err != nil {
				t.Fatalf("SliceManifestBody() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SliceManifestBody() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSliceManifestBody_Errors(t *testing.T) {
	const podYAML = `apiVersion: v1
kind: Pod
metadata:
  name: web-1
  namespace: default
spec:
  nodeName: gke-node-1
status:
  phase: Running
`

	t.Run("rejects empty window query", func(t *testing.T) {
		_, err := SliceManifestBody(podYAML, ManifestSliceOptions{
			Windows: []SliceWindow{{Query: ""}},
		})
		var gotErr *InvalidSliceWindowError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceManifestBody() err = %T (%v), want *InvalidSliceWindowError", err, err)
		}
		want := &InvalidSliceWindowError{WindowIndex: 0, Reason: "query must not be empty"}
		if diff := cmp.Diff(want, gotErr); diff != "" {
			t.Errorf("InvalidSliceWindowError mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("returns SliceQueryNotFoundError with top-level keys and YAML preview", func(t *testing.T) {
		_, err := SliceManifestBody(podYAML, ManifestSliceOptions{
			Windows: []SliceWindow{{Query: "nonExistentField"}},
		})
		var gotErr *SliceQueryNotFoundError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceManifestBody() err = %T (%v), want *SliceQueryNotFoundError", err, err)
		}
		want := &SliceQueryNotFoundError{
			WindowIndex:  0,
			Query:        "nonExistentField",
			TopLevelKeys: []string{"apiVersion", "kind", "metadata", "spec", "status"},
			Preview: `apiVersion: v1
kind: Pod
metadata:
  name: web-1
  namespace: default
spec:`,
		}
		if diff := cmp.Diff(want, gotErr); diff != "" {
			t.Errorf("SliceQueryNotFoundError mismatch (-want +got):\n%s", diff)
		}
	})
}
