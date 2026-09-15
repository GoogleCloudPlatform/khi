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

package k8saudit

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	commonk8saudit "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

// TaskIDPrefix is the prefix for all task IDs in the googlecloudlogk8saudit package.
const TaskIDPrefix = "cloud.google.com/log/k8s-audit/"

var GCPK8sAuditLogListLogEntriesTaskID = taskid.NewImplementationID(commonk8saudit.K8sAuditLogProviderRef, "gcp")

var GCPK8sAuditLogExtractorTaskID = taskid.NewImplementationID(commonk8saudit.K8sAuditLogExtractorRef, "gcp")

var GCPK8sAuditLogErrorExtractorTaskID = taskid.NewImplementationID(commonk8saudit.K8sAuditLogErrorExtractorRef, "gcp")

var GCPK8sAuditLogParserTailTaskID = taskid.NewImplementationID(commonk8saudit.K8sAuditLogParserTailRef, "gcp")

// NEGToBackendServiceDiscoveryTaskID is the task ID for the discovery task that extracts NEG to BackendService mappings from Kubernetes Audit logs.
var NEGToBackendServiceDiscoveryTaskID = taskid.NewDefaultImplementationID[k8scommon.NEGToBackendServiceMap](TaskIDPrefix + "neg-discovery")
