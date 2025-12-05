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
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

// TaskIDPrefix is the prefix for all task IDs in the googlecloudlogk8saudit package.
const TaskIDPrefix = "cloud.google.com/log/k8s-audit/"

var GCPK8sAuditLogListLogEntriesTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "audit-list-log-entries")

var GCPK8sAuditLogCommonFieldSetReaderTaskID = taskid.NewImplementationID(commonlogk8sauditv2_contract.K8sAuditLogProviderRef, "gcp")

var GCPK8sAuditLogParserTailTaskID = taskid.NewImplementationID(commonlogk8sauditv2_contract.K8sAuditLogParserTailRef, "gcp")
