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
	"fmt"
	"slices"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

var pathFilterTestID = structured.CompileFieldPath("id")

func TestDefineLogFilterTask(t *testing.T) {
	sourceTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("source")
	keepIDsTaskID := taskid.NewDefaultImplementationID[[]string]("keep-ids")
	task := DefineLogFilterTask(taskid.NewDefaultImplementationID[[]*log.Log]("dest"), sourceTaskID.Ref(), func(b *coretask.Binder) LogFilterFunc {
		keepIDs := coretask.Use(b, keepIDsTaskID.Ref())
		return func(ctx context.Context, l *log.Log) bool {
			return slices.Contains(keepIDs.Get(ctx), l.ReadStringOrDefault(pathFilterTestID, "unknown"))
		}
	}, coretask.WithTaskDescription("filters logs by ID"))
	if got := typedmap.GetOrDefault(task.Labels(), coretask.LabelKeyTaskDescription, ""); got != "filters logs by ID" {
		t.Errorf("LabelKeyTaskDescription = %q, want %q", got, "filters logs by ID")
	}
	wantInputs := []string{"required source", "required keep-ids"}

	testCases := []struct {
		name       string
		taskMode   inspectioncore.InspectionTaskModeType
		logYAMLs   []string
		keepIDs    []string
		wantLogIDs []string
	}{
		{
			name:       "returns an empty slice for empty logs on run mode",
			taskMode:   inspectioncore.TaskModeRun,
			logYAMLs:   []string{},
			keepIDs:    []string{},
			wantLogIDs: []string{},
		},
		{
			name:     "keeps the logs selected by the input declared in bind on run mode",
			taskMode: inspectioncore.TaskModeRun,
			logYAMLs: []string{
				`id: foo`,
				`id: bar`,
				`id: qux`,
			},
			keepIDs:    []string{"foo", "qux"},
			wantLogIDs: []string{"foo", "qux"},
		},
		{
			name:     "preserves order when filtering many logs concurrently",
			taskMode: inspectioncore.TaskModeRun,
			logYAMLs: func() []string {
				yamls := make([]string, 100)
				for i := 0; i < 100; i++ {
					yamls[i] = fmt.Sprintf("id: item-%03d", i)
				}
				return yamls
			}(),
			keepIDs: func() []string {
				ids := make([]string, 50)
				for i := 0; i < 50; i++ {
					ids[i] = fmt.Sprintf("item-%03d", i*2)
				}
				return ids
			}(),
			wantLogIDs: func() []string {
				ids := make([]string, 50)
				for i := 0; i < 50; i++ {
					ids[i] = fmt.Sprintf("item-%03d", i*2)
				}
				return ids
			}(),
		},
		{
			name:     "returns an empty slice without filtering on dry run",
			taskMode: inspectioncore.TaskModeDryRun,
			logYAMLs: []string{
				`id: foo`,
				`id: bar`,
				`id: qux`,
			},
			keepIDs:    []string{"foo"},
			wantLogIDs: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(wantInputs, describeInputs(task.Inputs())); diff != "" {
				t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
			}

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			logs := []*log.Log{}
			for _, logYAML := range tc.logYAMLs {
				logs = append(logs, mustNewLogFromYAML(t, ctx, logYAML))
			}

			result, _, err := inspectiontest.Run(t, ctx, task, tc.taskMode, map[string]any{},
				tasktest.Given(sourceTaskID.Ref(), logs),
				tasktest.Given(keepIDsTaskID.Ref(), tc.keepIDs),
			)
			if err != nil {
				t.Fatalf("Run() returned an unexpected error: %v", err)
			}

			logIDs := []string{}
			for _, resultLog := range result {
				logIDs = append(logIDs, resultLog.ReadStringOrDefault(pathFilterTestID, "unknown"))
			}

			if diff := cmp.Diff(tc.wantLogIDs, logIDs); diff != "" {
				t.Errorf("log IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
