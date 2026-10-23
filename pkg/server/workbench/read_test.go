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
	"google.golang.org/protobuf/proto"
)

var testHeaderMetadataChunks = []*khifilev6.MetadataChunk{
	{
		Metadata: []*khifilev6.MetadataItem{
			{
				Payload: &khifilev6.MetadataItem_Header{
					Header: &khifilev6.HeaderMetadata{
						StartTimeUnixSeconds: proto.Int64(1700000000),
						EndTimeUnixSeconds:   proto.Int64(1700003600),
					},
				},
			},
		},
	},
}

func TestFormatTimelinePathAndBuildTimelineQuery(t *testing.T) {
	testCases := []struct {
		name      string
		segments  []TimelineSegment
		wantPath  string
		wantQuery string
	}{
		{
			name:      "empty segments",
			segments:  nil,
			wantPath:  "",
			wantQuery: "",
		},
		{
			name: "hierarchy with cluster namespace pod",
			segments: []TimelineSegment{
				{Type: "APIVersion", Name: "core/v1"},
				{Type: "Kind", Name: "pod"},
				{Type: "Namespace", Name: "kube-system"},
				{Type: "Resource", Name: "coredns-123"},
			},
			wantPath:  "[APIVersion] core/v1 > [Kind] pod > [Namespace] kube-system > [Resource] coredns-123",
			wantQuery: `path["apiversion"] == "core/v1" && path["kind"] == "pod" && path["namespace"] == "kube-system" && path["resource"] == "coredns-123"`,
		},
		{
			name: "hyphenated timeline type uses bracket syntax",
			segments: []TimelineSegment{
				{Type: "Namespace", Name: "kube-system"},
				{Type: "node-component", Name: "kubelet"},
			},
			wantPath:  "[Namespace] kube-system > [node-component] kubelet",
			wantQuery: `path["namespace"] == "kube-system" && path["node-component"] == "kubelet"`,
		},
		{
			name: "escaping quotes and backslashes in type and value",
			segments: []TimelineSegment{
				{Type: `custom"type\name`, Name: `pod"s\name`},
			},
			wantPath:  `[custom"type\name] pod"s\name`,
			wantQuery: `path["custom\"type\\name"] == "pod\"s\\name"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath := FormatTimelinePath(tc.segments)
			if gotPath != tc.wantPath {
				t.Errorf("FormatTimelinePath() = %q, want %q", gotPath, tc.wantPath)
			}

			gotQuery := BuildTimelineQuery(tc.segments)
			if gotQuery != tc.wantQuery {
				t.Errorf("BuildTimelineQuery() = %q, want %q", gotQuery, tc.wantQuery)
			}

			if err := cel.ValidateTimelineQuery(gotQuery); err != nil {
				t.Errorf("ValidateTimelineQuery(%q) unexpected error: %v", gotQuery, err)
			}
		})
	}
}

func TestTimelineSegments(t *testing.T) {
	wb := createSampleWorkbench()

	testCases := []struct {
		name       string
		timelineID uint32
		want       []TimelineSegment
		wantErr    bool
	}{
		{
			name:       "leaf container timeline",
			timelineID: 4,
			want: []TimelineSegment{
				{Type: "Namespace", Name: "default"},
				{Type: "Pod", Name: "pod-b"},
				{Type: "Container", Name: "container-b"},
			},
			wantErr: false,
		},
		{
			name:       "root namespace timeline",
			timelineID: 1,
			want: []TimelineSegment{
				{Type: "Namespace", Name: "default"},
			},
			wantErr: false,
		},
		{
			name:       "non-existent timeline",
			timelineID: 999,
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := wb.TimelineSegments(tc.timelineID)
			if (err != nil) != tc.wantErr {
				t.Fatalf("TimelineSegments(%d) error = %v, wantErr = %v", tc.timelineID, err, tc.wantErr)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("TimelineSegments(%d) mismatch (-want +got):\n%s", tc.timelineID, diff)
				}
			}
		})
	}
}

func TestInspectionTimeRange(t *testing.T) {
	testCases := []struct {
		name           string
		metadataChunks []*khifilev6.MetadataChunk
		wantStart      time.Time
		wantEnd        time.Time
	}{
		{
			name:           "from HeaderMetadata",
			metadataChunks: testHeaderMetadataChunks,
			wantStart:      time.Unix(1700000000, 0).UTC(),
			wantEnd:        time.Unix(1700003600, 0).UTC(),
		},
		{
			name:           "fallback to indexed timestamps",
			metadataChunks: nil,
			wantStart:      time.Unix(0, 1000).UTC(),
			wantEnd:        time.Unix(0, 3000).UTC(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := createSampleWorkbench()
			wb.metadataChunks = tc.metadataChunks
			start, end := wb.InspectionTimeRange()
			if !start.Equal(tc.wantStart) {
				t.Errorf("start time = %v, want %v", start, tc.wantStart)
			}
			if !end.Equal(tc.wantEnd) {
				t.Errorf("end time = %v, want %v", end, tc.wantEnd)
			}
		})
	}
}

func TestResolveFilter(t *testing.T) {
	wb := createSampleWorkbench()
	wb.metadataChunks = testHeaderMetadataChunks

	testCases := []struct {
		name        string
		filter      Filter
		wantApplied AppliedFilter
	}{
		{
			name: "default resolution without log query",
			filter: Filter{
				TimelineQuery: "path.kind == 'Pod'",
			},
			wantApplied: AppliedFilter{
				TimelineQuery: "path.kind == 'Pod'",
				StartTime:     time.Unix(1700000000, 0).UTC(),
				EndTime:       time.Unix(1700003600, 0).UTC(),
			},
		},
		{
			name: "default resolution with log query sets ExcludeTimelinesWithoutLogs to true",
			filter: Filter{
				LogQuery: "severity >= ERROR",
			},
			wantApplied: AppliedFilter{
				LogQuery:                    "severity >= ERROR",
				StartTime:                   time.Unix(1700000000, 0).UTC(),
				EndTime:                     time.Unix(1700003600, 0).UTC(),
				ExcludeTimelinesWithoutLogs: true,
			},
		},
		{
			name: "explicit ExcludeTimelinesWithoutLogs overrides default",
			filter: Filter{
				LogQuery:                    "severity >= ERROR",
				ExcludeTimelinesWithoutLogs: proto.Bool(false),
			},
			wantApplied: AppliedFilter{
				LogQuery:  "severity >= ERROR",
				StartTime: time.Unix(1700000000, 0).UTC(),
				EndTime:   time.Unix(1700003600, 0).UTC(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := wb.ResolveFilter(tc.filter)
			if diff := cmp.Diff(tc.wantApplied, got); diff != "" {
				t.Errorf("ResolveFilter() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateFilter(t *testing.T) {
	testCases := []struct {
		name      string
		filter    Filter
		wantField string
		wantErr   bool
	}{
		{
			name: "valid empty filter",
		},
		{
			name: "invalid timeline query",
			filter: Filter{
				TimelineQuery: "unknownVar == 'default'",
			},
			wantField: "filter.timelineQuery",
			wantErr:   true,
		},
		{
			name: "invalid timeline exclusion query",
			filter: Filter{
				TimelineExclusionQuery: "unknownFunc('foo')",
			},
			wantField: "filter.timelineExclusionQuery",
			wantErr:   true,
		},
		{
			name: "invalid log query",
			filter: Filter{
				LogQuery: "severit >= WARNING",
			},
			wantField: "filter.logQuery",
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			field, err := ValidateFilter(tc.filter)
			if field != tc.wantField {
				t.Errorf("ValidateFilter() field = %q, wantField = %q", field, tc.wantField)
			}
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateFilter() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestExecuteFilter(t *testing.T) {
	testCases := []struct {
		name            string
		filter          Filter
		modify          func(*Workbench)
		wantTimelineIDs []uint32
		wantLogIDs      []uint32
		wantErr         bool
		wantErrTarget   error
	}{
		{
			name:            "empty filter matches all timelines and logs",
			wantTimelineIDs: []uint32{1, 2, 3, 4},
			wantLogIDs:      []uint32{1, 2, 3},
		},
		{
			name: "timeline include query keeps matched timeline and ancestors",
			filter: Filter{
				TimelineQuery: "name == 'pod-a'",
			},
			wantTimelineIDs: []uint32{1, 2},
			wantLogIDs:      []uint32{1, 2},
		},
		{
			name: "timeline exclusion query removes excluded subtree",
			filter: Filter{
				TimelineExclusionQuery: "name == 'pod-a'",
			},
			wantTimelineIDs: []uint32{1, 3, 4},
			wantLogIDs:      []uint32{3},
		},
		{
			name: "log query with default exclude timelines without logs",
			filter: Filter{
				LogQuery: "severity >= ERROR",
			},
			wantTimelineIDs: []uint32{1, 2},
			wantLogIDs:      []uint32{2},
		},
		{
			name: "invalid log query returns validation error",
			filter: Filter{
				LogQuery: "severity >",
			},
			wantErr: true,
		},
		{
			name: "closed workbench returns ErrWorkbenchClosed",
			modify: func(wb *Workbench) {
				wb.Close()
			},
			wantErr:       true,
			wantErrTarget: ErrWorkbenchClosed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := createSampleWorkbench()
			if tc.modify != nil {
				tc.modify(wb)
			}

			out, err := wb.ExecuteFilter(context.Background(), tc.filter)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ExecuteFilter() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
					t.Errorf("ExecuteFilter() error = %v, want %v", err, tc.wantErrTarget)
				}
				return
			}

			if diff := cmp.Diff(tc.wantTimelineIDs, out.TimelineIDs.ToArray()); diff != "" {
				t.Errorf("ExecuteFilter() timeline IDs mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantLogIDs, out.LogIDs.ToArray()); diff != "" {
				t.Errorf("ExecuteFilter() log IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
