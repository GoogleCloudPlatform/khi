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

package constants

import "github.com/GoogleCloudPlatform/khi/pkg/task"

// CommonAuditLogSource is a task ID for the task to inject logs and dependencies specific to the log source.
// The task needs to return types.AuditLogParserLogSource as its result.
const CommonAuitLogSource = task.KHISystemPrefix + "audit-log-source"

const k8sAuditTaskIDPrefix = task.KHISystemPrefix + "feature/k8s_audit/"

const TimelineGroupingTaskID = k8sAuditTaskIDPrefix + "timelne-grouping"
const ManifestGenerateTaskID = k8sAuditTaskIDPrefix + "manifest-generate"
const LogConvertTaskID = k8sAuditTaskIDPrefix + "log-convert"
const CommonLogParseTaskID = k8sAuditTaskIDPrefix + "common-fields-parse"
