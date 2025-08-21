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

package apacheairflow

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/stretchr/testify/assert"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func Test__airflowWorkerRunningHostFn(t *testing.T) {
	p := &airflowWorkerRunningHostFn{}
	testCases := []struct {
		name     string
		yaml     string
		expected *model.AirflowTaskInstance
	}{
		{
			"queued",
			`textPayload: "Running <TaskInstance: example.query3 scheduled__2024-04-22T05:30:00+00:00 [queued]> on host airflow-worker-dpvl7"`,
			model.NewAirflowTaskInstance(
				"example",
				"query3",
				"scheduled__2024-04-22T05:30:00+00:00",
				"-1",
				"airflow-worker-dpvl7",
				"queued",
			),
		},
		{
			"mapIndex",
			`textPayload: "Running <TaskInstance: example.query3 scheduled__2024-04-22T05:30:00+00:00 map_index=2 [running]> on host airflow-worker-dpvl7"`,
			model.NewAirflowTaskInstance(
				"example",
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
			l := testlog.MustLogFromYAML(test.yaml, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
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
		isErr    bool
		expected *model.AirflowTaskInstance
	}{
		{
			"success(before 2.8)",
			`
labels:
  worker_id: "airflow-worker-5fqxd"
textPayload: "Marking task as SUCCESS. dag_id=airflow_monitoring, task_id=echo, execution_date=20240423T072000, start_date=20240423T073002, end_date=20240423T073007"`,
			true,
			nil,
		},
		{
			"success(after 2.9)",
			`
labels:
  worker_id: "airflow-worker-5fqxd"
textPayload: "Marking task as SUCCESS. dag_id=airflow_monitoring, task_id=echo, run_id=scheduled__2025-04-14T01:30:00+00:00, execution_date=20250414T013000, start_date=20250414T014000, end_date=20250414T014001"`,
			false,
			model.NewAirflowTaskInstance(
				"airflow_monitoring",
				"echo",
				"scheduled__2025-04-14T01:30:00+00:00",
				"-1",
				"airflow-worker-5fqxd",
				"success",
			),
		},
		{
			"success(after 2.9) with mapid",
			`
labels:
  worker_id: "airflow-worker-5fqxd"
textPayload: "Marking task as SUCCESS. dag_id=airflow_monitoring, task_id=echo, run_id=scheduled__2025-04-14T01:30:00+00:00, map_index=2, execution_date=20250414T013000, start_date=20250414T014000, end_date=20250414T014001"`,
			false,
			model.NewAirflowTaskInstance(
				"airflow_monitoring",
				"echo",
				"scheduled__2025-04-14T01:30:00+00:00",
				"2",
				"airflow-worker-5fqxd",
				"success",
			),
		},
	}

	for _, test := range testCases {
		t.Run("Test-"+test.name, func(t *testing.T) {
			l := testlog.MustLogFromYAML(test.yaml, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
			ti, err := p.fn(l)
			assert.Equal(t, test.isErr, err != nil)
			assert.Equal(t, test.expected, ti)
		})
	}
}
