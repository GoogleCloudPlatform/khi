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

package bigquery

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
)

type BigQueryJob struct {
	Statistics BigQueryJobStatistics `yaml:"jobStatistics"`
	Name       BigqueryJobName       `yaml:"jobName"`
	Status     BigQueryJobStatus     `yaml:"jobStatus"`
}

func (b *BigQueryJob) ToResourcePath() resourcepath.ResourcePath {
	jobid := fmt.Sprintf("%s:%s.%s", b.Name.ProjectId, b.Name.Location, b.Name.JobId)
	// BigQuery > Reservation > ProjectID > JobID
	return resourcepath.NameLayerGeneralItem("BigQuery", b.Statistics.Reservation, b.Name.ProjectId, jobid)
}

type BigQueryJobStatistics struct {
	CreateTime  string `yaml:"createTime"`
	StartTime   string `yaml:"startTime"`
	EndTime     string `yaml:"endTime"`
	Reservation string `yaml:"reservation"`
}

type BigqueryJobName struct {
	JobId     string `yaml:"jobId"`
	ProjectId string `yaml:"projectId"`
	Location  string `yaml:"location"`
}

type BigQueryJobStatus struct {
	State           string                `yaml:"state"`
	Error           BigQueryStatusError   `yaml:"error"`
	AdditionalError []BigQueryStatusError `yaml:"additionalErrors"`
}

type BigQueryStatusError struct {
	Code    int64  `yaml:"code"`
	Message string `yaml:"message"`
}
