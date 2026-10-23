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
	type wantResult struct {
		Path  string
		Query string
	}

	testCases := []struct {
		name     string
		segments []TimelineSegment
		want     wantResult
	}{
		{
			name:     "empty segments",
			segments: nil,
			want:     wantResult{},
		},
		{
			name: "hierarchy with cluster namespace pod",
			segments: []TimelineSegment{
				{Type: "APIVersion", Name: "core/v1"},
				{Type: "Kind", Name: "pod"},
				{Type: "Namespace", Name: "kube-system"},
				{Type: "Resource", Name: "coredns-123"},
			},
			want: wantResult{
				Path:  "[APIVersion] core/v1 > [Kind] pod > [Namespace] kube-system > [Resource] coredns-123",
				Query: `path["apiversion"] == "core/v1" && path["kind"] == "pod" && path["namespace"] == "kube-system" && path["resource"] == "coredns-123"`,
			},
		},
		{
			name: "hyphenated timeline type uses bracket syntax",
			segments: []TimelineSegment{
				{Type: "Namespace", Name: "kube-system"},
				{Type: "node-component", Name: "kubelet"},
			},
			want: wantResult{
				Path:  "[Namespace] kube-system > [node-component] kubelet",
				Query: `path["namespace"] == "kube-system" && path["node-component"] == "kubelet"`,
			},
		},
		{
			name: "escaping quotes and backslashes in type and value",
			segments: []TimelineSegment{
				{Type: `custom"type\name`, Name: `pod"s\name`},
			},
			want: wantResult{
				Path:  `[custom"type\name] pod"s\name`,
				Query: `path["custom\"type\\name"] == "pod\"s\\name"`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := wantResult{
				Path:  FormatTimelinePath(tc.segments),
				Query: BuildTimelineQuery(tc.segments),
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FormatTimelinePath/BuildTimelineQuery mismatch (-want +got):\n%s", diff)
			}

			if err := cel.ValidateTimelineQuery(got.Query); err != nil {
				t.Errorf("ValidateTimelineQuery(%q) unexpected error: %v", got.Query, err)
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
	type wantTimeRange struct {
		Start time.Time
		End   time.Time
	}

	testCases := []struct {
		name           string
		metadataChunks []*khifilev6.MetadataChunk
		want           wantTimeRange
	}{
		{
			name:           "from HeaderMetadata",
			metadataChunks: testHeaderMetadataChunks,
			want: wantTimeRange{
				Start: time.Unix(1700000000, 0).UTC(),
				End:   time.Unix(1700003600, 0).UTC(),
			},
		},
		{
			name:           "fallback to indexed timestamps",
			metadataChunks: nil,
			want: wantTimeRange{
				Start: time.Unix(0, 1000).UTC(),
				End:   time.Unix(0, 3000).UTC(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := createSampleWorkbench()
			wb.metadataChunks = tc.metadataChunks
			start, end := wb.InspectionTimeRange()
			got := wantTimeRange{Start: start, End: end}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("InspectionTimeRange() mismatch (-want +got):\n%s", diff)
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
	type wantFilterOutput struct {
		TimelineIDs []uint32
		LogIDs      []uint32
	}

	testCases := []struct {
		name          string
		filter        Filter
		modify        func(*Workbench)
		want          wantFilterOutput
		wantErr       bool
		wantErrTarget error
	}{
		{
			name: "empty filter matches all timelines and logs",
			want: wantFilterOutput{
				TimelineIDs: []uint32{1, 2, 3, 4},
				LogIDs:      []uint32{1, 2, 3},
			},
		},
		{
			name: "timeline include query keeps matched timeline and ancestors",
			filter: Filter{
				TimelineQuery: "name == 'pod-a'",
			},
			want: wantFilterOutput{
				TimelineIDs: []uint32{1, 2},
				LogIDs:      []uint32{1, 2},
			},
		},
		{
			name: "timeline exclusion query removes excluded subtree",
			filter: Filter{
				TimelineExclusionQuery: "name == 'pod-a'",
			},
			want: wantFilterOutput{
				TimelineIDs: []uint32{1, 3, 4},
				LogIDs:      []uint32{3},
			},
		},
		{
			name: "log query with default exclude timelines without logs",
			filter: Filter{
				LogQuery: "severity >= ERROR",
			},
			want: wantFilterOutput{
				TimelineIDs: []uint32{1, 2},
				LogIDs:      []uint32{2},
			},
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

			got := wantFilterOutput{
				TimelineIDs: out.TimelineIDs.ToArray(),
				LogIDs:      out.LogIDs.ToArray(),
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExecuteFilter() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
