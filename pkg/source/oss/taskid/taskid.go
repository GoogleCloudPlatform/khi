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

package oss_taskid

import (
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	common_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// OSSTaskPrefix is the prefixes of IDs used in OSS related tasks.
const OSSTaskPrefix = "khi.google.com/oss/"

var OSSK8sAuditLogSourceTaskID = taskid.NewImplementationID(common_k8saudit_taskid.CommonAuitLogSource, "oss")
var OSSAPIServerAuditLogFileInputTask = taskid.NewDefaultImplementationID[upload.UploadResult](OSSTaskPrefix + "form/kube-apiserver-audit-log-files")
var OSSAPIServerAuditLogFileReader = taskid.NewDefaultImplementationID[[]*log.LogEntity](OSSTaskPrefix + "log-reader")
var OSSAPIServerAuditLogFilterAuditTaskID = taskid.NewDefaultImplementationID[[]*log.LogEntity](OSSTaskPrefix + "log-filter/audit")
var OSSAPIServerAuditLogFilterNonAuditTaskID = taskid.NewDefaultImplementationID[[]*log.LogEntity](OSSTaskPrefix + "log-filter/non-audit")
var OSSAuditLogFileReader = taskid.NewDefaultImplementationID[[]*log.LogEntity](OSSTaskPrefix + "log-reader")
var OSSK8sAuditLogParserTaskID = taskid.NewDefaultImplementationID[struct{}](OSSTaskPrefix + "audit-parser")
var OSSK8sEventLogParserTaskID = taskid.NewDefaultImplementationID[struct{}](OSSTaskPrefix + "event-parser")
