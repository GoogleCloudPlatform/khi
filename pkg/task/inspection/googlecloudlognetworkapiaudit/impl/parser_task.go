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

package googlecloudlognetworkapiaudit_impl

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	googlecloudlognetworkapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlognetworkapiaudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"gopkg.in/yaml.v3"
)

// LogIngesterTask is the task id to finalize the logs to be included in the final output.
var LogIngesterTask = gcpcommon.NewGCPOperationLogIngesterTask(
	googlecloudlognetworkapiaudit_contract.LogIngesterTaskID,
	googlecloudlognetworkapiaudit_contract.ListLogEntriesTaskID.Ref(),
	googlecloudlognetworkapiaudit_contract.LogTypeNetworkAPI,
)

// LogGrouperTask groups logs by the NEG resource name.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlognetworkapiaudit_contract.LogGrouperTaskID, googlecloudlognetworkapiaudit_contract.ListLogEntriesTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		audit, err := gcpcommon.ExtractGCPAuditLog(l.NodeReader)
		if err != nil {
			return "unknown"
		}
		return audit.ResourceName
	},
)

type negAttachOrDetachRequestEndpoint struct {
	Instance  string `yaml:"instance"`
	IpAddress string `yaml:"ipAddress"`
	Port      string `yaml:"port"`
}

type negAttachOrDetachRequest struct {
	NetworkEndpoints []*negAttachOrDetachRequestEndpoint `yaml:"networkEndpoints"`
}

type pendingNEGOperation struct {
	Method  string
	Request *negAttachOrDetachRequest
}

type perNEGHistoryModificationStatus struct {
	PendingOperations map[string]*pendingNEGOperation
	OperationTracker  *gcpcommon.GCPOperationTracker
	KnownEndpoints    map[string]bool
}

type networkAPITimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*perNEGHistoryModificationStatus]
}

// LogIngesterTask is the task reference that provides the ingested logs.
func (m *networkAPITimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return googlecloudlognetworkapiaudit_contract.LogIngesterTaskID.Ref()
}

