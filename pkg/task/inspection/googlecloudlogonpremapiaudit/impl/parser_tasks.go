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

package googlecloudlogonpremapiaudit_impl

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
	googlecloudlogonpremapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogonpremapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReaderTask is a task that reads and parses field sets from MulticloudAPI audit logs.
// It uses GCPOperationAuditLogFieldSetReader, OnPremAPIAuditResourceFieldSetReader, and GCPDefaultSeverityFieldSetReader
// to extract common GCP audit log fields, multicloud api-specific resource fields, and severity.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudlogonpremapiaudit_contract.FieldSetReaderTaskID,
	googlecloudlogonpremapiaudit_contract.ListLogEntriesTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
		&googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSetReader{},
		&googlecloudcommon_contract.GCPDefaultSeverityFieldSetReader{},
	},
)

// LogIngesterTask is a task that serializes MulticloudAPI audit logs for storage in the history builder.
var LogIngesterTask = googlecloudcommon_contract.NewGCPOperationLogIngesterTask(
	googlecloudlogonpremapiaudit_contract.LogIngesterTaskID,
	googlecloudlogonpremapiaudit_contract.FieldSetReaderTaskID.Ref(),
	googlecloudlogonpremapiaudit_contract.LogTypeOnPremAPI,
)

// LogGrouperTask is a task that groups MulticloudAPI audit logs by their resource path.
// This grouping allows for parallel processing of logs related to the same resource.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogonpremapiaudit_contract.LogGrouperTaskID,
	googlecloudlogonpremapiaudit_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		resourceFieldSet, err := log.GetFieldSet(l, &googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{})
		if err != nil {
			return ""
		}
		return fmt.Sprintf("%s/%s/%s", resourceFieldSet.Project, resourceFieldSet.ClusterName, resourceFieldSet.NodepoolName)
	},
)

// OnPremAPIAuditTimelineMapper maps On-Prem API audit logs to timeline elements.
type OnPremAPIAuditTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*googlecloudcommon_contract.GCPOperationTracker]
}

// LogIngesterTask returns the task reference providing the ingested logs.
func (m *OnPremAPIAuditTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogonpremapiaudit_contract.LogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies of the mapper.
func (m *OnPremAPIAuditTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *OnPremAPIAuditTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogonpremapiaudit_contract.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps log entries to timeline elements.
func (m *OnPremAPIAuditTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *googlecloudcommon_contract.GCPOperationTracker) (*khifilev6.TimelineChangeSet, *googlecloudcommon_contract.GCPOperationTracker, error) {
	if tracker == nil {
		tracker = googlecloudcommon_contract.NewGCPOperationTracker()
	}
	commonFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, tracker, err
	}
	auditFieldSet, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return nil, tracker, err
	}
	resourceFieldSet, err := log.GetFieldSet(l, &googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{})
	if err != nil {
		return nil, tracker, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	projectPath := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, resourceFieldSet.Project)
	clusterPath := googlecloudlogonpremapiaudit_contract.MustOnPremClusterTimeline(ctx, projectPath, resourceFieldSet.ClusterName)

	var targetPath *khifilev6.TimelinePath
	if resourceFieldSet.IsCluster() {
		targetPath = clusterPath
	} else {
		targetPath = googlecloudlogonpremapiaudit_contract.MustOnPremNodePoolTimeline(ctx, clusterPath, resourceFieldSet.NodepoolName)
	}

	if !auditFieldSet.ImmediateOperation() {
		resourceBodyField := ""
		if resourceFieldSet.IsCluster() {
			resourceBodyField = "cluster"
		} else {
			resourceBodyField = "nodePool"
		}

		clusterTypeToFragmentInMethodNameMapping := map[googlecloudlogonpremapiaudit_contract.OnPremClusterType]string{
			googlecloudlogonpremapiaudit_contract.ClusterTypeBaremetalAdmin:      "BaremetalAdmin",
			googlecloudlogonpremapiaudit_contract.ClusterTypeBaremetalStandalone: "BaremetalStandalone",
			googlecloudlogonpremapiaudit_contract.ClusterTypeBaremetalUser:       "Baremetal",
			googlecloudlogonpremapiaudit_contract.ClusterTypeVMWareAdmin:         "VmwareAdmin",
			googlecloudlogonpremapiaudit_contract.ClusterTypeVMWareUser:          "Vmware",
		}

		methodNameParts := strings.Split(auditFieldSet.MethodName, ".")
		shortMethodName := methodNameParts[len(methodNameParts)-1]
		shortMethodName = strings.ReplaceAll(shortMethodName, clusterTypeToFragmentInMethodNameMapping[resourceFieldSet.ClusterType], "")

		switch shortMethodName {
		case "CreateCluster", "CreateNodePool", "EnrollCluster", "EnrollNodePool":
			var bodyNode structured.Node
			state := commonlogk8saudit_contract.RevisionStateK8sClusterProvisioning
			if auditFieldSet.Ending() {
				state = commonlogk8saudit_contract.RevisionStateK8sClusterExisting
			}
			if auditFieldSet.Request != nil {
				if reqReader, err := auditFieldSet.Request.GetReader(resourceBodyField); err == nil {
					bodyNode = reqReader.Node
				}
			}
			cs.AddRevision(targetPath, &khifilev6.StagingRevision{
				ChangedTime:  commonFieldSet.Timestamp,
				ResourceBody: bodyNode,
				Principal:    auditFieldSet.PrincipalEmail,
				VerbType:     commonlogk8saudit_contract.VerbCreate,
				StateType:    state,
			})
		case "DeleteCluster", "DeleteNodePool", "UnenrollCluster", "UnenrollNodePool":
			state := commonlogk8saudit_contract.RevisionStateK8sClusterDeleting
			if auditFieldSet.Ending() {
				state = commonlogk8saudit_contract.RevisionStateK8sClusterDeleted
			}
			cs.AddRevision(targetPath, &khifilev6.StagingRevision{
				ChangedTime:  commonFieldSet.Timestamp,
				ResourceBody: nil,
				Principal:    auditFieldSet.PrincipalEmail,
				VerbType:     commonlogk8saudit_contract.VerbDelete,
				StateType:    state,
			})
		}

		methodNameSplitted := strings.Split(auditFieldSet.MethodName, ".")
		originalShortMethodName := "unknown"
		if len(methodNameSplitted) > 0 {
			originalShortMethodName = methodNameSplitted[len(methodNameSplitted)-1]
		}
		operationPath := googlecloudcommon_contract.MustGCPOperationTimeline(ctx, targetPath, originalShortMethodName, auditFieldSet.OperationID)

		tracker.ProcessOperationLog(ctx, cs, operationPath, auditFieldSet, commonFieldSet.Timestamp)
	} else {
		cs.AddEvent(targetPath)
	}

	return cs, tracker, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[*googlecloudcommon_contract.GCPOperationTracker] = (*OnPremAPIAuditTimelineMapper)(nil)

// LogToTimelineMapperTask is a task that adds revisions/events regarding logs.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
	googlecloudlogonpremapiaudit_contract.LogToTimelineMapperTaskID,
	&OnPremAPIAuditTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabelV2(
		"On-Premises API Logs",
		"Gather Anthos On-Premises audit logs to visualize cluster lifecycle events (creation, deletion, enrollment, unenrollment, and upgrades) on timelines.",
		9500,
		true,
	),
)
