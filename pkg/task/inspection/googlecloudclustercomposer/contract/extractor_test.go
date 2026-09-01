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

package googlecloudclustercomposer_contract

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func TestExtractComposerTaskInstance(t *testing.T) {
	tests := []struct {
		name        string
		textPayload string
		want        ComposerTaskInstanceFieldSet
		wantErr     bool
	}{
		{
			name:        "scheduled task instance",
			textPayload: `\t<TaskInstance: airflow_monitoring.echo scheduled__2024-04-17T04:50:00+00:00 [scheduled]>`,
			want: ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("airflow_monitoring", "echo", "scheduled__2024-04-17T04:50:00+00:00", "-1", "", TASKINSTANCE_SCHEDULED),
			},
		},
		{
			name:        "queued task instance",
			textPayload: `Received executor event with state queued for task instance TaskInstanceKey(dag_id='khi_dag', task_id='add_one', run_id='scheduled__2023-11-30T05:00:00+00:00', try_number=1, map_index=0)`,
			want: ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("khi_dag", "add_one", "scheduled__2023-11-30T05:00:00+00:00", "0", "", TASKINSTANCE_QUEUED),
			},
		},
		{
			name:        "success task instance",
			textPayload: `TaskInstance Finished: dag_id=airflow_monitoring, task_id=echo, run_id=scheduled__2024-04-17T06:00:00+00:00, map_index=-1, run_start_date=2024-04-17 06:10:01.486093+00:00, run_end_date=2024-04-17 06:10:03.568974+00:00, run_duration=2.082881, state=success, executor_state=success, try_number=1, max_tries=1, job_id=4747, pool=default_pool, queue=default, priority_weight=2147483647, operator=BashOperator, queued_dttm=2024-04-17 06:10:00.625711+00:00, queued_by_job_id=4746, pid=145568`,
			want: ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"airflow_monitoring",
					"echo",
					"scheduled__2024-04-17T06:00:00+00:00",
					"-1",
					"",
					TASKINSTANCE_SUCCESS,
				),
			},
		},
		{
			name:        "deferred task instance",
			textPayload: `TaskInstance Finished: dag_id=airflow_monitoring, task_id=echo, run_id=scheduled__2024-04-17T06:00:00+00:00, map_index=-1, run_start_date=2024-04-17 06:10:01.486093+00:00, run_end_date=2024-04-17 06:10:03.568974+00:00, run_duration=2.082881, state=deferred, executor_state=success, try_number=1, max_tries=1, job_id=4747, pool=default_pool, queue=default, priority_weight=2147483647, operator=BashOperator, queued_dttm=2024-04-17 06:10:00.625711+00:00, queued_by_job_id=4746, pid=145568`,
			want: ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"airflow_monitoring",
					"echo",
					"scheduled__2024-04-17T06:00:00+00:00",
					"-1",
					"",
					TASKINSTANCE_DEFERRED,
				),
			},
		},
		{
			name:        "support task group",
			textPayload: `\t<TaskInstance: demo_data_generate.tg.gke-basic-1.store_per_test_case_xcom manual__2024-04-16T08:38:39.822800+00:00 [scheduled]>`,
			want: ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"demo_data_generate",
					"tg.gke-basic-1.store_per_test_case_xcom",
					"manual__2024-04-16T08:38:39.822800+00:00",
					"-1",
					"",
					TASKINSTANCE_SCHEDULED,
				),
			},
		},
		{
			name:        "zombie detection",
			textPayload: `Detected zombie job: {'full_filepath': '/home/airflow/gcs/dags/memory.py', 'processor_subdir': '/home/airflow/gcs/dags', 'msg': \"{'DAG Id': 'Workload', 'Task Id': 'Aggregate', 'Run Id': 'manual__2024-05-21T07:55:33.285896+00:00', 'Hostname': 'airflow-worker-fs7hj', 'External Executor Id': '6b40704f-96ca-413d-91c7-1b50efdad27f'}\", 'simple_task_instance': <airflow.models.taskinstance.SimpleTaskInstance object at 0x7cc8063ea520>, 'is_failure_callback': True}`,
			want: ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"Workload",
					"Aggregate",
					"manual__2024-05-21T07:55:33.285896+00:00",
					"-1",
					"airflow-worker-fs7hj",
					TASKINSTANCE_ZOMBIE,
				),
			},
		},
		{
			name:        "invalid log",
			textPayload: `Some other log message here`,
			wantErr:     true,
		},
		{
			name:        "no textPayload",
			textPayload: ``,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yamlStr string
			if tt.textPayload != "" {
				yamlStr = fmt.Sprintf("textPayload: \"%s\"", tt.textPayload)
			} else {
				yamlStr = "{}"
			}
			yamlNode, err := structured.FromYAML(yamlStr)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			nodeReader := structured.NewNodeReader(yamlNode)

			got, err := ExtractComposerTaskInstance(nodeReader)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExtractComposerTaskInstance() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if diff := cmp.Diff(tt.want.TaskInstance, got.TaskInstance, cmp.AllowUnexported(AirflowTaskInstance{})); diff != "" {
					t.Errorf("ExtractComposerTaskInstance() result mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}

	t.Run("MockNode returns mock data directly", func(t *testing.T) {
		want := ComposerTaskInstanceFieldSet{
			TaskInstance: NewAirflowTaskInstance("mock_dag", "mock_task", "mock_run", "-1", "mock_host", TASKINSTANCE_SUCCESS),
		}
		reader := structured.NewNodeReader(structured.NewMockNode(want))
		got, err := ExtractComposerTaskInstance(reader)
		if err != nil {
			t.Fatalf("ExtractComposerTaskInstance() returned error: %v", err)
		}
		if diff := cmp.Diff(want.TaskInstance, got.TaskInstance, cmp.AllowUnexported(AirflowTaskInstance{})); diff != "" {
			t.Errorf("ExtractComposerTaskInstance() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestExtractComposerWorkerTaskInstance(t *testing.T) {
	tests := []struct {
		name        string
		textPayload string
		workerId    string
		labels      map[string]string
		want        ComposerWorkerTaskInstanceFieldSet
		wantErr     bool
	}{
		{
			name:        "queued",
			textPayload: `Running <TaskInstance: example.query3 scheduled__2024-04-22T05:30:00+00:00 [queued]> on host airflow-worker-dpvl7`,
			workerId:    "worker-id-dpvl7",
			want: ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"example",
					"query3",
					"scheduled__2024-04-22T05:30:00+00:00",
					"-1",
					"airflow-worker-dpvl7",
					TASKINSTANCE_QUEUED,
				),
			},
		},
		{
			name:        "mapIndex",
			textPayload: `Running <TaskInstance: example.query3 scheduled__2024-04-22T05:30:00+00:00 map_index=2 [running]> on host airflow-worker-dpvl7`,
			workerId:    "worker-id-dpvl7",
			want: ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"example",
					"query3",
					"scheduled__2024-04-22T05:30:00+00:00",
					"2",
					"airflow-worker-dpvl7",
					TASKINSTANCE_RUNNING,
				),
			},
		},
		{
			name:        "task group",
			textPayload: `Running <TaskInstance: taskgroup_example.this_is_group.task_1 manual__2024-05-09T08:28:49.778920+00:00 [running]> on host airflow-worker-8vrrm`,
			workerId:    "airflow-worker-8vrrm",
			want: ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"taskgroup_example",
					"this_is_group.task_1",
					"manual__2024-05-09T08:28:49.778920+00:00",
					"-1",
					"airflow-worker-8vrrm",
					TASKINSTANCE_RUNNING,
				),
			},
		},
		{
			name:        "success(before 2.8)",
			textPayload: `Marking task as SUCCESS. dag_id=airflow_monitoring, task_id=echo, execution_date=20240423T072000, start_date=20240423T073002, end_date=20240423T073007`,
			workerId:    "airflow-worker-5fqxd", // This is currently unsupported because KHI can't extract assume the run_id for this.
			wantErr:     true,
		},
		{
			name:        "success(after 2.9)",
			textPayload: `Marking task as SUCCESS. dag_id=airflow_monitoring, task_id=echo, run_id=scheduled__2025-04-14T01:30:00+00:00, execution_date=20250414T013000, start_date=20250414T014000, end_date=20250414T014001`,
			workerId:    "airflow-worker-5fqxd",
			want: ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"airflow_monitoring",
					"echo",
					"scheduled__2025-04-14T01:30:00+00:00",
					"-1",
					"airflow-worker-5fqxd",
					TASKINSTANCE_SUCCESS,
				),
			},
		},
		{
			name:        "success(after 2.9) with mapid",
			textPayload: `Marking task as SUCCESS. dag_id=airflow_monitoring, task_id=echo, run_id=scheduled__2025-04-14T01:30:00+00:00, map_index=2, execution_date=20250414T013000, start_date=20250414T014000, end_date=20250414T014001`,
			workerId:    "airflow-worker-5fqxd",
			want: ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"airflow_monitoring",
					"echo",
					"scheduled__2025-04-14T01:30:00+00:00",
					"2",
					"airflow-worker-5fqxd",
					TASKINSTANCE_SUCCESS,
				),
			},
		},
		{
			name:        "from labels (Airflow 3)",
			textPayload: `Any text payload`,
			labels: map[string]string{
				"worker_id": "airflow-worker-test",
				"run-id":    "scheduled__2025-04-14T01:30:00+00:00",
				"workflow":  "airflow_monitoring",
				"task-id":   "echo",
				"map-index": "2",
			},
			want: ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"airflow_monitoring",
					"echo",
					"scheduled__2025-04-14T01:30:00+00:00",
					"2",
					"airflow-worker-test",
					TASKINSTANCE_NONE,
				),
			},
		},
		{
			name:        "running with labels (Airflow 2/3 on Cloud Composer)",
			textPayload: `Running <TaskInstance: sample_khi_multi_step_dag.step2_process scheduled__2026-08-05T16:36:00+00:00 [running]> on host airflow-worker-mwtx5`,
			labels: map[string]string{
				"worker_id": "airflow-worker-mwtx5",
				"run-id":    "scheduled__2026-08-05T16:36:00+00:00",
				"workflow":  "sample_khi_multi_step_dag",
				"task-id":   "step2_process",
				"map-index": "-1",
			},
			want: ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"sample_khi_multi_step_dag",
					"step2_process",
					"scheduled__2026-08-05T16:36:00+00:00",
					"-1",
					"airflow-worker-mwtx5",
					TASKINSTANCE_RUNNING,
				),
			},
		},
		{
			name:        "final state skipped with labels (Airflow 3)",
			textPayload: `Task finished [task_instance_id=019d10e4-71f5-7016-b412-aa1fbcfd16fc] [exit_code=0] [duration=0.41656468300061533] [final_state=skipped]`,
			labels: map[string]string{
				"worker_id": "airflow-worker-test",
				"run-id":    "scheduled__2025-04-14T01:30:00+00:00",
				"workflow":  "airflow_monitoring",
				"task-id":   "echo",
				"map-index": "2",
			},
			want: ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"airflow_monitoring",
					"echo",
					"scheduled__2025-04-14T01:30:00+00:00",
					"2",
					"airflow-worker-test",
					TASKINSTANCE_SKIPPED,
				),
			},
		},
		{
			name:        "marking status with labels",
			textPayload: `Marking task as SUCCESS. dag_id=airflow_monitoring, task_id=echo, run_id=scheduled__2025-04-14T01:30:00+00:00, map_index=2, execution_date=20250414T013000, start_date=20250414T014000, end_date=20250414T014001`,
			labels: map[string]string{
				"worker_id": "airflow-worker-5fqxd",
				"run-id":    "scheduled__2025-04-14T01:30:00+00:00",
				"workflow":  "airflow_monitoring",
				"task-id":   "echo",
				"map-index": "2",
			},
			want: ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance(
					"airflow_monitoring",
					"echo",
					"scheduled__2025-04-14T01:30:00+00:00",
					"2",
					"airflow-worker-5fqxd",
					TASKINSTANCE_SUCCESS,
				),
			},
		},
		{
			name:        "missing worker_id with generic payload",
			textPayload: `Any text payload`,
			labels: map[string]string{
				"run-id":    "scheduled__2025-04-14T01:30:00+00:00",
				"workflow":  "airflow_monitoring",
				"task-id":   "echo",
				"map-index": "2",
			},
			wantErr: true,
		},
		{
			name:        "missing run-id with generic payload",
			textPayload: `Any text payload`,
			labels: map[string]string{
				"worker_id": "airflow-worker-test",
				"workflow":  "airflow_monitoring",
				"task-id":   "echo",
				"map-index": "2",
			},
			wantErr: true,
		},
		{
			name:        "missing workflow with generic payload",
			textPayload: `Any text payload`,
			labels: map[string]string{
				"worker_id": "airflow-worker-test",
				"run-id":    "scheduled__2025-04-14T01:30:00+00:00",
				"task-id":   "echo",
				"map-index": "2",
			},
			wantErr: true,
		},
		{
			name:        "missing task-id with generic payload",
			textPayload: `Any text payload`,
			labels: map[string]string{
				"worker_id": "airflow-worker-test",
				"run-id":    "scheduled__2025-04-14T01:30:00+00:00",
				"workflow":  "airflow_monitoring",
				"map-index": "2",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yamlStr string
			if tt.textPayload != "" {
				yamlStr = fmt.Sprintf("textPayload: \"%s\"", tt.textPayload)
			}
			hasLabels := false
			if tt.workerId != "" {
				yamlStr += fmt.Sprintf("\nlabels:\n  worker_id: '%s'", tt.workerId)
				hasLabels = true
			}
			if len(tt.labels) > 0 {
				if !hasLabels {
					yamlStr += "\nlabels:"
				}
				for k, v := range tt.labels {
					yamlStr += fmt.Sprintf("\n  \"%s\": '%s'", k, v)
				}
			}
			yamlNode, err := structured.FromYAML(yamlStr)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			nodeReader := structured.NewNodeReader(yamlNode)

			got, err := ExtractComposerWorkerTaskInstance(nodeReader)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExtractComposerWorkerTaskInstance() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if diff := cmp.Diff(tt.want.TaskInstance, got.TaskInstance, cmp.AllowUnexported(AirflowTaskInstance{})); diff != "" {
					t.Errorf("ExtractComposerWorkerTaskInstance() result mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}

	t.Run("MockNode returns mock data directly", func(t *testing.T) {
		want := ComposerWorkerTaskInstanceFieldSet{
			TaskInstance: NewAirflowTaskInstance("mock_dag", "mock_task", "mock_run", "1", "mock_worker", TASKINSTANCE_SUCCESS),
		}
		reader := structured.NewNodeReader(structured.NewMockNode(want))
		got, err := ExtractComposerWorkerTaskInstance(reader)
		if err != nil {
			t.Fatalf("ExtractComposerWorkerTaskInstance() returned error: %v", err)
		}
		if diff := cmp.Diff(want.TaskInstance, got.TaskInstance, cmp.AllowUnexported(AirflowTaskInstance{})); diff != "" {
			t.Errorf("ExtractComposerWorkerTaskInstance() mismatch (-want +got):\n%s", diff)
		}
	})
}
