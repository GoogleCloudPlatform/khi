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

package coretask

import (
	"strings"
	"testing"
)

func TestVerifyAcyclic(t *testing.T) {
	testCases := []struct {
		name            string
		outgoingAdjList map[string][]string
		inDegree        map[string]int
		nodeCount       int
		wantErr         bool
		wantSubstring   string
	}{
		{
			name:            "empty graph is acyclic",
			outgoingAdjList: map[string][]string{},
			inDegree:        map[string]int{},
			nodeCount:       0,
			wantErr:         false,
		},
		{
			name: "single node without edges is acyclic",
			outgoingAdjList: map[string][]string{
				"A": nil,
			},
			inDegree: map[string]int{
				"A": 0,
			},
			nodeCount: 1,
			wantErr:   false,
		},
		{
			name: "linear DAG A -> B -> C is acyclic",
			outgoingAdjList: map[string][]string{
				"A": {"B"},
				"B": {"C"},
				"C": nil,
			},
			inDegree: map[string]int{
				"A": 0,
				"B": 1,
				"C": 1,
			},
			nodeCount: 3,
			wantErr:   false,
		},
		{
			name: "diamond DAG is acyclic",
			outgoingAdjList: map[string][]string{
				"A": {"B", "C"},
				"B": {"D"},
				"C": {"D"},
				"D": nil,
			},
			inDegree: map[string]int{
				"A": 0,
				"B": 1,
				"C": 1,
				"D": 2,
			},
			nodeCount: 4,
			wantErr:   false,
		},
		{
			name: "self-loop cycle A -> A fails",
			outgoingAdjList: map[string][]string{
				"A": {"A"},
			},
			inDegree: map[string]int{
				"A": 1,
			},
			nodeCount:     1,
			wantErr:       true,
			wantSubstring: "cyclic dependency",
		},
		{
			name: "2-node cycle A -> B -> A fails",
			outgoingAdjList: map[string][]string{
				"A": {"B"},
				"B": {"A"},
			},
			inDegree: map[string]int{
				"A": 1,
				"B": 1,
			},
			nodeCount:     2,
			wantErr:       true,
			wantSubstring: "cyclic dependency",
		},
		{
			name: "cycle with upstream entry S -> A -> B -> A fails",
			outgoingAdjList: map[string][]string{
				"S": {"A"},
				"A": {"B"},
				"B": {"A"},
			},
			inDegree: map[string]int{
				"S": 0,
				"A": 2,
				"B": 1,
			},
			nodeCount:     3,
			wantErr:       true,
			wantSubstring: "cyclic dependency",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyAcyclic(tc.outgoingAdjList, tc.inDegree, tc.nodeCount)
			if (err != nil) != tc.wantErr {
				t.Fatalf("verifyAcyclic() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr && tc.wantSubstring != "" {
				if !strings.Contains(err.Error(), tc.wantSubstring) {
					t.Errorf("verifyAcyclic() error %q does not contain %q", err.Error(), tc.wantSubstring)
				}
			}
		})
	}
}

func TestIsReachable(t *testing.T) {
	outgoingAdjList := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"E"},
		"D": {"F"},
		"E": nil,
		"F": nil,
		"X": {"Y"},
		"Y": nil,
	}

	testCases := []struct {
		name       string
		startNode  string
		targetNode string
		want       bool
	}{
		{
			name:       "direct reachability A -> B",
			startNode:  "A",
			targetNode: "B",
			want:       true,
		},
		{
			name:       "transitive reachability A -> B -> D -> F",
			startNode:  "A",
			targetNode: "F",
			want:       true,
		},
		{
			name:       "transitive reachability A -> C -> E",
			startNode:  "A",
			targetNode: "E",
			want:       true,
		},
		{
			name:       "reverse direction is not reachable B -> A",
			startNode:  "B",
			targetNode: "A",
			want:       false,
		},
		{
			name:       "disjoint component is not reachable A -> X",
			startNode:  "A",
			targetNode: "X",
			want:       false,
		},
		{
			name:       "same node without self edge is not reachable",
			startNode:  "A",
			targetNode: "A",
			want:       false,
		},
		{
			name:       "node with no outgoing edges cannot reach anything",
			startNode:  "F",
			targetNode: "A",
			want:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isReachable(tc.startNode, tc.targetNode, outgoingAdjList)
			if got != tc.want {
				t.Errorf("isReachable(%q, %q) = %v, want %v", tc.startNode, tc.targetNode, got, tc.want)
			}
		})
	}
}

func TestExtractCyclicDependencyPath(t *testing.T) {
	testCases := []struct {
		name            string
		outgoingAdjList map[string][]string
		inDegree        map[string]int
		wantSubstring   string
	}{
		{
			name: "simple cycle A -> B -> A",
			outgoingAdjList: map[string][]string{
				"A": {"B"},
				"B": {"A"},
			},
			inDegree: map[string]int{
				"A": 1,
				"B": 1,
			},
			wantSubstring: "A -> B",
		},
		{
			name:            "no unresolved nodes returns empty string",
			outgoingAdjList: map[string][]string{},
			inDegree: map[string]int{
				"A": 0,
			},
			wantSubstring: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractCyclicDependencyPath(tc.outgoingAdjList, tc.inDegree)
			if tc.wantSubstring == "" {
				if got != "" {
					t.Errorf("extractCyclicDependencyPath() = %q, want empty string", got)
				}
			} else if !strings.Contains(got, tc.wantSubstring) {
				t.Errorf("extractCyclicDependencyPath() = %q, want containing %q", got, tc.wantSubstring)
			}
		})
	}
}
