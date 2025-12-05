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
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// TaskIDPrefix is the prefix for all task IDs in this package.
var TaskIDPrefix = "khi.google.com/k8s-common-auditlog-v2/"

// K8sAuditLogProviderRef is the task reference for the task to fetch k8s audit log.
// The actual implementation for this reference must provide log array with the K8sAuditLogFieldSet.
var K8sAuditLogProviderRef = taskid.NewTaskReference[[]*log.Log](TaskIDPrefix + "k8s-auditlog-provider")

// K8sAuditLogParserTailRef is the task reference for the task to depend all enabled k8s audit log parsing sub tasks.
var K8sAuditLogParserTailRef = taskid.NewTaskReference[struct{}](TaskIDPrefix + "k8s-auditlog-parser-tail")

// K8sAuditLogSerializerTaskID is the task ID for the task to serialize the k8s audit log.
var K8sAuditLogSerializerTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "k8s-auditlog-serializer")

// SuccessLogFilterTaskID is the task ID for the task to filter success logs.
var SuccessLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "success-log-filter")

// NonSuccessLogFilterTaskID is the task ID for the task to filter non-success logs.
var NonSuccessLogFilterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "non-success-log-filter")

// LogSorterTaskID is the task ID for the task to sort logs by time.
var LogSorterTaskID = taskid.NewDefaultImplementationID[[]*log.Log](TaskIDPrefix + "log-sorter")

// LogSummaryGrouperTaskID is the task ID for the task to group logs for summary generation.
var LogSummaryGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "log-summary-grouper")

// NonSuccessLogGrouperTaskID is the task ID for the task to group non-success logs.
var NonSuccessLogGrouperTaskID = taskid.NewDefaultImplementationID[inspectiontaskbase.LogGroupMap](TaskIDPrefix + "non-success-log-grouper")

// ChangeTargetGrouperTaskID is the task ID for the task to group logs by the target resource.
var ChangeTargetGrouperTaskID = taskid.NewDefaultImplementationID[ResourceLogGroupMap](TaskIDPrefix + "change-target-grouper")

// NamespaceRequestHistoryModifierTaskID is the task ID for the task recording events for requests against entire resources in namespace.
var NamespaceRequestHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "namespace-request-history-modifier")

// ManifestGeneratorTaskID is the task ID for the task to generate manifests.
var ManifestGeneratorTaskID = taskid.NewDefaultImplementationID[ResourceManifestLogGroupMap](TaskIDPrefix + "manifest-generator")

// ResourceLifetimeTrackerTaskID is the task ID for the task to track resource lifetime.
var ResourceLifetimeTrackerTaskID = taskid.NewDefaultImplementationID[ResourceManifestLogGroupMap](TaskIDPrefix + "resource-lifetime-tracker")

// LogSummaryHistoryModifierTaskID is the task ID for the task to generate log summary from given k8s audit log.
var LogSummaryHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "log-summary-history-modifier")

// NonSuccessLogHistoryModifierTaskID is the task ID for the task to generate history from non-success logs.
var NonSuccessLogHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "non-success-history-modifier")

// ResourceRevisionHistoryModifierTaskID is the task ID for the task to modify resource revision history.
var ResourceRevisionHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "resource-revision-history-modifier")

// ResourceOwnerReferenceModifierTaskID is the task ID for the task to modify resource owner reference.
var ResourceOwnerReferenceModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "resource-owner-reference-modifier")

// EndpointResourceHistoryModifierTaskID is the task ID for the task to modify endpoint resource history.
var EndpointResourceHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "endpoint-resource-history-modifier")

// PodPhaseHistoryModifierTaskID is the task ID for the task to modify pod phase history.
var PodPhaseHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "pod-phase-history-modifier")

// ContainerHistoryModifierTaskID is the task ID for the task to modify container history.
var ContainerHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "container-history-modifier")

// ConditionHistoryModifierTaskID is the task ID for the task to generate condition history.
var ConditionHistoryModifierTaskID = taskid.NewDefaultImplementationID[struct{}](TaskIDPrefix + "condition-history-modifier")

// NodeNameDiscoveryTaskID is the task ID for extracting node names from audit logs.
var NodeNameDiscoveryTaskID = taskid.NewDefaultImplementationID[[]string](TaskIDPrefix + "node-name-discovery")

// ResourceUIDDiscoveryTaskID is the task ID for extracting resource uids from audit logs.
var ResourceUIDDiscoveryTaskID = taskid.NewDefaultImplementationID[UIDToResourceIdentity](TaskIDPrefix + "resource-uid-discovery")

// ResourceUIDPatternFinderTaskID is the task ID to build the PatternFinder from aggregated UIDs obtained from the inventory task.
var ResourceUIDPatternFinderTaskID = taskid.NewDefaultImplementationID[patternfinder.PatternFinder[*ResourceIdentity]](TaskIDPrefix + "resource-uid-pattern-finder")

// ContainerIDDiscoveryTaskID is the task ID for extracting container ids from audit logs.
var ContainerIDDiscoveryTaskID = taskid.NewDefaultImplementationID[ContainerIDToContainerIdentity](TaskIDPrefix + "container-id-discovery")

// ContainerIDPatternFinderTaskID is the task ID to build the PatternFinder from aggregated container ids obtained from the inventory task.
var ContainerIDPatternFinderTaskID = taskid.NewDefaultImplementationID[patternfinder.PatternFinder[*ContainerIdentity]](TaskIDPrefix + "container-id-pattern-finder")

// IPLeaseHistoryDiscoveryTaskID is the task ID for extracting IP lease history from audit logs.
var IPLeaseHistoryDiscoveryTaskID = taskid.NewDefaultImplementationID[IPLeaseHistory](TaskIDPrefix + "ip-lease-history-discovery")
