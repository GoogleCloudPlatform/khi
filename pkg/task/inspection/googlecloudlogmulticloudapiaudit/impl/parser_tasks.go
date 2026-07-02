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

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
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

// LogIngesterTask is a task that serializes MulticloudAPI audit logs for storage in the history builder.
var LogIngesterTask = googlecloudcommon_contract.NewGCPOperationLogIngesterTask(
	googlecloudlogmulticloudapiaudit_contract.LogIngesterTaskID,
	googlecloudlogmulticloudapiaudit_contract.FieldSetReaderTaskID.Ref(),
	googlecloudlogmulticloudapiaudit_contract.LogTypeMulticloudAPI,
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
	inspectiontaskbase.SinglePassMapperBase[*googlecloudcommon_contract.GCPOperationTracker]
}

// Dependencies implements LogToTimelineMapper.
func (m *multicloudAuditLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements LogToTimelineMapper.
func (m *multicloudAuditLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogmulticloudapiaudit_contract.LogGrouperTaskID.Ref()
}

// LogIngesterTask implements LogToTimelineMapper.
func (m *multicloudAuditLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogmulticloudapiaudit_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup maps grouped logs to resource timelines and operations in KHI V6 format.
func (m *multicloudAuditLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, tracker *googlecloudcommon_contract.GCPOperationTracker) (*khifilev6.TimelineChangeSet, *googlecloudcommon_contract.GCPOperationTracker, error) {
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
	resourceFieldSet, err := log.GetFieldSet(l, &googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{})
	if err != nil {
		return nil, tracker, err
	}

	projectPath := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, auditFieldSet.ProjectID)
	clusterPath := googlecloudlogmulticloudapiaudit_contract.MustMultiCloudClusterTimeline(ctx, projectPath, resourceFieldSet.ClusterName)

	var targetPath *khifilev6.TimelinePath
	if resourceFieldSet.IsCluster() {
		targetPath = clusterPath
	} else {
		targetPath = googlecloudlogmulticloudapiaudit_contract.MustMultiCloudNodepoolTimeline(ctx, clusterPath, resourceFieldSet.NodepoolName)
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	clusterTypeToFragmentInMethodNameMapping := map[googlecloudlogmulticloudapiaudit_contract.MultiCloudClusterType]string{
		googlecloudlogmulticloudapiaudit_contract.ClusterTypeAWS:   "Aws",
		googlecloudlogmulticloudapiaudit_contract.ClusterTypeAzure: "Azure",
	}

	methodNameParts := strings.Split(auditFieldSet.MethodName, ".")
	shortMethodName := methodNameParts[len(methodNameParts)-1]
	shortMethodName = strings.ReplaceAll(shortMethodName, clusterTypeToFragmentInMethodNameMapping[resourceFieldSet.ClusterType], "") // Remove type specific part.

	opPath := googlecloudlogmulticloudapiaudit_contract.MustOperationTimeline(ctx, targetPath, shortMethodName, auditFieldSet.OperationID)
	googlecloudcommon_contract.ProcessGCPClusterNodepoolOperationLog(ctx, cs, tracker, targetPath, opPath, auditFieldSet, commonFieldSet, shortMethodName, resourceFieldSet.IsCluster())

	return cs, tracker, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[*googlecloudcommon_contract.GCPOperationTracker] = (*multicloudAuditLogLogToTimelineMapperSetting)(nil)

// LogToTimelineMapperTask is a task that adds revisions/events regarding logs.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[*googlecloudcommon_contract.GCPOperationTracker](
	googlecloudlogmulticloudapiaudit_contract.LogToTimelineMapperTaskID,
	&multicloudAuditLogLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabel(`Multi-Cloud API Logs`,
		`Gather Anthos Multi-Cloud audit logs to visualize cluster lifecycle events (creation, deletion, and upgrades) on timelines.`,
		5000,
		true,
	),
)