// Dependencies are the additional references used in timeline mapper.
func (m *networkAPITimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8scommon.ClusterIdentityTaskID.Ref(),
		k8scommon.NEGNamesInventoryTaskID.Ref(),
		k8saudit.IPLeaseHistoryInventoryTaskID.Ref(),
		k8scommon.NEGToBackendServiceInventoryTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *networkAPITimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlognetworkapiaudit_contract.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps the NEG audit log to resource timelines as state revisions.
func (m *networkAPITimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData *perNEGHistoryModificationStatus) (*khifilev6.TimelineChangeSet, *perNEGHistoryModificationStatus, error) {
	auditFieldSet, err := gcpcommon.ExtractGCPAuditLog(l.NodeReader)
	if err != nil {
		return nil, prevGroupData, err
	}
	if prevGroupData == nil {
		prevGroupData = &perNEGHistoryModificationStatus{
			PendingOperations: make(map[string]*pendingNEGOperation),
			OperationTracker:  gcpcommon.NewGCPOperationTracker(),
			KnownEndpoints:    make(map[string]bool),
		}
	}

	clusterIdentity := coretask.GetTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref())
	negs := coretask.GetTaskResult(ctx, k8scommon.NEGNamesInventoryTaskID.Ref())
	var negResourcePath *khifilev6.TimelinePath
	negName := getNegNameFromResourceName(auditFieldSet.ResourceName)

	if negResource, found := negs[negName]; found {
		negResourcePath = googlecloudlognetworkapiaudit_contract.MustNEGTimeline(ctx, clusterIdentity.ClusterName, negResource.Namespace, negName)
	} else {
		negResourcePath = googlecloudlognetworkapiaudit_contract.MustNEGTimeline(ctx, clusterIdentity.ClusterName, "unknown", negName)
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	// Add operation subresource under neg resource.
	var negOperationPath *khifilev6.TimelinePath
	if auditFieldSet.ImmediateOperation() {
		negOperationPath = negResourcePath
	} else {
		negOperationPath = googlecloudlognetworkapiaudit_contract.MustNEGOperationTimeline(ctx, negResourcePath, auditFieldSet.MethodName, auditFieldSet.OperationID)
	}

	if auditFieldSet.ImmediateOperation() {
		cs.AddEvent(negOperationPath)
	} else {
		prevGroupData.OperationTracker.ProcessOperationLog(ctx, cs, negOperationPath, &auditFieldSet, l.Timestamp)
	}

	// Add neg subresource under resources with the same IP of the endpoint.
	shortMethodName := getShortMethodNameFromMethodName(auditFieldSet.MethodName)
	var startVerb, endVerb *pb.Verb
	var startState, endState *pb.RevisionState

	switch shortMethodName {
	case "attachNetworkEndpoints":
		startVerb = k8saudit.VerbCreate
		startState = googlecloudlognetworkapiaudit_contract.RevisionStateNEGEndpointAttaching
		endVerb = k8saudit.VerbReady
		endState = googlecloudlognetworkapiaudit_contract.RevisionStateNEGEndpointAttached
	case "detachNetworkEndpoints":
		startVerb = k8saudit.VerbNonReady
		startState = googlecloudlognetworkapiaudit_contract.RevisionStateNEGEndpointDetaching
		endVerb = k8saudit.VerbDelete
		endState = googlecloudlognetworkapiaudit_contract.RevisionStateNEGEndpointDetached
	default:
		return cs, prevGroupData, nil
	}

	var negRequest *negAttachOrDetachRequest
	var verb *pb.Verb
	var state *pb.RevisionState

	switch {
	case auditFieldSet.Starting():
		var err error
		negRequest, err = parseNEGAttachOrDetachRequest(&auditFieldSet)
		if err != nil {
			return nil, prevGroupData, err
		}
		if auditFieldSet.OperationID != "" {
			prevGroupData.PendingOperations[auditFieldSet.OperationID] = &pendingNEGOperation{
				Method:  shortMethodName,
				Request: negRequest,
			}
		}
		verb = startVerb
		state = startState
	case auditFieldSet.Ending():
		if op, found := prevGroupData.PendingOperations[auditFieldSet.OperationID]; found {
			delete(prevGroupData.PendingOperations, auditFieldSet.OperationID)
			if auditFieldSet.Status <= 0 {
				negRequest = op.Request
				verb = endVerb
				state = endState
			}
		}
	case auditFieldSet.ImmediateOperation():
		var err error
		negRequest, err = parseNEGAttachOrDetachRequest(&auditFieldSet)
		if err != nil {
			return nil, prevGroupData, err
		}
		verb = endVerb
		state = endState
	}

	if negRequest != nil {
		m.processEndpointRevisions(ctx, cs, l, &auditFieldSet, prevGroupData, clusterIdentity, negName, shortMethodName, negRequest, verb, state)
	}

	return cs, prevGroupData, nil
}

// processEndpointRevisions resolves resource endpoints (Pod or Node) and records the corresponding NEG subresource revisions.
func (m *networkAPITimelineMapper) processEndpointRevisions(
	ctx context.Context,
	cs *khifilev6.TimelineChangeSet,
	l *log.Log,
	auditFieldSet *gcpcommon.GCPAuditLogFieldSet,
	prevGroupData *perNEGHistoryModificationStatus,
	clusterIdentity k8scommon.GoogleCloudClusterIdentity,
	negName string,
	shortMethodName string,
	negRequest *negAttachOrDetachRequest,
	verb *pb.Verb,
	state *pb.RevisionState,
) {
	negToBS := coretask.GetTaskResult(ctx, k8scommon.NEGToBackendServiceInventoryTaskID.Ref())
	ipLeases := coretask.GetTaskResult(ctx, k8saudit.IPLeaseHistoryInventoryTaskID.Ref())

	for _, endpoint := range negRequest.NetworkEndpoints {
		var resourceTimelinePath *khifilev6.TimelinePath
		var bsSubresourceName string
		var endpointKey string

		switch {
		case endpoint.IpAddress != "" && endpoint.Port != "":
			lease, err := ipLeases.GetResourceLeaseHolderAt(endpoint.IpAddress, l.Timestamp)
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Failed to identify the holder of the IP %s.\n This might be because the IP holder resource wasn't updated during the log period ", endpoint.IpAddress))
				continue
			}
			holder := lease.Holder
			if holder.Kind != "pod" {
				slog.DebugContext(ctx, fmt.Sprintf("IP %s is held by non-pod resource %s/%s, skipping NEG mapping", endpoint.IpAddress, holder.Kind, holder.Name))
				continue
			}

			clusterPath := k8saudit.MustK8sClusterTimeline(ctx, clusterIdentity.ClusterName)
			apiPath := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
			kindPath := k8saudit.MustK8sKindTimeline(ctx, apiPath, "pod")
			nsPath := k8saudit.MustK8sNamespaceTimeline(ctx, kindPath, holder.Namespace)
			resourceTimelinePath = k8saudit.MustK8sNamespacedResourceTimeline(ctx, nsPath, holder.Name)
			bsSubresourceName = holder.Name
			endpointKey = getPodEndpointKey(endpoint.IpAddress, endpoint.Port)
		case endpoint.Instance != "":
			nodeName := getInstanceNameFromResourceName(endpoint.Instance)
			clusterPath := k8saudit.MustK8sClusterTimeline(ctx, clusterIdentity.ClusterName)
			apiPath := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
			kindPath := k8saudit.MustK8sKindTimeline(ctx, apiPath, "node")
			resourceTimelinePath = k8saudit.MustK8sClusterScopeResourceTimeline(ctx, kindPath, nodeName)
			bsSubresourceName = nodeName
			endpointKey = getNodeEndpointKey(nodeName)
		default:
			continue
		}

		// Add revisions to the resource-level NEG subresource timeline.
		negSubresourcePath := googlecloudlognetworkapiaudit_contract.MustNEGUnderResourceTimeline(ctx, resourceTimelinePath, negName)
		isKnown := prevGroupData.KnownEndpoints[endpointKey]
		addEndpointRevisions(cs, negSubresourcePath, shortMethodName, isKnown, l.Timestamp, verb, state, auditFieldSet.PrincipalEmail)

		// Add revisions to the BackendService-level NEG subresource timeline if associated.
		if bsName, found := negToBS[negName]; found {
			// BackendService is usually global in the context of gsmrsvd backends.
			bsPath := googlecloudlognetworkapiaudit_contract.MustGCPResourceTimeline(ctx, clusterIdentity.ProjectID, "backendServices", bsName)
			bsNegSubresourcePath := googlecloudlognetworkapiaudit_contract.MustNEGUnderResourceTimeline(ctx, bsPath, bsSubresourceName)
			addEndpointRevisions(cs, bsNegSubresourcePath, shortMethodName, isKnown, l.Timestamp, verb, state, auditFieldSet.PrincipalEmail)
		}

		prevGroupData.KnownEndpoints[endpointKey] = true
	}
}

