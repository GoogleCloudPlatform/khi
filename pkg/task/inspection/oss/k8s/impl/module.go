// Copyright 2026 Google LLC
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

package ossk8s_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	ossk8s "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/oss/k8s"
)

// Module declares the OSS Kubernetes log files inspection type and the tasks that build timelines from uploaded kube-apiserver audit logs.
var Module = coreinspection.Module{
	Name: "oss/k8s",
	Scope: coreinspection.Scope{
		inspectioncore.InspectionTypeLabelKeyLogSource:    "file",
		inspectioncore.InspectionTypeLabelKeyEnvironment:  "oss",
		inspectioncore.InspectionTypeLabelKeyBasePlatform: "kubernetes",
	},
	InspectionTypes: []coreinspection.InspectionType{ossk8s.OSSKubernetesLogFilesInspectionType},
	Tasks: []coretask.UntypedTask{
		ossK8sAuditLogExtractorTask,
		ossK8sAuditLogErrorExtractorTask,
		inputAuditLogFilesTask,
		auditLogFileReaderTask,
		eventAuditLogFilterTask,
		nonEventAuditLogFilterTask,
		ossK8sEventLogIngesterTask,
		ossK8sEventLogGrouperTask,
		ossK8sEventLogToTimelineMapperTask,
		ossK8sAuditLogParserTailTask,
	},
}
