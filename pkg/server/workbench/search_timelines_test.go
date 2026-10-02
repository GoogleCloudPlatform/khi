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
	"testing"
	"time"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func setupSearchTimelinesTestWorkbench() *Workbench {
	wb := NewWorkbench("wb-test", "test-inspection")
	wb.searchIndex = &SearchIndex{
		TimelineMap: make(map[uint32]*cel.TimelineData),
	}

	wb.searchIndex.StyleResolver = &cel.SimpleStyleResolver{
		LogTypes: map[uint32]string{1: "k8s-event"},
		Severities: map[uint32]uint32{
			1: 1, // INFO
			2: 2, // WARN
			3: 3, // ERROR
		},
	}

	wb.searchIndex.Logs = []cel.LogData{
		{ID: 1, LogTypeID: 1, SeverityTypeID: 1, Timestamp: 1000},
		{ID: 2, LogTypeID: 1, SeverityTypeID: 2, Timestamp: 2000},
		{ID: 3, LogTypeID: 1, SeverityTypeID: 2, Timestamp: 2500},
		{ID: 4, LogTypeID: 1, SeverityTypeID: 1, Timestamp: 3000},
		{ID: 5, LogTypeID: 1, SeverityTypeID: 3, Timestamp: 4000},
		{ID: 6, LogTypeID: 1, SeverityTypeID: 3, Timestamp: 5000},
	}

	tl1 := &cel.TimelineData{
		ID:           1,
		ParentID:     0,
		ChildrenIDs:  []uint32{2, 3},
		Name:         "ns-1",
		TimelineType: "Namespace",
	}

	tl2 := &cel.TimelineData{
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
	}

	tl3 := &cel.TimelineData{
		ID:           3,
		ParentID:     1,
		ChildrenIDs:  []uint32{5},
		Name:         "pod-b",
		TimelineType: "Pod",
		Events: []cel.EventInfo{
			{LogID: 4, Timestamp: 3000},
		},
	}

	tl4 := &cel.TimelineData{
		ID:           4,
		ParentID:     2,
		ChildrenIDs:  nil,
		Name:         "container-a1",
		TimelineType: "Container",
		Events: []cel.EventInfo{
			{LogID: 5, Timestamp: 4000},
		},
		// The revision has no linked log, so it adds no severity count.
		Revisions: []cel.RevisionInfo{
			{LogID: 0, ChangedTime: 4500},
		},
	}

	tl5 := &cel.TimelineData{
		ID:           5,
		ParentID:     3,
		ChildrenIDs:  nil,
		Name:         "container-b1",
		TimelineType: "Container",
		// The event and the revision link to the same log, so its severity is counted once.
		Events: []cel.EventInfo{
			{LogID: 6, Timestamp: 5000},
		},
		Revisions: []cel.RevisionInfo{
			{LogID: 6, ChangedTime: 5000},
		},
	}

	wb.searchIndex.Timelines = []*cel.TimelineData{tl1, tl2, tl3, tl4, tl5}
	wb.searchIndex.TimelineMap[1] = tl1
	wb.searchIndex.TimelineMap[2] = tl2
	wb.searchIndex.TimelineMap[3] = tl3
	wb.searchIndex.TimelineMap[4] = tl4
	wb.searchIndex.TimelineMap[5] = tl5

	wb.styleChunk = &khifilev6.TimelineStyleChunk{
		Severities: []*khifilev6.Severity{testSeverityInfo, testSeverityWarning, testSeverityError},
		TimelineTypes: []*khifilev6.TimelineType{
			{Label: proto.String("Namespace"), Description: proto.String("Kubernetes namespace")},
			{Label: proto.String("Pod"), Description: proto.String("Kubernetes pod")},
			{Label: proto.String("Container"), Description: proto.String("Container inside pod")},
		},
	}
	// Build the style maps from the style chunk as BuildBaseSearchIndex does when the workbench loads.
	wb.styles = wb.buildStyleMaps()

	return wb
}

