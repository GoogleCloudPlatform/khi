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
	"context"
	"errors"
	"testing"
	"time"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func setupSearchTimelinesTestWorkbench() *Workbench {
	logs := []cel.LogData{
		{ID: 1, LogTypeID: 1, SeverityTypeID: 1, Timestamp: 1000},
		{ID: 2, LogTypeID: 1, SeverityTypeID: 2, Timestamp: 2000},
		{ID: 3, LogTypeID: 1, SeverityTypeID: 2, Timestamp: 2500},
		{ID: 4, LogTypeID: 1, SeverityTypeID: 1, Timestamp: 3000},
		{ID: 5, LogTypeID: 1, SeverityTypeID: 3, Timestamp: 4000},
		{ID: 6, LogTypeID: 1, SeverityTypeID: 3, Timestamp: 5000},
	}
	timelines := []*cel.TimelineData{
		{
			ID:           1,
			ParentID:     0,
			ChildrenIDs:  []uint32{2, 3},
			Name:         "ns-1",
			TimelineType: "Namespace",
		},
		{
			ID:           2,
			ParentID:     1,
			ChildrenIDs:  []uint32{4},
			Name:         "pod-a",
			TimelineType: "Pod",
			Events: []cel.EventInfo{
				{LogID: 1, Timestamp: 1000},
				{LogID: 2, Timestamp: 2000},
			},
			Revisions: []cel.RevisionInfo{
				{LogID: 3, ChangedTime: 2500},
			},
		},
		{
			ID:           3,
			ParentID:     1,
			ChildrenIDs:  []uint32{5},
			Name:         "pod-b",
			TimelineType: "Pod",
			Events: []cel.EventInfo{
				{LogID: 4, Timestamp: 3000},
			},
		},
		{
			ID:           4,
			ParentID:     2,
			Name:         "container-a1",
			TimelineType: "Container",
			Events: []cel.EventInfo{
				{LogID: 5, Timestamp: 4000},
			},
			// The revision has no linked log, so it adds no severity count.
			Revisions: []cel.RevisionInfo{
				{LogID: 0, ChangedTime: 4500},
			},
		},
		{
			ID:           5,
			ParentID:     3,
			Name:         "container-b1",
			TimelineType: "Container",
			// The event and the revision link to the same log, so its severity is counted once.
			Events: []cel.EventInfo{
				{LogID: 6, Timestamp: 5000},
			},
			Revisions: []cel.RevisionInfo{
				{LogID: 6, ChangedTime: 5000},
			},
		},
	}
	severities := []*khifilev6.Severity{testSeverityInfo, testSeverityWarning, testSeverityError}
	wb := newTestIndexedWorkbench(logs, timelines, severities)
	wb.styleChunk.TimelineTypes = []*khifilev6.TimelineType{
		{Label: proto.String("Namespace"), Description: proto.String("Kubernetes namespace")},
		{Label: proto.String("Pod"), Description: proto.String("Kubernetes pod")},
		{Label: proto.String("Container"), Description: proto.String("Container inside pod")},
	}
	wb.styles = wb.buildStyleMaps()
	return wb
}

