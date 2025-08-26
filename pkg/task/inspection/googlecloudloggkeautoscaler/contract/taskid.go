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
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

const gkeAutoscalerTaskIDPrefix = "cloud.google.com/gke/log/autoscaler/"

// AutoscalerQueryTaskID is the task id for the task that queries GKE autoscaler logs from Cloud Logging.
var AutoscalerQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](gkeAutoscalerTaskIDPrefix + "query")

// AutoscalerParserTaskID is the task id for the task that parses GKE autoscaler logs.
var AutoscalerParserTaskID = taskid.NewDefaultImplementationID[struct{}](gkeAutoscalerTaskIDPrefix + "parser")
