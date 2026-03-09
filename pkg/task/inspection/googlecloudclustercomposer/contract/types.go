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

package googlecloudclustercomposer_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"gopkg.in/yaml.v3"
)

type Tistate string

const (
	// ref: https://airflow.apache.org/docs/apache-airflow/stable/core-concepts/tasks.html#task-instances
	TASKINSTANCE_NONE              Tistate = "none"
	TASKINSTANCE_SCHEDULED         Tistate = "scheduled"
	TASKINSTANCE_QUEUED            Tistate = "queued"
	TASKINSTANCE_RUNNING           Tistate = "running"
	TASKINSTANCE_SUCCESS           Tistate = "success"
	TASKINSTANCE_SHUTDOWN          Tistate = "shutdown"
	TASKINSTANCE_RESTARTING        Tistate = "restarting"
	TASKINSTANCE_FAILED            Tistate = "failed"
	TASKINSTANCE_SKIPPED           Tistate = "skipped"
	TASKINSTANCE_UP_FOR_RETRY      Tistate = "up_for_retry"
	TASKINSTANCE_DEFERRED          Tistate = "deferred"
	TASKINSTANCE_UP_FOR_RESCHEDULE Tistate = "up_for_reschedule"
	TASKINSTANCE_REMOVED           Tistate = "removed"
	TASKINSTANCE_UPSTREAM_FAILED   Tistate = "upstream_failed"

	// Original States //
	// Zombie status for KHI view
	TASKINSTANCE_ZOMBIE Tistate = "zombie"
)

// ref: https://github.com/apache/airflow/blob/main/airflow/models/taskinstance.py#L1187
type AirflowTaskInstance struct {
	dagId    string // primary key
	taskId   string // primary key
	runId    string // primary key
	mapIndex string // primary key
	host     string
	status   Tistate
}

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

func (a *AirflowTaskInstance) Status() Tistate {
	return a.status
}

func (a *AirflowTaskInstance) ToYaml() string {
	b, err := yaml.Marshal(a)
	if err != nil {
		return ""
	}
	return string(b)
}

func (a *AirflowTaskInstance) ResourcePath() resourcepath.ResourcePath {
	var detail = a.TaskId()
	if a.MapIndex() != "-1" {
		detail += "+" + a.MapIndex()
	}
	rp := resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "TaskInstance", a.DagId(), a.RunId(), detail)
	rp.ParentRelationship = enum.RelationshipAirflowTaskInstance
	return rp
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

func (a *AirflowWorker) ResourcePath() resourcepath.ResourcePath {
	return resourcepath.NameLayerGeneralItem("Apache Airflow", "AirflowWorker", "cluster-scope", a.Host())
}

type AirflowScheduler struct {
	host          string
	componentName string
}

func NewAirflowScheduler(host string, componentName string) *AirflowScheduler {
	return &AirflowScheduler{
		host:          host,
		componentName: componentName,
	}
}

func (a *AirflowScheduler) Host() string {
	return a.host
}

func (a *AirflowScheduler) ToYaml() string {
	b, err := yaml.Marshal(a)
	if err != nil {
		return ""
	}
	return string(b)
}

func (a *AirflowScheduler) ResourcePath() resourcepath.ResourcePath {
	return resourcepath.SubresourceLayerGeneralItem("Apache Airflow", "AirflowScheduler", "cluster-scope", a.Host(), a.componentName)
}
