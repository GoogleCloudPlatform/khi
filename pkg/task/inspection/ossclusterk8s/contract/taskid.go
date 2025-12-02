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

package ossclusterk8s_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

// OSSTaskPrefix is the prefixes of IDs used in OSS related tasks.
const OSSTaskPrefix = "khi.google.com/oss/"

var InputAuditLogFilesFormTaskID = taskid.NewDefaultImplementationID[upload.UploadResult](OSSTaskPrefix + "form/kube-apiserver-audit-log-files")
var AuditLogFileReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](OSSTaskPrefix + "audit-log-reader")
var NonEventAuditLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](OSSTaskPrefix + "audit-log-filter-non-event-audit")
var EventAuditLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](OSSTaskPrefix + "audit-log-filter-event-audit")
var OSSK8sEventLogParserTaskID = taskid.NewDefaultImplementationID[struct{}](OSSTaskPrefix + "event-parser")

var OSSK8sAuditLogProviderTaskID = taskid.NewImplementationID(commonlogk8sauditv2_contract.K8sAuditLogProviderRef, "oss")
var OSSK8sAuditLogParserTailTaskID = taskid.NewImplementationID(commonlogk8sauditv2_contract.K8sAuditLogParserTailRef, "oss")
