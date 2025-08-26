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

package googlecloudlogonpremapiaudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// OnPremCloudAPITaskIDPrefix is the prefix for all task ids related to google cloud on-prem API audit.
var OnPremCloudAPITaskIDPrefix = "cloud.google.com/onprem-api/"

// OnPremCloudAuditLogQueryTaskID is the task id for the task that queries on-prem API audit logs from Cloud Logging.
var OnPremCloudAuditLogQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](OnPremCloudAPITaskIDPrefix + "query")

// OnPremCloudAuditLogParseTaskID is the task id for the task that parses on-prem API audit logs.
var OnPremCloudAuditLogParseTaskID = taskid.NewDefaultImplementationID[struct{}](OnPremCloudAPITaskIDPrefix + "parser")
