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

package googlecloudlogmulticloudapiaudit_impl

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
	googlecloudlogmulticloudapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogmulticloudapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReaderTask is a task that reads and parses field sets from MulticloudAPI audit logs.
// It uses GCPOperationAuditLogFieldSetReader, MulticloudAPIAuditResourceFieldSetReader, and
// GCPDefaultSeverityFieldSetReader to extract common GCP audit log fields, severity, and resource fields.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudlogmulticloudapiaudit_contract.FieldSetReaderTaskID,
	googlecloudlogmulticloudapiaudit_contract.ListLogEntriesTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
		&googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSetReader{},
		&googlecloudcommon_contract.GCPDefaultSeverityFieldSetReader{},
	},
)

// multicloudAuditLogIngester is a V2 log ingester that parses log metadata.
type multicloudAuditLogIngester struct{}

// RawLogTask returns the task reference that provides raw logs.
func (i *multicloudAuditLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogmulticloudapiaudit_contract.FieldSetReaderTaskID.Ref()
}

// Dependencies returns additional task dependencies.
func (i *multicloudAuditLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog customizes the log metadata and returns a LogChangeSet.
func (i *multicloudAuditLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(googlecloudlogmulticloudapiaudit_contract.LogTypeMulticloudAPI)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severityFS.Severity)
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

var _ inspectiontaskbase.LogIngesterV2 = (*multicloudAuditLogIngester)(nil)

// LogIngesterTask is a task that serializes MulticloudAPI audit logs for storage in the history builder.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
	googlecloudlogmulticloudapiaudit_contract.LogIngesterTaskID,
	&multicloudAuditLogIngester{},
)

// LogGrouperTask is a task that groups MulticloudAPI audit logs by resource identifier.
// This grouping allows for parallel processing of logs related to the same resource.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogmulticloudapiaudit_contract.LogGrouperTaskID,
	googlecloudlogmulticloudapiaudit_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		resourceFieldSet, err := log.GetFieldSet(l, &googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{})
		if err != nil {
			return ""
		}
		if resourceFieldSet.IsCluster() {
			return fmt.Sprintf("cluster/%s/%s", resourceFieldSet.ClusterType, resourceFieldSet.ClusterName)
		}
		return fmt.Sprintf("nodepool/%s/%s/%s", resourceFieldSet.ClusterType, resourceFieldSet.ClusterName, resourceFieldSet.NodepoolName)
	},
)

type multicloudAuditLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.StatelessMapperBase
}

