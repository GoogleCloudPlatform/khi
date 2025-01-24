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
)

// composer#taskinstance#DAGID#RUNID#TASKID-MAPINDEX
func ComposerTaskInstance(ti *model.AirflowTaskInstance) ResourcePath {
	var detail = ti.TaskId()
	if ti.MapIndex() != "-1" {
		detail += "+" + ti.MapIndex()
	}
	return SubresourceLayerGeneralItem("Cloud Composer", "Task Instance", ti.DagId(), ti.RunId(), detail)
}

// composer#airflow-worker#HOST
func ComposerAirflowWorker(wo *model.AirflowWorker) ResourcePath {
	return NameLayerGeneralItem("Cloud Composer", "Airflow Worker", "cluster-scope", wo.Host())
}

func DagFileProcessorStats(stats *model.DagFileProcessorStats) ResourcePath {
	return NameLayerGeneralItem("Cloud Composer", "Dag File Processor Stats", "cluster-scope", stats.DagFilePath())
}
