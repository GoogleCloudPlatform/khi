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
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestAirflowWorkerMapperTask_ProcessLogByGroup(t *testing.T) {
	timestamp := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name      string
		logs      []*log.Log
		asserters [][]testchangeset.ChangeSetAsserter
	}{
		{
			name: "Worker basic identification and TaskInstance extraction",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: timestamp},
					&log.MainMessageFieldSet{MainMessage: "Executing task"},
					&googlecloudclustercomposer_contract.ComposerWorkerFieldSet{
						WorkerID: "airflow-worker-abc",
					},
					&googlecloudclustercomposer_contract.ComposerWorkerTaskInstanceFieldSet{
						TaskInstance: googlecloudclustercomposer_contract.NewAirflowTaskInstance(
							"my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "airflow-worker-abc", googlecloudclustercomposer_contract.TASKINSTANCE_RUNNING,
						),
					},
				),
			},
			asserters: [][]testchangeset.ChangeSetAsserter{
				{
					// Worker Identity Event
					&testchangeset.HasEvent{
						ResourcePath: googlecloudclustercomposer_contract.NewAirflowWorker("airflow-worker-abc").ResourcePath().Path,
					},
					// TaskInstance Revision
					&testchangeset.HasRevision{
						ResourcePath: googlecloudclustercomposer_contract.NewAirflowTaskInstance("my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "airflow-worker-abc", googlecloudclustercomposer_contract.TASKINSTANCE_RUNNING).ResourcePath().Path,
						WantRevision: history.StagingResourceRevision{
							Verb:       enum.RevisionVerbComposerTaskInstanceRunning, // Based on "running"
							State:      enum.RevisionStateComposerTiRunning,
							ChangeTime: timestamp,
							Requestor:  "airflow-worker",
							Body:       googlecloudclustercomposer_contract.NewAirflowTaskInstance("my_dag", "task_id_1", "2023-01-01T00:00:00Z", "1", "airflow-worker-abc", googlecloudclustercomposer_contract.TASKINSTANCE_RUNNING).ToYaml(),
						},
					},
					// Log Summary
					&testchangeset.HasLogSummary{WantLogSummary: "Executing task"},
				},
			},
		},
		{
			name: "Worker log without TaskInstance",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: timestamp},
					&log.MainMessageFieldSet{MainMessage: "Worker Heartbeat"},
					&googlecloudclustercomposer_contract.ComposerWorkerFieldSet{
						WorkerID: "airflow-worker-abc",
					},
				),
			},
			asserters: [][]testchangeset.ChangeSetAsserter{
				{
					// Worker Identity Event only
					&testchangeset.HasEvent{
						ResourcePath: googlecloudclustercomposer_contract.NewAirflowWorker("airflow-worker-abc").ResourcePath().Path,
					},
					// Log Summary
					&testchangeset.HasLogSummary{WantLogSummary: "Worker Heartbeat"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapper := airflowWorkerLogToTimelineMapperSetting{
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
