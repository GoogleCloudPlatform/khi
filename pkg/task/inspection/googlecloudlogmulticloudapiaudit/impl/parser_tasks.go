// Copyright 2025 Google LLC
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
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogmulticloudapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogmulticloudapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReaderTask is a task that reads and parses field sets from MulticloudAPI audit logs.
// It uses GCPOperationAuditLogFieldSetReader and MulticloudAPIAuditResourceFieldSetReader
// to extract common GCP audit log fields and multicloud api-specific resource fields.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogmulticloudapiaudit_contract.FieldSetReaderTaskID, googlecloudlogmulticloudapiaudit_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
	&googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSetReader{},
})

// LogIngesterTask is a task that serializes MulticloudAPI audit logs for storage in the history builder.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(googlecloudlogmulticloudapiaudit_contract.LogIngesterTaskID, googlecloudlogmulticloudapiaudit_contract.ListLogEntriesTaskID.Ref())

// LogGrouperTask is a task that groups MulticloudAPI audit logs by their resource path.
// This grouping allows for parallel processing of logs related to the same resource.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlogmulticloudapiaudit_contract.LogGrouperTaskID, googlecloudlogmulticloudapiaudit_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		resourceFieldSet, err := log.GetFieldSet(l, &googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{})
		if err != nil {
			return ""
		}
		return resourceFieldSet.ResourcePath().Path
	},
)

// LogToTimelineMapperTask is a task that adds revisions/events regarding logs.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](googlecloudlogmulticloudapiaudit_contract.LogToTimelineMapperTaskID, &multicloudAuditLogLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabel(`MultiCloud API logs`,
		`Gather Anthos Multicloud audit log including cluster creation,deletion and upgrades.`,
		enum.LogTypeGkeAudit,
		5000,
		true,
		googlecloudinspectiontypegroup_contract.GKEMultiCloudClusterInspectionTypes...),
)

type multicloudAuditLogLogToTimelineMapperSetting struct {
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (m *multicloudAuditLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (m *multicloudAuditLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogmulticloudapiaudit_contract.LogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (m *multicloudAuditLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogmulticloudapiaudit_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (m *multicloudAuditLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	commonFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return struct{}{}, err
	}
	auditFieldSet, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return struct{}{}, err
	}
	resourceFieldSet, err := log.GetFieldSet(l, &googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{})
	if err != nil {
		return struct{}{}, err
	}

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
		shortMethodName = strings.ReplaceAll(shortMethodName, clusterTypeToFragmentInMethodNameMapping[resourceFieldSet.ClusterType], "") // Remove type specific part. Example: converting CreateAwsCluster to CreateCluster.

		switch shortMethodName {
		case "CreateCluster", "CreateNodePool":
			var bodyRaw []byte
			state := enum.RevisionStateProvisioning
			if auditFieldSet.Ending() {
				state = enum.RevisionStateExisting
			}
			if auditFieldSet.Request != nil {
				bodyRaw, _ = auditFieldSet.Request.Serialize(resourceBodyField, &structured.YAMLNodeSerializer{})
			}
			cs.AddRevision(resourceFieldSet.ResourcePath(), &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbCreate,
				State:      state,
				Requestor:  auditFieldSet.PrincipalEmail,
				ChangeTime: commonFieldSet.Timestamp,
				Partial:    false,
				Body:       string(bodyRaw),
			})
		case "DeleteCluster", "DeleteNodePool":
			state := enum.RevisionStateDeleting
			if auditFieldSet.Ending() {
				state = enum.RevisionStateDeleted
			}
			cs.AddRevision(resourceFieldSet.ResourcePath(), &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbDelete,
				State:      state,
				Requestor:  auditFieldSet.PrincipalEmail,
				ChangeTime: commonFieldSet.Timestamp,
				Partial:    false,
				Body:       "",
			})
		}

		state := enum.RevisionStateOperationStarted
		verb := enum.RevisionVerbOperationStart
		if auditFieldSet.Ending() {
			state = enum.RevisionStateOperationFinished
			verb = enum.RevisionVerbOperationFinish
		}
		requestBody, _ := auditFieldSet.RequestString()
		cs.AddRevision(auditFieldSet.OperationPath(resourceFieldSet.ResourcePath()), &history.StagingResourceRevision{
			Body:       requestBody,
			Verb:       verb,
			State:      state,
			Requestor:  auditFieldSet.PrincipalEmail,
			ChangeTime: commonFieldSet.Timestamp,
			Partial:    false,
		})
	} else {
		cs.AddEvent(resourceFieldSet.ResourcePath())
	}

	switch {
	case auditFieldSet.Starting():
		cs.SetLogSummary(fmt.Sprintf("%s Started", auditFieldSet.MethodName))
	case auditFieldSet.Ending():
		cs.SetLogSummary(fmt.Sprintf("%s Finished", auditFieldSet.MethodName))
	default:
		cs.SetLogSummary(auditFieldSet.MethodName)
	}
	return struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*multicloudAuditLogLogToTimelineMapperSetting)(nil)
