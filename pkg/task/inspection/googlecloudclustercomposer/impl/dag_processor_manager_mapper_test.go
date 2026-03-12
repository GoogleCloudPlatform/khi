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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestDagProcessorMapperTask_ProcessLogByGroup(t *testing.T) {
	timestamp2 := time.Date(2024, 5, 8, 2, 44, 1, 0, time.UTC)
	timestamp3 := time.Date(2024, 5, 8, 2, 44, 2, 0, time.UTC)
	timestamp4 := time.Date(2024, 5, 8, 2, 44, 3, 0, time.UTC)
	timestamp5 := time.Date(2024, 5, 8, 2, 44, 4, 0, time.UTC)

	testCases := []struct {
		name         string
		logs         []*log.Log
		initialState *DagProcessorState
		wantState    *DagProcessorState
		asserters    [][]testchangeset.ChangeSetAsserter // Expected changesets for each log
	}{
		{
			name: "Header detection and dynamic extraction",
			logs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: timestamp2},
					&log.MainMessageFieldSet{MainMessage: "DAG_PROCESSOR_MANAGER_LOG: =========== DAG File Processing Stats ============"},
				),
				log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: timestamp3},
					&log.MainMessageFieldSet{MainMessage: "DAG_PROCESSOR_MANAGER_LOG: File Path                                           PID    Runtime      # DAGs    # Errors  Last Runtime    Last Run"},
				),
				log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: timestamp4},
					&log.MainMessageFieldSet{MainMessage: "DAG_PROCESSOR_MANAGER_LOG: --------------------------------------------------  -----  ---------  --------  ----------  --------------  -------------------"},
				),
				log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: timestamp5},
					&log.MainMessageFieldSet{MainMessage: "DAG_PROCESSOR_MANAGER_LOG: /home/airflow/gcs/dags/airflow_monitoring.py                                 1           0  0.36s           2026-03-08T04:49:37"},
				),
			},
			initialState: nil,
			wantState: &DagProcessorState{
				Reader: &logutil.TabulateReader{
					Headers: []string{dagProcessorManagerColumnFilePath, dagProcessorManagerColumnPID, dagProcessorManagerColumnRuntime, dagProcessorManagerColumnNumDags, dagProcessorManagerColumnNumErrors, dagProcessorManagerColumnLastRuntime, dagProcessorManagerColumnLastRun},
					ColumnBoundaries: []logutil.ColumnBoundary{
						{Name: dagProcessorManagerColumnFilePath, Left: 0, Right: 51},
						{Name: dagProcessorManagerColumnPID, Left: 51, Right: 58},
						{Name: dagProcessorManagerColumnRuntime, Left: 58, Right: 69},
						{Name: dagProcessorManagerColumnNumDags, Left: 69, Right: 79},
						{Name: dagProcessorManagerColumnNumErrors, Left: 79, Right: 91},
						{Name: dagProcessorManagerColumnLastRuntime, Left: 91, Right: 107},
						{Name: dagProcessorManagerColumnLastRun, Left: 107, Right: 2147483647},
					},
				},
			},
			asserters: [][]testchangeset.ChangeSetAsserter{
				{}, // "==========="
				{}, // "File Path" (HeaderCandidate)
				{}, // "-----------" (Separator)
				{ // Data line processing
					&testchangeset.HasRevision{
						ResourcePath: resourcepath.NameLayerGeneralItem("Apache Airflow", "Dag File Processor Stats", "unknown-parser", "/home/airflow/gcs/dags/airflow_monitoring.py").Path,
						WantRevision: history.StagingResourceRevision{
							Verb:       enum.RevisionVerbComposerTaskInstanceStats,
							State:      enum.RevisionStateConditionTrue,
							ChangeTime: timestamp5,
							Requestor:  "dag-processor-manager",
						},
					},
					&testchangeset.HasLogSummary{WantLogSummary: "File Path: /home/airflow/gcs/dags/airflow_monitoring.py PID:  #DAGs: 1 #Errors: 0"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapper := airflowDagProcessorManagerLogToTimelineMapperSetting{
				dagFilePath: "/home/airflow/gcs/dags",
			}

			state := tc.initialState
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

			if diff := cmp.Diff(tc.wantState, state, cmp.AllowUnexported(DagProcessorState{}, logutil.TabulateReader{})); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
