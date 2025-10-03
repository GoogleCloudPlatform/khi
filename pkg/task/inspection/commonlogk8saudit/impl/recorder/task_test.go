// Copyright 2024 Google LLC
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

package recorder

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/google/go-cmp/cmp"
)

func TestConvertTimelineGroupListToHierarchicalTimelineGroup(t *testing.T) {
	tests := []struct {
		name  string
		input []*commonlogk8saudit_contract.TimelineGrouperResult
		want  *hierarchicalTimelineGroupWorker
	}{
		{
			name:  "empty",
			input: []*commonlogk8saudit_contract.TimelineGrouperResult{},
			want: &hierarchicalTimelineGroupWorker{
				children: map[string]*hierarchicalTimelineGroupWorker{},
				group:    nil,
			},
		},
		{
			name: "flat",
			input: []*commonlogk8saudit_contract.TimelineGrouperResult{
				{TimelineResourcePath: "a"},
				{TimelineResourcePath: "b"},
			},
			want: &hierarchicalTimelineGroupWorker{
				children: map[string]*hierarchicalTimelineGroupWorker{
					"a": {
						children: map[string]*hierarchicalTimelineGroupWorker{},
						group:    &commonlogk8saudit_contract.TimelineGrouperResult{TimelineResourcePath: "a"},
					},
					"b": {
						children: map[string]*hierarchicalTimelineGroupWorker{},
						group:    &commonlogk8saudit_contract.TimelineGrouperResult{TimelineResourcePath: "b"},
					},
				},
			},
		},
		{
			name: "three levels",
			input: []*commonlogk8saudit_contract.TimelineGrouperResult{
				{TimelineResourcePath: "a"},
				{TimelineResourcePath: "a#b"},
				{TimelineResourcePath: "a#b#c"},
			},
			want: &hierarchicalTimelineGroupWorker{
				children: map[string]*hierarchicalTimelineGroupWorker{
					"a": {
						children: map[string]*hierarchicalTimelineGroupWorker{
							"b": {
								children: map[string]*hierarchicalTimelineGroupWorker{
									"c": {
										children: map[string]*hierarchicalTimelineGroupWorker{},
										group:    &commonlogk8saudit_contract.TimelineGrouperResult{TimelineResourcePath: "a#b#c"},
									},
								},
								group: &commonlogk8saudit_contract.TimelineGrouperResult{TimelineResourcePath: "a#b"},
							},
						},
						group: &commonlogk8saudit_contract.TimelineGrouperResult{TimelineResourcePath: "a"},
					},
				},
			},
		},
		{
			name: "multiple children",
			input: []*commonlogk8saudit_contract.TimelineGrouperResult{
				{TimelineResourcePath: "a"},
				{TimelineResourcePath: "a#b"},
				{TimelineResourcePath: "a#c"},
			},
			want: &hierarchicalTimelineGroupWorker{
				children: map[string]*hierarchicalTimelineGroupWorker{
					"a": {
						children: map[string]*hierarchicalTimelineGroupWorker{
							"b": {
								children: map[string]*hierarchicalTimelineGroupWorker{},
								group:    &commonlogk8saudit_contract.TimelineGrouperResult{TimelineResourcePath: "a#b"},
							},
							"c": {
								children: map[string]*hierarchicalTimelineGroupWorker{},
								group:    &commonlogk8saudit_contract.TimelineGrouperResult{TimelineResourcePath: "a#c"},
							},
						},
						group: &commonlogk8saudit_contract.TimelineGrouperResult{TimelineResourcePath: "a"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertTimelineGroupListToHierarcicalTimelineGroup(tt.input)
			opts := []cmp.Option{
				cmp.AllowUnexported(hierarchicalTimelineGroupWorker{}),
			}
			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("convertTimelineGroupListToHierarcicalTimelineGroup() mismatch (-want +got): \n %s", diff)
			}
		})
	}
}

func TestHierarcicalTimelineGroupWorker_Run(t *testing.T) {
	// callRecord is a type to memory call timing for each group paths. This is used for verifying if its parent is called before its children.
	type callRecord struct {
		path      string
		timestamp time.Time
	}
	tests := []struct {
		name          string
		inputGroups   []*commonlogk8saudit_contract.TimelineGrouperResult
		expectedPaths []string
	}{
		{
			name:          "empty tree",
			inputGroups:   []*commonlogk8saudit_contract.TimelineGrouperResult{},
			expectedPaths: []string{},
		},
		{
			name: "flat tree",
			inputGroups: []*commonlogk8saudit_contract.TimelineGrouperResult{
				{TimelineResourcePath: "a"},
				{TimelineResourcePath: "b"},
			},
			expectedPaths: []string{"a", "b"},
		},
		{
			name: "hierarchical tree",
			inputGroups: []*commonlogk8saudit_contract.TimelineGrouperResult{
				{TimelineResourcePath: "a"},
				{TimelineResourcePath: "a#b"},
				{TimelineResourcePath: "a#b#c"},
				{TimelineResourcePath: "d"},
			},
			expectedPaths: []string{"a", "a#b", "a#b#c", "d"},
		},
		{
			name: "hierarchical tree with intermediate node without group",
			inputGroups: []*commonlogk8saudit_contract.TimelineGrouperResult{
				// "a" is an intermediate node without a group
				{TimelineResourcePath: "a#b"},
				{TimelineResourcePath: "c"},
			},
			expectedPaths: []string{"a#b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := convertTimelineGroupListToHierarcicalTimelineGroup(tt.inputGroups)

			ctx := context.Background()

			var records []callRecord
			var mu sync.Mutex

			f := func(group *commonlogk8saudit_contract.TimelineGrouperResult) {
				time.Sleep(1 * time.Millisecond)
				mu.Lock()
				defer mu.Unlock()
				records = append(records, callRecord{
					path:      group.TimelineResourcePath,
					timestamp: time.Now(),
				})
			}

			root.Run(ctx, f)

			// Verification1: if all groups are processed.
			processedPaths := make([]string, len(records))
			for i, r := range records {
				processedPaths[i] = r.path
			}
			sort.Strings(processedPaths)
			sort.Strings(tt.expectedPaths)
			if diff := cmp.Diff(tt.expectedPaths, processedPaths); diff != "" {
				t.Errorf("Run() processed paths mismatch (-want +got):\n%s", diff)
			}

			// Verification2: if parent element is processed before its children
			recordMap := make(map[string]time.Time)
			for _, r := range records {
				recordMap[r.path] = r.timestamp
			}

			for path, ts := range recordMap {
				if i := strings.LastIndex(path, "#"); i != -1 {
					parentPath := path[:i]
					if parentTs, ok := recordMap[parentPath]; ok {
						if parentTs.After(ts) {
							t.Errorf("parent %q (%v) was processed after child %q (%v)", parentPath, parentTs, path, ts)
						}
					}
				}
			}
		})
	}
}
