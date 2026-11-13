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

package composerairflow

import (
	"gopkg.in/yaml.v3"
)

const (
	// TASKINSTANCE_NONE represents the unset task instance state.
	// See https://airflow.apache.org/docs/apache-airflow/stable/core-concepts/tasks.html#task-instances.
	TASKINSTANCE_NONE Tistate = "none"
	// TASKINSTANCE_SCHEDULED indicates the task instance is scheduled.
	TASKINSTANCE_SCHEDULED Tistate = "scheduled"
	// TASKINSTANCE_QUEUED indicates the task instance is queued for execution.
	TASKINSTANCE_QUEUED Tistate = "queued"
	// TASKINSTANCE_RUNNING indicates the task instance is currently running.
	TASKINSTANCE_RUNNING Tistate = "running"
	// TASKINSTANCE_SUCCESS indicates the task instance finished successfully.
	TASKINSTANCE_SUCCESS Tistate = "success"
	// TASKINSTANCE_SHUTDOWN indicates the task instance was shut down.
	TASKINSTANCE_SHUTDOWN Tistate = "shutdown"
	// TASKINSTANCE_RESTARTING indicates the task instance is restarting.
	TASKINSTANCE_RESTARTING Tistate = "restarting"
	// TASKINSTANCE_FAILED indicates the task instance failed.
	TASKINSTANCE_FAILED Tistate = "failed"
	// TASKINSTANCE_SKIPPED indicates the task instance was skipped.
	TASKINSTANCE_SKIPPED Tistate = "skipped"
	// TASKINSTANCE_UP_FOR_RETRY indicates the task instance is waiting for retry.
	TASKINSTANCE_UP_FOR_RETRY Tistate = "up_for_retry"
	// TASKINSTANCE_DEFERRED indicates the task instance is deferred.
	TASKINSTANCE_DEFERRED Tistate = "deferred"
	// TASKINSTANCE_UP_FOR_RESCHEDULE indicates the task instance is waiting for sensor reschedule.
	TASKINSTANCE_UP_FOR_RESCHEDULE Tistate = "up_for_reschedule"
	// TASKINSTANCE_REMOVED indicates the task instance was removed from the DAG run.
	TASKINSTANCE_REMOVED Tistate = "removed"
	// TASKINSTANCE_UPSTREAM_FAILED indicates an upstream task failed.
	TASKINSTANCE_UPSTREAM_FAILED Tistate = "upstream_failed"
	// TASKINSTANCE_ZOMBIE indicates the task instance is a zombie process detected by KHI.
	TASKINSTANCE_ZOMBIE Tistate = "zombie"
)

// Tistate represents an Airflow task instance execution state.
type Tistate string

// AirflowTaskInstance represents an Airflow task instance parsed from Composer logs.
// See https://github.com/apache/airflow/blob/main/airflow/models/taskinstance.py#L1187.
type AirflowTaskInstance struct {
	dagId    string // primary key
	taskId   string // primary key
	runId    string // primary key
	mapIndex string // primary key
	host     string
	status   Tistate
}

// AirflowWorker represents an Airflow worker host parsed from Composer logs.
type AirflowWorker struct {
	host string
}

// NewAirflowTaskInstance creates a new AirflowTaskInstance.
func NewAirflowTaskInstance(dagId string, taskId string, runId string, mapIndex string, host string, status Tistate) *AirflowTaskInstance {
	return &AirflowTaskInstance{
		dagId:    dagId,
		taskId:   taskId,
		runId:    runId,
		mapIndex: mapIndex,
		host:     host,
		status:   status,
	}
}

// DagId returns the DAG ID of the task instance.
func (a *AirflowTaskInstance) DagId() string {
	return a.dagId
}

// TaskId returns the task ID of the task instance.
func (a *AirflowTaskInstance) TaskId() string {
	return a.taskId
}

// RunId returns the DAG run ID of the task instance.
func (a *AirflowTaskInstance) RunId() string {
	return a.runId
}

// MapIndex returns the dynamic task mapping index of the task instance.
func (a *AirflowTaskInstance) MapIndex() string {
	return a.mapIndex
}

// Host returns the worker host running the task instance.
func (a *AirflowTaskInstance) Host() string {
	return a.host
}

// Status returns the execution state of the task instance.
func (a *AirflowTaskInstance) Status() Tistate {
	return a.status
}

// ToYaml serializes the task instance to a YAML string.
func (a *AirflowTaskInstance) ToYaml() string {
	b, err := yaml.Marshal(a)
	if err != nil {
		return ""
	}
	return string(b)
}

// NewAirflowWorker creates a new AirflowWorker for the given host.
func NewAirflowWorker(host string) *AirflowWorker {
	return &AirflowWorker{
		host: host,
	}
}

// Host returns the worker host name.
func (a *AirflowWorker) Host() string {
	return a.host
}

// ToYaml serializes the worker to a YAML string.
func (a *AirflowWorker) ToYaml() string {
	b, err := yaml.Marshal(a)
	if err != nil {
		return ""
	}
	return string(b)
}
