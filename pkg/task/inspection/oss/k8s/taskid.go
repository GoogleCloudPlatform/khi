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

package ossk8s

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
)

// OSSTaskPrefix is the prefixes of IDs used in OSS related tasks.
const OSSTaskPrefix = "khi.google.com/oss/"

// InputAuditLogFilesFormTaskID is the task ID for the audit log file upload form.
var InputAuditLogFilesFormTaskID = taskid.NewDefaultImplementationID[upload.UploadResult](OSSTaskPrefix + "form/kube-apiserver-audit-log-files")

// AuditLogFileReaderTaskID is the task ID for reading uploaded OSS Kubernetes audit log files.
var AuditLogFileReaderTaskID = taskid.NewDefaultImplementationID[[]*log.Log](OSSTaskPrefix + "audit-log-reader")

// NonEventAuditLogFilterTaskID is the task ID for filtering non-Event audit logs in OSS Kubernetes.
var NonEventAuditLogFilterTaskID = taskid.NewImplementationID(k8saudit.K8sAuditLogProviderRef, "oss")

// EventAuditLogFilterTaskID is the task ID for filtering Event audit logs in OSS Kubernetes.
var EventAuditLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](OSSTaskPrefix + "audit-log-filter-event-audit")

// OSSK8sEventLogIngesterTaskID is the task ID for ingesting OSS Kubernetes Event logs.
var OSSK8sEventLogIngesterTaskID = taskid.NewDefaultImplementationID[struct{}](OSSTaskPrefix + "event-log-ingester")

// OSSK8sEventLogGrouperTaskID is the task ID for grouping OSS Kubernetes Event logs.
var OSSK8sEventLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](OSSTaskPrefix + "event-log-grouper")

// OSSK8sEventLogToTimelineMapperTaskID is the task ID for mapping OSS Kubernetes Event logs to timelines.
var OSSK8sEventLogToTimelineMapperTaskID = taskid.NewDefaultImplementationID[struct{}](OSSTaskPrefix + "event-timeline-mapper")

// OSSK8sAuditLogExtractorTaskID is the task ID for extracting OSS Kubernetes audit log fields.
var OSSK8sAuditLogExtractorTaskID = taskid.NewImplementationID(k8saudit.K8sAuditLogExtractorRef, "oss")

// OSSK8sAuditLogErrorExtractorTaskID is the task ID for extracting errors from OSS Kubernetes audit logs.
var OSSK8sAuditLogErrorExtractorTaskID = taskid.NewImplementationID(k8saudit.K8sAuditLogErrorExtractorRef, "oss")

// OSSK8sAuditLogParserTailTaskID is the task ID for the OSS Kubernetes audit log parser tail.
var OSSK8sAuditLogParserTailTaskID = taskid.NewImplementationID(k8saudit.K8sAuditLogParserTailRef, "oss")
