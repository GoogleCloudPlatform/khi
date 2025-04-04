// Copyright 2024 Google LLC
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

package gke_k8saudit_taskid

import (
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var K8sAuditQueryTaskID = taskid.NewDefaultImplementationID[[]*log.LogEntity](gcp_task.GCPPrefix + "query/k8s_audit")
var K8sAuditParseTaskID = taskid.NewDefaultImplementationID[struct{}](gcp_task.GCPPrefix + "/feature/audit-parser-v2")
