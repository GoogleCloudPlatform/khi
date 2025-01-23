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

package model

import (
	"gopkg.in/yaml.v3"
)

const (
	// ref: https://airflow.apache.org/docs/apache-airflow/stable/core-concepts/tasks.html#task-instances
	TASKINSTANCE_NONE              string = "none"
	TASKINSTANCE_SCHEDULED         string = "scheduled"
	TASKINSTANCE_QUEUED            string = "queued"
	TASKINSTANCE_RUNNING           string = "running"
	TASKINSTANCE_SUCCESS           string = "success"
	TASKINSTANCE_SHUTDOWN          string = "shutdown"
	TASKINSTANCE_RESTARTING        string = "restarting"
	TASKINSTANCE_FAILED            string = "failed"
	TASKINSTANCE_SKIPPED           string = "skipped"
	TASKINSTANCE_UP_FOR_RETRY      string = "up_for_retry"
	TASKINSTANCE_DEFERRED          string = "deferred"
	TASKINSTANCE_UP_FOR_RESCHEDULE string = "up_for_reschedule"
	TASKINSTANCE_REMOVED           string = "removed"
	TASKINSTANCE_UPSTREAM_FAILED   string = "upstream_failed"

	// Original States //
	// Zombie status for KHI view
	TASKINSTANCE_ZOMBIE string = "zombie"
)

// ref: https://github.com/apache/airflow/blob/main/airflow/models/taskinstance.py#L1187
type AirflowTaskInstance struct {
	dagId    string // primary key
	taskId   string // primary key
	runId    string // primary key
	mapIndex string // primary key
	host     string
	status   string
}

func NewAirflowTaskInstance(dagId string, taskId string, runId string, mapIndex string, host string, status string) *AirflowTaskInstance {
	return &AirflowTaskInstance{
		dagId:    dagId,
		taskId:   taskId,
		runId:    runId,
		mapIndex: mapIndex,
		host:     host,
		status:   status,
	}
}

func (a *AirflowTaskInstance) DagId() string {
	return a.dagId
}

func (a *AirflowTaskInstance) TaskId() string {
	return a.taskId
}

func (a *AirflowTaskInstance) RunId() string {
	return a.runId
}

func (a *AirflowTaskInstance) MapIndex() string {
	return a.mapIndex
}

func (a *AirflowTaskInstance) Host() string {
	return a.host
}

func (a *AirflowTaskInstance) Status() string {
	return a.status
}

func (a *AirflowTaskInstance) ToYaml() string {
	b, err := yaml.Marshal(a)
	if err != nil {
		return ""
	}
	return string(b)
}

type AirflowWorker struct {
	host string
}

func NewAirflowWorker(host string) *AirflowWorker {
	return &AirflowWorker{
		host: host,
	}
}

func (a *AirflowWorker) Host() string {
	return a.host
}

func (a *AirflowWorker) ToYaml() string {
	b, err := yaml.Marshal(a)
	if err != nil {
		return ""
	}
	return string(b)
}

type DagFileProcessorStats struct {
	dagFilePath    string `yaml:"dagFilePath"`
	runtime        string `yaml:"runtime"`
	numberOfDags   string `yaml:"numberOfDags"`
	numberOfErrors string `yaml:"numberOfErrors"`
}

func NewDagFileProcessorStats(dagFilePath string, runtime string, numberOfDags string, numberOfErrors string) *DagFileProcessorStats {
	return &DagFileProcessorStats{
		dagFilePath:    dagFilePath,
		runtime:        runtime,
		numberOfDags:   numberOfDags,
		numberOfErrors: numberOfErrors,
	}
}

func (s *DagFileProcessorStats) ToYaml() string {
	b, err := yaml.Marshal(s)
	if err != nil {
		return ""
	}
	return string(b)
}

func (s *DagFileProcessorStats) DagFilePath() string {
	return s.dagFilePath
}

func (s *DagFileProcessorStats) Runtime() string {
	return s.runtime
}

func (s *DagFileProcessorStats) NumberOfDags() string {
	return s.numberOfDags
}

func (s *DagFileProcessorStats) NumberOfErrors() string {
	return s.numberOfErrors
}
