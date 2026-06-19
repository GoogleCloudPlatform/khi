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

package googlecloudclustercomposer_impl

import (
	"context"
	"testing"
	"time"
	"unique"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestAirflowSchedulerMapperTask_ProcessLogByGroup(t *testing.T) {
	timestamp := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "Scheduler basic identification and TaskInstance extraction",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: timestamp},
				&googlecloudcommon_contract.GCPMainMessageFieldSet{MainMessage: "Processing /app/models.py"},
				&googlecloudclustercomposer_contract.ComposerFieldSet{
					SchedulerID: "airflow-scheduler-7b5f",
				},
				&googlecloudclustercomposer_contract.ComposerTaskInstanceFieldSet{
					TaskInstance: googlecloudclustercomposer_contract.NewAirflowTaskInstance(
						"my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "worker-1", googlecloudclustercomposer_contract.TASKINSTANCE_SUCCESS,
					),
				},
			),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				envPath := googlecloudclustercomposer_contract.MustComposerEnvironmentTimeline(ctx, "test-project", "test-environment")
				schedulerPath := googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowScheduler, "airflow-scheduler-7b5f")
				ti := googlecloudclustercomposer_contract.NewAirflowTaskInstance("my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "worker-1", googlecloudclustercomposer_contract.TASKINSTANCE_SUCCESS)
				runPath := googlecloudclustercomposer_contract.MustAirflowDAGRunTimeline(ctx, envPath, ti.DagId(), ti.RunId())
				tiPath := googlecloudclustercomposer_contract.MustAirflowTaskInstanceTimeline(ctx, runPath, "task_id_1+1")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(schedulerPath).
					HasEvent(tiPath).
					HasRevision(tiPath, &khifilev6.StagingRevision{
						ChangedTime:  timestamp,
						ResourceBody: mustParseYAMLNode(t, ti.ToYaml()),
						Principal:    "airflow-scheduler",
						VerbType:     googlecloudclustercomposer_contract.VerbComposerTaskInstanceSuccess,
						StateType:    googlecloudclustercomposer_contract.RevisionStateComposerTiSuccess,
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
			name: "Zombie task adds event to worker",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: timestamp},
				&googlecloudcommon_contract.GCPMainMessageFieldSet{MainMessage: "Detected zombie task"},
				&googlecloudclustercomposer_contract.ComposerFieldSet{
					SchedulerID: "airflow-scheduler-7b5f",
				},
				&googlecloudclustercomposer_contract.ComposerTaskInstanceFieldSet{
					TaskInstance: googlecloudclustercomposer_contract.NewAirflowTaskInstance(
						"my_dag", "task_id_zombie", "2023-01-01T00:00:00Z", "1", "worker-bad", googlecloudclustercomposer_contract.TASKINSTANCE_ZOMBIE,
					),
				},
			),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				envPath := googlecloudclustercomposer_contract.MustComposerEnvironmentTimeline(ctx, "test-project", "test-environment")
				schedulerPath := googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowScheduler, "airflow-scheduler-7b5f")
				workerPath := googlecloudclustercomposer_contract.MustAirflowWorkerTimeline(ctx, envPath, "worker-bad")
				ti := googlecloudclustercomposer_contract.NewAirflowTaskInstance("my_dag", "task_id_zombie", "2023-01-01T00:00:00Z", "1", "worker-bad", googlecloudclustercomposer_contract.TASKINSTANCE_ZOMBIE)
				runPath := googlecloudclustercomposer_contract.MustAirflowDAGRunTimeline(ctx, envPath, ti.DagId(), ti.RunId())
				tiPath := googlecloudclustercomposer_contract.MustAirflowTaskInstanceTimeline(ctx, runPath, "task_id_zombie+1")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(schedulerPath).
					HasEvent(workerPath).
					HasEvent(tiPath).
					HasRevision(tiPath, &khifilev6.StagingRevision{
						ChangedTime:  timestamp,
						ResourceBody: mustParseYAMLNode(t, ti.ToYaml()),
						Principal:    "airflow-scheduler",
						VerbType:     googlecloudclustercomposer_contract.VerbComposerTaskInstanceZombie,
						StateType:    googlecloudclustercomposer_contract.RevisionStateComposerTiZombie,
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
			name: "Scheduler log without TaskInstance",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: timestamp},
				&googlecloudcommon_contract.GCPMainMessageFieldSet{MainMessage: "Heartbeat"},
				&googlecloudclustercomposer_contract.ComposerFieldSet{
					SchedulerID: "airflow-scheduler-7b5f",
				},
			),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				envPath := googlecloudclustercomposer_contract.MustComposerEnvironmentTimeline(ctx, "test-project", "test-environment")
				schedulerPath := googlecloudclustercomposer_contract.MustAirflowComponentTimeline(ctx, envPath, googlecloudclustercomposer_contract.TimelineTypeAirflowScheduler, "airflow-scheduler-7b5f")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(schedulerPath)
			},
		},
	}

	mapper := &schedulerLogToTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			taskDependentValues := typedmap.NewTypedMap()
			typedmap.Set(taskDependentValues, typedmap.NewTypedKey[googlecloudk8scommon_contract.GoogleCloudClusterIdentity](googlecloudclustercomposer_contract.ClusterIdentityTaskID.ReferenceIDString()), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{ProjectID: "test-project"})
			typedmap.Set(taskDependentValues, typedmap.NewTypedKey[string](googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.ReferenceIDString()), "test-environment")
			ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskDependentValues)

			cs, _, err := mapper.ProcessLogByGroup(ctx, tc.input, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}

			tc.assert(t, ctx, cs)
		})
	}
}

func mustParseYAMLNode(t *testing.T, yamlStr string) structured.Node {
	t.Helper()
	node, err := structured.FromYAML(yamlStr)
	if err != nil {
		t.Fatalf("failed to parse yaml node in test: %v", err)
	}
	return node
}
