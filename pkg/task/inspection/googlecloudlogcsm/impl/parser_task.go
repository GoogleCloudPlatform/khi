// Copyright 2026 Google LLC
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
package googlecloudlogcsm_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FieldSetReaderTask is a task that reads CSM access logs field sets.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogcsm_contract.FieldSetReaderTaskID, googlecloudlogcsm_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudcommon_contract.GCPAccessLogFieldSetReader{},
	&googlecloudlogcsm_contract.IstioAccessLogFieldSetReader{},
	&googlecloudcommon_contract.GCPDefaultSeverityFieldSetReader{},
})

// CSMAccessLogLogIngester ingests CSM access logs.
type CSMAccessLogLogIngester struct{}

// RawLogTask returns the task reference that provides raw logs.
func (i *CSMAccessLogLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcsm_contract.FieldSetReaderTaskID.Ref()
}

// Dependencies returns the task dependencies.
func (i *CSMAccessLogLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog parses raw log entry and populates the LogChangeSet.
func (i *CSMAccessLogLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, err
	}
	cs.SetTimestamp(commonFS.Timestamp)

	gcpCommonAccessLog, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAccessLogFieldSet{})
	if err != nil {
		return nil, err
	}
	istioAccessLog, err := log.GetFieldSet(l, &googlecloudlogcsm_contract.IstioAccessLogFieldSet{})
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("%d %s %s", gcpCommonAccessLog.Status, gcpCommonAccessLog.Method, gcpCommonAccessLog.RequestURL)
	if istioAccessLog.ResponseFlag != googlecloudlogcsm_contract.ResponseFlagNoError {
		summary = fmt.Sprintf("【%s(%s)】", istioAccessLog.ResponseFlagMessage(), istioAccessLog.ResponseFlag) + summary
	}
	cs.SetSummary(summary)
	cs.SetLogType(googlecloudlogcsm_contract.LogTypeCSMAccessLog)

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severityFS.Severity)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngester = (*CSMAccessLogLogIngester)(nil)

// LogIngesterTask is the task that executes CSMAccessLogLogIngester.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudlogcsm_contract.LogIngesterTaskID,
	&CSMAccessLogLogIngester{},
)

// LogGrouperTask groups CSM access logs by their reporter pod.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlogcsm_contract.LogGrouperTaskID, googlecloudlogcsm_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		istioAccessLogFieldSet := log.MustGetFieldSet(l, &googlecloudlogcsm_contract.IstioAccessLogFieldSet{})
		return fmt.Sprintf("%s-%s", istioAccessLogFieldSet.ReporterPodNamespace, istioAccessLogFieldSet.ReporterPodName)
	},
)

// CSMAccessLogLogToTimelineMapper maps CSM access logs to resource timelines.
type CSMAccessLogLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns a reference to the task that provides ingested logs.
func (m *CSMAccessLogLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcsm_contract.LogIngesterTaskID.Ref()
}

// Dependencies returns additional task dependencies.
func (m *CSMAccessLogLogToTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogcsm_contract.ClusterIdentityTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *CSMAccessLogLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogcsm_contract.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps each log inside a group to one or more timeline events.
func (m *CSMAccessLogLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	istioAccessLog, err := log.GetFieldSet(l, &googlecloudlogcsm_contract.IstioAccessLogFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}

	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.ClusterIdentityTaskID.Ref())
	clusterName := clusterIdentity.ClusterName

	cs := khifilev6.NewTimelineChangeSet(l)

	switch istioAccessLog.Type {
	case googlecloudlogcsm_contract.AccessLogTypeServer:
		cs.AddEvent(googlecloudlogcsm_contract.MustCSMServerAccessTimeline(ctx, clusterName, istioAccessLog.ReporterPodNamespace, istioAccessLog.ReporterPodName, istioAccessLog.ReporterContainerName))
		if istioAccessLog.SourceName != "" && istioAccessLog.SourceNamespace != "" {
			cs.AddEvent(googlecloudlogcsm_contract.MustCSMClientAccessTimeline(ctx, clusterName, istioAccessLog.SourceNamespace, istioAccessLog.SourceName))
		}
		if istioAccessLog.DestinationServiceName != "" && istioAccessLog.DestinationServiceNamespace != "" {
			cs.AddEvent(googlecloudlogcsm_contract.MustCSMServiceServerAccessTimeline(ctx, clusterName, istioAccessLog.DestinationServiceNamespace, istioAccessLog.DestinationServiceName))
		}
	case googlecloudlogcsm_contract.AccessLogTypeClient:
		cs.AddEvent(googlecloudlogcsm_contract.MustCSMClientAccessTimeline(ctx, clusterName, istioAccessLog.ReporterPodNamespace, istioAccessLog.ReporterPodName))
		if istioAccessLog.DestinationName != "" && istioAccessLog.DestinationNamespace != "" {
			cs.AddEvent(googlecloudlogcsm_contract.MustCSMServerAccessTimeline(ctx, clusterName, istioAccessLog.DestinationNamespace, istioAccessLog.DestinationName, ""))
		}
		if istioAccessLog.DestinationServiceName != "" && istioAccessLog.DestinationServiceNamespace != "" {
			cs.AddEvent(googlecloudlogcsm_contract.MustCSMServiceClientAccessTimeline(ctx, clusterName, istioAccessLog.DestinationServiceNamespace, istioAccessLog.DestinationServiceName))
		}
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*CSMAccessLogLogToTimelineMapper)(nil)

// LogToTimelineMapperTask maps CSM access logs to timelines.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	googlecloudlogcsm_contract.LogToTimelineMapperTaskID,
	&CSMAccessLogLogToTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabel(
		"CSM Access Logs",
		"Gather CSM access logs to visualize network traffic flows and latency under client or server Pod timelines.",
		10000,
		false,
	),
)
