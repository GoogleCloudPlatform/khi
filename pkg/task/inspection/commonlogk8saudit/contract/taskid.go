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

package commonlogk8saudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

var CommonK8sAuditLogTaskIDPrefix = "khi.google.com/k8s-common-auditlog/"

// CommonAuditLogSource is a task ID for the task to inject logs and dependencies specific to the log source.
// The task needs to return AuditLogParserLogSource as its result.
var CommonAuitLogSource = taskid.NewTaskReference[*AuditLogParserLogSource](CommonK8sAuditLogTaskIDPrefix + "audit-log-source")
var TimelineGroupingTaskID = taskid.NewDefaultImplementationID[[]*TimelineGrouperResult](CommonK8sAuditLogTaskIDPrefix + "timelne-grouping")
var ManifestGenerateTaskID = taskid.NewDefaultImplementationID[[]*TimelineGrouperResult](CommonK8sAuditLogTaskIDPrefix + "manifest-generate")
var LogConvertTaskID = taskid.NewDefaultImplementationID[struct{}](CommonK8sAuditLogTaskIDPrefix + "log-convert")
var CommonLogParseTaskID = taskid.NewDefaultImplementationID[[]*AuditLogParserInput](CommonK8sAuditLogTaskIDPrefix + "common-fields-parse")
