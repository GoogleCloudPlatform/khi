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

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

type stubAuditLogFieldExtractor struct {
	Extractor func(ctx context.Context, log *log.Log) (*commonlogk8saudit_contract.AuditLogParserInput, error)
}

// ExtractFields implements commonlogk8saudit_contract.AuditLogFieldExtractor.
func (f *stubAuditLogFieldExtractor) ExtractFields(ctx context.Context, log *log.Log) (*commonlogk8saudit_contract.AuditLogParserInput, error) {
	return f.Extractor(ctx, log)
}

var _ commonlogk8saudit_contract.AuditLogFieldExtractor = (*stubAuditLogFieldExtractor)(nil)

func TestGroupByTimelineTask(t *testing.T) {
	t.Run("it ignores dryrun mode", func(t *testing.T) {
		ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
		result, _, err := inspectiontest.RunInspectionTask(ctx, TimelineGroupingTask, inspectioncore_contract.TaskModeDryRun, map[string]any{},
			tasktest.NewTaskDependencyValuePair(commonlogk8saudit_contract.CommonLogParseTaskID.Ref(), nil))
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
		result, _, err := inspectiontest.RunInspectionTaskWithDependency(ctx, TimelineGroupingTask, []coretask.UntypedTask{
			CommonLogParserTask,
			tasktest.StubTaskFromReferenceID(commonlogk8saudit_contract.CommonAuitLogSource, &commonlogk8saudit_contract.AuditLogParserLogSource{
				Logs: logs,
				Extractor: &stubAuditLogFieldExtractor{
					Extractor: func(ctx context.Context, log *log.Log) (*commonlogk8saudit_contract.AuditLogParserInput, error) {
						resourceName := log.ReadStringOrDefault("protoPayload.resourceName", "")
						if resourceName == "core/v1/namespaces/default/pods/foo" {
							return &commonlogk8saudit_contract.AuditLogParserInput{
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
							return &commonlogk8saudit_contract.AuditLogParserInput{
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