var _ inspectiontaskbase.LogToTimelineMapper[*perNEGHistoryModificationStatus] = (*networkAPITimelineMapper)(nil)

// LogToTimelineMapperTask registers the mapper to resolve network status in timeline.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(googlecloudlognetworkapiaudit_contract.LogToTimelineMapperTaskID, &networkAPITimelineMapper{},
	inspectioncore.FeatureTaskLabel(`GCE Network Logs`,
		`Gather GCE Network API logs to visualize the provisioning and status transitions of Network Endpoint Groups (NEGs) on timelines.`,
		7000,
		true,
	),
)

func getNegNameFromResourceName(resourceName string) string {
	lastSlashIndex := strings.LastIndex(resourceName, "/")
	if lastSlashIndex == -1 {
		return resourceName
	}
	return resourceName[lastSlashIndex+1:]
}

func getInstanceNameFromResourceName(instance string) string {
	lastSlashIndex := strings.LastIndex(instance, "/")
	if lastSlashIndex == -1 {
		return instance
	}
	return instance[lastSlashIndex+1:]
}

func getShortMethodNameFromMethodName(methodName string) string {
	lastDotIndex := strings.LastIndex(methodName, ".")
	if lastDotIndex == -1 {
		return methodName
	}
	return methodName[lastDotIndex+1:]
}

func parseNEGAttachOrDetachRequest(auditFieldSet *gcpcommon.GCPAuditLogFieldSet) (*negAttachOrDetachRequest, error) {
	requestBody, err := auditFieldSet.RequestString()
	if err != nil {
		return nil, err
	}
	var negRequest negAttachOrDetachRequest
	err = yaml.Unmarshal([]byte(requestBody), &negRequest)
	if err != nil {
		return nil, err
	}
	return &negRequest, nil
}

func getPodEndpointKey(ip, port string) string {
	return fmt.Sprintf("pod:%s:%s", ip, port)
}

func getNodeEndpointKey(nodeName string) string {
	return fmt.Sprintf("node:%s", nodeName)
}

// addEndpointRevisions stages revisions for a NEG endpoint on the specified timeline path.
func addEndpointRevisions(
	cs *khifilev6.TimelineChangeSet,
	targetPath *khifilev6.TimelinePath,
	shortMethodName string,
	isKnown bool,
	changedTime time.Time,
	verb *pb.Verb,
	state *pb.RevisionState,
	principal string,
) {
	if shortMethodName == "detachNetworkEndpoints" && !isKnown {
		cs.AddRevision(targetPath, &khifilev6.StagingRevision{
			ChangedTime: time.Unix(0, 0),
			VerbType:    k8saudit.VerbUnknown,
			StateType:   googlecloudlognetworkapiaudit_contract.RevisionStateNEGEndpointExistingLogNotFound,
			Principal:   "N/A",
		})
	}
	cs.AddRevision(targetPath, &khifilev6.StagingRevision{
		ChangedTime: changedTime,
		VerbType:    verb,
		StateType:   state,
		Principal:   principal,
	})
}
