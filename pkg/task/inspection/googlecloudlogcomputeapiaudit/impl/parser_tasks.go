// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package googlecloudlogcomputeapiaudit_impl defines the implementation of the googlecloudlogcomputeapiaudit task.
package googlecloudlogcomputeapiaudit_impl

import (
	"context"
	"fmt"
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogcomputeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomputeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogcomputeapiaudit_contract.FieldSetReaderTaskID, googlecloudlogcomputeapiaudit_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
})

var LogSerializerTask = inspectiontaskbase.NewLogSerializerTask(googlecloudlogcomputeapiaudit_contract.LogSerializerTaskID, googlecloudlogcomputeapiaudit_contract.ListLogEntriesTaskID.Ref())

var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlogcomputeapiaudit_contract.LogGrouperTaskID, googlecloudlogcomputeapiaudit_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		// Compute Engine Audit log parser is stateless and it doesn't require grouping to work, but grouping them by its associated instance resource name for better performance to process them in parallel.
		audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
		if err != nil {
			return "unknown"
		}
		return getInstanceNameFromResourceName(audit.ResourceName)
	})

var HistoryModifierTask = inspectiontaskbase.NewHistoryModifierTask[struct{}](googlecloudlogcomputeapiaudit_contract.HistoryModifierTaskID, &gcpComputeAuditLogHistoryModifierSetting{},
	inspectioncore_contract.FeatureTaskLabel(`Compute API Logs`,
		`Gather Compute API audit logs to show the timings of the provisioning of resources(e.g creating/deleting GCE VM,mounting Persistent Disk...etc) on associated timelines.`,
		enum.LogTypeComputeApi,
		6000,
		true,
		googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes...),
)

type gcpComputeAuditLogHistoryModifierSetting struct {
}

// Dependencies implements inspectiontaskbase.HistoryModifer.
func (g *gcpComputeAuditLogHistoryModifierSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.HistoryModifer.
func (g *gcpComputeAuditLogHistoryModifierSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogcomputeapiaudit_contract.LogGrouperTaskID.Ref()
}

// LogSerializerTask implements inspectiontaskbase.HistoryModifer.
func (g *gcpComputeAuditLogHistoryModifierSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcomputeapiaudit_contract.LogSerializerTaskID.Ref()
}

// ModifyChangeSetFromLog implements inspectiontaskbase.HistoryModifer.
func (g *gcpComputeAuditLogHistoryModifierSetting) ModifyChangeSetFromLog(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	commonLogFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return struct{}{}, err
	}
	audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return struct{}{}, err
	}

	nodeResourcePath := resourcepath.Node(getInstanceNameFromResourceName(audit.ResourceName))
	resourcePath := audit.OperationPath(nodeResourcePath)

	if audit.ImmediateOperation() {
		cs.AddEvent(resourcePath)
	} else {
		state := enum.RevisionStateOperationStarted
		verb := enum.RevisionVerbOperationStart
		if audit.Ending() {
			state = enum.RevisionStateOperationFinished
			verb = enum.RevisionVerbOperationFinish
		}
		requestBody, _ := audit.RequestString()
		cs.AddRevision(resourcePath, &history.StagingResourceRevision{
			Body:       requestBody,
			Verb:       verb,
			State:      state,
			Requestor:  audit.PrincipalEmail,
			ChangeTime: commonLogFieldSet.Timestamp,
			Partial:    false,
		})
	}

	switch {
	case audit.Starting():
		cs.SetLogSummary(fmt.Sprintf("%s Started", audit.MethodName))
	case audit.Ending():
		cs.SetLogSummary(fmt.Sprintf("%s Finished", audit.MethodName))
	default:
		cs.SetLogSummary(audit.MethodName)
	}

	return struct{}{}, nil
}

var _ inspectiontaskbase.HistoryModifer[struct{}] = (*gcpComputeAuditLogHistoryModifierSetting)(nil)

func getInstanceNameFromResourceName(resourceName string) string {
	resourceNameSplitted := strings.Split(resourceName, "/")
	if len(resourceNameSplitted) < 1 {
		return ""
	}
	return resourceNameSplitted[len(resourceNameSplitted)-1]
}
