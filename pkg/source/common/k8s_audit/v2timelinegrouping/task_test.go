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

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	common_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/v2commonlogparse"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type stubAuditLogFieldExtractor struct {
	Extractor func(ctx context.Context, log *log.Log) (*types.AuditLogParserInput, error)
}

// ExtractFields implements types.AuditLogFieldExtractor.
func (f *stubAuditLogFieldExtractor) ExtractFields(ctx context.Context, log *log.Log) (*types.AuditLogParserInput, error) {
	return f.Extractor(ctx, log)
}

var _ types.AuditLogFieldExtractor = (*stubAuditLogFieldExtractor)(nil)

func TestGroupByTimelineTask(t *testing.T) {
	t.Run("it ignores dryrun mode", func(t *testing.T) {
		ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
		result, _, err := inspectiontest.RunInspectionTask(ctx, Task, inspectioncore_contract.TaskModeDryRun, map[string]any{},
			tasktest.NewTaskDependencyValuePair(common_k8saudit_taskid.CommonLogParseTaskID.Ref(), nil))
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
		tl := testlog.New(testlog.YAML(baseLog))
		logs := []*log.Log{}
		for _, opt := range logOpts {
			logs = append(logs, tl.With(opt...).MustBuildLogEntity())
		}

		ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
		result, _, err := inspectiontest.RunInspectionTaskWithDependency(ctx, Task, []coretask.UntypedTask{
			v2commonlogparse.Task,
			tasktest.StubTaskFromReferenceID(common_k8saudit_taskid.CommonAuitLogSource, &types.AuditLogParserLogSource{
				Logs: logs,
				Extractor: &stubAuditLogFieldExtractor{
					Extractor: func(ctx context.Context, log *log.Log) (*types.AuditLogParserInput, error) {
						resourceName := log.ReadStringOrDefault("protoPayload.resourceName", "")
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
		}, inspectioncore_contract.TaskModeRun, map[string]any{})
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
