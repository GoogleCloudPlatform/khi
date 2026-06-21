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
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
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

func TestDagProcessorMapperTask_ProcessLogByGroup(t *testing.T) {
	timestamp2 := time.Date(2024, 5, 8, 2, 44, 1, 0, time.UTC)
	timestamp3 := time.Date(2024, 5, 8, 2, 44, 2, 0, time.UTC)
	timestamp4 := time.Date(2024, 5, 8, 2, 44, 3, 0, time.UTC)
	timestamp5 := time.Date(2024, 5, 8, 2, 44, 4, 0, time.UTC)

	logsCase1 := []*log.Log{
		log.NewLogWithFieldSetsForTest(
			&log.CommonFieldSet{Timestamp: timestamp2},
			&googlecloudcommon_contract.GCPMainMessageFieldSet{MainMessage: "DAG_PROCESSOR_MANAGER_LOG: =========== DAG File Processing Stats ============"},
		),
		log.NewLogWithFieldSetsForTest(
			&log.CommonFieldSet{Timestamp: timestamp3},
			&googlecloudcommon_contract.GCPMainMessageFieldSet{MainMessage: "DAG_PROCESSOR_MANAGER_LOG: File Path                                           PID    Runtime      # DAGs    # Errors  Last Runtime    Last Run"},
		),
		log.NewLogWithFieldSetsForTest(
			&log.CommonFieldSet{Timestamp: timestamp4},
			&googlecloudcommon_contract.GCPMainMessageFieldSet{MainMessage: "DAG_PROCESSOR_MANAGER_LOG: --------------------------------------------------  -----  ---------  --------  ----------  --------------  -------------------"},
		),
		log.NewLogWithFieldSetsForTest(
			&log.CommonFieldSet{Timestamp: timestamp5},
			&googlecloudcommon_contract.GCPMainMessageFieldSet{MainMessage: "DAG_PROCESSOR_MANAGER_LOG: /home/airflow/gcs/dags/airflow_monitoring.py                                 1           0  0.36s           2026-03-08T04:49:37"},
		),
	}

	testCases := []struct {
		name         string
		logs         []*log.Log
		initialState *DagProcessorState
		wantState    *DagProcessorState
		asserts      []func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name:         "Header detection and dynamic extraction",
			logs:         logsCase1,
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
			asserts: []func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet){
				func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
					// "==========="
					if cs != nil && len(cs.Revisions) > 0 {
						t.Error("expected no timeline revisions for header boundary")
					}
				},
				func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
					// HeaderCandidate
					if cs != nil && len(cs.Revisions) > 0 {
						t.Error("expected no timeline revisions for header candidates")
					}
				},
				func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
					// Separator
					if cs != nil && len(cs.Revisions) > 0 {
						t.Error("expected no timeline revisions for separators")
					}
				},
				func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
					// Data line
					envPath := googlecloudclustercomposer_contract.MustComposerEnvironmentTimeline(ctx, "test-project", "test-environment")
					timelinePath := googlecloudclustercomposer_contract.MustAirflowDAGProcessorManagerInstanceTimeline(ctx, envPath, "/home/airflow/gcs/dags/airflow_monitoring.py", "unknown-parser")
					testchangeset.AssertTimeline(t, cs).
						HasRevision(timelinePath, &khifilev6.StagingRevision{
							ChangedTime: timestamp5,
							Principal:   "dag-processor-manager",
							VerbType:    googlecloudclustercomposer_contract.VerbComposerTaskInstanceStats,
							StateType:   googlecloudclustercomposer_contract.RevisionStateComposerDagProcessorNoError,
						}, cmp.AllowUnexported(
							structured.StandardMapNode{},
							structured.StandardScalarNode[string]{},
							structured.StandardScalarNode[any]{},
							structured.StandardSequenceNode{},
							unique.Handle[string]{},
						))
				},
			},
		},
	}

	mapper := &dagProcessorManagerTimelineMapper{
		targetLogType: googlecloudclustercomposer_contract.LogTypeComposerEnvironment,
		dagFilePath:   "/home/airflow/gcs/dags",
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			taskDependentValues := typedmap.NewTypedMap()
			typedmap.Set(taskDependentValues, typedmap.NewTypedKey[googlecloudk8scommon_contract.GoogleCloudClusterIdentity](googlecloudclustercomposer_contract.ClusterIdentityTaskID.ReferenceIDString()), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{ProjectID: "test-project"})
			typedmap.Set(taskDependentValues, typedmap.NewTypedKey[string](googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.ReferenceIDString()), "test-environment")
			ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskDependentValues)

			state := tc.initialState
			for i, l := range tc.logs {
				cs, nextState, err := mapper.ProcessLogByGroup(ctx, l, state)
				if err != nil {
					t.Fatalf("ProcessLogByGroup failed at message %d: %v", i, err)
				}
				tc.asserts[i](t, ctx, cs)
				state = nextState
			}

			if diff := cmp.Diff(tc.wantState, state, cmp.AllowUnexported(DagProcessorState{}, logutil.TabulateReader{})); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDagProcessorLogIngester_ProcessLog(t *testing.T) {
	timestamp := time.Date(2024, 5, 8, 2, 44, 1, 0, time.UTC)

	testCases := []struct {
		name        string
		messages    []string
		wantSummary string
	}{
		{
			name: "header detection",
			messages: []string{
				"DAG_PROCESSOR_MANAGER_LOG: =========== DAG File Processing Stats ============",
			},
			wantSummary: "=========== DAG File Processing Stats ============",
		},
		{
			name: "data line parsing success",
			messages: []string{
				"DAG_PROCESSOR_MANAGER_LOG: =========== DAG File Processing Stats ============",
				"DAG_PROCESSOR_MANAGER_LOG: File Path                                           PID    Runtime      # DAGs    # Errors  Last Runtime    Last Run",
				"DAG_PROCESSOR_MANAGER_LOG: --------------------------------------------------  -----  ---------  --------  ----------  --------------  -------------------",
				"DAG_PROCESSOR_MANAGER_LOG: /home/airflow/gcs/dags/airflow_monitoring.py                                 1           0  0.36s           2026-03-08T04:49:37",
			},
			wantSummary: "File Path: /home/airflow/gcs/dags/airflow_monitoring.py PID:  #DAGs: 1 #Errors: 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ingester := &dagProcessorManagerLogIngester{}
			var finalCS *khifilev6.LogChangeSet
			var state *DagProcessorState
			for _, msg := range tc.messages {
				inputLog := log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: timestamp},
					&googlecloudcommon_contract.GCPMainMessageFieldSet{MainMessage: msg},
				)
				cs, nextState, err := ingester.ProcessLogByGroup(context.Background(), inputLog, state)
				if err != nil {
					t.Fatalf("ProcessLogByGroup failed: %v", err)
				}
				finalCS = cs
				state = nextState
			}
			if finalCS.Summary != tc.wantSummary {
				t.Errorf("summary mismatch: want %q, got %q", tc.wantSummary, finalCS.Summary)
			}
		})
	}
}
