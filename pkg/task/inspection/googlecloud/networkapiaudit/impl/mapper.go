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

package networkapiaudit_impl

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/networkapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"gopkg.in/yaml.v3"
)

// logIngesterTask ingests the metadata of GCE Network API audit logs into the KHI v6 format.
var logIngesterTask = gcpcommon.DefineGCPOperationLogIngesterTask(
	networkapiaudit.LogIngesterTaskID,
	networkapiaudit.ListLogEntriesTaskID.Ref(),
	networkapiaudit.LogTypeNetworkAPI,
)

// logGrouperTask groups logs by the NEG resource name.
var logGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	networkapiaudit.LogGrouperTaskID,
	networkapiaudit.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			audit, err := gcpcommon.ExtractGCPAuditLog(l.NodeReader)
			if err != nil {
				return "unknown"
			}
			return audit.ResourceName
		}
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

// networkAuditInputs holds the input values that mapNetworkAuditLog reads.
type networkAuditInputs struct {
	clusterIdentity     k8scommon.GoogleCloudClusterIdentity
	negs                k8scommon.NEGNameToResourceIdentityMap
	ipLeases            k8saudit.IPLeaseHistory
	negToBackendService k8scommon.NEGToBackendServiceMap
}

// networkAuditTimelineMapper maps GCE Network API audit logs to resource timelines and operations in KHI v6 format.
type networkAuditTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*perNEGHistoryModificationStatus]
	clusterIdentity     coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	negs                coretask.Input[k8scommon.NEGNameToResourceIdentityMap]
	ipLeases            coretask.Input[k8saudit.IPLeaseHistory]
	negToBackendService coretask.Input[k8scommon.NEGToBackendServiceMap]
}

// ProcessLogByGroup maps the NEG audit log to resource timelines as state revisions.
func (m *networkAuditTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData *perNEGHistoryModificationStatus) (*khifilev6.TimelineChangeSet, *perNEGHistoryModificationStatus, error) {
	return mapNetworkAuditLog(ctx, l, prevGroupData, networkAuditInputs{
		clusterIdentity:     m.clusterIdentity.Get(ctx),
		negs:                m.negs.Get(ctx),
		ipLeases:            m.ipLeases.Get(ctx),
		negToBackendService: m.negToBackendService.Get(ctx),
	})
}

