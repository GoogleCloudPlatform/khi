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

package googlecloudlogmulticloudapiaudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

var MultiCloudAPIAuditLogTaskIDPrefix = "cloud.google.com/log/multicloud-api/"

// MultiCloudAPIQueryTaskID is the task id for the task that queries multicloud API logs from Cloud Logging.
var MultiCloudAPIQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](MultiCloudAPIAuditLogTaskIDPrefix + "query")

// MultiCloudAPIParserTaskID is the task id for the task that parses multicloud API logs.
var MultiCloudAPIParserTaskID = taskid.NewDefaultImplementationID[struct{}](MultiCloudAPIAuditLogTaskIDPrefix + "parser")
