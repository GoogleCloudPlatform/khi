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

package googlecloudlogcsm_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

const TaskIDPrefix = "cloud.google.com/log/csm-accesslog/"

// InputCSMResponseFlagsTaskID is the task ID for the form input that specifies which Envoy response flags to filter CSM access logs by.
var InputCSMResponseFlagsTaskID = taskid.NewDefaultImplementationID[*gcpqueryutil.SetFilterParseResult](TaskIDPrefix + "input/response-flags")

// ListLogEntriesTaskID is the task ID for the task that queries CSM access logs from Cloud Logging.
var ListLogEntriesTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "list-log-entries")

// FieldSetReaderTaskID is the task id to read the CSM related fieldset for processing the log in the later task.
var FieldSetReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "fieldset-reader")

// LogSerializerTaskID is the task id to finalize the logs to be included in the final output.
var LogSerializerTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "log-serializer")

// LogGrouperTaskID is the task ID to group CSM access logs by their reporter pod for parallel processing.
var LogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "grouper")

// HistoryModifierTaskID is the task ID for associating CSM access log events with resource timelines.
var HistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "history-modifier")
