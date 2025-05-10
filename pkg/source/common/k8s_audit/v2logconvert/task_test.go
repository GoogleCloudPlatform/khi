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

package v2logconvert

import (
	"context"
	"testing"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/ioconfig"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	inspection_task_test "github.com/GoogleCloudPlatform/khi/pkg/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	common_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	gcp_log "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/log"
	task_test "github.com/GoogleCloudPlatform/khi/pkg/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestLogFillerTask(t *testing.T) {
	builder := history.NewBuilder(&ioconfig.IOConfig{})
	baseLog := `protoPayload:
  authenticationInfo:
    principalEmail: user@example.com
  methodName: io.k8s.core.v1.pods.create
  resourceName: core/v1/namespaces/default/pods/my-pod
  response:
    '@type': core.k8s.io/v1.Pod
    foo: bar
  status:
    code: 0
timestamp: "2024-01-01T00:00:00+09:00"`
	logOpts := [][]testlog.TestLogOpt{
		{
			testlog.StringField("insertId", "insertid-1"),
		},
		{
			testlog.StringField("insertId", "insertid-2"),
			testlog.StringField("timestamp", "2024-01-01T00:01:00+09:00"),
		},
		{
			testlog.StringField("insertId", "insertid-3"),
			testlog.StringField("timestamp", "2024-01-01T00:02:00+09:00"),
		},
	}
	logs := []*log.Log{}
	for _, opt := range logOpts {
		logs = append(logs, testlog.New(testlog.YAML(baseLog)).With(opt...).MustBuildLogEntity(&gcp_log.GCPCommonFieldSetReader{}, &gcp_log.GCPMainMessageFieldSetReader{}))
	}

	ctx := inspection_task_test.WithDefaultTestInspectionTaskContext(context.Background())
	_, _, err := inspection_task_test.RunInspectionTask(ctx, Task, inspection_task_interface.TaskModeRun, map[string]any{},
		task_test.NewTaskDependencyValuePair(inspection_task.BuilderGeneratorTaskID.Ref(), builder),
		task_test.NewTaskDependencyValuePair(common_k8saudit_taskid.CommonAuitLogSource, &types.AuditLogParserLogSource{
			Logs:      logs,
			Extractor: nil,
		}))

	if err != nil {
		t.Fatal(err.Error())
	}
	for i := 0; i < len(logs); i++ {
		logId := logs[i].ID
		_, err := builder.GetLog(logId)
		if err != nil {
			t.Errorf("failed to get log %s", logId)
		}
	}
}
