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

// Package googlecloudloggkeautoscaler_contract contains the task IDs for the GKE autoscaler tasks.
package googlecloudloggkeautoscaler_contract

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

const gkeAutoscalerTaskIDPrefix = "cloud.google.com/gke/log/autoscaler/"

// ListLogEntriesTaskID is the task id for the task that queries GKE autoscaler logs from Cloud Logging.
var ListLogEntriesTaskID = taskid.NewDefaultImplementationID[[]*log.Log](gkeAutoscalerTaskIDPrefix + "query")

// FieldSetReaderTaskID is the task id for the task that reads the common field set from GKE autoscaler logs.
var FieldSetReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](gkeAutoscalerTaskIDPrefix + "fieldset-reader")

// LogGrouperTaskID is the task id for the task that groups GKE autoscaler logs.
var LogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](gkeAutoscalerTaskIDPrefix + "log-grouper")

// LogIngesterTaskID is the task id for the task that serializes GKE autoscaler logs.
var LogIngesterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](gkeAutoscalerTaskIDPrefix + "log-serializer")

// LogToTimelineMapperTaskID is the task id for the task that modifies the history based on GKE autoscaler logs.
var LogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](gkeAutoscalerTaskIDPrefix + "timeline-mapper")