func TestSearchTimelines(t *testing.T) {
	testCases := []struct {
		name              string
		filter            Filter
		maxDepth          int
		maxNodes          int
		setupWorkbench    func() *Workbench
		wantErr           bool
		wantMatchedCount  int
		wantReturnedCount int
		wantMaxDepth      int
		wantMaxNodes      int
		wantTimelineTypes []TimelineTypeDescription
		wantNodes         []TimelineTreeNode
	}{
		{
			name:              "full tree returned when maxNodes is large enough",
			filter:            Filter{},
			maxDepth:          0,
			maxNodes:          10,
			wantMatchedCount:  5,
			wantReturnedCount: 5,
			wantMaxDepth:      0,
			wantMaxNodes:      10,
			wantTimelineTypes: []TimelineTypeDescription{
				{Type: "Namespace", Description: "Kubernetes namespace"},
				{Type: "Pod", Description: "Kubernetes pod"},
				{Type: "Container", Description: "Container inside pod"},
			},
			wantNodes: []TimelineTreeNode{
				{
					ID:            1,
					Depth:         0,
					Type:          "Namespace",
					Name:          "ns-1",
					EventCount:    0,
					RevisionCount: 0,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 2},
						{Severity: testSeverityWarning, Count: 2},
						{Severity: testSeverityInfo, Count: 2},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Time{},
					LastMatchTime:   time.Time{},
				},
				{
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
					OmittedChildren: 0,
					FirstMatchTime:  time.Unix(0, 1000).UTC(),
					LastMatchTime:   time.Unix(0, 2500).UTC(),
				},
				{
					ID:            4,
					Depth:         2,
					Type:          "Container",
					Name:          "container-a1",
					EventCount:    1,
					RevisionCount: 1,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 1},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Unix(0, 4000).UTC(),
					LastMatchTime:   time.Unix(0, 4500).UTC(),
				},
				{
					ID:            3,
					Depth:         1,
					Type:          "Pod",
					Name:          "pod-b",
					EventCount:    1,
					RevisionCount: 0,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 1},
						{Severity: testSeverityInfo, Count: 1},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Unix(0, 3000).UTC(),
					LastMatchTime:   time.Unix(0, 3000).UTC(),
				},
				{
					ID:            5,
					Depth:         2,
					Type:          "Container",
					Name:          "container-b1",
					EventCount:    1,
					RevisionCount: 1,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 1},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Unix(0, 5000).UTC(),
					LastMatchTime:   time.Unix(0, 5000).UTC(),
				},
			},
		},
		{
			name:     "shared log across parent child and sibling timelines is counted once in ancestor",
			filter:   Filter{},
			maxDepth: 0,
			maxNodes: 10,
			setupWorkbench: func() *Workbench {
				wb := setupSearchTimelinesTestWorkbench()
				// Link log 5 (ERROR, already on child container-a1) and log 4 (INFO, already on sibling pod-b) to pod-a.
				tl2 := wb.searchIndex.TimelineMap[2]
				tl2.Events = append(tl2.Events,
					cel.EventInfo{LogID: 4, Timestamp: 3000},
					cel.EventInfo{LogID: 5, Timestamp: 4000},
				)
				return wb
			},
			wantMatchedCount:  5,
			wantReturnedCount: 5,
			wantMaxDepth:      0,
			wantMaxNodes:      10,
			wantNodes: []TimelineTreeNode{
				{
					ID:            1,
					Depth:         0,
					Type:          "Namespace",
					Name:          "ns-1",
					EventCount:    0,
					RevisionCount: 0,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 2},
						{Severity: testSeverityWarning, Count: 2},
						{Severity: testSeverityInfo, Count: 2},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Time{},
					LastMatchTime:   time.Time{},
				},
				{
					ID:            2,
					Depth:         1,
					Type:          "Pod",
					Name:          "pod-a",
					EventCount:    4,
					RevisionCount: 1,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 1},
						{Severity: testSeverityWarning, Count: 2},
						{Severity: testSeverityInfo, Count: 2},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Unix(0, 1000).UTC(),
					LastMatchTime:   time.Unix(0, 4000).UTC(),
				},
				{
					ID:            4,
					Depth:         2,
					Type:          "Container",
					Name:          "container-a1",
					EventCount:    1,
					RevisionCount: 1,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 1},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Unix(0, 4000).UTC(),
					LastMatchTime:   time.Unix(0, 4500).UTC(),
				},
				{
					ID:            3,
					Depth:         1,
					Type:          "Pod",
					Name:          "pod-b",
					EventCount:    1,
					RevisionCount: 0,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 1},
						{Severity: testSeverityInfo, Count: 1},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Unix(0, 3000).UTC(),
					LastMatchTime:   time.Unix(0, 3000).UTC(),
				},
				{
					ID:            5,
					Depth:         2,
					Type:          "Container",
					Name:          "container-b1",
					EventCount:    1,
					RevisionCount: 1,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 1},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Unix(0, 5000).UTC(),
					LastMatchTime:   time.Unix(0, 5000).UTC(),
				},
			},
		},
		{
			name:              "breadth-first truncation by maxNodes",
			filter:            Filter{},
			maxDepth:          0,
			maxNodes:          3,
			wantMatchedCount:  5,
			wantReturnedCount: 3,
			wantMaxDepth:      0,
			wantMaxNodes:      3,
			wantTimelineTypes: []TimelineTypeDescription{
				{Type: "Namespace", Description: "Kubernetes namespace"},
				{Type: "Pod", Description: "Kubernetes pod"},
			},
			wantNodes: []TimelineTreeNode{
				{
					ID:            1,
					Depth:         0,
					Type:          "Namespace",
					Name:          "ns-1",
					EventCount:    0,
					RevisionCount: 0,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 2},
						{Severity: testSeverityWarning, Count: 2},
						{Severity: testSeverityInfo, Count: 2},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Time{},
					LastMatchTime:   time.Time{},
				},
				{
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
					OmittedChildren: 1,
					FirstMatchTime:  time.Unix(0, 1000).UTC(),
					LastMatchTime:   time.Unix(0, 2500).UTC(),
				},
				{
					ID:            3,
					Depth:         1,
					Type:          "Pod",
					Name:          "pod-b",
					EventCount:    1,
					RevisionCount: 0,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 1},
						{Severity: testSeverityInfo, Count: 1},
					},
					OmittedChildren: 1,
					FirstMatchTime:  time.Unix(0, 3000).UTC(),
					LastMatchTime:   time.Unix(0, 3000).UTC(),
				},
			},
		},
		{
			name:              "truncation by maxDepth",
			filter:            Filter{},
			maxDepth:          1,
			maxNodes:          10,
			wantMatchedCount:  5,
			wantReturnedCount: 3,
			wantMaxDepth:      1,
			wantMaxNodes:      10,
			wantTimelineTypes: []TimelineTypeDescription{
				{Type: "Namespace", Description: "Kubernetes namespace"},
				{Type: "Pod", Description: "Kubernetes pod"},
			},
			wantNodes: []TimelineTreeNode{
				{
					ID:            1,
					Depth:         0,
					Type:          "Namespace",
					Name:          "ns-1",
					EventCount:    0,
					RevisionCount: 0,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 2},
						{Severity: testSeverityWarning, Count: 2},
						{Severity: testSeverityInfo, Count: 2},
					},
					OmittedChildren: 0,
					FirstMatchTime:  time.Time{},
					LastMatchTime:   time.Time{},
				},
				{
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
					OmittedChildren: 1,
					FirstMatchTime:  time.Unix(0, 1000).UTC(),
					LastMatchTime:   time.Unix(0, 2500).UTC(),
				},
				{
					ID:            3,
					Depth:         1,
					Type:          "Pod",
					Name:          "pod-b",
					EventCount:    1,
					RevisionCount: 0,
					SeverityCounts: []SeverityCount{
						{Severity: testSeverityError, Count: 1},
						{Severity: testSeverityInfo, Count: 1},
					},
					OmittedChildren: 1,
					FirstMatchTime:  time.Unix(0, 3000).UTC(),
					LastMatchTime:   time.Unix(0, 3000).UTC(),
				},
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
			name:              "default maxNodes is used when non-positive",
			filter:            Filter{},
			maxDepth:          0,
			maxNodes:          0,
			wantMatchedCount:  5,
			wantReturnedCount: 5,
			wantMaxDepth:      0,
			wantMaxNodes:      DefaultMaxTimelineNodes,
		},
		{
			name:              "zero matching timelines returns empty result",
			filter:            Filter{TimelineQuery: `name == "non-existent"`},
			maxDepth:          0,
			maxNodes:          10,
			wantMatchedCount:  0,
			wantReturnedCount: 0,
			wantMaxDepth:      0,
			wantMaxNodes:      10,
		},
		{
			name:     "closed workbench returns error",
			filter:   Filter{},
			maxDepth: 0,
			maxNodes: 10,
			setupWorkbench: func() *Workbench {
				wb := setupSearchTimelinesTestWorkbench()
				wb.Close()
				return wb
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupSearchTimelinesTestWorkbench()
			if tc.setupWorkbench != nil {
				wb = tc.setupWorkbench()
			}

			got, err := wb.SearchTimelines(context.Background(), tc.filter, tc.maxDepth, tc.maxNodes)
			if (err != nil) != tc.wantErr {
				t.Fatalf("SearchTimelines() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if got.MatchedTimelineCount != tc.wantMatchedCount {
				t.Errorf("SearchTimelines() MatchedTimelineCount = %d, want %d", got.MatchedTimelineCount, tc.wantMatchedCount)
			}
			if got.ReturnedNodeCount != tc.wantReturnedCount {
				t.Errorf("SearchTimelines() ReturnedNodeCount = %d, want %d", got.ReturnedNodeCount, tc.wantReturnedCount)
			}
			if got.MaxDepth != tc.wantMaxDepth {
				t.Errorf("SearchTimelines() MaxDepth = %d, want %d", got.MaxDepth, tc.wantMaxDepth)
			}
			if got.MaxNodes != tc.wantMaxNodes {
				t.Errorf("SearchTimelines() MaxNodes = %d, want %d", got.MaxNodes, tc.wantMaxNodes)
			}
			if tc.wantTimelineTypes != nil {
				if diff := cmp.Diff(tc.wantTimelineTypes, got.TimelineTypes); diff != "" {
					t.Errorf("SearchTimelines() TimelineTypes mismatch (-want +got):\n%s", diff)
				}
			}
			if tc.wantNodes != nil {
				if diff := cmp.Diff(tc.wantNodes, got.Nodes, protocmp.Transform()); diff != "" {
					t.Errorf("SearchTimelines() Nodes mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
