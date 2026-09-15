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

package googlecloudloggkeapiaudit_impl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudloggkeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var (
	pathUpdate   = structured.CompileFieldPath("update")
	pathCluster  = structured.CompileFieldPath("cluster")
	pathNodePool = structured.CompileFieldPath("nodePool")

	clusterUpdateCandidatePaths  = []structured.FieldPath{pathUpdate, pathCluster}
	nodePoolUpdateCandidatePaths = []structured.FieldPath{pathUpdate, pathNodePool}
)

// LogIngesterTask is a task that serializes GKE audit logs for storage in the history builder.
var LogIngesterTask = googlecloudcommon_contract.NewGCPOperationLogIngesterTask(
	googlecloudloggkeapiaudit_contract.LogIngesterTaskID,
	googlecloudloggkeapiaudit_contract.ListLogEntriesTaskID.Ref(),
	googlecloudloggkeapiaudit_contract.LogTypeGkeAudit,
)

// LogGrouperTask is a task that groups GKE audit logs by GKE cluster or nodepool name.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudloggkeapiaudit_contract.LogGrouperTaskID, googlecloudloggkeapiaudit_contract.ListLogEntriesTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		resourceFieldSet, err := googlecloudloggkeapiaudit_contract.ExtractGKEAuditLogResource(l.NodeReader)
		if err != nil {
			return ""
		}
		if resourceFieldSet.IsCluster() {
			return fmt.Sprintf("cluster/%s", resourceFieldSet.ClusterName)
		}
		return fmt.Sprintf("nodepool/%s/%s", resourceFieldSet.ClusterName, resourceFieldSet.NodepoolName)
	},
)

// LogToTimelineMapperTask is a task that maps GKE audit logs to timeline elements.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[*googlecloudcommon_contract.GCPOperationTracker](googlecloudloggkeapiaudit_contract.LogToTimelineMapperTaskID, &gkeAuditLogLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabel(`GKE Audit Logs`,
		`Gather GKE audit logs to visualize the creation, upgrade, and deletion of clusters and node pools on timelines.`,
		5000,
		true),
)

// gkeAuditLogLogToTimelineMapperSetting implements the LogToTimelineMapper interface for GKE audit logs.
type gkeAuditLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.SinglePassMapperBase[*googlecloudcommon_contract.GCPOperationTracker]
}

// Dependencies returns additional task dependencies used in timeline mapper.
func (g *gkeAuditLogLogToTimelineMapperSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		googlecloudloggkeapiaudit_contract.InitialResourceStateProviderRef,
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (g *gkeAuditLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudloggkeapiaudit_contract.LogGrouperTaskID.Ref()
}

// LogIngesterTask returns a reference to the log ingester task.
func (g *gkeAuditLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[struct{}] {
	return googlecloudloggkeapiaudit_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup maps GKE audit log resource updates to timeline events/revisions.
func (g *gkeAuditLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *googlecloudcommon_contract.GCPOperationTracker) (*khifilev6.TimelineChangeSet, *googlecloudcommon_contract.GCPOperationTracker, error) {
	if tracker == nil {
		tracker = googlecloudcommon_contract.NewGCPOperationTracker()
	}
	auditFieldSet, err := googlecloudcommon_contract.ExtractGCPAuditLog(l.NodeReader)
	if err != nil {
		return nil, tracker, err
	}
	resourceFieldSet, err := googlecloudloggkeapiaudit_contract.ExtractGKEAuditLogResource(l.NodeReader)
	if err != nil {
		return nil, tracker, err
	}

	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, auditFieldSet.ProjectID)
	clusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, projectTimeline, resourceFieldSet.ClusterName)

	var targetTimeline *khifilev6.TimelinePath
	if resourceFieldSet.IsCluster() {
		targetTimeline = clusterTimeline
	} else {
		targetTimeline = googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, resourceFieldSet.NodepoolName)
	}

	initialStateProvider := coretask.GetTaskResult(ctx, googlecloudloggkeapiaudit_contract.InitialResourceStateProviderRef)
	var initialState *googlecloudloggkeapiaudit_contract.InitialResourceState
	var hasInitialState bool
	if resourceFieldSet.IsCluster() {
		initialState, hasInitialState = initialStateProvider.ClusterInitialState(resourceFieldSet.ClusterName)
	} else {
		initialState, hasInitialState = initialStateProvider.NodePoolInitialState(resourceFieldSet.ClusterName, resourceFieldSet.NodepoolName)
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	methodNameParts := strings.Split(auditFieldSet.MethodName, ".")
	shortMethodName := methodNameParts[len(methodNameParts)-1]

	operationTimeline := googlecloudcommon_contract.MustGCPOperationTimeline(ctx, targetTimeline, shortMethodName, auditFieldSet.OperationID)

	isCreate := strings.HasPrefix(shortMethodName, "Create") || strings.HasPrefix(shortMethodName, "Enroll")
	isDelete := strings.HasPrefix(shortMethodName, "Delete") || strings.HasPrefix(shortMethodName, "Unenroll")

	if hasInitialState && !tracker.HasResourceRevision(targetTimeline) {
		tracker.MarkResourceRevision(targetTimeline)
	}

	if !isCreate && !isDelete && !auditFieldSet.ImmediateOperation() {
		g.processUpdateOperationLog(ctx, cs, tracker, targetTimeline, operationTimeline, &auditFieldSet, l.Timestamp, resourceFieldSet.IsCluster(), initialState)
	} else {
		googlecloudcommon_contract.ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetTimeline, operationTimeline, &auditFieldSet, l.Timestamp, shortMethodName, resourceFieldSet.IsCluster())
		if isCreate && auditFieldSet.Request != nil {
			resourceBodyField := pathNodePool
			if resourceFieldSet.IsCluster() {
				resourceBodyField = pathCluster
			}
			if subReader, err := auditFieldSet.Request.GetReader(resourceBodyField); err == nil {
				tracker.SetCurrentManifest(subReader.Node)
			}
		} else if isDelete && auditFieldSet.Ending() {
			tracker.SetCurrentManifest(nil)
		}
	}

	return cs, tracker, nil
}

