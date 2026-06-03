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

// OnPremAPIAuditLogIngester ingests log metadata for OnPrem API audit logs.
type OnPremAPIAuditLogIngester struct{}

// RawLogTask returns the task reference providing the raw logs.
func (i *OnPremAPIAuditLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogonpremapiaudit_contract.FieldSetReaderTaskID.Ref()
}

// Dependencies returns the task dependencies of the ingester.
func (i *OnPremAPIAuditLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog parses raw log entry and manually populates the LogChangeSet.
func (i *OnPremAPIAuditLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(googlecloudlogonpremapiaudit_contract.LogTypeOnPremAPI)

	if commonSet, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonSet.Timestamp)
	}

	if severitySet, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severitySet.Severity)
	}

	auditFieldSet, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return nil, err
	}

	switch {
	case auditFieldSet.Starting():
		cs.SetSummary(fmt.Sprintf("%s Started", auditFieldSet.MethodName))
	case auditFieldSet.Ending():
		cs.SetSummary(fmt.Sprintf("%s Finished", auditFieldSet.MethodName))
	default:
		cs.SetSummary(auditFieldSet.MethodName)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*OnPremAPIAuditLogIngester)(nil)

// LogIngesterTask is a task that serializes MulticloudAPI audit logs for storage in the history builder.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
	googlecloudlogonpremapiaudit_contract.LogIngesterTaskID,
	&OnPremAPIAuditLogIngester{},
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
	inspectiontaskbase.StatelessMapperBase
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
func (m *OnPremAPIAuditTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	commonFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	auditFieldSet, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	resourceFieldSet, err := log.GetFieldSet(l, &googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
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
			state := googlecloudlogonpremapiaudit_contract.RevisionStateProvisioning
			if auditFieldSet.Ending() {
				state = googlecloudlogonpremapiaudit_contract.RevisionStateExisting
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
			state := googlecloudlogonpremapiaudit_contract.RevisionStateDeleting
			if auditFieldSet.Ending() {
				state = googlecloudlogonpremapiaudit_contract.RevisionStateDeleted
			}
			cs.AddRevision(targetPath, &khifilev6.StagingRevision{
				ChangedTime:  commonFieldSet.Timestamp,
				ResourceBody: nil,
				Principal:    auditFieldSet.PrincipalEmail,
				VerbType:     commonlogk8saudit_contract.VerbDelete,
				StateType:    state,
			})
		}

		state := googlecloudlogonpremapiaudit_contract.RevisionStateOperationStarted
		verb := googlecloudcommon_contract.VerbOperationStart
		if auditFieldSet.Ending() {
			state = googlecloudlogonpremapiaudit_contract.RevisionStateOperationFinished
			verb = googlecloudcommon_contract.VerbOperationFinish
		}

		var requestBodyNode structured.Node
		if auditFieldSet.Request != nil {
			requestBodyNode = auditFieldSet.Request.Node
		}

		methodNameSplitted := strings.Split(auditFieldSet.MethodName, ".")
		originalShortMethodName := "unknown"
		if len(methodNameSplitted) > 0 {
			originalShortMethodName = methodNameSplitted[len(methodNameSplitted)-1]
		}
		operationPath := googlecloudcommon_contract.MustGCPOperationTimeline(ctx, targetPath, originalShortMethodName, auditFieldSet.OperationID)

		cs.AddRevision(operationPath, &khifilev6.StagingRevision{
			ChangedTime:  commonFieldSet.Timestamp,
			ResourceBody: requestBodyNode,
			Principal:    auditFieldSet.PrincipalEmail,
			VerbType:     verb,
			StateType:    state,
		})
	} else {
		cs.AddEvent(targetPath)
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*OnPremAPIAuditTimelineMapper)(nil)

// LogToTimelineMapperTask is a task that adds revisions/events regarding logs.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
	googlecloudlogonpremapiaudit_contract.LogToTimelineMapperTaskID,
	&OnPremAPIAuditTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabelV2(
		"OnPrem API logs",
		"Gather Anthos OnPrem audit log including cluster creation,deletion,enroll,unenroll and upgrades.",
		9500,
		true,
	),
)
