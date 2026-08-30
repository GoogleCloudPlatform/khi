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

package googlecloudlogcomposerapiaudit_impl

import (
	"context"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcomposerapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomposerapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// LogIngesterTask ingests Cloud Composer audit logs into KHI v6 format.
var LogIngesterTask = googlecloudcommon_contract.NewGCPOperationLogIngesterTask(
	googlecloudlogcomposerapiaudit_contract.LogIngesterTaskID,
	googlecloudlogcomposerapiaudit_contract.ListLogEntriesTaskID.Ref(),
	googlecloudlogcomposerapiaudit_contract.LogTypeManagedAirflowAPI,
)

// LogGrouperTask groups Cloud Composer audit logs by environment name.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogcomposerapiaudit_contract.LogGrouperTaskID,
	googlecloudlogcomposerapiaudit_contract.ListLogEntriesTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		resourceFieldSet, err := googlecloudlogcomposerapiaudit_contract.ExtractComposerAuditLogResource(l.NodeReader)
		if err != nil {
			return "unknown"
		}
		return resourceFieldSet.EnvironmentName
	},
)

// LogToTimelineMapperTask maps Cloud Composer audit logs to timeline events and operation revisions.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[*googlecloudcommon_contract.GCPOperationTracker](
	googlecloudlogcomposerapiaudit_contract.LogToTimelineMapperTaskID,
	&composerAuditLogLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabel(
		"Managed Airflow API Logs",
		"Gather Managed Airflow API audit logs to visualize environment operations (creation, update, and deletion) on timelines.",
		5500,
		true,
	),
)

type composerAuditLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.SinglePassMapperBase[*googlecloudcommon_contract.GCPOperationTracker]
}

// Dependencies returns additional task dependencies.
func (s *composerAuditLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask returns a reference to the log grouper task.
func (s *composerAuditLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogcomposerapiaudit_contract.LogGrouperTaskID.Ref()
}

// LogIngesterTask returns a reference to the log ingester task.
func (s *composerAuditLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcomposerapiaudit_contract.LogIngesterTaskID.Ref()
}

var pathEnvironment = structured.CompileFieldPath("environment")

// ProcessLogByGroup maps a single Composer audit log entry to timeline events and revisions.
func (s *composerAuditLogLogToTimelineMapperSetting) ProcessLogByGroup(
	ctx context.Context,
	l *log.Log,
	tracker *googlecloudcommon_contract.GCPOperationTracker,
) (*khifilev6.TimelineChangeSet, *googlecloudcommon_contract.GCPOperationTracker, error) {
	if tracker == nil {
		tracker = googlecloudcommon_contract.NewGCPOperationTracker()
	}
	auditFieldSet, err := googlecloudcommon_contract.ExtractGCPAuditLog(l.NodeReader)
	if err != nil {
		return nil, tracker, err
	}
	resourceFieldSet, err := googlecloudlogcomposerapiaudit_contract.ExtractComposerAuditLogResource(l.NodeReader)
	if err != nil {
		return nil, tracker, err
	}

	projectID := auditFieldSet.ProjectID
	if projectID == "" || projectID == "unknown" {
		projectID = resourceFieldSet.ProjectID
	}

	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, projectID)
	envTimeline := googlecloudcommon_contract.MustManagedAirflowEnvironmentTimeline(ctx, projectTimeline, resourceFieldSet.EnvironmentName)

	cs := khifilev6.NewTimelineChangeSet(l)

	methodNameParts := strings.Split(auditFieldSet.MethodName, ".")
	shortMethodName := methodNameParts[len(methodNameParts)-1]

	if auditFieldSet.ImmediateOperation() {
		cs.AddEvent(envTimeline)
		return cs, tracker, nil
	}

	operationTimeline := googlecloudcommon_contract.MustGCPOperationTimeline(ctx, envTimeline, shortMethodName, auditFieldSet.OperationID)

	switch shortMethodName {
	case "CreateEnvironment":
		var bodyNode structured.Node
		if auditFieldSet.Request != nil {
			if subReader, err := auditFieldSet.Request.GetReader(pathEnvironment); err == nil {
				bodyNode = subReader.Node
			} else {
				bodyNode = auditFieldSet.Request.Node
			}
		}

		if auditFieldSet.Ending() && !tracker.HasStarted(auditFieldSet.OperationID) {
			cs.AddRevision(envTimeline, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbCreate,
				StateType:    googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentProvisioningLogNotFound,
				Principal:    auditFieldSet.PrincipalEmail,
				ChangedTime:  time.Unix(0, 0),
				ResourceBody: nil,
			})
			tracker.MarkResourceRevision(envTimeline)
		}

		var state *pb.RevisionState
		if auditFieldSet.Ending() {
			state = googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentExisting
		} else {
			state = googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentProvisioning
		}

		cs.AddRevision(envTimeline, &khifilev6.StagingRevision{
			VerbType:     commonlogk8saudit_contract.VerbCreate,
			StateType:    state,
			Principal:    auditFieldSet.PrincipalEmail,
			ChangedTime:  l.Timestamp,
			ResourceBody: bodyNode,
		})
		tracker.MarkResourceRevision(envTimeline)

	case "DeleteEnvironment":
		if !auditFieldSet.Ending() && !tracker.HasResourceRevision(envTimeline) {
			cs.AddRevision(envTimeline, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbCreate,
				StateType:    googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentExistingLogNotFound,
				Principal:    auditFieldSet.PrincipalEmail,
				ChangedTime:  time.Unix(0, 0),
				ResourceBody: nil,
			})
			tracker.MarkResourceRevision(envTimeline)
		}

		if auditFieldSet.Ending() && !tracker.HasStarted(auditFieldSet.OperationID) {
			cs.AddRevision(envTimeline, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbDelete,
				StateType:    googlecloudclustercomposer_contract.RevisionManagedAirflowEnvironmentDeletingLogNotFound,
				Principal:    auditFieldSet.PrincipalEmail,
				ChangedTime:  time.Unix(0, 0),
				ResourceBody: nil,
			})
			tracker.MarkResourceRevision(envTimeline)
		}

		var state *pb.RevisionState
		if auditFieldSet.Ending() {
			state = googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentDeleted
		} else {
			state = googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentDeleting
		}

		cs.AddRevision(envTimeline, &khifilev6.StagingRevision{
			VerbType:     commonlogk8saudit_contract.VerbDelete,
			StateType:    state,
			Principal:    auditFieldSet.PrincipalEmail,
			ChangedTime:  l.Timestamp,
			ResourceBody: nil,
		})
		tracker.MarkResourceRevision(envTimeline)
	}

	tracker.ProcessOperationLog(ctx, cs, operationTimeline, &auditFieldSet, l.Timestamp)

	return cs, tracker, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[*googlecloudcommon_contract.GCPOperationTracker] = (*composerAuditLogLogToTimelineMapperSetting)(nil)
