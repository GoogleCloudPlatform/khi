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

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
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

func TestBuildSearchIndex(t *testing.T) {
	testCases := []struct {
		name             string
		setupWorkbench   func() *Workbench
		wantTotalLogs    int
		wantTotalTLs     int
		wantTL1Path      map[string]string
		wantTL2Path      map[string]string
		wantTL1Children  []uint32
		wantTL1MaxSev    uint32
		wantTL2MaxSev    uint32
		wantTL2LogIDs    []uint32
		wantLog1Severity uint32
		wantLog2Severity uint32
		wantErr          bool
	}{
		{
			name: "single namespace and pod hierarchy with events and revisions",
			setupWorkbench: func() *Workbench {
				wb := NewWorkbench("test-wb", "test-inspection")
				idGen := &khifilev6model.IDGenerator{}
				pool := khifilev6model.NewInternPool(idGen)
				wb.internPool = pool

				// Style Chunk
				wb.styleChunk = &khifilev6.TimelineStyleChunk{
					Severities: []*khifilev6.Severity{
						{Id: proto.Uint32(10), Label: proto.String("INFO"), Order: proto.Int32(1)},
						{Id: proto.Uint32(20), Label: proto.String("WARNING"), Order: proto.Int32(2)},
						{Id: proto.Uint32(30), Label: proto.String("ERROR"), Order: proto.Int32(3)},
					},
					LogTypes: []*khifilev6.LogType{
						{Id: proto.Uint32(100), Label: proto.String("k8s-event")},
						{Id: proto.Uint32(200), Label: proto.String("k8s-audit")},
					},
					TimelineTypes: []*khifilev6.TimelineType{
						{Id: proto.Uint32(1), Label: proto.String("Namespace")},
						{Id: proto.Uint32(2), Label: proto.String("Pod")},
					},
					Verbs: []*khifilev6.Verb{
						{Id: proto.Uint32(1), Label: proto.String("create")},
						{Id: proto.Uint32(2), Label: proto.String("update")},
					},
					RevisionStates: []*khifilev6.RevisionState{
						{Id: proto.Uint32(1), Label: proto.String("active")},
					},
				}

				nsNameRef := pool.InternString("default")
				podNameRef := pool.InternString("pod-sample")
				principalRef := pool.InternString("admin")

				node, _ := structured.FromYAML("kind: Pod\nmetadata:\n  name: pod-sample\n")
				sRef, _ := khifilev6model.ToInternedStruct(node, pool)

				// Log Chunks (split into 2 chunks to test parallel indexing)
				wb.logChunks = []*khifilev6.LogChunk{
					{
						Logs: []*khifilev6.Log{
							{
								Id:             proto.Uint32(1),
								SeverityTypeId: proto.Uint32(10), // INFO (order 1)
								LogTypeId:      proto.Uint32(100),
								BodyStructId:   proto.Uint32(sRef.ID()),
							},
						},
					},
					{
						Logs: []*khifilev6.Log{
							{
								Id:             proto.Uint32(2),
								SeverityTypeId: proto.Uint32(30), // ERROR (order 3)
								LogTypeId:      proto.Uint32(200),
								BodyStructId:   proto.Uint32(sRef.ID()),
							},
						},
					},
				}

				// Timeline Chunks
				wb.timelineChunks = []*khifilev6.TimelineChunk{
					{
						TimelineItems: []*khifilev6.TimelineItems{
							{
								Id: proto.Uint32(1001),
								Events: []*khifilev6.Event{
									{LogId: proto.Uint32(1)},
								},
							},
							{
								Id: proto.Uint32(1002),
								Revisions: []*khifilev6.Revision{
									{
										LogId:                proto.Uint32(2),
										PrincipalStringId:    proto.Uint32(principalRef.ToProto().GetId()),
										VerbType:             proto.Uint32(1),
										StateType:            proto.Uint32(1),
										ResourceBodyStructId: proto.Uint32(sRef.ID()),
									},
								},
							},
						},
						Timelines: []*khifilev6.Timeline{
							{
								Id:              proto.Uint32(1),
								NameStringId:    proto.Uint32(nsNameRef.ToProto().GetId()),
								TimelineType:    proto.Uint32(1), // Namespace
								TimelineItemsId: proto.Uint32(1001),
							},
							{
								Id:               proto.Uint32(2),
								ParentTimelineId: proto.Uint32(1),
								NameStringId:     proto.Uint32(podNameRef.ToProto().GetId()),
								TimelineType:     proto.Uint32(2), // Pod
								TimelineItemsId:  proto.Uint32(1002),
							},
						},
					},
				}

				return wb
			},
			wantTotalLogs: 2,
			wantTotalTLs:  2,
			wantTL1Path: map[string]string{
				"namespace": "default",
			},
			wantTL2Path: map[string]string{
				"namespace": "default",
				"pod":       "pod-sample",
			},
			wantTL1Children:  []uint32{2},
			wantTL1MaxSev:    1,
			wantTL2MaxSev:    3,
			wantTL2LogIDs:    []uint32{2},
			wantLog1Severity: 1,
			wantLog2Severity: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := tc.setupWorkbench()
			idx, err := wb.BuildSearchIndex()
			if (err != nil) != tc.wantErr {
				t.Fatalf("BuildSearchIndex() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if len(idx.Logs) != tc.wantTotalLogs {
				t.Errorf("BuildSearchIndex() total logs mismatch (-want +got):\n%s", cmp.Diff(tc.wantTotalLogs, len(idx.Logs)))
			}
			if len(idx.Timelines) != tc.wantTotalTLs {
				t.Errorf("BuildSearchIndex() total timelines mismatch (-want +got):\n%s", cmp.Diff(tc.wantTotalTLs, len(idx.Timelines)))
			}

			tl1 := idx.TimelineMap[1]
			if tl1 == nil {
				t.Fatal("timeline 1 not found in index")
			}
			if diff := cmp.Diff(tc.wantTL1Path, tl1.Data.Path); diff != "" {
				t.Errorf("timeline 1 Path mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantTL1Children, tl1.ChildrenIDs); diff != "" {
				t.Errorf("timeline 1 ChildrenIDs mismatch (-want +got):\n%s", diff)
			}
			if tl1.Data.MaxSeverity != tc.wantTL1MaxSev {
				t.Errorf("timeline 1 MaxSeverity mismatch (-want +got):\n%s", cmp.Diff(tc.wantTL1MaxSev, tl1.Data.MaxSeverity))
			}

			tl2 := idx.TimelineMap[2]
			if tl2 == nil {
				t.Fatal("timeline 2 not found in index")
			}
			if diff := cmp.Diff(tc.wantTL2Path, tl2.Data.Path); diff != "" {
				t.Errorf("timeline 2 Path mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantTL2MaxSev, tl2.Data.MaxSeverity); diff != "" {
				t.Errorf("timeline 2 MaxSeverity mismatch (-want +got):\n%s", cmp.Diff(tc.wantTL2MaxSev, tl2.Data.MaxSeverity))
			}
			if diff := cmp.Diff(tc.wantTL2LogIDs, tl2.LogIDs); diff != "" {
				t.Errorf("timeline 2 LogIDs mismatch (-want +got):\n%s", diff)
			}

			log1 := idx.LogMap[1]
			if log1 == nil {
				t.Fatal("log 1 not found in index")
			}
			if log1.Data.Severity != tc.wantLog1Severity {
				t.Errorf("log 1 Severity mismatch (-want +got):\n%s", cmp.Diff(tc.wantLog1Severity, log1.Data.Severity))
			}

			log2 := idx.LogMap[2]
			if log2 == nil {
				t.Fatal("log 2 not found in index")
			}
			if log2.Data.Severity != tc.wantLog2Severity {
				t.Errorf("log 2 Severity mismatch (-want +got):\n%s", cmp.Diff(tc.wantLog2Severity, log2.Data.Severity))
			}
		})
	}
}
