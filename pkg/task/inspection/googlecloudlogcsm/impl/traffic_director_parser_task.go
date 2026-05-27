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
	"fmt"
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

// CSMTrafficDirectorLogIngester V2 LogIngester.
type CSMTrafficDirectorLogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *CSMTrafficDirectorLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcsm_contract.CSMTrafficDirectorFieldSetReaderTaskID.Ref()
}

// Dependencies returns the task dependencies.
func (i *CSMTrafficDirectorLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog parses raw log entry and populates the LogChangeSet.
func (i *CSMTrafficDirectorLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, err
	}
	cs.SetTimestamp(commonFS.Timestamp)

	audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return nil, err
	}

	methodNameParts := strings.Split(audit.MethodName, ".")
	shortMethodName := methodNameParts[len(methodNameParts)-1]

	var summary string
	switch {
	case audit.Starting():
		summary = fmt.Sprintf("%s Started", shortMethodName)
	case audit.Ending():
		summary = fmt.Sprintf("%s Finished", shortMethodName)
	default:
		summary = shortMethodName
	}
	cs.SetSummary(summary)
	cs.SetLogType(googlecloudlogcsm_contract.LogTypeCSMAccessLog)

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severityFS.Severity)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*CSMTrafficDirectorLogIngester)(nil)

// CSMTrafficDirectorLogIngesterTask is a task that ingests CSM Traffic Director logs.
var CSMTrafficDirectorLogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
	googlecloudlogcsm_contract.CSMTrafficDirectorLogIngesterTaskID,
	&CSMTrafficDirectorLogIngester{},
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
	inspectiontaskbase.SinglePassMapperBase[*googlecloudcommon_contract.GCPOperationStateTracker]
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
func (m *CSMTrafficDirectorLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *googlecloudcommon_contract.GCPOperationStateTracker) (*khifilev6.TimelineChangeSet, *googlecloudcommon_contract.GCPOperationStateTracker, error) {
	if tracker == nil {
		tracker = googlecloudcommon_contract.NewGCPOperationStateTracker()
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

	projectID, resourceType, resourceName := parseGCPResource(audit.ResourceName)
	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, projectID)
	typeTimeline := googlecloudcommon_contract.MustGCPResourceTypeTimeline(ctx, projectTimeline, resourceType)
	resourceTimelinePath := googlecloudcommon_contract.MustGCPResourceTimeline(ctx, typeTimeline, resourceName)

	if !audit.ImmediateOperation() {
		state := googlecloudcommon_contract.RevisionStateOperationStarted
		opVerb := googlecloudcommon_contract.VerbOperationStart
		if audit.Ending() {
			state = googlecloudcommon_contract.RevisionStateOperationFinished
			opVerb = googlecloudcommon_contract.VerbOperationFinish
		}
		requestBody, _ := audit.RequestString()
		methodNameParts := strings.Split(audit.MethodName, ".")
		shortMethodName := methodNameParts[len(methodNameParts)-1]
		operationTimelinePath := googlecloudcommon_contract.MustGCPOperationTimeline(ctx, resourceTimelinePath, shortMethodName, audit.OperationID)
		cs.AddRevision(operationTimelinePath, &khifilev6.StagingRevision{
			ResourceBody: structured.NewStandardScalarNode(requestBody),
			VerbType:     opVerb,
			StateType:    state,
			Principal:    audit.PrincipalEmail,
			ChangedTime:  commonLogFieldSet.Timestamp,
		})
	}

	manifest, shouldUpdate := tracker.TrackAndGetManifest(audit)
	if shouldUpdate {
		switch {
		case verb == commonlogk8saudit_contract.VerbDelete:
			cs.AddRevision(resourceTimelinePath, &khifilev6.StagingRevision{
				VerbType:    commonlogk8saudit_contract.VerbDelete,
				StateType:   commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
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

// parseGCPResource parses a GCP resource name string into project, type, and name.
func parseGCPResource(resourceName string) (string, string, string) {
	if resourceName == "" || resourceName == "unknown" {
		return "unknown", "unknown", "unknown"
	}
	parts := strings.Split(resourceName, "/")
	project := "unknown"
	projectIdx := -1
	for i, p := range parts {
		if p == "projects" {
			projectIdx = i
			break
		}
	}
	if projectIdx != -1 && projectIdx+1 < len(parts) {
		project = parts[projectIdx+1]
	}

	var resourceType, name string
	if len(parts) >= 2 {
		resourceType = parts[len(parts)-2]
		name = parts[len(parts)-1]
	} else {
		resourceType = "unknown"
		name = resourceName
	}
	return project, resourceType, name
}

var _ inspectiontaskbase.LogToTimelineMapperV2[*googlecloudcommon_contract.GCPOperationStateTracker] = (*CSMTrafficDirectorLogToTimelineMapper)(nil)

// CSMTrafficDirectorLogToTimelineMapperTask maps CSM Traffic Director logs to timelines.
var CSMTrafficDirectorLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
	googlecloudlogcsm_contract.CSMTrafficDirectorLogToTimelineMapperTaskID,
	&CSMTrafficDirectorLogToTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabelV2(
		"CSM Resource Audit Logs",
		"Gather audit logs related to TrafficDirector resources created by the TD based CSM. Map them into pseudo timelines with other Kubernetes logs.",
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
