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
			name:      "standard dollar dot scalar path",
			path:      "$.spec.replicas",
			wantLines: [][2]int{{11, 11}},
		},
		{
			name:      "omitted dollar prefix",
			path:      "spec.replicas",
			wantLines: [][2]int{{11, 11}},
		},
		{
			name:      "leading dot without dollar",
			path:      ".metadata.name",
			wantLines: [][2]int{{7, 7}},
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
			wantLines: [][2]int{{17, 19}},
		},
		{
			name:      "negative array index selects last condition",
			path:      "$.status.conditions[-1]",
			wantLines: [][2]int{{29, 31}},
		},
		{
			name:      "array slice [0:1]",
			path:      "$.status.conditions[0:1]",
			wantLines: [][2]int{{27, 28}},
		},
		{
			name:      "compound filter with &&",
			path:      "$.status.conditions[?(@.type=='Ready' && @.status=='False')]",
			wantLines: [][2]int{{29, 31}},
		},
		{
			name:      "numeric filter and nested existence filter",
			path:      "$.spec.containers[?(@.restartCount > 0 && @.lastState.terminated)].lastState.terminated",
			wantLines: [][2]int{{23, 24}},
		},
		{
			name:      "recursive descent finds all status keys",
			path:      "$..exitCode",
			wantLines: [][2]int{{24, 24}},
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

func TestCompileJSONPaths_Errors(t *testing.T) {
	testCases := []struct {
		name      string
		paths     []string
		wantIndex int
	}{
		{name: "nil paths slice", paths: nil, wantIndex: -1},
		{name: "empty paths slice", paths: []string{}, wantIndex: -1},
		{name: "empty path string", paths: []string{""}, wantIndex: 0},
		{name: "root dollar only", paths: []string{"$"}, wantIndex: 0},
		{name: "trailing dot", paths: []string{"$.spec."}, wantIndex: 0},
		{name: "trailing recursive descent", paths: []string{"$.spec.."}, wantIndex: 0},
		{name: "dot before bracket", paths: []string{"$.spec.[0]"}, wantIndex: 0},
		{name: "unclosed bracket", paths: []string{"$.spec[0"}, wantIndex: 0},
		{name: "single equals in filter", paths: []string{"$.items[?(@.type='Ready')]"}, wantIndex: 0},
		{name: "unquoted string literal in filter", paths: []string{"$.items[?(@.type==Ready)]"}, wantIndex: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := compileJSONPaths(tc.paths)
			var gotErr *InvalidJSONPathError
			if !errors.As(err, &gotErr) {
				t.Fatalf("compileJSONPaths(%v) err = %T (%v), want *InvalidJSONPathError", tc.paths, err, err)
			}
			if gotErr.PathIndex != tc.wantIndex {
				t.Errorf("PathIndex = %d, want %d (err=%v)", gotErr.PathIndex, tc.wantIndex, gotErr)
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

	compiled, err := compileJSONPaths([]string{"$.status.currentMetrics"})
	if err != nil {
		t.Fatalf("compileJSONPaths() error = %v", err)
	}
	prefix, keys := diagnoseUnmatchedJSONPath(compiled[0], doc)
	if prefix != "$.status" {
		t.Errorf("matchedPrefix = %q, want \"$.status\"", prefix)
	}
	if diff := cmp.Diff([]string{"conditions", "phase"}, keys); diff != "" {
		t.Errorf("availableKeys mismatch (-want +got):\n%s", diff)
	}
}
