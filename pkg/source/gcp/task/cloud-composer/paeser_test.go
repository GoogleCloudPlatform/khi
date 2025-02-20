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

package composer_task

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	log_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/log"
	"github.com/stretchr/testify/assert"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestAirflowSchedulerParser(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected *model.AirflowTaskInstance
	}{
		{
			name: "scheduled",
			text: `textPayload: "\t<TaskInstance: airflow_monitoring.echo scheduled__2024-04-17T04:50:00+00:00 [scheduled]>"`,
			expected: model.NewAirflowTaskInstance(
				"airflow_monitoring",
				"echo",
				"scheduled__2024-04-17T04:50:00+00:00",
				"-1",
				"",
				model.TASKINSTANCE_SCHEDULED,
			),
		},
		{
			name: "queued",
			text: `textPayload: "Received executor event with state queued for task instance TaskInstanceKey(dag_id='khi_dag', task_id='add_one', run_id='scheduled__2023-11-30T05:00:00+00:00', try_number=1, map_index=0)"`,
			expected: model.NewAirflowTaskInstance(
				"khi_dag",
				"add_one",
				"scheduled__2023-11-30T05:00:00+00:00",
				"0",
				"",
				model.TASKINSTANCE_QUEUED,
			),
		},
		{
			name: "success",
			text: `textPayload: "TaskInstance Finished: dag_id=airflow_monitoring, task_id=echo, run_id=scheduled__2024-04-17T06:00:00+00:00, map_index=-1, run_start_date=2024-04-17 06:10:01.486093+00:00, run_end_date=2024-04-17 06:10:03.568974+00:00, run_duration=2.082881, state=success, executor_state=success, try_number=1, max_tries=1, job_id=4747, pool=default_pool, queue=default, priority_weight=2147483647, operator=BashOperator, queued_dttm=2024-04-17 06:10:00.625711+00:00, queued_by_job_id=4746, pid=145568"`,
			expected: model.NewAirflowTaskInstance(
				"airflow_monitoring",
				"echo",
				"scheduled__2024-04-17T06:00:00+00:00",
				"-1",
				"",
				model.TASKINSTANCE_SUCCESS,
			),
		},
		{
			name: "deferred",
			text: `textPayload: "TaskInstance Finished: dag_id=airflow_monitoring, task_id=echo, run_id=scheduled__2024-04-17T06:00:00+00:00, map_index=-1, run_start_date=2024-04-17 06:10:01.486093+00:00, run_end_date=2024-04-17 06:10:03.568974+00:00, run_duration=2.082881, state=deferred, executor_state=success, try_number=1, max_tries=1, job_id=4747, pool=default_pool, queue=default, priority_weight=2147483647, operator=BashOperator, queued_dttm=2024-04-17 06:10:00.625711+00:00, queued_by_job_id=4746, pid=145568"`,
			expected: model.NewAirflowTaskInstance(
				"airflow_monitoring",
				"echo",
				"scheduled__2024-04-17T06:00:00+00:00",
				"-1",
				"",
				model.TASKINSTANCE_DEFERRED,
			),
		},
		{
			name: "support hyphen",
			text: `textPayload: "\t<TaskInstance: demo_data_generate.tg.gke-basic-1.store_per_test_case_xcom manual__2024-04-16T08:38:39.822800+00:00 [scheduled]>"`,
			expected: model.NewAirflowTaskInstance(
				"demo_data_generate",
				"tg.gke-basic-1.store_per_test_case_xcom",
				"manual__2024-04-16T08:38:39.822800+00:00",
				"-1",
				"",
				model.TASKINSTANCE_SCHEDULED,
			),
		},
		{
			name: "detected zombie",
			text: `textPayload: "Detected zombie job: {'full_filepath': '/home/airflow/gcs/dags/memory.py', 'processor_subdir': '/home/airflow/gcs/dags', 'msg': \"{'DAG Id': 'Workload', 'Task Id': 'Aggregate', 'Run Id': 'manual__2024-05-21T07:55:33.285896+00:00', 'Hostname': 'airflow-worker-fs7hj', 'External Executor Id': '6b40704f-96ca-413d-91c7-1b50efdad27f'}\", 'simple_task_instance': <airflow.models.taskinstance.SimpleTaskInstance object at 0x7cc8063ea520>, 'is_failure_callback': True}"`,
			expected: model.NewAirflowTaskInstance(
				"Workload",
				"Aggregate",
				"manual__2024-05-21T07:55:33.285896+00:00",
				"-1",
				"airflow-worker-fs7hj",
				model.TASKINSTANCE_ZOMBIE,
			),
		},
	}

	p := &AirflowSchedulerParser{}

	for _, test := range testCases {
		t.Run(test.name, func(subt *testing.T) {
			entity := log_test.MustLogEntity(test.text)
			if entity == nil {
				t.Fatalf("entity is nil")
			}
			actual, err := p.parseInternal(entity)
			assert.Nil(t, err)
			assert.NotNil(t, actual)
		})
	}
}

