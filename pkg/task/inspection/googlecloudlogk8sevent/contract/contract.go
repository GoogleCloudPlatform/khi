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

// package googlecloudlogk8sevent_contract defines the task IDs for Kubernetes Event Log inspection.
package googlecloudlogk8sevent_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

var GKEK8sEventLogTaskIDPrefix = "cloud.google.com/log/k8s-event/"

// GKEK8sEventLogQueryTaskID is the task ID for the task that queries Kubernetes Event logs from Cloud Logging.
var GKEK8sEventLogQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](GKEK8sEventLogTaskIDPrefix + "query")

// GKEK8sEventLogParserTaskID is the task ID for the task that parses Kubernetes Event logs.
var GKEK8sEventLogParserTaskID = taskid.NewDefaultImplementationID[struct{}](GKEK8sEventLogTaskIDPrefix + "parser")
