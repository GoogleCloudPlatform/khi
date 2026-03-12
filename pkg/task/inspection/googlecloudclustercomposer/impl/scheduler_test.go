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

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestAirflowSchedulerMapperTask_ProcessLogByGroup(t *testing.T) {
	timestamp := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name      string
		logs      []*log.Log
		asserters [][]testchangeset.ChangeSetAsserter
	}{
		{
			name: "Scheduler basic identification and TaskInstance extraction",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: timestamp},
					&log.MainMessageFieldSet{MainMessage: "Processing /app/models.py"},
					&googlecloudclustercomposer_contract.ComposerFieldSet{
						SchedulerID: "airflow-scheduler-7b5f",
					},
					&googlecloudclustercomposer_contract.ComposerTaskInstanceFieldSet{
						TaskInstance: googlecloudclustercomposer_contract.NewAirflowTaskInstance(
							"my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "worker-1", googlecloudclustercomposer_contract.TASKINSTANCE_SUCCESS,
						),
					},
				),
			},
			asserters: [][]testchangeset.ChangeSetAsserter{
				{
					// Scheduler Identity Event
					&testchangeset.HasEvent{
						ResourcePath: resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowScheduler", "cluster-scope", "airflow-scheduler-7b5f", "airflow-scheduler").Path,
					},
					// TaskInstance Revision
					&testchangeset.HasRevision{
						ResourcePath: googlecloudclustercomposer_contract.NewAirflowTaskInstance("my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "worker-1", googlecloudclustercomposer_contract.TASKINSTANCE_SUCCESS).ResourcePath().Path,
						WantRevision: history.StagingResourceRevision{
							Verb:       enum.RevisionVerbComposerTaskInstanceSuccess, // Based on "success" mapped to verb
							State:      enum.RevisionStateComposerTiSuccess,
							ChangeTime: timestamp,
							Requestor:  "airflow-scheduler",
							Body:       googlecloudclustercomposer_contract.NewAirflowTaskInstance("my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "worker-1", googlecloudclustercomposer_contract.TASKINSTANCE_SUCCESS).ToYaml(),
						},
					},
					&testchangeset.HasEvent{
						ResourcePath: googlecloudclustercomposer_contract.NewAirflowTaskInstance("my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "worker-1", googlecloudclustercomposer_contract.TASKINSTANCE_SUCCESS).ResourcePath().Path,
					},
					// Log Summary
					&testchangeset.HasLogSummary{WantLogSummary: "Processing /app/models.py"},
				},
			},
		},
		{
			name: "Zombie task adds event to worker",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: timestamp},
					&log.MainMessageFieldSet{MainMessage: "Detected zombie task"},
					&googlecloudclustercomposer_contract.ComposerFieldSet{
						SchedulerID: "airflow-scheduler-7b5f",
					},
					&googlecloudclustercomposer_contract.ComposerTaskInstanceFieldSet{
						TaskInstance: googlecloudclustercomposer_contract.NewAirflowTaskInstance(
							"my_dag", "task_id_zombie", "2023-01-01T00:00:00Z", "1", "worker-bad", googlecloudclustercomposer_contract.TASKINSTANCE_ZOMBIE,
						),
					},
				),
			},
			asserters: [][]testchangeset.ChangeSetAsserter{
				{
					// Scheduler Identity Event
					&testchangeset.HasEvent{
						ResourcePath: resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowScheduler", "cluster-scope", "airflow-scheduler-7b5f", "airflow-scheduler").Path,
					},
					// Worker Event for Zombie
					&testchangeset.HasEvent{
						ResourcePath: googlecloudclustercomposer_contract.NewAirflowWorker("worker-bad").ResourcePath().Path,
					},
					// Log Summary
					&testchangeset.HasLogSummary{WantLogSummary: "Detected zombie task"},
				},
			},
		},
		{
			name: "Scheduler log without TaskInstance",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: timestamp},
					&log.MainMessageFieldSet{MainMessage: "Heartbeat"},
					&googlecloudclustercomposer_contract.ComposerFieldSet{
						SchedulerID: "airflow-scheduler-7b5f",
					},
				),
			},
			asserters: [][]testchangeset.ChangeSetAsserter{
				{
					// Scheduler Identity Event only
					&testchangeset.HasEvent{
						ResourcePath: resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowScheduler", "cluster-scope", "airflow-scheduler-7b5f", "airflow-scheduler").Path,
					},
					// Log Summary
					&testchangeset.HasLogSummary{WantLogSummary: "Heartbeat"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapper := airflowSchedulerLogToTimelineMapperSetting{
				targetLogType: enum.LogTypeComposerEnvironment,
			}

			state := struct{}{}
			for i, l := range tc.logs {
				changeSetAsserters := tc.asserters[i]
				cs := history.NewChangeSet(l)

				nextState, err := mapper.ProcessLogByGroup(context.Background(), l, cs, nil, state)
				if err != nil {
					t.Fatalf("ProcessLogByGroup failed at message %d: %v", i, err)
				}
				for _, asserter := range changeSetAsserters {
					asserter.Assert(t, cs)
				}
				state = nextState
			}

			if diff := cmp.Diff(struct{}{}, state); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
