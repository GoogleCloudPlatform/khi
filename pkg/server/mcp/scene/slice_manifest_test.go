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
			name: "slices disjoint spec and status subtrees via JSONPath with leading, interior, and trailing omitted bands",
			yaml: hpaManifestYAML,
			opts: ManifestSliceOptions{
				MaxLines: 12,
				Paths: []string{
					"$.spec.metrics[?(@.resource.name=='cpu')].resource.target",
					"$.status.currentMetrics[?(@.resource.name=='cpu')].resource.current",
				},
			},
			want: &ManifestSliceResult{
				TotalLines: 29,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 11, Text: "⋮ 11 lines omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 12,
						Text:       "      target:",
						Parts: []LinePart{
							{Text: "      "},
							{Text: "target:", Matched: true},
						},
					},
					{Kind: ExcerptLineKindContext, NewLineNum: 13, Text: "        averageUtilization: 40"},
					{Kind: ExcerptLineKindContext, NewLineNum: 14, Text: "        type: Utilization"},
					{Kind: ExcerptLineKindOmitted, Omitted: 8, Text: "⋮ 8 lines omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 23,
						Text:       "      current:",
						Parts: []LinePart{
							{Text: "      "},
							{Text: "current:", Matched: true},
						},
					},
					{Kind: ExcerptLineKindContext, NewLineNum: 24, Text: "        averageUtilization: 85"},
					{Kind: ExcerptLineKindContext, NewLineNum: 25, Text: "        averageValue: 850m"},
					{Kind: ExcerptLineKindOmitted, Omitted: 4, Text: "⋮ 4 lines omitted"},
				},
			},
		},
		{
			name: "merges adjacent JSONPath fields cleanly without duplicate lines",
			yaml: hpaManifestYAML,
			opts: ManifestSliceOptions{
				Paths: []string{
					"$.spec.maxReplicas",
					"$.spec.minReplicas",
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
			name: "caps large selected subtree at maxLines with trailing omitted band",
			yaml: hpaManifestYAML,
			opts: ManifestSliceOptions{
				MaxLines: 4,
				Paths:    []string{"$.spec"},
			},
			want: &ManifestSliceResult{
				TotalLines: 29,
				Lines: []ExcerptLine{
					{Kind: ExcerptLineKindOmitted, Omitted: 5, Text: "⋮ 5 lines omitted"},
					{
						Kind:       ExcerptLineKindContext,
						NewLineNum: 6,
						Text:       "spec:",
						Parts:      []LinePart{{Text: "spec:", Matched: true}},
					},
					{Kind: ExcerptLineKindContext, NewLineNum: 7, Text: "  maxReplicas: 10"},
					{Kind: ExcerptLineKindContext, NewLineNum: 8, Text: "  minReplicas: 2"},
					{Kind: ExcerptLineKindContext, NewLineNum: 9, Text: "  metrics:"},
					{Kind: ExcerptLineKindOmitted, Omitted: 20, Text: "⋮ 20 lines omitted"},
				},
			},
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

	t.Run("rejects omitted paths instead of returning implicit leading lines", func(t *testing.T) {
		_, err := SliceManifestBody(podYAML, ManifestSliceOptions{})
		var gotErr *InvalidJSONPathError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceManifestBody() err = %T (%v), want *InvalidJSONPathError", err, err)
		}
		want := &InvalidJSONPathError{
			PathIndex: -1,
			Reason:    "paths must contain at least one JSONPath selector",
		}
		if diff := cmp.Diff(want, gotErr); diff != "" {
			t.Errorf("InvalidJSONPathError mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("rejects empty manifest when path cannot match", func(t *testing.T) {
		_, err := SliceManifestBody("", ManifestSliceOptions{
			Paths: []string{"$.metadata.name"},
		})
		var gotErr *SliceQueryNotFoundError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceManifestBody() err = %T (%v), want *SliceQueryNotFoundError", err, err)
		}
		if gotErr.Path != "$.metadata.name" {
			t.Errorf("gotErr.Path = %q, want \"$.metadata.name\"", gotErr.Path)
		}
	})

	t.Run("returns SliceQueryNotFoundError with deepest matched prefix keys and YAML preview", func(t *testing.T) {
		_, err := SliceManifestBody(podYAML, ManifestSliceOptions{
			Paths: []string{"$.status.nonExistentField"},
		})
		var gotErr *SliceQueryNotFoundError
		if !errors.As(err, &gotErr) {
			t.Fatalf("SliceManifestBody() err = %T (%v), want *SliceQueryNotFoundError", err, err)
		}
		want := &SliceQueryNotFoundError{
			WindowIndex:   0,
			Path:          "$.status.nonExistentField",
			MatchedPrefix: "$.status",
			AvailableKeys: []string{"phase"},
			TopLevelKeys:  []string{"apiVersion", "kind", "metadata", "spec", "status"},
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
