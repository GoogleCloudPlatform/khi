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

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
)

func createSampleWorkbench() *Workbench {
	wb := NewWorkbench("wb-test", "test-inspection")
	wb.searchIndex = &SearchIndex{
		TimelineMap: make(map[uint32]*IndexedTimeline),
		LogMap:      make(map[uint32]*IndexedLog),
	}

	// 3 logs:
	// Log 1: Pod-A event, severity INFO (1)
	// Log 2: Pod-A event, severity ERROR (3)
	// Log 3: Pod-B event, severity INFO (1)
	log1 := &IndexedLog{
		ID: 1,
		Data: &cel.LogData{
			ID:       1,
			LogType:  "k8s-event",
			Severity: 1, // INFO
			Summary:  "Pod A started",
		},
	}
	log2 := &IndexedLog{
		ID: 2,
		Data: &cel.LogData{
			ID:       2,
			LogType:  "k8s-event",
			Severity: 3, // ERROR
			Summary:  "Pod A crashed",
		},
	}
	log3 := &IndexedLog{
		ID: 3,
		Data: &cel.LogData{
			ID:       3,
			LogType:  "k8s-event",
			Severity: 1, // INFO
			Summary:  "Pod B started",
		},
	}

	wb.searchIndex.Logs = []*IndexedLog{log1, log2, log3}
	wb.searchIndex.LogMap[1] = log1
	wb.searchIndex.LogMap[2] = log2
	wb.searchIndex.LogMap[3] = log3

	// Timeline 1: Root Namespace "default" (parent of Pod A and Pod B)
	// Timeline 2: Pod A (child of 1), has Log 1 and 2
	// Timeline 3: Pod B (child of 1), has Log 3
	// Timeline 4: Pod B Container C (child of 3), has Log 3
	tl1 := &IndexedTimeline{
		ID:          1,
		ParentID:    0,
		ChildrenIDs: []uint32{2, 3},
		Data: &cel.TimelineData{
			ID:           1,
			Name:         "default",
			TimelineType: "Namespace",
			Path: map[string]string{
				"namespace": "default",
			},
			MaxSeverity: 3,
		},
	}

	tl2 := &IndexedTimeline{
		ID:          2,
		ParentID:    1,
		ChildrenIDs: nil,
		LogIDs:      []uint32{1, 2},
		Data: &cel.TimelineData{
			ID:           2,
			Name:         "pod-a",
			TimelineType: "Pod",
			Path: map[string]string{
				"namespace": "default",
				"kind":      "Pod",
				"name":      "pod-a",
			},
			MaxSeverity: 3,
		},
	}

	tl3 := &IndexedTimeline{
		ID:          3,
		ParentID:    1,
		ChildrenIDs: []uint32{4},
		LogIDs:      []uint32{3},
		Data: &cel.TimelineData{
			ID:           3,
			Name:         "pod-b",
			TimelineType: "Pod",
			Path: map[string]string{
				"namespace": "default",
				"kind":      "Pod",
				"name":      "pod-b",
			},
			MaxSeverity: 1,
		},
	}

	tl4 := &IndexedTimeline{
		ID:          4,
		ParentID:    3,
		ChildrenIDs: nil,
		LogIDs:      []uint32{3},
		Data: &cel.TimelineData{
			ID:           4,
			Name:         "container-b",
			TimelineType: "Container",
			Path: map[string]string{
				"namespace": "default",
				"kind":      "Pod",
				"container": "container-b",
			},
			MaxSeverity: 1,
		},
	}

	wb.searchIndex.Timelines = []*IndexedTimeline{tl1, tl2, tl3, tl4}
	wb.searchIndex.TimelineMap[1] = tl1
	wb.searchIndex.TimelineMap[2] = tl2
	wb.searchIndex.TimelineMap[3] = tl3
	wb.searchIndex.TimelineMap[4] = tl4

	return wb
}

