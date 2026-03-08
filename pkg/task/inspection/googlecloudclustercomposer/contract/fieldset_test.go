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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func TestComposerTaskInstanceFieldSetReader_Read(t *testing.T) {
	reader := &ComposerTaskInstanceFieldSetReader{}

	tests := []struct {
		name        string
		textPayload string
		want        *ComposerTaskInstanceFieldSet
		wantErr     bool
	}{
		{
			name:        "airflowTiTemplate basic",
			textPayload: `TaskInstance: [<TaskInstance: my_dag.my_task my_run_name [up_for_retry]>]`,
			want: &ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("my_dag", "my_task", "my_run_name", "-1", "", TASKINSTANCE_UP_FOR_RETRY),
			},
		},
		{
			name:        "airflowTiTemplate mapped",
			textPayload: `TaskInstance: [<TaskInstance: my_dag.my_task my_run_name map_index=0 [up_for_retry]>]`,
			want: &ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("my_dag", "my_task", "my_run_name", "0", "", TASKINSTANCE_UP_FOR_RETRY),
			},
		},
		{
			name:        "airflowSchedulerReceivedEventTemplate",
			textPayload: `Received executor event with state success for task instance TaskInstanceKey(dag_id='my_dag', task_id='my_task', run_id='my_run_name', try_number=1, map_index=-1)`,
			want: &ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("my_dag", "my_task", "my_run_name", "-1", "", TASKINSTANCE_SUCCESS),
			},
		},
		{
			name:        "airflowSchedulerTaskFinishedTemplate",
			textPayload: `TaskInstance Finished: dag_id=my_dag, task_id=my_task, run_id=my_run_name, map_index=-1, run_start_date=2024-02-14 06:17:42.502905+00:00, run_end_date=2024-02-14 06:17:42.923171+00:00, run_duration=0.420266, state=success, executor_state=success`,
			want: &ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("my_dag", "my_task", "my_run_name", "-1", "", TASKINSTANCE_SUCCESS),
			},
		},
		{
			name:        "airflowSchedulerZombieDetectedTemplate",
			textPayload: `Detected zombie job: {'full_filepath': '/home/airflow/gcs/dags/zombie.py', 'processor_subdir': '/home/airflow/gcs/dags', 'msg': "{'DAG Id': 'zombie_dag', 'Task Id': 'hello_world', 'Run Id': 'manual__2024-05-02T13:46:17.391307+00:00', 'Hostname': 'airflow-worker-578bffd886-smm8m'}", 'simple_task_instance': <airflow.models.taskinstance.SimpleTaskInstance object at 0x7f4cae7502b0>, 'is_failure_callback': True}`,
			want: &ComposerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("zombie_dag", "hello_world", "manual__2024-05-02T13:46:17.391307+00:00", "-1", "airflow-worker-578bffd886-smm8m", TASKINSTANCE_ZOMBIE),
			},
		},
		{
			name:        "invalid log",
			textPayload: `Some other log message here`,
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "no textPayload",
			textPayload: ``,
			want:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yamlStr string
			if tt.textPayload != "" {
				yamlStr = "textPayload: |\n  " + tt.textPayload
			} else {
				yamlStr = "{}"
			}
			yamlNode, err := structured.FromYAML(yamlStr)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			nodeReader := structured.NewNodeReader(yamlNode)

			got, err := reader.Read(nodeReader)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ComposerTaskInstanceFieldSetReader.Read() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.want != nil {
				gotTyped, ok := got.(*ComposerTaskInstanceFieldSet)
				if !ok {
					t.Fatalf("expected ComposerTaskInstanceFieldSet, got %T", got)
				}
				if diff := cmp.Diff(tt.want.TaskInstance, gotTyped.TaskInstance, cmp.AllowUnexported(AirflowTaskInstance{})); diff != "" {
					t.Errorf("Read() result mismatch (-want +got):\n%s", diff)
				}
			} else if got != nil {
				t.Errorf("expected nil result, got %v", got)
			}
		})
	}
}

func TestComposerWorkerTaskInstanceFieldSetReader_Read(t *testing.T) {
	reader := &ComposerWorkerTaskInstanceFieldSetReader{}

	tests := []struct {
		name       string
		yamlString string
		want       *ComposerWorkerTaskInstanceFieldSet
		wantErr    bool
	}{
		{
			name:       "airflowWorkerRunningHostTemplate basic",
			yamlString: "textPayload: |\n  Running <TaskInstance: my_dag.my_task my_run_name [running]> on host airflow-worker-123\nlabels:\n  worker_id: 'worker-id-123'\n",
			want: &ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("my_dag", "my_task", "my_run_name", "-1", "airflow-worker-123", TASKINSTANCE_RUNNING),
			},
		},
		{
			name:       "airflowWorkerRunningHostTemplate dynamic dag",
			yamlString: "textPayload: |\n  Running <TaskInstance: my_dag.my_task my_run_name map_index=0 [running]> on host airflow-worker-123\nlabels:\n  worker_id: 'worker-id-456'\n",
			want: &ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("my_dag", "my_task", "my_run_name", "0", "airflow-worker-123", TASKINSTANCE_RUNNING),
			},
		},
		{
			name:       "airflowWorkerMarkingStatusTemplate",
			yamlString: "textPayload: 'Marking task as success. dag_id=my_dag, task_id=my_task, run_id=my_run_name, map_index=-1, execution_date=2024-02-14 06:17:42.502905+00:00'\nlabels:\n  worker_id: 'worker-id-123'\n",
			want: &ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("my_dag", "my_task", "my_run_name", "-1", "worker-id-123", TASKINSTANCE_SUCCESS),
			},
		},
		{
			name:       "airflowWorkerMarkingStatusTemplate uppercase state",
			yamlString: "textPayload: 'Marking task as SUCCESS. dag_id=my_dag, task_id=my_task, run_id=my_run_name, map_index=-1, execution_date=2024-02-14 06:17:42.502905+00:00'\nlabels:\n  worker_id: 'worker-id-123'\n",
			want: &ComposerWorkerTaskInstanceFieldSet{
				TaskInstance: NewAirflowTaskInstance("my_dag", "my_task", "my_run_name", "-1", "worker-id-123", TASKINSTANCE_SUCCESS),
			},
		},
		{
			name:       "airflowWorkerMarkingStatusTemplate missing worker id",
			yamlString: "textPayload: 'Marking task as success. dag_id=my_dag, task_id=my_task, run_id=my_run_name, map_index=-1, execution_date=2024-02-14 06:17:42.502905+00:00'\n",
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "invalid log",
			yamlString: "textPayload: |\n  Some other log message here\n",
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "no textPayload",
			yamlString: "{}",
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlNode, err := structured.FromYAML(tt.yamlString)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			nodeReader := structured.NewNodeReader(yamlNode)

			got, err := reader.Read(nodeReader)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ComposerWorkerTaskInstanceFieldSetReader.Read() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.want != nil {
				gotTyped, ok := got.(*ComposerWorkerTaskInstanceFieldSet)
				if !ok {
					t.Fatalf("expected ComposerWorkerTaskInstanceFieldSet, got %T", got)
				}
				if diff := cmp.Diff(tt.want.TaskInstance, gotTyped.TaskInstance, cmp.AllowUnexported(AirflowTaskInstance{})); diff != "" {
					t.Errorf("Read() result mismatch (-want +got):\n%s", diff)
				}
			} else if got != nil {
				t.Errorf("expected nil result, got %v", got)
			}
		})
	}
}
