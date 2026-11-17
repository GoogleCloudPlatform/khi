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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEvalCompiledJSONPath(t *testing.T) {
	const k8sYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    autoscaling.alpha.kubernetes.io/conditions: '[{"type":"AbleToScale"}]'
    deployment.kubernetes.io/revision: "4"
  name: checkout-web
  namespace: prod
spec:
  minReplicas: 2
  replicas: 4
  maxReplicas: 10
  configBlock: |
    listen: :8080
    workers: 4
  template:
    metadata:
      name: checkout-pod
  containers:
  -
    name: envoy
    restartCount: 0
  - name: app
    restartCount: 3
    lastState:
      terminated:
        exitCode: 137
status:
  conditions:
  - status: "True"
    type: Available
  - reason: MinimumReplicasUnavailable
    status: "False"
    type: Ready
  items:
  - first
  - second
`

	doc, err := parseIndexedYAMLDocument(k8sYAML)
	if err != nil {
		t.Fatalf("parseIndexedYAMLDocument() unexpected error = %v", err)
	}

	testCases := []struct {
		name      string
		path      string
		wantLines [][2]int
	}{
		{
			name:      "standard dollar dot scalar path does not match nested template metadata name",
			path:      "$.metadata.name",
			wantLines: [][2]int{{7, 7}},
		},
		{
			name:      "omitted dollar prefix",
			path:      "spec.replicas",
			wantLines: [][2]int{{11, 11}},
		},
		{
			name:      "leading dot without dollar",
			path:      ".spec.containers[1].name",
			wantLines: [][2]int{{23, 23}},
		},
		{
			name:      "bracket quoted key containing dots and slash",
			path:      "$.metadata.annotations['autoscaling.alpha.kubernetes.io/conditions']",
			wantLines: [][2]int{{5, 5}},
		},
		{
			name:      "escaped dot in bare key",
			path:      `$.metadata.annotations.deployment\.kubernetes\.io/revision`,
			wantLines: [][2]int{{6, 6}},
		},
		{
			name:      "quoted key union in brackets",
			path:      "$.spec['minReplicas', \"maxReplicas\"]",
			wantLines: [][2]int{{10, 10}, {12, 12}},
		},
		{
			name:      "multiline literal block scalar captures full block",
			path:      "$.spec.configBlock",
			wantLines: [][2]int{{13, 15}},
		},
		{
			name:      "sequence item with standalone dash line",
			path:      "$.spec.containers[0]",
			wantLines: [][2]int{{20, 22}},
		},
		{
			name:      "negative array index selects last condition",
			path:      "$.status.conditions[-1]",
			wantLines: [][2]int{{32, 34}},
		},
		{
			name:      "out of bounds array index returns empty",
			path:      "$.status.conditions[2]",
			wantLines: [][2]int{},
		},
		{
			name:      "array index on mapping returns empty",
			path:      "$.spec[0]",
			wantLines: [][2]int{},
		},
		{
			name:      "array slice [0:1]",
			path:      "$.status.conditions[0:1]",
			wantLines: [][2]int{{30, 31}},
		},
		{
			name:      "array slice [:0] returns empty",
			path:      "$.status.conditions[:0]",
			wantLines: [][2]int{},
		},
		{
			name:      "array slice [-1:] selects last element",
			path:      "$.status.conditions[-1:]",
			wantLines: [][2]int{{32, 34}},
		},
		{
			name:      "array slice on mapping returns empty",
			path:      "$.spec[0:1]",
			wantLines: [][2]int{},
		},
		{
			name:      "wildcard on mapping selects all keys in order",
			path:      "$.spec.template.metadata.*",
			wantLines: [][2]int{{18, 18}},
		},
		{
			name:      "wildcard on sequence selects all scalar items",
			path:      "$.status.items[*]",
			wantLines: [][2]int{{36, 36}, {37, 37}},
		},
		{
			name:      "wildcard on scalar returns empty",
			path:      "$.spec.replicas.*",
			wantLines: [][2]int{},
		},
		{
			name:      "compound filter with &&",
			path:      "$.status.conditions[?(@.type=='Ready' && @.status=='False')]",
			wantLines: [][2]int{{32, 34}},
		},
		{
			name:      "filter on mapping values",
			path:      "$.spec[?(@ == 4)]",
			wantLines: [][2]int{{11, 11}},
		},
		{
			name:      "filter on scalar returns empty",
			path:      "$.spec.replicas[?(@ == 4)]",
			wantLines: [][2]int{},
		},
		{
			name:      "numeric filter and nested existence filter",
			path:      "$.spec.containers[?(@.restartCount > 0 && @.lastState.terminated)].lastState.terminated",
			wantLines: [][2]int{{26, 27}},
		},
		{
			name:      "recursive descent finds nested exitCode",
			path:      "$..exitCode",
			wantLines: [][2]int{{27, 27}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compiled, err := compileJSONPaths([]string{tc.path})
			if err != nil {
				t.Fatalf("compileJSONPaths(%q) unexpected error = %v", tc.path, err)
			}
			sels := evalCompiledJSONPath(doc, compiled[0])
			got := make([][2]int, len(sels))
			for i, s := range sels {
				got[i] = [2]int{s.startLine, s.endLine}
			}
			if diff := cmp.Diff(tc.wantLines, got); diff != "" {
				t.Errorf("evalCompiledJSONPath(%q) line ranges mismatch (-want +got):\n%s", tc.path, diff)
			}
		})
	}
}

func TestParseIndexedYAMLDocument_EdgeCases(t *testing.T) {
	testCases := []struct {
		name      string
		yaml      string
		path      string
		wantLines [][2]int
	}{
		{
			name:      "comment only YAML has nil root and returns empty selections",
			yaml:      "# comment only\n",
			path:      "$.spec",
			wantLines: [][2]int{},
		},
		{
			name: "strips trailing blank lines and document end marker",
			yaml: `spec:
  name: web-1

...
`,
			path:      "$.spec",
			wantLines: [][2]int{{1, 2}},
		},
		{
			name: "resolves YAML anchor alias node",
			yaml: `defaults: &base
  replicas: 3
  cpu: 100m
service: *base
`,
			path:      "$.service.replicas",
			wantLines: [][2]int{{2, 2}},
		},
		{
			name: "top-level sequence keeps empty dash item separate from following item",
			yaml: `-
- a: 1
`,
			path:      "$[1]",
			wantLines: [][2]int{{2, 2}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := parseIndexedYAMLDocument(tc.yaml)
			if err != nil {
				t.Fatalf("parseIndexedYAMLDocument() unexpected error = %v", err)
			}
			compiled, err := compileJSONPaths([]string{tc.path})
			if err != nil {
				t.Fatalf("compileJSONPaths(%q) error = %v", tc.path, err)
			}
			sels := evalCompiledJSONPath(doc, compiled[0])
			got := make([][2]int, len(sels))
			for i, s := range sels {
				got[i] = [2]int{s.startLine, s.endLine}
			}
			if diff := cmp.Diff(tc.wantLines, got); diff != "" {
				t.Errorf("evalCompiledJSONPath(%q) line ranges mismatch (-want +got):\n%s", tc.path, diff)
			}
		})
	}
}

func TestTrimmedContentSpanOnLine(t *testing.T) {
	lines := []string{
		"  key: val",
		"   ",
		"  - item",
	}

	type spanResult struct {
		Line int
		Span byteSpan
	}

	testCases := []struct {
		name    string
		lineNum int
		want    spanResult
	}{
		{name: "below range returns zero", lineNum: 0},
		{name: "above range returns zero", lineNum: 4},
		{name: "blank line returns zero", lineNum: 2},
		{name: "normal indented line", lineNum: 1, want: spanResult{Line: 1, Span: byteSpan{start: 2, end: 10}}},
		{name: "sequence item strips dash prefix", lineNum: 3, want: spanResult{Line: 3, Span: byteSpan{start: 4, end: 8}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotLine, gotSpan := trimmedContentSpanOnLine(lines, tc.lineNum)
			got := spanResult{Line: gotLine, Span: gotSpan}
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(byteSpan{})); diff != "" {
				t.Errorf("trimmedContentSpanOnLine(%d) mismatch (-want +got):\n%s", tc.lineNum, diff)
			}
		})
	}
}

func TestDiagnoseUnmatchedJSONPath(t *testing.T) {
	doc, err := parseIndexedYAMLDocument(`apiVersion: v1
kind: Pod
status:
  conditions:
  - type: Ready
    status: "True"
  phase: Running
`)
	if err != nil {
		t.Fatalf("parseIndexedYAMLDocument() error = %v", err)
	}
	emptyDoc, err := parseIndexedYAMLDocument("# comment only\n")
	if err != nil {
		t.Fatalf("parseIndexedYAMLDocument(emptyDoc) error = %v", err)
	}

	type diagnoseOutcome struct {
		Prefix string
		Keys   []string
	}

	testCases := []struct {
		name string
		docs []*indexedYAMLDocument
		path string
		want diagnoseOutcome
	}{
		{
			name: "empty root documents return empty prefix and keys",
			docs: []*indexedYAMLDocument{emptyDoc},
			path: "$.status.phase",
			want: diagnoseOutcome{},
		},
		{
			name: "unmatched top-level key returns empty prefix and root keys",
			docs: []*indexedYAMLDocument{doc},
			path: "$.spec.replicas",
			want: diagnoseOutcome{
				Prefix: "",
				Keys:   []string{"apiVersion", "kind", "status"},
			},
		},
		{
			name: "unmatched nested key returns deepest matched prefix and child keys",
			docs: []*indexedYAMLDocument{doc},
			path: "$.status.currentMetrics",
			want: diagnoseOutcome{
				Prefix: "$.status",
				Keys:   []string{"conditions", "phase"},
			},
		},
		{
			name: "unmatched key after bare root step records dotted prefix",
			docs: []*indexedYAMLDocument{doc},
			path: "status.currentMetrics",
			want: diagnoseOutcome{
				Prefix: "$.status",
				Keys:   []string{"conditions", "phase"},
			},
		},
		{
			name: "unmatched key after bracket filter preserves bracket prefix",
			docs: []*indexedYAMLDocument{doc},
			path: "$.status.conditions[?(@.type=='Ready')].reason",
			want: diagnoseOutcome{
				Prefix: "$.status.conditions[?(@.type=='Ready')]",
				Keys:   []string{"status", "type"},
			},
		},
		{
			name: "unmatched key beneath sequence surfaces element keys",
			docs: []*indexedYAMLDocument{doc},
			path: "$.status.conditions.reason",
			want: diagnoseOutcome{
				Prefix: "$.status.conditions",
				Keys:   []string{"status", "type"},
			},
		},
		{
			name: "unmatched key after recursive descent step",
			docs: []*indexedYAMLDocument{doc},
			path: "$..conditions.reason",
			want: diagnoseOutcome{
				Prefix: "$..conditions",
				Keys:   []string{"status", "type"},
			},
		},
		{
			name: "unmatched key after wildcard step",
			docs: []*indexedYAMLDocument{doc},
			path: "$.*.missing",
			want: diagnoseOutcome{
				Prefix: "$.*",
				Keys:   []string{"conditions", "phase"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compiled, err := compileJSONPaths([]string{tc.path})
			if err != nil {
				t.Fatalf("compileJSONPaths(%q) error = %v", tc.path, err)
			}
			prefix, keys := diagnoseUnmatchedJSONPath(compiled[0], tc.docs...)
			got := diagnoseOutcome{Prefix: prefix, Keys: keys}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("diagnoseUnmatchedJSONPath(%q) mismatch (-want +got):\n%s", tc.path, diff)
			}
		})
	}
}
