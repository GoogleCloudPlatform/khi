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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestLogIngesterTask_DryRunMode(t *testing.T) {
	l := testlog.MustLogFromYAML("insertId: foo", &mockCommonLogFieldSetReader{})
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	inputTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("input")
	taskID := taskid.NewDefaultImplementationID[[]*log.Log]("test")
	task := NewLogIngesterTask(taskID, inputTaskID.Ref())

	result, _, err := inspectiontest.RunInspectionTask(ctx, task, inspectioncore_contract.TaskModeDryRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(inputTaskID.Ref(), []*log.Log{l}))
	if err != nil {
		t.Fatalf("RunInspectionTask returned an unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("LogIngesterTask returned a log result for dryrun mode")
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)
	_, err = builder.GetLog(l.ID)
	if err == nil {
		t.Errorf("LogIngesterTask must not write log to the builder when it run for dryrun, but it wrote a log.")
	}
}

func TestLogIngesterTask_RunMode(t *testing.T) {
	l := testlog.MustLogFromYAML("insertId: foo", &mockCommonLogFieldSetReader{})
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	inputTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("input")
	taskID := taskid.NewDefaultImplementationID[[]*log.Log]("test")
	task := NewLogIngesterTask(taskID, inputTaskID.Ref())

	result, _, err := inspectiontest.RunInspectionTask(ctx, task, inspectioncore_contract.TaskModeRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(inputTaskID.Ref(), []*log.Log{l}))
	if err != nil {
		t.Fatalf("RunInspectionTask returned an unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("LogIngesterTask didn't return a log result for run mode")
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)
	_, err = builder.GetLog(l.ID)
	if err != nil {
		t.Errorf("LogIngesterTask must write log to the builder when it run. err=%v", err)
	}
}
