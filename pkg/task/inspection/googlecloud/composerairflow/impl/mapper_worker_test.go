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

package composerairflow_impl

import (
	"context"
	"testing"
	"time"
	"unique"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"

	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func TestAirflowWorkerMapperTask_ProcessLogByGroup(t *testing.T) {
	timestamp := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "Worker basic identification and TaskInstance extraction",
			input: testlog.NewMockLog(
				timestamp,
				gcpcommon.GCPMainMessageFieldSet{MainMessage: "Executing task"},
				composerairflow.ComposerFieldSet{
					WorkerID: "airflow-worker-abc",
				},
				composerairflow.ComposerWorkerTaskInstanceFieldSet{
					TaskInstance: composerairflow.NewAirflowTaskInstance(
						"my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "airflow-worker-abc", composerairflow.TASKINSTANCE_RUNNING,
					),
				},
			),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				envPath := composerairflow.MustAirflowTimeline(ctx, "test-environment")
				workerPath := composerairflow.MustAirflowComponentTimeline(ctx, envPath, "airflow-worker-abc")
				ti := composerairflow.NewAirflowTaskInstance("my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "airflow-worker-abc", composerairflow.TASKINSTANCE_RUNNING)
				runPath := composerairflow.MustAirflowDAGRunTimeline(ctx, envPath, ti.DagId(), ti.RunId())
				tiPath := composerairflow.MustAirflowTaskInstanceTimeline(ctx, runPath, "task_id_1+1")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(workerPath).
					HasRevision(tiPath, &khifilev6.StagingRevision{
						ChangedTime:  timestamp,
						ResourceBody: mustParseYAMLNode(t, ti.ToYaml()),
						Principal:    "airflow-worker",
						VerbType:     composerairflow.VerbComposerTaskInstanceRunning,
						StateType:    composerairflow.RevisionStateComposerTiRunning,
					}, cmp.AllowUnexported(
						structured.StandardMapNode{},
						structured.StandardScalarNode[string]{},
						structured.StandardScalarNode[any]{},
						structured.StandardSequenceNode{},
						unique.Handle[string]{},
					))
			},
		},
		{
			name: "Worker log without TaskInstance",
			input: testlog.NewMockLog(
				timestamp,
				gcpcommon.GCPMainMessageFieldSet{MainMessage: "Worker Heartbeat"},
				composerairflow.ComposerFieldSet{
					WorkerID: "airflow-worker-abc",
				},
			),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				envPath := composerairflow.MustAirflowTimeline(ctx, "test-environment")
				workerPath := composerairflow.MustAirflowComponentTimeline(ctx, envPath, "airflow-worker-abc")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(workerPath)
			},
		},
		{
			name: "Worker TaskInstance with none status generates event",
			input: testlog.NewMockLog(
				timestamp,
				gcpcommon.GCPMainMessageFieldSet{MainMessage: "Any user task log"},
				composerairflow.ComposerFieldSet{
					WorkerID: "airflow-worker-abc",
				},
				composerairflow.ComposerWorkerTaskInstanceFieldSet{
					TaskInstance: composerairflow.NewAirflowTaskInstance(
						"my_dag", "task_id_1", "2023-01-01T00:00:00Z", "-1", "airflow-worker-abc", composerairflow.TASKINSTANCE_NONE,
					),
				},
			),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				envPath := composerairflow.MustAirflowTimeline(ctx, "test-environment")
				workerPath := composerairflow.MustAirflowComponentTimeline(ctx, envPath, "airflow-worker-abc")
				runPath := composerairflow.MustAirflowDAGRunTimeline(ctx, envPath, "my_dag", "2023-01-01T00:00:00Z")
				tiPath := composerairflow.MustAirflowTaskInstanceTimeline(ctx, runPath, "task_id_1")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(workerPath).
					HasEvent(tiPath)
			},
		},
	}

	mapper := &workerLogToTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewTestBuilder(id.NewGenerator())
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

			taskDependentValues := typedmap.NewTypedMap()
			typedmap.Set(taskDependentValues, typedmap.NewTypedKey[string](composercluster.InputComposerEnvironmentNameTaskID.ReferenceIDString()), "test-environment")
			ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskDependentValues)

			cs, _, err := mapper.ProcessLogByGroup(ctx, tc.input, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}

			tc.assert(t, ctx, cs)
		})
	}
}
