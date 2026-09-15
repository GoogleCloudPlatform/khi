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

package csm_impl

import (
	"context"
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csm"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// CSMTrafficDirectorLogIngesterTask is a task that ingests CSM Traffic Director logs.
var CSMTrafficDirectorLogIngesterTask = gcpcommon.NewGCPOperationLogIngesterTask(
	csm.CSMTrafficDirectorLogIngesterTaskID,
	csm.ListCSMTrafficDirectorLogEntriesTaskID.Ref(),
	csm.LogTypeCSMTrafficLog,
)

// CSMTrafficDirectorLogGrouperTask is a task that groups CSM Traffic Director logs by their resource name.
var CSMTrafficDirectorLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	csm.CSMTrafficDirectorLogGrouperTaskID,
	csm.ListCSMTrafficDirectorLogEntriesTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		audit, err := gcpcommon.ExtractGCPAuditLog(l.NodeReader)
		if err != nil {
			return "unknown"
		}
		return audit.ResourceName
	},
)

// CSMTrafficDirectorLogToTimelineMapper maps CSM Traffic Director logs to resource timelines.
type CSMTrafficDirectorLogToTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*gcpcommon.GCPOperationTracker]
}

// LogIngesterTask returns a reference to the task that provides ingested logs.
func (m *CSMTrafficDirectorLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return csm.CSMTrafficDirectorLogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies.
func (m *CSMTrafficDirectorLogToTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		csm.ClusterIdentityTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *CSMTrafficDirectorLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return csm.CSMTrafficDirectorLogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps each log inside a group to one or more timeline events or revisions.
func (m *CSMTrafficDirectorLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *gcpcommon.GCPOperationTracker) (*khifilev6.TimelineChangeSet, *gcpcommon.GCPOperationTracker, error) {
	if tracker == nil {
		tracker = gcpcommon.NewGCPOperationTracker()
	}
	audit, err := gcpcommon.ExtractGCPAuditLog(l.NodeReader)
	if err != nil {
		return nil, tracker, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	verb := guessRevisionVerb(audit.MethodName)

	resourceType, resourceName := parseGCPResource(audit.ResourceName)
	projectTimeline := gcpcommon.MustGCPProjectTimeline(ctx, audit.ProjectID)
	typeTimeline := gcpcommon.MustGCPResourceTypeTimeline(ctx, projectTimeline, resourceType)
	resourceTimelinePath := gcpcommon.MustGCPResourceTimeline(ctx, typeTimeline, resourceName)

	if !audit.ImmediateOperation() {
		methodNameParts := strings.Split(audit.MethodName, ".")
		shortMethodName := methodNameParts[len(methodNameParts)-1]
		operationTimelinePath := gcpcommon.MustGCPOperationTimeline(ctx, resourceTimelinePath, shortMethodName, audit.OperationID)
		tracker.ProcessOperationLog(ctx, cs, operationTimelinePath, &audit, l.Timestamp)
	}

	manifestNode, shouldUpdate := tracker.TrackAndGetManifest(&audit)
	if shouldUpdate {
		switch {
		case verb == k8saudit.VerbDelete:
			cs.AddRevision(resourceTimelinePath, &khifilev6.StagingRevision{
				VerbType:    k8saudit.VerbDelete,
				StateType:   k8saudit.RevisionStateK8sResourceDeleted,
				Principal:   audit.PrincipalEmail,
				ChangedTime: l.Timestamp,
			})
		case audit.ImmediateOperation():
			cs.AddEvent(resourceTimelinePath)
		default:
			cs.AddRevision(resourceTimelinePath, &khifilev6.StagingRevision{
				ResourceBody: manifestNode,
				VerbType:     verb,
				StateType:    k8saudit.RevisionStateK8sResourceExisting,
				Principal:    audit.PrincipalEmail,
				ChangedTime:  l.Timestamp,
			})
		}
	}

	return cs, tracker, nil
}

// parseGCPResource parses a GCP resource name string into type and name.
func parseGCPResource(resourceName string) (string, string) {
	if resourceName == "" || resourceName == "unknown" {
		return "unknown", "unknown"
	}
	parts := strings.Split(resourceName, "/")

	var resourceType, name string
	if len(parts) >= 2 {
		resourceType = parts[len(parts)-2]
		name = parts[len(parts)-1]
	} else {
		resourceType = "unknown"
		name = resourceName
	}
	return resourceType, name
}

var _ inspectiontaskbase.LogToTimelineMapper[*gcpcommon.GCPOperationTracker] = (*CSMTrafficDirectorLogToTimelineMapper)(nil)

// CSMTrafficDirectorLogToTimelineMapperTask maps CSM Traffic Director logs to timelines.
var CSMTrafficDirectorLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	csm.CSMTrafficDirectorLogToTimelineMapperTaskID,
	&CSMTrafficDirectorLogToTimelineMapper{},
	inspectioncore.FeatureTaskLabel(
		"CSM Resource Audit Logs",
		"Gather audit logs for Traffic Director resources created by the TD-based CSM to map them to timelines alongside associated Kubernetes resource logs.",
		10100,
		false,
	),
)

// guessRevisionVerb guesses the styled verb type based on the method name.
func guessRevisionVerb(methodName string) *pb.Verb {
	methodNameSplitted := strings.Split(methodName, ".")
	shortMethodName := "unknown"
	if len(methodNameSplitted) > 0 {
		shortMethodName = methodNameSplitted[len(methodNameSplitted)-1]
	}
	shortMethodName = strings.ToLower(shortMethodName)

	switch {
	case strings.HasPrefix(shortMethodName, "create"), strings.HasPrefix(shortMethodName, "insert"):
		return k8saudit.VerbCreate
	case strings.HasPrefix(shortMethodName, "delete"):
		return k8saudit.VerbDelete
	case strings.HasPrefix(shortMethodName, "update"), strings.HasPrefix(shortMethodName, "patch"):
		return k8saudit.VerbUpdate
	default:
		return k8saudit.VerbUpdate
	}
}
