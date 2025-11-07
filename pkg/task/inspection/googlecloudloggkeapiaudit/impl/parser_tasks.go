// Copyright 2024 Google LLC
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
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudloggkeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReaderTask is a task that reads and parses field sets from GKE audit logs.
// It uses GCPOperationAuditLogFieldSetReader and GKEAuditLogResourceFieldSetReader
// to extract common GCP audit log fields and GKE-specific resource fields.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudloggkeapiaudit_contract.FieldSetReaderTaskID, googlecloudloggkeapiaudit_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
	&googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSetReader{},
})

// LogSerializerTask is a task that serializes GKE audit logs for storage in the history builder.
var LogSerializerTask = inspectiontaskbase.NewLogSerializerTask(googlecloudloggkeapiaudit_contract.LogSerializerTaskID, googlecloudloggkeapiaudit_contract.ListLogEntriesTaskID.Ref())

// LogGrouperTask is a task that groups GKE audit logs by their resource path.
// This grouping allows for parallel processing of logs related to the same resource.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudloggkeapiaudit_contract.LogGrouperTaskID, googlecloudloggkeapiaudit_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		resourceFieldSet, err := log.GetFieldSet(l, &googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{})
		if err != nil {
			return ""
		}
		return resourceFieldSet.ResourcePath().Path
	},
)

// HistoryModifierTask is a task that adds revisions/events regarding logs.
var HistoryModifierTask = inspectiontaskbase.NewHistoryModifierTask[struct{}](googlecloudloggkeapiaudit_contract.HistoryModifierTaskID, &gkeAuditLogHistoryModifierSetting{},
	inspectioncore_contract.FeatureTaskLabel(`GKE Audit logs`,
		`Gather GKE audit log to show creation/upgrade/deletion of logs cluster/nodepool`,
		enum.LogTypeGkeAudit,
		5000,
		true,
		googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes...),
)

// gkeAuditLogHistoryModifierSetting implements the HistoryModifer interface for GKE audit logs.
type gkeAuditLogHistoryModifierSetting struct {
}

// Dependencies implements inspectiontaskbase.HistoryModifer.
// It returns a list of task references that this history modifier depends on.
func (g *gkeAuditLogHistoryModifierSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.HistoryModifer.
func (g *gkeAuditLogHistoryModifierSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudloggkeapiaudit_contract.LogGrouperTaskID.Ref()
}

// LogSerializerTask implements inspectiontaskbase.HistoryModifer.
func (g *gkeAuditLogHistoryModifierSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudloggkeapiaudit_contract.LogSerializerTaskID.Ref()
}

// ModifyChangeSetFromLog implements inspectiontaskbase.HistoryModifer.
func (g *gkeAuditLogHistoryModifierSetting) ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	commonFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return struct{}{}, err
	}
	auditFieldSet, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return struct{}{}, err
	}
	resourceFieldSet, err := log.GetFieldSet(l, &googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{})
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

		methodNameParts := strings.Split(auditFieldSet.MethodName, ".")
		shortMethodName := methodNameParts[len(methodNameParts)-1]

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

var _ inspectiontaskbase.HistoryModifer[struct{}] = (*gkeAuditLogHistoryModifierSetting)(nil)
