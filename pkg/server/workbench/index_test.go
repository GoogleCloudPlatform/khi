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
	"testing"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
)

func TestIndexedTimeline_ComputePath(t *testing.T) {
	testCases := []struct {
		name     string
		targetID uint32
		tlMap    map[uint32]*IndexedTimeline
		wantPath map[string]string
	}{
		{
			name:     "single root timeline",
			targetID: 1,
			tlMap: map[uint32]*IndexedTimeline{
				1: {
					ID:       1,
					ParentID: 0,
					Data: &cel.TimelineData{
						ID:           1,
						Name:         "cluster-1",
						TimelineType: "Cluster",
					},
				},
			},
			wantPath: map[string]string{
				"cluster": "cluster-1",
			},
		},
		{
			name:     "multi-level hierarchical timeline",
			targetID: 3,
			tlMap: map[uint32]*IndexedTimeline{
				1: {
					ID:       1,
					ParentID: 0,
					Data: &cel.TimelineData{
						ID:           1,
						Name:         "default",
						TimelineType: "Namespace",
					},
				},
				2: {
					ID:       2,
					ParentID: 1,
					Data: &cel.TimelineData{
						ID:           2,
						Name:         "frontend-deployment",
						TimelineType: "Deployment",
					},
				},
				3: {
					ID:       3,
					ParentID: 2,
					Data: &cel.TimelineData{
						ID:           3,
						Name:         "frontend-pod-abc",
						TimelineType: "Pod",
					},
				},
			},
			wantPath: map[string]string{
				"namespace":  "default",
				"deployment": "frontend-deployment",
				"pod":        "frontend-pod-abc",
			},
		},
		{
			name:     "handles cycle safely without infinite loop",
			targetID: 1,
			tlMap: map[uint32]*IndexedTimeline{
				1: {
					ID:       1,
					ParentID: 2,
					Data: &cel.TimelineData{
						ID:           1,
						Name:         "node-1",
						TimelineType: "Node",
					},
				},
				2: {
					ID:       2,
					ParentID: 1, // Cycle back to 1
					Data: &cel.TimelineData{
						ID:           2,
						Name:         "node-parent",
						TimelineType: "ParentNode",
					},
				},
			},
			wantPath: map[string]string{
				"node":       "node-1",
				"parentnode": "node-parent",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetTL := tc.tlMap[tc.targetID]
			got := targetTL.ComputePath(tc.tlMap)
			if diff := cmp.Diff(tc.wantPath, got); diff != "" {
				t.Errorf("ComputePath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWorkbench_BuildSearchIndex(t *testing.T) {
	str1ID := uint32(1)
	str1Val := "cluster"
	str2ID := uint32(2)
	str2Val := "my-cluster"
	str3ID := uint32(3)
	str3Val := "my log summary"

	poolChunk := &khifilev6.InterningPoolChunk{
		Strings: []*khifilev6.InternString{
			{Id: &str1ID, Value: &str1Val},
			{Id: &str2ID, Value: &str2Val},
			{Id: &str3ID, Value: &str3Val},
		},
	}

	styleChunk := &khifilev6.TimelineStyleChunk{
		TimelineTypes: []*khifilev6.TimelineType{
			{Id: proto.Uint32(1), Label: proto.String("Cluster")},
		},
		LogTypes: []*khifilev6.LogType{
			{Id: proto.Uint32(1), Label: proto.String("kubernetes")},
		},
		Severities: []*khifilev6.Severity{
			{Id: proto.Uint32(1), Order: proto.Int32(1), Label: proto.String("INFO")},
		},
	}

	logChunk := &khifilev6.LogChunk{
		Logs: []*khifilev6.Log{
			{
				Id:              proto.Uint32(10),
				LogTypeId:       proto.Uint32(1),
				SeverityTypeId:  proto.Uint32(1),
				SummaryStringId: proto.Uint32(str3ID),
			},
		},
	}

	timelineChunk := &khifilev6.TimelineChunk{
		Timelines: []*khifilev6.Timeline{
			{
				Id:               proto.Uint32(1),
				ParentTimelineId: proto.Uint32(0),
				NameStringId:     proto.Uint32(str2ID),
				TimelineType:     proto.Uint32(1),
				TimelineItemsId:  proto.Uint32(1),
			},
		},
		TimelineItems: []*khifilev6.TimelineItems{
			{
				Id: proto.Uint32(1),
				Events: []*khifilev6.Event{
					{LogId: proto.Uint32(10)},
				},
			},
		},
	}

	wb := NewWorkbench("wb-test", "insp-test")
	wb.internPool.IngestChunk(poolChunk)
	wb.styleChunk = styleChunk
	wb.logChunks = append(wb.logChunks, logChunk)
	wb.timelineChunks = append(wb.timelineChunks, timelineChunk)

	index, err := wb.BuildSearchIndex()
	if err != nil {
		t.Fatalf("BuildSearchIndex() unexpected error = %v", err)
	}

	if len(index.Timelines) != 1 {
		t.Fatalf("len(index.Timelines) = %d, want 1", len(index.Timelines))
	}
	if len(index.Logs) != 1 {
		t.Fatalf("len(index.Logs) = %d, want 1", len(index.Logs))
	}

	tl := index.Timelines[0]
	if tl.Path["cluster"] != "my-cluster" {
		t.Errorf("tl.Path[\"cluster\"] = %q, want %q", tl.Path["cluster"], "my-cluster")
	}
	if tl.Data.Path["cluster"] != "my-cluster" {
		t.Errorf("tl.Data.Path[\"cluster\"] = %q, want %q", tl.Data.Path["cluster"], "my-cluster")
	}
}
