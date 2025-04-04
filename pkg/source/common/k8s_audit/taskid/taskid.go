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

package common_k8saudit_taskid

import (
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// CommonAuditLogSource is a task ID for the task to inject logs and dependencies specific to the log source.
// The task needs to return types.AuditLogParserLogSource as its result.
var CommonAuitLogSource = taskid.NewTaskReference[*types.AuditLogParserLogSource](task.KHISystemPrefix + "audit-log-source")

var k8sAuditTaskIDPrefix = task.KHISystemPrefix + "feature/k8s_audit/"

var TimelineGroupingTaskID = taskid.NewDefaultImplementationID[[]*types.TimelineGrouperResult](k8sAuditTaskIDPrefix + "timelne-grouping")
var ManifestGenerateTaskID = taskid.NewDefaultImplementationID[[]*types.TimelineGrouperResult](k8sAuditTaskIDPrefix + "manifest-generate")
var LogConvertTaskID = taskid.NewDefaultImplementationID[struct{}](k8sAuditTaskIDPrefix + "log-convert")
var CommonLogParseTaskID = taskid.NewDefaultImplementationID[[]*types.AuditLogParserInput](k8sAuditTaskIDPrefix + "common-fields-parse")
