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

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudloggkeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReaderTask is a task that reads and parses field sets from GKE audit logs.
// It uses GCPOperationAuditLogFieldSetReader, GKEAuditLogResourceFieldSetReader,
// and GCPDefaultSeverityFieldSetReader to extract fields.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudloggkeapiaudit_contract.FieldSetReaderTaskID, googlecloudloggkeapiaudit_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
	&googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSetReader{},
	&googlecloudcommon_contract.GCPDefaultSeverityFieldSetReader{},
})

// LogIngesterTask is a task that serializes GKE audit logs for storage in the history builder.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(googlecloudloggkeapiaudit_contract.LogIngesterTaskID, &gkeAuditLogLogIngester{})

// gkeAuditLogLogIngester implements LogIngesterV2.
type gkeAuditLogLogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *gkeAuditLogLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudloggkeapiaudit_contract.FieldSetReaderTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *gkeAuditLogLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog parses raw log entry and manually populates the LogChangeSet.
func (i *gkeAuditLogLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(googlecloudloggkeapiaudit_contract.LogTypeGkeAudit)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severityFS.Severity)
	}

	if auditFieldSet, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{}); err == nil {
		switch {
		case auditFieldSet.Starting():
			cs.SetSummary(fmt.Sprintf("%s Started", auditFieldSet.MethodName))
		case auditFieldSet.Ending():
			cs.SetSummary(fmt.Sprintf("%s Finished", auditFieldSet.MethodName))
		default:
			cs.SetSummary(auditFieldSet.MethodName)
		}
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*gkeAuditLogLogIngester)(nil)

// LogGrouperTask is a task that groups GKE audit logs by GKE cluster or nodepool name.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudloggkeapiaudit_contract.LogGrouperTaskID, googlecloudloggkeapiaudit_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		resourceFieldSet, err := log.GetFieldSet(l, &googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{})
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
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2[struct{}](googlecloudloggkeapiaudit_contract.LogToTimelineMapperTaskID, &gkeAuditLogLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabelV2(`GKE Audit Logs`,
		`Gather GKE audit logs to visualize the creation, upgrade, and deletion of clusters and node pools on timelines.`,
		5000,
		true),
)

// gkeAuditLogLogToTimelineMapperSetting implements the LogToTimelineMapperV2 interface for GKE audit logs.
type gkeAuditLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.StatelessMapperBase
}

// Dependencies returns additional task references used in timeline mapper.
func (g *gkeAuditLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (g *gkeAuditLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudloggkeapiaudit_contract.LogGrouperTaskID.Ref()
}

// LogIngesterTask returns a reference to the log ingester task.
func (g *gkeAuditLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudloggkeapiaudit_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup maps GKE audit log resource updates to timeline events/revisions.
func (g *gkeAuditLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	commonFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	auditFieldSet, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	resourceFieldSet, err := log.GetFieldSet(l, &googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}

	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, auditFieldSet.ProjectID)
	clusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, projectTimeline, resourceFieldSet.ClusterName)

	var targetTimeline *khifilev6.TimelinePath
	if resourceFieldSet.IsCluster() {
		targetTimeline = clusterTimeline
	} else {
		targetTimeline = googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, resourceFieldSet.NodepoolName)
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	if !auditFieldSet.ImmediateOperation() {
		resourceBodyField := ""
		if resourceFieldSet.IsCluster() {
			resourceBodyField = "cluster"
		} else {
			resourceBodyField = "nodePool"
		}

		methodNameParts := strings.Split(auditFieldSet.MethodName, ".")
		shortMethodName := methodNameParts[len(methodNameParts)-1]

		switch shortMethodName {
		case "CreateCluster", "CreateNodePool":
			var bodyNode structured.Node
			state := googlecloudloggkeapiaudit_contract.RevisionStateProvisioning
			if auditFieldSet.Ending() {
				state = commonlogk8saudit_contract.RevisionStateK8sResourceExisting
			}
			if auditFieldSet.Request != nil {
				if subReader, err := auditFieldSet.Request.GetReader(resourceBodyField); err == nil {
					bodyNode = subReader.Node
				}
			}
			cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbCreate,
				StateType:    state,
				Principal:    auditFieldSet.PrincipalEmail,
				ChangedTime:  commonFieldSet.Timestamp,
				ResourceBody: bodyNode,
			})
		case "DeleteCluster", "DeleteNodePool":
			state := commonlogk8saudit_contract.RevisionStateK8sResourceDeleting
			if auditFieldSet.Ending() {
				state = commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted
			}
			cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbDelete,
				StateType:    state,
				Principal:    auditFieldSet.PrincipalEmail,
				ChangedTime:  commonFieldSet.Timestamp,
				ResourceBody: nil,
			})
		}

		state := googlecloudcommon_contract.RevisionStateOperationStarted
		verb := googlecloudcommon_contract.VerbOperationStart
		if auditFieldSet.Ending() {
			state = googlecloudcommon_contract.RevisionStateOperationFinished
			verb = googlecloudcommon_contract.VerbOperationFinish
		}
		var bodyNode structured.Node
		if auditFieldSet.Request != nil {
			bodyNode = auditFieldSet.Request.Node
		}

		operationTimeline := googlecloudcommon_contract.MustGCPOperationTimeline(ctx, targetTimeline, shortMethodName, auditFieldSet.OperationID)
		cs.AddRevision(operationTimeline, &khifilev6.StagingRevision{
			VerbType:     verb,
			StateType:    state,
			Principal:    auditFieldSet.PrincipalEmail,
			ChangedTime:  commonFieldSet.Timestamp,
			ResourceBody: bodyNode,
		})
	} else {
		cs.AddEvent(targetTimeline)
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*gkeAuditLogLogToTimelineMapperSetting)(nil)
