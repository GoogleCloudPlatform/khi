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

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcomputeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomputeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReaderTask parses GCPOperationAuditLogFieldSet from raw Compute API logs.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogcomputeapiaudit_contract.FieldSetReaderTaskID, googlecloudlogcomputeapiaudit_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
})

// LogIngesterTask is a V2 task that ingests log metadata (timestamp, severity, summary, log type) into KHI v6 format.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(googlecloudlogcomputeapiaudit_contract.LogIngesterTaskID, &gcpComputeAuditLogLogIngester{})

// LogGrouperTask groups GCE API audit logs by node resource name for parallel mapper processing.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlogcomputeapiaudit_contract.LogGrouperTaskID, googlecloudlogcomputeapiaudit_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
		if err != nil {
			return "unknown"
		}
		return getInstanceNameFromResourceName(audit.ResourceName)
	})

// LogToTimelineMapperTask maps GCE API audit logs to timeline events and revisions in parallel.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2[struct{}](googlecloudlogcomputeapiaudit_contract.LogToTimelineMapperTaskID, &gcpComputeAuditLogLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabelV2("Compute API Logs",
		"Gather Compute API audit logs to visualize the provisioning of infrastructure resources (e.g., GCE VM creation/deletion, Persistent Disk mounting) on associated timelines.",
		6000,
		true,
	),
)

type gcpComputeAuditLogLogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *gcpComputeAuditLogLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcomputeapiaudit_contract.FieldSetReaderTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *gcpComputeAuditLogLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog parses raw log entry and manually populates the LogChangeSet.
func (i *gcpComputeAuditLogLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	commonSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, err
	}
	cs.SetTimestamp(commonSet.Timestamp)

	if severitySet, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severitySet.Severity)
	} else {
		cs.SetSeverity(inspectioncore_contract.SeverityUnknown)
	}

	cs.SetLogType(googlecloudlogcomputeapiaudit_contract.LogTypeComputeApi)

	audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return nil, err
	}

	var summary string
	switch {
	case audit.Starting():
		summary = fmt.Sprintf("%s Started", audit.MethodName)
	case audit.Ending():
		summary = fmt.Sprintf("%s Finished", audit.MethodName)
	default:
		summary = audit.MethodName
	}
	cs.SetSummary(summary)

	return cs, nil
}

// Explicit interface compliance assertion.
var _ inspectiontaskbase.LogIngesterV2 = (*gcpComputeAuditLogLogIngester)(nil)

type gcpComputeAuditLogLogToTimelineMapperSetting struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns a reference to the log ingester task.
func (g *gcpComputeAuditLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcomputeapiaudit_contract.LogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies.
func (g *gcpComputeAuditLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogcomputeapiaudit_contract.ClusterIdentityTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the log grouper task.
func (g *gcpComputeAuditLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogcomputeapiaudit_contract.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup translates a single GCE API audit log into timeline event/revision changesets.
func (g *gcpComputeAuditLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	commonLogFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}

	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogcomputeapiaudit_contract.ClusterIdentityTaskID.Ref())
	nodeTimelinePath := googlecloudlogcomputeapiaudit_contract.MustNodeTimelinePath(ctx, clusterIdentity.ClusterName, getInstanceNameFromResourceName(audit.ResourceName))

	var targetPath *khifilev6.TimelinePath
	if audit.ImmediateOperation() {
		targetPath = nodeTimelinePath
	} else {
		methodNameSplitted := strings.Split(audit.MethodName, ".")
		shortMethodName := "unknown"
		if len(methodNameSplitted) > 0 {
			shortMethodName = methodNameSplitted[len(methodNameSplitted)-1]
		}
		targetPath = googlecloudcommon_contract.MustGCPOperationTimeline(ctx, nodeTimelinePath, shortMethodName, audit.OperationID)
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	if audit.ImmediateOperation() {
		cs.AddEvent(targetPath)
	} else {
		state := googlecloudcommon_contract.RevisionStateOperationStarted
		verb := googlecloudcommon_contract.VerbOperationStart
		if audit.Ending() {
			state = googlecloudcommon_contract.RevisionStateOperationFinished
			verb = googlecloudcommon_contract.VerbOperationFinish
		}
		var body structured.Node
		if audit.Request != nil {
			body = audit.Request.Node
		}
		cs.AddRevision(targetPath, &khifilev6.StagingRevision{
			ResourceBody: body,
			VerbType:     verb,
			StateType:    state,
			Principal:    audit.PrincipalEmail,
			ChangedTime:  commonLogFieldSet.Timestamp,
		})
	}

	return cs, struct{}{}, nil
}

// Explicit interface compliance assertion.
var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*gcpComputeAuditLogLogToTimelineMapperSetting)(nil)

func getInstanceNameFromResourceName(resourceName string) string {
	resourceNameSplitted := strings.Split(resourceName, "/")
	if len(resourceNameSplitted) < 1 {
		return ""
	}
	return resourceNameSplitted[len(resourceNameSplitted)-1]
}
