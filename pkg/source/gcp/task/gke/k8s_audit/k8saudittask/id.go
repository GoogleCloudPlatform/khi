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

package k8saudittask

import (
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
)

const K8sAuditQueryTaskID = gcp_task.GCPPrefix + "query/k8s_audit"
const K8sAuditParseTaskID = gcp_task.GCPPrefix + "/feature/audit-parser-v2"
const k8sAuditTaskIDPrefix = gcp_task.GCPPrefix + "feature/k8s_audit/"

const TimelineGroupingTaskID = k8sAuditTaskIDPrefix + "timelne-grouping"
const ManifestGenerateTaskID = k8sAuditTaskIDPrefix + "manifest-generate"
const LogConvertTaskID = k8sAuditTaskIDPrefix + "log-convert"
const CommonLogParseTaskID = k8sAuditTaskIDPrefix + "common-fields-parse"
