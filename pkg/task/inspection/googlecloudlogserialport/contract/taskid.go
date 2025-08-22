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
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

const TaskIDPrefix = "cloud.google.com/log/serialport/"

// SerialPortLogQueryTaskID is the task id for the task that queries serial port logs from Cloud Logging.
var SerialPortLogQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "query")

// SerialPortLogParserTaskID is the task id for the task that parses serial port logs.
var SerialPortLogParserTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "parser")
