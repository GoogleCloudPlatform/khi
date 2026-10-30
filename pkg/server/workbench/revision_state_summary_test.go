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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
)

func TestSummarizeRevisionStates(t *testing.T) {
	ts := func(ns int64) time.Time {
		return time.Unix(0, ns).UTC()
	}
	revsFromStates := func(states ...string) []cel.RevisionInfo {
		revs := make([]cel.RevisionInfo, len(states))
		for i, st := range states {
			revs[i] = cel.RevisionInfo{
				ChangedTime: int64((i + 1) * 1000),
				State:       st,
			}
		}
		return revs
	}

	testCases := []struct {
		name      string
		revs      []cel.RevisionInfo
		startTime time.Time
		endTime   time.Time
		want      RevisionStateSummary
	}{
		{
			name: "empty revisions returns empty summary",
			revs: nil,
			want: RevisionStateSummary{},
		},
		{
			name: "revisions with empty state labels are ignored",
			revs: []cel.RevisionInfo{
				{ChangedTime: 1000, State: ""},
				{ChangedTime: 2000, State: ""},
			},
			want: RevisionStateSummary{},
		},
		{
			name: "consecutive identical states collapse to single step",
			revs: revsFromStates(
				"Resource exists",
				"Resource exists",
				"Resource exists",
			),
			want: RevisionStateSummary{
				Prefix: []RevisionStateStep{
					{States: []string{"Resource exists"}, Count: 1},
				},
				TotalTransitions: 0,
			},
		},
		{
			name: "normal 8-step container lifecycle fits in full without truncation",
			revs: revsFromStates(
				"Container status is unavailable",
				"Container is waiting",
				"Container is started, readiness unknown",
				"Container is not ready",
				"Container is ready",
				"Container is not ready",
				"Container exited successfully",
				"Resource is deleted",
			),
			want: RevisionStateSummary{
				Prefix: []RevisionStateStep{
					{States: []string{"Container status is unavailable"}, Count: 1},
					{States: []string{"Container is waiting"}, Count: 1},
					{States: []string{"Container is started, readiness unknown"}, Count: 1},
					{States: []string{"Container is not ready"}, Count: 1},
					{States: []string{"Container is ready"}, Count: 1},
					{States: []string{"Container is not ready"}, Count: 1},
					{States: []string{"Container exited successfully"}, Count: 1},
					{States: []string{"Resource is deleted"}, Count: 1},
				},
				TotalTransitions: 7,
			},
		},
		{
			name: "container crash loop folds 4-state tandem cycle with exact repeat count",
			revs: revsFromStates(
				"Container status is unavailable",
				"Container is waiting",
				"Container is started, readiness unknown",
				"Container is not ready",
				"Container exited with error",
				"Container is waiting",
				"Container is started, readiness unknown",
				"Container is not ready",
				"Container exited with error",
				"Container is waiting",
				"Container is started, readiness unknown",
				"Container is not ready",
				"Container exited with error",
				"Container is waiting",
				"Container is started, readiness unknown",
				"Container is not ready",
				"Container exited with error",
				"Container is waiting",
				"Container is started, readiness unknown",
				"Container is not ready",
				"Container exited with error",
				"Container is waiting",
			),
			want: RevisionStateSummary{
				Prefix: []RevisionStateStep{
					{States: []string{"Container status is unavailable"}, Count: 1},
					{
						States: []string{
							"Container is waiting",
							"Container is started, readiness unknown",
							"Container is not ready",
							"Container exited with error",
						},
						Count: 5,
					},
					{States: []string{"Container is waiting"}, Count: 1},
				},
				TotalTransitions: 21,
			},
		},
		{
			name: "2-state oscillation prefers primitive period 2 over multiple period 4",
			revs: revsFromStates(
				"Condition is True",
				"Condition is False",
				"Condition is True",
				"Condition is False",
				"Condition is True",
				"Condition is False",
				"Condition is True",
				"Condition is False",
			),
			want: RevisionStateSummary{
				Prefix: []RevisionStateStep{
					{
						States: []string{"Condition is True", "Condition is False"},
						Count:  4,
					},
				},
				TotalTransitions: 7,
			},
		},
		{
			name: "6-state container cycle with readiness recovery before crash folds cleanly",
			revs: revsFromStates(
				"Container is waiting",
				"Container is started, readiness unknown",
				"Container is not ready",
				"Container is ready",
				"Container is not ready",
				"Container exited with error",
				"Container is waiting",
				"Container is started, readiness unknown",
				"Container is not ready",
				"Container is ready",
				"Container is not ready",
				"Container exited with error",
			),
			want: RevisionStateSummary{
				Prefix: []RevisionStateStep{
					{
						States: []string{
							"Container is waiting",
							"Container is started, readiness unknown",
							"Container is not ready",
							"Container is ready",
							"Container is not ready",
							"Container exited with error",
						},
						Count: 2,
					},
				},
				TotalTransitions: 11,
			},
		},
		{
			name: "time window includes effective revision at startTime and excludes superseded and future revisions",
			revs: []cel.RevisionInfo{
				{ChangedTime: 0, State: "Pod status is unavailable"},
				{ChangedTime: 1000, State: "Pod is pending"},
				{ChangedTime: 2000, State: "Pod is running"},
				{ChangedTime: 3000, State: "Pod has failed"},
				{ChangedTime: 4000, State: "Resource is deleted"},
			},
			startTime: ts(2500),
			endTime:   ts(3500),
			want: RevisionStateSummary{
				Prefix: []RevisionStateStep{
					{States: []string{"Pod is running"}, Count: 1},
					{States: []string{"Pod has failed"}, Count: 1},
				},
				TotalTransitions: 1,
			},
		},
		{
			name: "revision changed at exact startTime supersedes earlier revision",
			revs: []cel.RevisionInfo{
				{ChangedTime: 1000, State: "Pod is pending"},
				{ChangedTime: 2000, State: "Pod is running"},
				{ChangedTime: 3000, State: "Pod has succeeded"},
			},
			startTime: ts(2000),
			endTime:   ts(3000),
			want: RevisionStateSummary{
				Prefix: []RevisionStateStep{
					{States: []string{"Pod is running"}, Count: 1},
					{States: []string{"Pod has succeeded"}, Count: 1},
				},
				TotalTransitions: 1,
			},
		},
		{
			name: "all revisions strictly after endTime returns empty summary",
			revs: []cel.RevisionInfo{
				{ChangedTime: 5000, State: "Pod is pending"},
			},
			startTime: ts(1000),
			endTime:   ts(4000),
			want:      RevisionStateSummary{},
		},
		{
			name: "startTime after endTime returns empty summary",
			revs: []cel.RevisionInfo{
				{ChangedTime: 2000, State: "Pod is running"},
			},
			startTime: ts(3000),
			endTime:   ts(1000),
			want:      RevisionStateSummary{},
		},
		{
			name: "irregular transitions exceeding slot budget elide middle and preserve unseen middle states",
			revs: revsFromStates(
				"S1", "S2", "S3", "S4", "S5", "S6", "S7", "S8",
				"S1", "Container exited with error", "S2", "S3",
				"S13", "S14", "S15", "S16",
			),
			want: RevisionStateSummary{
				Prefix: []RevisionStateStep{
					{States: []string{"S1"}, Count: 1},
					{States: []string{"S2"}, Count: 1},
					{States: []string{"S3"}, Count: 1},
					{States: []string{"S4"}, Count: 1},
					{States: []string{"S5"}, Count: 1},
					{States: []string{"S6"}, Count: 1},
					{States: []string{"S7"}, Count: 1},
					{States: []string{"S8"}, Count: 1},
				},
				Suffix: []RevisionStateStep{
					{States: []string{"S13"}, Count: 1},
					{States: []string{"S14"}, Count: 1},
					{States: []string{"S15"}, Count: 1},
					{States: []string{"S16"}, Count: 1},
				},
				OmittedStates:    []string{"Container exited with error"},
				TotalTransitions: 15,
				Truncated:        true,
			},
		},
		{
			name: "more than maxOmittedDistinctStates unseen middle states sets ExtraOmittedStateCount",
			revs: revsFromStates(
				"S1", "S2", "S3", "S4", "S5", "S6", "S7", "S8",
				"M1", "M2", "M3", "M4", "M5", "M6", "M7",
				"E1", "E2", "E3", "E4",
			),
			want: RevisionStateSummary{
				Prefix: []RevisionStateStep{
					{States: []string{"S1"}, Count: 1},
					{States: []string{"S2"}, Count: 1},
					{States: []string{"S3"}, Count: 1},
					{States: []string{"S4"}, Count: 1},
					{States: []string{"S5"}, Count: 1},
					{States: []string{"S6"}, Count: 1},
					{States: []string{"S7"}, Count: 1},
					{States: []string{"S8"}, Count: 1},
				},
				Suffix: []RevisionStateStep{
					{States: []string{"E1"}, Count: 1},
					{States: []string{"E2"}, Count: 1},
					{States: []string{"E3"}, Count: 1},
					{States: []string{"E4"}, Count: 1},
				},
				OmittedStates:          []string{"M1", "M2", "M3", "M4", "M5"},
				ExtraOmittedStateCount: 2,
				TotalTransitions:       18,
				Truncated:              true,
			},
		},
		{
			name: "two adjacent 8-state cycles with no middle step between them are not marked truncated",
			revs: revsFromStates(
				"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8",
				"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8",
				"B1", "B2", "B3", "B4", "B5", "B6", "B7", "B8",
				"B1", "B2", "B3", "B4", "B5", "B6", "B7", "B8",
			),
			want: RevisionStateSummary{
				Prefix: []RevisionStateStep{
					{States: []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8"}, Count: 2},
					{States: []string{"B1", "B2", "B3", "B4", "B5", "B6", "B7", "B8"}, Count: 2},
				},
				TotalTransitions: 31,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeRevisionStates(tc.revs, tc.startTime, tc.endTime)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SummarizeRevisionStates() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