// Dependencies implements LogToTimelineMapperV2.
func (m *multicloudAuditLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements LogToTimelineMapperV2.
func (m *multicloudAuditLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogmulticloudapiaudit_contract.LogGrouperTaskID.Ref()
}

// LogIngesterTask implements LogToTimelineMapperV2.
func (m *multicloudAuditLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogmulticloudapiaudit_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup maps grouped logs to resource timelines and operations in KHI V6 format.
func (m *multicloudAuditLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	commonFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	auditFieldSet, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	resourceFieldSet, err := log.GetFieldSet(l, &googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}

	// Extract the project ID/number from the resourceName.
	// Format: projects/<PROJECT_NUMBER>/locations/<LOCATION>/...
	var projectID string
	resourceName, _ := l.ReadString("protoPayload.resourceName")
	splited := strings.Split(resourceName, "/")
	if len(splited) > 1 && splited[0] == "projects" {
		projectID = splited[1]
	} else {
		projectID = "unknown"
	}

	projectPath := googlecloudlogmulticloudapiaudit_contract.MustProjectTimeline(ctx, projectID)
	clusterPath := googlecloudlogmulticloudapiaudit_contract.MustMultiCloudClusterTimeline(ctx, projectPath, resourceFieldSet.ClusterName)

	var targetPath *khifilev6.TimelinePath
	if resourceFieldSet.IsCluster() {
		targetPath = clusterPath
	} else {
		targetPath = googlecloudlogmulticloudapiaudit_contract.MustMultiCloudNodepoolTimeline(ctx, clusterPath, resourceFieldSet.NodepoolName)
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	if !auditFieldSet.ImmediateOperation() {
		resourceBodyField := ""

		if resourceFieldSet.IsCluster() {
			resourceBodyField = "cluster"
		} else {
			resourceBodyField = "nodePool"
		}

		clusterTypeToFragmentInMethodNameMapping := map[googlecloudlogmulticloudapiaudit_contract.MultiCloudClusterType]string{
			googlecloudlogmulticloudapiaudit_contract.ClusterTypeAWS:   "Aws",
			googlecloudlogmulticloudapiaudit_contract.ClusterTypeAzure: "Azure",
		}

		methodNameParts := strings.Split(auditFieldSet.MethodName, ".")
		shortMethodName := methodNameParts[len(methodNameParts)-1]
		shortMethodName = strings.ReplaceAll(shortMethodName, clusterTypeToFragmentInMethodNameMapping[resourceFieldSet.ClusterType], "") // Remove type specific part.

		switch shortMethodName {
		case "CreateCluster", "CreateNodePool":
			var bodyNode structured.Node
			state := googlecloudlogmulticloudapiaudit_contract.RevisionStateProvisioning
			if auditFieldSet.Ending() {
				state = commonlogk8saudit_contract.RevisionStateK8sResourceExisting
			}
			if auditFieldSet.Request != nil {
				if r, err := auditFieldSet.Request.GetReader(resourceBodyField); err == nil {
					bodyNode = r.Node
				}
			}
			cs.AddRevision(targetPath, &khifilev6.StagingRevision{
				ChangedTime:  commonFieldSet.Timestamp,
				ResourceBody: bodyNode,
				Principal:    auditFieldSet.PrincipalEmail,
				VerbType:     commonlogk8saudit_contract.VerbCreate,
				StateType:    state,
			})
		case "DeleteCluster", "DeleteNodePool":
			state := commonlogk8saudit_contract.RevisionStateK8sResourceDeleting
			if auditFieldSet.Ending() {
				state = commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted
			}
			cs.AddRevision(targetPath, &khifilev6.StagingRevision{
				ChangedTime:  commonFieldSet.Timestamp,
				ResourceBody: nil,
				Principal:    auditFieldSet.PrincipalEmail,
				VerbType:     commonlogk8saudit_contract.VerbDelete,
				StateType:    state,
			})
		}

		state := googlecloudcommon_contract.RevisionStateOperationStarted
		verb := googlecloudcommon_contract.VerbOperationStart
		if auditFieldSet.Ending() {
			state = googlecloudcommon_contract.RevisionStateOperationFinished
			verb = googlecloudcommon_contract.VerbOperationFinish
		}

		var opBody structured.Node
		if auditFieldSet.Request != nil {
			opBody = auditFieldSet.Request.Node
		}

		opPath := googlecloudlogmulticloudapiaudit_contract.MustOperationTimeline(ctx, targetPath, shortMethodName, auditFieldSet.OperationID)
		cs.AddRevision(opPath, &khifilev6.StagingRevision{
			ChangedTime:  commonFieldSet.Timestamp,
			ResourceBody: opBody,
			Principal:    auditFieldSet.PrincipalEmail,
			VerbType:     verb,
			StateType:    state,
		})
	} else {
		cs.AddEvent(targetPath)
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*multicloudAuditLogLogToTimelineMapperSetting)(nil)

// LogToTimelineMapperTask is a task that adds revisions/events regarding logs.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
	googlecloudlogmulticloudapiaudit_contract.LogToTimelineMapperTaskID,
	&multicloudAuditLogLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabelV2(`MultiCloud API logs`,
		`Gather Anthos MultiCloud audit log including cluster creation, deletion and upgrades.`,
		5000,
		true,
	),
)
