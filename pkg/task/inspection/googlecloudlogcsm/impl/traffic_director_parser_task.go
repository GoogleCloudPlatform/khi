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

package googlecloudlogcsm_impl

import (
	"context"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// CSMTrafficDirectorFieldSetReaderTask is a task that reads and parses field sets from CSM Traffic Director logs.
var CSMTrafficDirectorFieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudlogcsm_contract.CSMTrafficDirectorFieldSetReaderTaskID,
	googlecloudlogcsm_contract.ListCSMTrafficDirectorLogEntriesTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
		&googlecloudcommon_contract.GCPDefaultSeverityFieldSetReader{},
	},
)

// CSMTrafficDirectorLogIngesterTask is a task that ingests CSM Traffic Director logs.
var CSMTrafficDirectorLogIngesterTask = googlecloudcommon_contract.NewGCPOperationLogIngesterTask(
	googlecloudlogcsm_contract.CSMTrafficDirectorLogIngesterTaskID,
	googlecloudlogcsm_contract.CSMTrafficDirectorFieldSetReaderTaskID.Ref(),
	googlecloudlogcsm_contract.LogTypeCSMAccessLog,
)

// CSMTrafficDirectorLogGrouperTask is a task that groups CSM Traffic Director logs by their resource name.
var CSMTrafficDirectorLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogcsm_contract.CSMTrafficDirectorLogGrouperTaskID,
	googlecloudlogcsm_contract.CSMTrafficDirectorFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
		if err != nil {
			return "unknown"
		}
		return audit.ResourceName
	},
)

// CSMTrafficDirectorLogToTimelineMapper maps CSM Traffic Director logs to resource timelines.
type CSMTrafficDirectorLogToTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*googlecloudcommon_contract.GCPOperationTracker]
}

// LogIngesterTask returns a reference to the task that provides ingested logs.
func (m *CSMTrafficDirectorLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcsm_contract.CSMTrafficDirectorLogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies.
func (m *CSMTrafficDirectorLogToTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogcsm_contract.ClusterIdentityTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *CSMTrafficDirectorLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogcsm_contract.CSMTrafficDirectorLogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps each log inside a group to one or more timeline events or revisions.
func (m *CSMTrafficDirectorLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *googlecloudcommon_contract.GCPOperationTracker) (*khifilev6.TimelineChangeSet, *googlecloudcommon_contract.GCPOperationTracker, error) {
	if tracker == nil {
		tracker = googlecloudcommon_contract.NewGCPOperationTracker()
	}
	commonLogFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, tracker, err
	}
	audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return nil, tracker, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	verb := guessRevisionVerb(audit.MethodName)

	resourceType, resourceName := parseGCPResource(audit.ResourceName)
	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, audit.ProjectID)
	typeTimeline := googlecloudcommon_contract.MustGCPResourceTypeTimeline(ctx, projectTimeline, resourceType)
	resourceTimelinePath := googlecloudcommon_contract.MustGCPResourceTimeline(ctx, typeTimeline, resourceName)

	if !audit.ImmediateOperation() {
		methodNameParts := strings.Split(audit.MethodName, ".")
		shortMethodName := methodNameParts[len(methodNameParts)-1]
		operationTimelinePath := googlecloudcommon_contract.MustGCPOperationTimeline(ctx, resourceTimelinePath, shortMethodName, audit.OperationID)
		tracker.ProcessOperationLog(ctx, cs, operationTimelinePath, audit, commonLogFieldSet.Timestamp)
	}

	manifest, shouldUpdate := tracker.TrackAndGetManifest(audit)
	if shouldUpdate {
		switch {
		case verb == commonlogk8saudit_contract.VerbDelete:
			cs.AddRevision(resourceTimelinePath, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbDelete,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sResourceDeleted,
				Principal:   audit.PrincipalEmail,
				ChangedTime: commonLogFieldSet.Timestamp,
			})
		case audit.ImmediateOperation():
			cs.AddEvent(resourceTimelinePath)
		default:
			cs.AddRevision(resourceTimelinePath, &khifilev6.StagingRevision{
				ResourceBody: structured.NewStandardScalarNode(manifest),
				VerbType:     verb,
				StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExisting,
				Principal:    audit.PrincipalEmail,
				ChangedTime:  commonLogFieldSet.Timestamp,
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

var _ inspectiontaskbase.LogToTimelineMapper[*googlecloudcommon_contract.GCPOperationTracker] = (*CSMTrafficDirectorLogToTimelineMapper)(nil)

// CSMTrafficDirectorLogToTimelineMapperTask maps CSM Traffic Director logs to timelines.
var CSMTrafficDirectorLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	googlecloudlogcsm_contract.CSMTrafficDirectorLogToTimelineMapperTaskID,
	&CSMTrafficDirectorLogToTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabel(
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
		return commonlogk8saudit_contract.VerbCreate
	case strings.HasPrefix(shortMethodName, "delete"):
		return commonlogk8saudit_contract.VerbDelete
	case strings.HasPrefix(shortMethodName, "update"), strings.HasPrefix(shortMethodName, "patch"):
		return commonlogk8saudit_contract.VerbUpdate
	default:
		return commonlogk8saudit_contract.VerbUpdate
	}
}
