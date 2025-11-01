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

package googlecloudlogserialport_contract

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

const TaskIDPrefix = "cloud.google.com/log/serialport/"

// LogQueryTaskID is the task id for the task that queries serial port logs from GCE nodes.
var LogQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "query")

// LogFilterTaskID is the task id for filtering empty messages incldued in the serial port logs.
var LogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "filter")

// FieldSetReadTaskID is the task id for reading serial port node specific fields(GCESerialPortLogFieldSet).
var FieldSetReadTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "field-set-read")

// LogSerializerTaskID is the task id to serialize logs to history.
var LogSerializerTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "log-serializer")

// LogGrouperTaskID is the task id to group logs by node name and serial port number.
var LogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "log-grouper")

// HistoryModifierTaskID is the task id to relate serialized logs to events on timeline.
var HistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "history-modifier")
