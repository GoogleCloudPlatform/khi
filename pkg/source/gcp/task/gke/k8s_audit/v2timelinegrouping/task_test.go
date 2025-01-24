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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/k8saudittask"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/v2commonlogparse"
	base_task "github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testtask"
)

func TestGroupByTimelineTask(t *testing.T) {
	t.Run("it ignores dryrun mode", func(t *testing.T) {
		result, err := testtask.RunSingleTask[struct{}](Task, task.TaskModeDryRun,
			testtask.PriorTaskResultFromID(task.MetadataVariableName, metadata.NewSet()),
			testtask.PriorTaskResultFromID(k8saudittask.CommonLogParseTaskID, struct{}{}))
		if err != nil {
			t.Error(err)
		}
		if result != struct{}{} {
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

		result, err := testtask.RunMultipleTask[[]*types.TimelineGrouperResult](Task, []base_task.Definition{v2commonlogparse.Task}, task.TaskModeRun,
			testtask.PriorTaskResultFromID(task.MetadataVariableName, metadata.NewSet()),
			testtask.PriorTaskResultFromID(k8saudittask.K8sAuditQueryTaskID, logs))
		if err != nil {
			t.Error(err)
		}
		for _, result := range result {
			if count, found := expectedLogCounts[result.TimelineResourcePath]; !found {
				t.Errorf("unexpected timeline %s not found", result.TimelineResourcePath)
			} else {
				if count != len(result.PreParsedLogs) {
					t.Errorf("expected log count is not matching in a timeline:%s", result.TimelineResourcePath)
				}
			}
		}
	})
}
