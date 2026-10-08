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

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
)

func TestBuildFilterVocabulary(t *testing.T) {
	styles := &cel.SimpleStyleResolver{
		LogTypes: map[uint32]string{
			1: "k8s_audit",
			2: "k8s_event",
			3: "",
		},
	}
	testCases := []struct {
		name      string
		timelines []*cel.TimelineData
		logs      []cel.LogData
		want      FilterVocabulary
	}{
		{
			name: "distinct sorted values with lowercased path keys",
			timelines: []*cel.TimelineData{
				{ID: 1, TimelineType: "pod"},
				{ID: 2, TimelineType: "k8sCluster"},
				{ID: 3, TimelineType: "pod"},
				{ID: 4, TimelineType: "apiVersion"},
			},
			logs: []cel.LogData{
				{ID: 1, LogTypeID: 2},
				{ID: 2, LogTypeID: 1},
				{ID: 3, LogTypeID: 2},
			},
			want: FilterVocabulary{
				PathKeys:      []string{"apiversion", "k8scluster", "pod"},
				TimelineTypes: []string{"apiVersion", "k8sCluster", "pod"},
				LogTypes:      []string{"k8s_audit", "k8s_event"},
			},
		},
		{
			name: "empty timeline types and unlabeled log types are skipped",
			timelines: []*cel.TimelineData{
				{ID: 1, TimelineType: ""},
				{ID: 2, TimelineType: "node"},
			},
			logs: []cel.LogData{
				{ID: 1, LogTypeID: 3},
				{ID: 2, LogTypeID: 99},
				{ID: 3, LogTypeID: 1},
			},
			want: FilterVocabulary{
				PathKeys:      []string{"node"},
				TimelineTypes: []string{"node"},
				LogTypes:      []string{"k8s_audit"},
			},
		},
		{
			name: "no data",
			want: FilterVocabulary{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildFilterVocabulary(tc.timelines, tc.logs, styles)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("buildFilterVocabulary() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