// mapNetworkAuditLog maps a GCE Network API audit log to resource timelines and operations in KHI v6 format.
func mapNetworkAuditLog(ctx context.Context, l *log.Log, prevGroupData *perNEGHistoryModificationStatus, inputs networkAuditInputs) (*khifilev6.TimelineChangeSet, *perNEGHistoryModificationStatus, error) {
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

	clusterIdentity := inputs.clusterIdentity
	negs := inputs.negs
	var negResourcePath *khifilev6.TimelinePath
	negName := getNegNameFromResourceName(auditFieldSet.ResourceName)

	if negResource, found := negs[negName]; found {
		negResourcePath = networkapiaudit.MustNEGTimeline(ctx, clusterIdentity.ClusterName, negResource.Namespace, negName)
	} else {
		negResourcePath = networkapiaudit.MustNEGTimeline(ctx, clusterIdentity.ClusterName, "unknown", negName)
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	// Add operation subresource under neg resource.
	var negOperationPath *khifilev6.TimelinePath
	if auditFieldSet.ImmediateOperation() {
		negOperationPath = negResourcePath
	} else {
		negOperationPath = networkapiaudit.MustNEGOperationTimeline(ctx, negResourcePath, auditFieldSet.MethodName, auditFieldSet.OperationID)
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
		startState = networkapiaudit.RevisionStateNEGEndpointAttaching
		endVerb = k8saudit.VerbReady
		endState = networkapiaudit.RevisionStateNEGEndpointAttached
	case "detachNetworkEndpoints":
		startVerb = k8saudit.VerbNonReady
		startState = networkapiaudit.RevisionStateNEGEndpointDetaching
		endVerb = k8saudit.VerbDelete
		endState = networkapiaudit.RevisionStateNEGEndpointDetached
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
		processEndpointRevisions(ctx, cs, l, &auditFieldSet, prevGroupData, inputs, negName, shortMethodName, negRequest, verb, state)
	}

	return cs, prevGroupData, nil
}

// processEndpointRevisions resolves resource endpoints (Pod or Node) and records the corresponding NEG subresource revisions.
func processEndpointRevisions(
	ctx context.Context,
	cs *khifilev6.TimelineChangeSet,
	l *log.Log,
	auditFieldSet *gcpcommon.GCPAuditLogFieldSet,
	prevGroupData *perNEGHistoryModificationStatus,
	inputs networkAuditInputs,
	negName string,
	shortMethodName string,
	negRequest *negAttachOrDetachRequest,
	verb *pb.Verb,
	state *pb.RevisionState,
) {
	negToBS := inputs.negToBackendService
	ipLeases := inputs.ipLeases

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

			clusterPath := k8saudit.MustK8sClusterTimeline(ctx, inputs.clusterIdentity.ClusterName)
			apiPath := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
			kindPath := k8saudit.MustK8sKindTimeline(ctx, apiPath, "pod")
			nsPath := k8saudit.MustK8sNamespaceTimeline(ctx, kindPath, holder.Namespace)
			resourceTimelinePath = k8saudit.MustK8sNamespacedResourceTimeline(ctx, nsPath, holder.Name)
			bsSubresourceName = holder.Name
			endpointKey = getPodEndpointKey(endpoint.IpAddress, endpoint.Port)
		case endpoint.Instance != "":
			nodeName := getInstanceNameFromResourceName(endpoint.Instance)
			clusterPath := k8saudit.MustK8sClusterTimeline(ctx, inputs.clusterIdentity.ClusterName)
			apiPath := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
			kindPath := k8saudit.MustK8sKindTimeline(ctx, apiPath, "node")
			resourceTimelinePath = k8saudit.MustK8sClusterScopeResourceTimeline(ctx, kindPath, nodeName)
			bsSubresourceName = nodeName
			endpointKey = getNodeEndpointKey(nodeName)
		default:
			continue
		}

		// Add revisions to the resource-level NEG subresource timeline.
		negSubresourcePath := networkapiaudit.MustNEGUnderResourceTimeline(ctx, resourceTimelinePath, negName)
		isKnown := prevGroupData.KnownEndpoints[endpointKey]
		addEndpointRevisions(cs, negSubresourcePath, shortMethodName, isKnown, l.Timestamp, verb, state, auditFieldSet.PrincipalEmail)

		// Add revisions to the BackendService-level NEG subresource timeline if associated.
		if bsName, found := negToBS[negName]; found {
			// BackendService is usually global in the context of gsmrsvd backends.
			bsPath := networkapiaudit.MustGCPResourceTimeline(ctx, inputs.clusterIdentity.ProjectID, "backendServices", bsName)
			bsNegSubresourcePath := networkapiaudit.MustNEGUnderResourceTimeline(ctx, bsPath, bsSubresourceName)
			addEndpointRevisions(cs, bsNegSubresourcePath, shortMethodName, isKnown, l.Timestamp, verb, state, auditFieldSet.PrincipalEmail)
		}

		prevGroupData.KnownEndpoints[endpointKey] = true
	}
}

var _ inspectiontaskbase.TimelineMapper[*perNEGHistoryModificationStatus] = (*networkAuditTimelineMapper)(nil)

// logToTimelineMapperTask maps GCE Network API audit logs to NEG timelines.
var logToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	networkapiaudit.LogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: networkapiaudit.LogIngesterTaskID.Ref(),
		GroupedLogs: networkapiaudit.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[*perNEGHistoryModificationStatus] {
		return &networkAuditTimelineMapper{
			clusterIdentity:     coretask.Use(b, k8scommon.ClusterIdentityTaskID.Ref()),
			negs:                coretask.Use(b, k8scommon.NEGNamesInventoryTaskID.Ref()),
			ipLeases:            coretask.Use(b, k8saudit.IPLeaseHistoryInventoryTaskID.Ref()),
			negToBackendService: coretask.Use(b, k8scommon.NEGToBackendServiceInventoryTaskID.Ref()),
		}
	},
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
			StateType:   networkapiaudit.RevisionStateNEGEndpointExistingLogNotFound,
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
