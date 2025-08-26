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

package googlecloudlogk8saudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

// TaskIDPrefix is the prefix for all task IDs in the googlecloudlogk8saudit package.
const TaskIDPrefix = "cloud.google.com/log/k8s-audit/"

// K8sAuditQueryTaskID is the task ID for querying Kubernetes audit logs from Google Cloud Logging.
var K8sAuditQueryTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "query-k8s_audit")

// K8sAuditParseTaskID is the task ID for the root task that parses Kubernetes audit logs.
var K8sAuditParseTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "audit-parser-v2")

// GKEK8sAuditLogSourceTaskID is the task ID for providing a log source of GKE Kubernetes audit logs for parsing.
var GKEK8sAuditLogSourceTaskID = taskid.NewImplementationID(commonlogk8saudit_contract.CommonAuitLogSource, "gcp")
