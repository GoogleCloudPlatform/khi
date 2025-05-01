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

package v2timelinegrouping

import (
	"context"
	"testing"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	inspection_task_test "github.com/GoogleCloudPlatform/khi/pkg/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	common_k8saudit_fieldextactor "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/fieldextractor"
	common_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/v2commonlogparse"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	task_test "github.com/GoogleCloudPlatform/khi/pkg/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestGroupByTimelineTask(t *testing.T) {
	t.Run("it ignores dryrun mode", func(t *testing.T) {
		ctx := inspection_task_test.WithDefaultTestInspectionTaskContext(context.Background())
		result, _, err := inspection_task_test.RunInspectionTask(ctx, Task, inspection_task_interface.TaskModeDryRun, map[string]any{},
			task_test.NewTaskDependencyValuePair(common_k8saudit_taskid.CommonLogParseTaskID.Ref(), nil))
		if err != nil {
			t.Error(err)
		}
		if result != nil {
			t.Errorf("the result is not valid")
		}
	})

	t.Run("it grups logs by timleines", func(t *testing.T) {
		baseLog := `insertId: foo
protoPayload:
  authenticationInfo:
    principalEmail: user@example.com
  methodName: io.k8s.core.v1.pods.create
  status:
    code: 200
timestamp: 2024-01-01T00:00:00+09:00`
		logOpts := [][]testlog.TestLogOpt{
			{
				testlog.StringField("protoPayload.resourceName", "core/v1/namespaces/default/pods/foo"),
			},
			{
				testlog.StringField("protoPayload.resourceName", "core/v1/namespaces/default/pods/foo"),
			},
			{
				testlog.StringField("protoPayload.resourceName", "core/v1/namespaces/default/pods/bar"),
			},
		}
		expectedLogCounts := map[string]int{
			"core/v1#pod#default#foo": 2,
			"core/v1#pod#default#bar": 1,
		}
		tl := testlog.New(testlog.BaseYaml(baseLog))
		logs := []*log.LogEntity{}
		for _, opt := range logOpts {
			logs = append(logs, tl.With(opt...).MustBuildLogEntity(&log.UnreachableCommonFieldExtractor{}))
		}

		ctx := inspection_task_test.WithDefaultTestInspectionTaskContext(context.Background())
		result, _, err := inspection_task_test.RunInspectionTaskWithDependency(ctx, Task, []task.UntypedTask{
			v2commonlogparse.Task,
			task_test.StubTaskFromReferenceID(common_k8saudit_taskid.CommonAuitLogSource, &types.AuditLogParserLogSource{
				Logs: logs,
				Extractor: &common_k8saudit_fieldextactor.StubFieldExtractor{
					Extractor: func(ctx context.Context, log *log.LogEntity) (*types.AuditLogParserInput, error) {
						resourceName := log.GetStringOrDefault("protoPayload.resourceName", "")
						if resourceName == "core/v1/namespaces/default/pods/foo" {
							return &types.AuditLogParserInput{
								Log: log,
								Operation: &model.KubernetesObjectOperation{
									APIVersion: "core/v1",
									PluralKind: "pods",
									Namespace:  "default",
									Name:       "foo",
									Verb:       enum.RevisionVerbCreate,
								},
							}, nil
						} else {
							return &types.AuditLogParserInput{
								Log: log,
								Operation: &model.KubernetesObjectOperation{
									APIVersion: "core/v1",
									PluralKind: "pods",
									Namespace:  "default",
									Name:       "bar",
									Verb:       enum.RevisionVerbCreate,
								},
							}, nil
						}
					},
				},
			}, nil),
		}, inspection_task_interface.TaskModeRun, map[string]any{})
		if err != nil {
			t.Error(err)
		}
		for _, result := range result {
			if count, found := expectedLogCounts[result.TimelineResourcePath]; !found {
				t.Errorf("unexpected timeline %s not found", result.TimelineResourcePath)
			} else if count != len(result.PreParsedLogs) {
				t.Errorf("expected log count is not matching in a timeline:%s", result.TimelineResourcePath)
			}
		}
	})
}