func TestFilterTimelinePipeline(t *testing.T) {
	testCases := []struct {
		name          string
		params        FilterPipelineParams
		wantTimelines []uint32
		wantLogs      []uint32
		wantErr       bool
	}{
		{
			name: "no filters pass everything",
			params: FilterPipelineParams{
				TimelineQuery: "",
				LogQuery:      "",
				ExcludeNoLogs: false,
			},
			wantTimelines: []uint32{1, 2, 3, 4},
			wantLogs:      []uint32{1, 2, 3},
		},
		{
			name: "timeline CEL inclusion matches Pod A and includes ancestors",
			params: FilterPipelineParams{
				TimelineQuery: `name == "pod-a"`,
			},
			wantTimelines: []uint32{1, 2},
			wantLogs:      []uint32{1, 2},
		},
		{
			name: "timeline CEL inclusion on parent includes descendants",
			params: FilterPipelineParams{
				TimelineQuery: `name == "pod-b"`,
			},
			wantTimelines: []uint32{1, 3, 4},
			wantLogs:      []uint32{3},
		},
		{
			name: "timeline exclusion removes subtree",
			params: FilterPipelineParams{
				TimelineQuery:          "",
				TimelineExclusionQuery: `name == "pod-b"`,
			},
			wantTimelines: []uint32{1, 2},
			wantLogs:      []uint32{1, 2},
		},
		{
			name: "log CEL filter filters logs and retains timelines with logs",
			params: FilterPipelineParams{
				LogQuery:      `severity >= ERROR`,
				ExcludeNoLogs: true,
			},
			wantTimelines: []uint32{2},
			wantLogs:      []uint32{2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := createSampleWorkbench()
			var progressReports []*apiv1.FilterProgress
			res, err := wb.FilterTimeline(context.Background(), tc.params, func(p *apiv1.FilterProgress) error {
				progressReports = append(progressReports, p)
				return nil
			})
			if (err != nil) != tc.wantErr {
				t.Fatalf("FilterTimeline() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			allTLIDs := []uint32{1, 2, 3, 4}
			allLogIDs := []uint32{1, 2, 3}
			gotTimelineIDs := decodeSparseBitset(res.GetTimelineMode(), res.GetTimelineBitset(), allTLIDs)
			gotLogIDs := decodeSparseBitset(res.GetLogMode(), res.GetLogBitset(), allLogIDs)

			if diff := cmp.Diff(tc.wantTimelines, gotTimelineIDs); diff != "" {
				t.Errorf("FilterTimeline() timeline IDs mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantLogs, gotLogIDs); diff != "" {
				t.Errorf("FilterTimeline() log IDs mismatch (-want +got):\n%s", diff)
			}
			if len(progressReports) == 0 {
				t.Errorf("FilterTimeline() expected progress reports but got none")
			}
		})
	}
}

func decodeSparseBitset(mode apiv1.FilterResultMode, bitset *apiv1.SparseBitset, allIDs []uint32) []uint32 {
	if bitset == nil {
		return nil
	}
	blockMap := make(map[uint32]uint32)
	for i, idx := range bitset.Indices {
		blockMap[idx] = bitset.Masks[i]
	}
	isSet := func(id uint32) bool {
		mask, ok := blockMap[id/32]
		if !ok {
			return false
		}
		return (mask & (1 << (id % 32))) != 0
	}

	var result []uint32
	if mode == apiv1.FilterResultMode_FILTER_RESULT_MODE_INCLUDE {
		for _, id := range allIDs {
			if isSet(id) {
				result = append(result, id)
			}
		}
	} else {
		for _, id := range allIDs {
			if !isSet(id) {
				result = append(result, id)
			}
		}
	}
	return result
}

func TestFilterTimelineCancellation(t *testing.T) {
	wb := createSampleWorkbench()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := wb.FilterTimeline(ctx, FilterPipelineParams{}, nil)
	if err == nil {
		t.Errorf("FilterTimeline() expected context cancellation error but got nil")
	}
}
