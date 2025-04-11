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

package resourcepath

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// airflow#taskinstance#DAGID#RUNID#TASKID-MAPINDEX
func AirflowTaskInstance(ti *model.AirflowTaskInstance) ResourcePath {
	var detail = ti.TaskId()
	if ti.MapIndex() != "-1" {
		detail += "+" + ti.MapIndex()
	}
	resourcepath := SubresourceLayerGeneralItem("Apache Airflow", "TaskInstance", ti.DagId(), ti.RunId(), detail)
	resourcepath.ParentRelationship = enum.RelationshipAirflowTaskInstance
	return resourcepath
}

// airflow#airflow-worker#HOST
func AirflowWorker(wo *model.AirflowWorker) ResourcePath {
	return NameLayerGeneralItem("Apache Airflow", "AirflowWorker", "cluster-scope", wo.Host())
}

func DagFileProcessorStats(stats *model.DagFileProcessorStats) ResourcePath {
	return NameLayerGeneralItem("Apache Airflow", "Dag File Processor Stats", "cluster-scope", stats.DagFilePath())
}
