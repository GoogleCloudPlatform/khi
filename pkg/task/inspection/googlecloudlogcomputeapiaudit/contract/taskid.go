// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package googlecloudlogcomputeapiaudit_contract defines the contract for the googlecloudlogcomputeapiaudit task.
package googlecloudlogcomputeapiaudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// ComputeAPIAuditLogTaskIDPrefix is the prefix for the task IDs of the compute API audit log tasks.
var ComputeAPIAuditLogTaskIDPrefix = "cloud.google.com/log/compute-api/"

// ComputeAPIQueryTaskID is the task id for the task that queries compute API logs from Cloud Logging.
var ComputeAPIQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](ComputeAPIAuditLogTaskIDPrefix + "query")

// ComputeAPIParserTaskID is the task id for the task that parses compute API logs.
var ComputeAPIParserTaskID = taskid.NewDefaultImplementationID[struct{}](ComputeAPIAuditLogTaskIDPrefix + "parser")
