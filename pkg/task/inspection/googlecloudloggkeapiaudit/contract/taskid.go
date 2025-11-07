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

package googlecloudloggkeapiaudit_contract

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

var GKEAPIAuditLogTaskIDPrefix = "cloud.google.com/log/gke-api/"

// ListLogEntriesTaskID is the task id for the task that queries compute API logs from Cloud Logging.
var ListLogEntriesTaskID = taskid.NewDefaultImplementationID[[]*log.Log](GKEAPIAuditLogTaskIDPrefix + "query")

// FieldSetReaderTaskID is the task id to read the common fieldset for processing the log in the later task.
var FieldSetReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](GKEAPIAuditLogTaskIDPrefix + "fieldset-reader")

// LogSerializerTaskID is the task id to finalize the logs to be included in the final output.
var LogSerializerTaskID = taskid.NewDefaultImplementationID[[]*log.Log](GKEAPIAuditLogTaskIDPrefix + "log-serializer")

// LogGrouperTaskID is the task id to group logs by target instance to process logs in HistoryModifier in parallel.
var LogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](GKEAPIAuditLogTaskIDPrefix + "grouper")

// HistoryModifierTaskID is the task id for associating events/revisions with a given logs.
var HistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](GKEAPIAuditLogTaskIDPrefix + "history-modifier")
