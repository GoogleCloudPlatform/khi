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

package commonlogk8saudit_impl

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestCommonLogConvertTask(t *testing.T) {
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
		logs = append(logs, testlog.New(testlog.YAML(baseLog)).With(opt...).MustBuildLogEntity(&gcpqueryutil.GCPCommonFieldSetReader{}, &gcpqueryutil.GCPMainMessageFieldSetReader{}))
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)
	_, _, err := inspectiontest.RunInspectionTask(ctx, CommonLogConvertTask, inspectioncore_contract.TaskModeRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(commonlogk8saudit_contract.CommonAuitLogSource, &commonlogk8saudit_contract.AuditLogParserLogSource{
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