// processUpdateOperationLog handles non-create, non-delete asynchronous operations on clusters and node pools.
// It stages a dummy LogNotFound revision at Unix time 0 if the resource was not observed beforehand and has no CAI snapshot,
// and stages a VerbUpdate revision with merged manifests upon operation completion.
func (g *gkeAuditLogLogToTimelineMapperSetting) processUpdateOperationLog(
	ctx context.Context,
	cs *khifilev6.TimelineChangeSet,
	tracker *googlecloudcommon_contract.GCPOperationTracker,
	targetTimeline *khifilev6.TimelinePath,
	operationTimeline *khifilev6.TimelinePath,
	audit *googlecloudcommon_contract.GCPAuditLogFieldSet,
	logTimestamp time.Time,
	isCluster bool,
	initialState *googlecloudloggkeapiaudit_contract.InitialResourceState,
) {
	hasInitialState := initialState != nil && initialState.ResourceBody != nil
	if tracker.CurrentManifest() == nil && hasInitialState {
		tracker.SetCurrentManifest(initialState.ResourceBody)
	}

	if !tracker.HasResourceRevision(targetTimeline) {
		var stateNotFound *pb.RevisionState
		if isCluster {
			stateNotFound = commonlogk8saudit_contract.RevisionStateK8sClusterExistingLogNotFound
		} else {
			stateNotFound = commonlogk8saudit_contract.RevisionStateK8sNodepoolExistingLogNotFound
		}
		cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
			VerbType:     commonlogk8saudit_contract.VerbCreate,
			StateType:    stateNotFound,
			Principal:    audit.PrincipalEmail,
			ChangedTime:  time.Unix(0, 0),
			ResourceBody: nil,
		})
		tracker.MarkResourceRevision(targetTimeline)
	}

	if audit.Ending() && audit.Status <= 0 {
		var state *pb.RevisionState
		if isCluster {
			state = commonlogk8saudit_contract.RevisionStateK8sClusterExisting
		} else {
			state = commonlogk8saudit_contract.RevisionStateK8sNodepoolExisting
		}

		var patchNode structured.Node
		if audit.Request != nil {
			candidatePaths := nodePoolUpdateCandidatePaths
			if isCluster {
				candidatePaths = clusterUpdateCandidatePaths
			}
			for _, fieldPath := range candidatePaths {
				if reader, err := audit.Request.GetReader(fieldPath); err == nil && reader.Node != nil {
					patchNode = reader.Node
					break
				}
			}
			if patchNode == nil {
				patchNode = audit.Request.Node
			}
		}

		var bodyNode structured.Node
		if base := tracker.CurrentManifest(); base != nil && patchNode != nil {
			merged, err := structured.MergeNode(base, patchNode, structured.MergeConfiguration{
				MergeMapOrderStrategy: &structured.DefaultMergeMapOrderStrategy{},
			})
			if err == nil && merged != nil {
				bodyNode = merged
			} else {
				bodyNode = patchNode
			}
		} else if patchNode != nil {
			bodyNode = patchNode
		} else {
			bodyNode = tracker.CurrentManifest()
		}
		tracker.SetCurrentManifest(bodyNode)

		cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
			VerbType:     commonlogk8saudit_contract.VerbUpdate,
			StateType:    state,
			Principal:    audit.PrincipalEmail,
			ChangedTime:  logTimestamp,
			ResourceBody: bodyNode,
		})
		tracker.MarkResourceRevision(targetTimeline)
	}

	tracker.ProcessOperationLog(ctx, cs, operationTimeline, audit, logTimestamp)
}

var _ inspectiontaskbase.LogToTimelineMapper[*googlecloudcommon_contract.GCPOperationTracker] = (*gkeAuditLogLogToTimelineMapperSetting)(nil)
