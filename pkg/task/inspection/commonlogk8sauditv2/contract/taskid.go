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

package commonlogk8sauditv2_contract

import (
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

var TaskIDPrefix = "khi.google.com/k8s-common-auditlog-v2/"

// K8sAuditLogProviderRef is the task reference for the task to fetch k8s audit log.
// The actual implementation for this reference must provide log array with the K8sAuditLogFieldSet.
var K8sAuditLogProviderRef = taskid.NewTaskReference[[]*log.Log](TaskIDPrefix + "k8s-auditlog-provider")

// K8sAuditLogSerializerTaskID is the task ID for the task to serialize the k8s audit log.
var K8sAuditLogSerializerTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "k8s-auditlog-serializer")

// LogSummaryGrouperTaskID is the task ID for the task to group logs for summary generation.
var LogSummaryGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "log-summary-grouper")

// ChangeTargetGrouperTaskID is the task ID for the task to group logs by the target resource.
var ChangeTargetGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "change-target-grouper")

// LogSummaryHistoryModifierTaskID is the task ID for the task to generate log summary from given k8s audit log.
var LogSummaryHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "log-summary-history-modifier")

// ConditionHistoryModifierTaskID is the task ID for the task to generate condition history.
var ConditionHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "condition-history-modifier")

// SuccessLogFilterTaskID is the task ID for the task to filter success logs.
var SuccessLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "success-log-filter")

// NonSuccessLogFilterTaskID is the task ID for the task to filter non-success logs.
var NonSuccessLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "non-success-log-filter")

// NonSuccessLogGrouperTaskID is the task ID for the task to group non-success logs.
var NonSuccessLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "non-success-log-grouper")

// NonSuccessLogHistoryModifierTaskID is the task ID for the task to generate history from non-success logs.
var NonSuccessLogHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "non-success-history-modifier")

// LogSorterTaskID is the task ID for the task to sort logs by time.
var LogSorterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "log-sorter")

var ResourceLifetimeTrackerTaskID = taskid.NewDefaultImplementationID[ResourceChangeLogGroupMap](TaskIDPrefix + "resource-lifetime-tracker")

var ResourceRevisionHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "resource-revision-history-modifier")

var ResourceOwnerReferenceModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "resource-owner-reference-modifier")

var EndpointResourceHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "endpoint-resource-history-modifier")

var ManifestGeneratorTaskID = taskid.NewDefaultImplementationID[ResourceChangeLogGroupMap](TaskIDPrefix + "manifest-generator")
