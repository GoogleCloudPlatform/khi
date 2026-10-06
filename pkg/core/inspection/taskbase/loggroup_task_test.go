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

var pathGroupTestID = structured.CompileFieldPath("id")

func TestDefineLogGrouperTask(t *testing.T) {
	sourceTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("source")
	prefixTaskID := taskid.NewDefaultImplementationID[string]("prefix")
	task := DefineLogGrouperTask(taskid.NewDefaultImplementationID[LogGroupMap]("dest"), sourceTaskID.Ref(), func(b *coretask.Binder) LogGrouperFunc {
		prefix := coretask.Use(b, prefixTaskID.Ref())
		return func(ctx context.Context, l *log.Log) string {
			return prefix.Get(ctx) + l.ReadStringOrDefault(pathGroupTestID, "unknown")[:1]
		}
	}, coretask.WithTaskDescription("groups logs by ID prefix"))
	if got := typedmap.GetOrDefault(task.Labels(), coretask.LabelKeyTaskDescription, ""); got != "groups logs by ID prefix" {
		t.Errorf("LabelKeyTaskDescription = %q, want %q", got, "groups logs by ID prefix")
	}
	wantInputs := []string{"required source", "required prefix"}

	testCases := []struct {
		name       string
		taskMode   inspectioncore.InspectionTaskModeType
		logYAMLs   []string
		wantGroups map[string][]string
	}{
		{
			name:     "groups logs with the key built from an input declared in bind",
			taskMode: inspectioncore.TaskModeRun,
			logYAMLs: []string{
				`id: foo`,
				`id: bar`,
				`id: qux`,
				`id: quux`,
			},
			wantGroups: map[string][]string{
				"prefix-f": {"foo"},
				"prefix-b": {"bar"},
				"prefix-q": {"qux", "quux"},
			},
		},
		{
			name:       "returns an empty map for empty logs",
			taskMode:   inspectioncore.TaskModeRun,
			logYAMLs:   []string{},
			wantGroups: map[string][]string{},
		},
		{
			name:     "returns an empty map on dry run",
			taskMode: inspectioncore.TaskModeDryRun,
			logYAMLs: []string{
				`id: foo`,
			},
			wantGroups: map[string][]string{},
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
				tasktest.Given(prefixTaskID.Ref(), "prefix-"),
			)
			if err != nil {
				t.Fatalf("Run() returned an unexpected error: %v", err)
			}

			gotGroups := map[string][]string{}
			for key, group := range result {
				for _, l := range group.Logs {
					gotGroups[key] = append(gotGroups[key], l.ReadStringOrDefault(pathGroupTestID, "unknown"))
				}
			}
			if diff := cmp.Diff(tc.wantGroups, gotGroups); diff != "" {
				t.Errorf("grouped log IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
