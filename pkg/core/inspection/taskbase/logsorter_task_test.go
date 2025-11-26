// Copyright 2025 Google LLC
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

package inspectiontaskbase

import (
	"context"
	"testing"
	"time"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestLogSorterByTimeTask(t *testing.T) {
	testCases := []struct {
		name string
		mode inspectioncore_contract.InspectionTaskModeType
		logs []*log.CommonFieldSet
		want []*log.CommonFieldSet
	}{
		{
			name: "should return an empty list for empty log input on task run mode",
			mode: inspectioncore_contract.TaskModeRun,
			logs: []*log.CommonFieldSet{},
			want: make([]*log.CommonFieldSet, 0),
		},
		{
			name: "should return an empty list for empty log input on task dry run mode",
			mode: inspectioncore_contract.TaskModeDryRun,
			logs: []*log.CommonFieldSet{
				{
					DisplayID: "foo",
					Timestamp: time.Date(2025, 11, 21, 13, 16, 34, 0, time.UTC),
				},
				{
					DisplayID: "bar",
					Timestamp: time.Date(2025, 11, 21, 13, 16, 33, 0, time.UTC),
				},
				{
					DisplayID: "qux",
					Timestamp: time.Date(2025, 11, 21, 13, 16, 32, 0, time.UTC),
				},
			},
			want: make([]*log.CommonFieldSet, 0),
		},
		{
			name: "should return sorted logs on task run mode",
			mode: inspectioncore_contract.TaskModeRun,
			logs: []*log.CommonFieldSet{
				{
					DisplayID: "foo",
					Timestamp: time.Date(2025, 11, 21, 13, 16, 34, 0, time.UTC),
				},
				{
					DisplayID: "bar",
					Timestamp: time.Date(2025, 11, 21, 13, 16, 33, 0, time.UTC),
				},
				{
					DisplayID: "qux",
					Timestamp: time.Date(2025, 11, 21, 13, 16, 32, 0, time.UTC),
				},
			},
			want: []*log.CommonFieldSet{
				{
					DisplayID: "qux",
					Timestamp: time.Date(2025, 11, 21, 13, 16, 32, 0, time.UTC),
				},
				{
					DisplayID: "bar",
					Timestamp: time.Date(2025, 11, 21, 13, 16, 33, 0, time.UTC),
				},
				{
					DisplayID: "foo",
					Timestamp: time.Date(2025, 11, 21, 13, 16, 34, 0, time.UTC),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logs := []*log.Log{}
			for _, commonFieldSet := range tc.logs {
				l := log.NewLogWithFieldSetsForTest(commonFieldSet)
				logs = append(logs, l)
			}

			testSourceTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("source")
			testTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("dest")
			task := NewLogSorterByTimeTask(testTaskID, testSourceTaskID.Ref())

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			sortedLogs, _, err := inspectiontest.RunInspectionTask(ctx, task, tc.mode, map[string]any{}, tasktest.NewTaskDependencyValuePair(testSourceTaskID.Ref(), logs))
			if err != nil {
				t.Fatalf("RunInspectionTask returned an unexpected error: %v", err)
			}

			got := []*log.CommonFieldSet{}
			for _, l := range sortedLogs {
				fieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
				got = append(got, fieldSet)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("log sort mismatch (-want +got):\n%s", diff)
			}

		})
	}
}