func TestSearchTimelines(t *testing.T) {
	allTimelineTypes := []TimelineTypeDescription{
		{Type: "Namespace", Description: "Kubernetes namespace"},
		{Type: "Pod", Description: "Kubernetes pod"},
		{Type: "Container", Description: "Container inside pod"},
	}
	truncatedTimelineTypes := []TimelineTypeDescription{
		{Type: "Namespace", Description: "Kubernetes namespace"},
		{Type: "Pod", Description: "Kubernetes pod"},
	}

	nsNode := TimelineTreeNode{
		ID:   1,
		Type: "Namespace",
		Name: "ns-1",
		SeverityCounts: []SeverityCount{
			{Severity: testSeverityError, Count: 2},
			{Severity: testSeverityWarning, Count: 2},
			{Severity: testSeverityInfo, Count: 2},
		},
	}
	podANode := TimelineTreeNode{
		ID:            2,
		Depth:         1,
		Type:          "Pod",
		Name:          "pod-a",
		EventCount:    2,
		RevisionCount: 1,
		SeverityCounts: []SeverityCount{
			{Severity: testSeverityError, Count: 1},
			{Severity: testSeverityWarning, Count: 2},
			{Severity: testSeverityInfo, Count: 1},
		},
		FirstMatchTime: time.Unix(0, 1000).UTC(),
		LastMatchTime:  time.Unix(0, 2500).UTC(),
	}
	containerA1Node := TimelineTreeNode{
		ID:            4,
		Depth:         2,
		Type:          "Container",
		Name:          "container-a1",
		EventCount:    1,
		RevisionCount: 1,
		SeverityCounts: []SeverityCount{
			{Severity: testSeverityError, Count: 1},
		},
		FirstMatchTime: time.Unix(0, 4000).UTC(),
		LastMatchTime:  time.Unix(0, 4500).UTC(),
	}
	podBNode := TimelineTreeNode{
		ID:         3,
		Depth:      1,
		Type:       "Pod",
		Name:       "pod-b",
		EventCount: 1,
		SeverityCounts: []SeverityCount{
			{Severity: testSeverityError, Count: 1},
			{Severity: testSeverityInfo, Count: 1},
		},
		FirstMatchTime: time.Unix(0, 3000).UTC(),
		LastMatchTime:  time.Unix(0, 3000).UTC(),
	}
	containerB1Node := TimelineTreeNode{
		ID:            5,
		Depth:         2,
		Type:          "Container",
		Name:          "container-b1",
		EventCount:    1,
		RevisionCount: 1,
		SeverityCounts: []SeverityCount{
			{Severity: testSeverityError, Count: 1},
		},
		FirstMatchTime: time.Unix(0, 5000).UTC(),
		LastMatchTime:  time.Unix(0, 5000).UTC(),
	}

	fullTreeNodes := []TimelineTreeNode{nsNode, podANode, containerA1Node, podBNode, containerB1Node}

	truncatedPodA := podANode
	truncatedPodA.OmittedChildren = 1
	truncatedPodB := podBNode
	truncatedPodB.OmittedChildren = 1
	truncatedNodes := []TimelineTreeNode{nsNode, truncatedPodA, truncatedPodB}

	sharedLogPodA := podANode
	sharedLogPodA.EventCount = 4
	sharedLogPodA.SeverityCounts = []SeverityCount{
		{Severity: testSeverityError, Count: 1},
		{Severity: testSeverityWarning, Count: 2},
		{Severity: testSeverityInfo, Count: 2},
	}
	sharedLogPodA.LastMatchTime = time.Unix(0, 4000).UTC()

	testCases := []struct {
		name          string
		filter        Filter
		maxDepth      int
		maxNodes      int
		modify        func(*Workbench)
		wantErr       bool
		wantErrTarget error
		want          *TimelineTreeResult
	}{
		{
			name:     "full tree returned when maxNodes is large enough",
			filter:   Filter{},
			maxDepth: 0,
			maxNodes: 10,
			want: &TimelineTreeResult{
				MatchedTimelineCount: 5,
				DirectMatchCount:     4,
				AncestorNodeCount:    1,
				ReturnedNodeCount:    5,
				MaxDepth:             0,
				MaxNodes:             10,
				TimelineTypes:        allTimelineTypes,
				Nodes:                fullTreeNodes,
			},
		},
		{
			name:     "shared log across parent child and sibling timelines is counted once in ancestor",
			filter:   Filter{},
			maxDepth: 0,
			maxNodes: 10,
			modify: func(wb *Workbench) {
				// Link log 5 (ERROR, already on child container-a1) and log 4 (INFO, already on sibling pod-b) to pod-a.
				tl2 := wb.searchIndex.TimelineMap[2]
				tl2.Events = append(tl2.Events,
					cel.EventInfo{LogID: 4, Timestamp: 3000},
					cel.EventInfo{LogID: 5, Timestamp: 4000},
				)
			},
			want: &TimelineTreeResult{
				MatchedTimelineCount: 5,
				DirectMatchCount:     4,
				AncestorNodeCount:    1,
				ReturnedNodeCount:    5,
				MaxDepth:             0,
				MaxNodes:             10,
				TimelineTypes:        allTimelineTypes,
				Nodes:                []TimelineTreeNode{nsNode, sharedLogPodA, containerA1Node, podBNode, containerB1Node},
			},
		},
		{
			name:     "breadth-first truncation by maxNodes",
			filter:   Filter{},
			maxDepth: 0,
			maxNodes: 3,
			want: &TimelineTreeResult{
				MatchedTimelineCount: 5,
				DirectMatchCount:     4,
				AncestorNodeCount:    1,
				ReturnedNodeCount:    3,
				MaxDepth:             0,
				MaxNodes:             3,
				TimelineTypes:        truncatedTimelineTypes,
				Nodes:                truncatedNodes,
			},
		},
		{
			name:     "truncation by maxDepth",
			filter:   Filter{},
			maxDepth: 1,
			maxNodes: 10,
			want: &TimelineTreeResult{
				MatchedTimelineCount: 5,
				DirectMatchCount:     4,
				AncestorNodeCount:    1,
				ReturnedNodeCount:    3,
				MaxDepth:             1,
				MaxNodes:             10,
				TimelineTypes:        truncatedTimelineTypes,
				Nodes:                truncatedNodes,
			},
		},
		{
			name:     "invalid CEL in filter returns error",
			filter:   Filter{TimelineQuery: "invalid &&& syntax"},
			maxDepth: 0,
			maxNodes: 10,
			wantErr:  true,
		},
		{
			name:     "default maxNodes is used when non-positive",
			filter:   Filter{},
			maxDepth: 0,
			maxNodes: 0,
			want: &TimelineTreeResult{
				MatchedTimelineCount: 5,
				DirectMatchCount:     4,
				AncestorNodeCount:    1,
				ReturnedNodeCount:    5,
				MaxDepth:             0,
				MaxNodes:             DefaultMaxTimelineNodes,
				TimelineTypes:        allTimelineTypes,
				Nodes:                fullTreeNodes,
			},
		},
		{
			name:     "zero matching timelines returns empty result",
			filter:   Filter{TimelineQuery: `name == "non-existent"`},
			maxDepth: 0,
			maxNodes: 10,
			want: &TimelineTreeResult{
				MatchedTimelineCount: 0,
				DirectMatchCount:     0,
				AncestorNodeCount:    0,
				ReturnedNodeCount:    0,
				MaxDepth:             0,
				MaxNodes:             10,
			},
		},
		{
			name:          "closed workbench returns error",
			filter:        Filter{},
			maxDepth:      0,
			maxNodes:      10,
			modify:        func(wb *Workbench) { wb.Close() },
			wantErr:       true,
			wantErrTarget: ErrWorkbenchClosed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupSearchTimelinesTestWorkbench()
			if tc.modify != nil {
				tc.modify(wb)
			}

			got, err := wb.SearchTimelines(context.Background(), tc.filter, tc.maxDepth, tc.maxNodes)
			if (err != nil) != tc.wantErr {
				t.Fatalf("SearchTimelines() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Fatalf("SearchTimelines() error = %v, want %v", err, tc.wantErrTarget)
			}
			if tc.wantErr {
				return
			}

			if diff := cmp.Diff(tc.want, got, protocmp.Transform(), cmpopts.IgnoreFields(TimelineTreeResult{}, "Applied", "FilterVocabulary")); diff != "" {
				t.Errorf("SearchTimelines() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