func Test__airflowWorkerRunningHostFn(t *testing.T) {
	p := &airflowWorkerRunningHostFn{}
	testCases := []struct {
		name     string
		yaml     string
		expected *model.AirflowTaskInstance
	}{
		{
			"queued",
			`textPayload: "Running <TaskInstance: Afghanistan_2__generated.query3 scheduled__2024-04-22T05:30:00+00:00 [queued]> on host airflow-worker-dpvl7"`,
			model.NewAirflowTaskInstance(
				"Afghanistan_2__generated",
				"query3",
				"scheduled__2024-04-22T05:30:00+00:00",
				"-1",
				"airflow-worker-dpvl7",
				"queued",
			),
		},
		{
			"mapIndex",
			`textPayload: "Running <TaskInstance: Afghanistan_2__generated.query3 scheduled__2024-04-22T05:30:00+00:00 map_index=2 [running]> on host airflow-worker-dpvl7"`,
			model.NewAirflowTaskInstance(
				"Afghanistan_2__generated",
				"query3",
				"scheduled__2024-04-22T05:30:00+00:00",
				"2",
				"airflow-worker-dpvl7",
				"running",
			),
		},
		{
			"TaskGroup",
			`textPayload: "Running <TaskInstance: taskgroup_example.this_is_group.task_1 manual__2024-05-09T08:28:49.778920+00:00 [running]> on host airflow-worker-8vrrm"`,
			model.NewAirflowTaskInstance(
				"taskgroup_example",
				"this_is_group.task_1",
				"manual__2024-05-09T08:28:49.778920+00:00",
				"-1",
				"airflow-worker-8vrrm",
				"running",
			),
		},
	}

	for _, test := range testCases {
		t.Run("Test-"+test.name, func(t *testing.T) {
			l := log_test.MustLogEntity(test.yaml)
			ti, err := p.fn(l)
			assert.Nil(t, err)
			assert.Equal(t, test.expected, ti)
		})
	}
}

func Test__airflowWorkerMarkingStatusFn(t *testing.T) {
	p := &airflowWorkerMarkingStatusFn{}
	testCases := []struct {
		name     string
		yaml     string
		expected *model.AirflowTaskInstance
	}{
		{
			"success",
			`
labels:
  worker_id: "airflow-worker-5fqxd"
textPayload: "Marking task as SUCCESS. dag_id=airflow_monitoring, task_id=echo, execution_date=20240423T072000, start_date=20240423T073002, end_date=20240423T073007"`,
			model.NewAirflowTaskInstance(
				"airflow_monitoring",
				"echo",
				"unknown",
				"-1",
				"airflow-worker-5fqxd",
				"success",
			),
		},
		{
			"success",
			`
labels:
  worker_id: "airflow-worker-5fqxd"
textPayload: "Marking task as SUCCESS. dag_id=airflow_monitoring, task_id=echo, map_index=2, execution_date=20240423T072000, start_date=20240423T073002, end_date=20240423T073007"`,
			model.NewAirflowTaskInstance(
				"airflow_monitoring",
				"echo",
				"unknown",
				"2",
				"airflow-worker-5fqxd",
				"success",
			),
		},
	}

	for _, test := range testCases {
		t.Run("Test-"+test.name, func(t *testing.T) {
			l := log_test.MustLogEntity(test.yaml)
			ti, err := p.fn(l)
			assert.Nil(t, err)
			assert.Equal(t, test.expected, ti)
		})
	}
}

func TestDagProcessor(t *testing.T) {

	testCases := []struct {
		name     string
		text     string
		expected *model.DagFileProcessorStats
	}{
		{
			"Real Data(with 7)",
			"/home/airflow/gcs/dags/airflow_monitoring.py  19517  0.08s             1           0  0.51s           2024-05-08T02:44:13",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"0.08s",
				"1",
				"0",
			),
		},
		{
			"minimum",
			"/home/airflow/gcs/dags/airflow_monitoring.py 1 0",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"",
				"1",
				"0",
			),
		},
		{
			"4 with PID",
			"/home/airflow/gcs/dags/airflow_monitoring.py  18419                   1           0",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"",
				"1",
				"0",
			),
		},
		{
			"4 with RUNTIME",
			"/home/airflow/gcs/dags/airflow_monitoring.py 2.58s 1 0",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"2.58s",
				"1",
				"0",
			),
		},
		{
			"5 with PID and RUNTIME",
			"/home/airflow/gcs/dags/airflow_monitoring.py 19517 0.08s 1 0",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"0.08s",
				"1",
				"0",
			),
		},
		{
			"5 with LAST_*",
			"/home/airflow/gcs/dags/airflow_monitoring.py 1 0  0.51s 2024-05-08T02:44:13",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"",
				"1",
				"0",
			),
		},
		{
			"6 with RUNTIME",
			"/home/airflow/gcs/dags/airflow_monitoring.py 0.08s 1 0  0.51s 2024-05-08T02:44:13",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"0.08s",
				"1",
				"0",
			),
		},
		{
			"6 with PID and RUNTIME and LAST_*",
			"/home/airflow/gcs/dags/airflow_monitoring.py 19517  0.08s 1 0  0.51s",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"0.08s",
				"1",
				"0",
			),
		},
		{
			"6 with PID and LAST_*",
			"/home/airflow/gcs/dags/airflow_monitoring.py  19517 1 0  0.51s 2024-05-08T02:44:13",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"",
				"1",
				"0",
			),
		},
	}

	p := &AirflowDagProcessorParser{
		dagFilePath: "/home/airflow/gcs/dags/",
	}

	for _, test := range testCases {
		t.Run("Test_"+test.name, func(t *testing.T) {
			stats := p.fromLogEntity(test.text)
			assert.NotNil(t, stats)
			assert.Equal(t, test.expected, stats)
		})
	}

}
