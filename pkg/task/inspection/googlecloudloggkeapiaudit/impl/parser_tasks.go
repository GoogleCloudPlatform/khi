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

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
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
var LogIngesterTask = googlecloudcommon_contract.NewGCPOperationLogIngesterTask(
	googlecloudloggkeapiaudit_contract.LogIngesterTaskID,
	googlecloudloggkeapiaudit_contract.FieldSetReaderTaskID.Ref(),
	googlecloudloggkeapiaudit_contract.LogTypeGkeAudit,
)

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
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[*googlecloudcommon_contract.GCPOperationTracker](googlecloudloggkeapiaudit_contract.LogToTimelineMapperTaskID, &gkeAuditLogLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabel(`GKE Audit Logs`,
		`Gather GKE audit logs to visualize the creation, upgrade, and deletion of clusters and node pools on timelines.`,
		5000,
		true),
)

// gkeAuditLogLogToTimelineMapperSetting implements the LogToTimelineMapper interface for GKE audit logs.
type gkeAuditLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.SinglePassMapperBase[*googlecloudcommon_contract.GCPOperationTracker]
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
func (g *gkeAuditLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *googlecloudcommon_contract.GCPOperationTracker) (*khifilev6.TimelineChangeSet, *googlecloudcommon_contract.GCPOperationTracker, error) {
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
	resourceFieldSet, err := log.GetFieldSet(l, &googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{})
	if err != nil {
		return nil, tracker, err
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

	methodNameParts := strings.Split(auditFieldSet.MethodName, ".")
	shortMethodName := methodNameParts[len(methodNameParts)-1]

	operationTimeline := googlecloudcommon_contract.MustGCPOperationTimeline(ctx, targetTimeline, shortMethodName, auditFieldSet.OperationID)
	googlecloudcommon_contract.ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetTimeline, operationTimeline, auditFieldSet, commonFieldSet, shortMethodName, resourceFieldSet.IsCluster())

	return cs, tracker, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[*googlecloudcommon_contract.GCPOperationTracker] = (*gkeAuditLogLogToTimelineMapperSetting)(nil)
