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

package googlecloudlogcsm_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogcsm_contract.FieldSetReaderTaskID, googlecloudlogcsm_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudcommon_contract.GCPAccessLogFieldSetReader{},
	&googlecloudlogcsm_contract.IstioAccessLogFieldSetReader{},
})

var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudlogcsm_contract.LogIngesterTaskID,
	googlecloudlogcsm_contract.ListLogEntriesTaskID.Ref(),
)

var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlogcsm_contract.LogGrouperTaskID, googlecloudlogcsm_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		istioAccessLogFieldSet := log.MustGetFieldSet(l, &googlecloudlogcsm_contract.IstioAccessLogFieldSet{})
		return fmt.Sprintf("%s-%s", istioAccessLogFieldSet.ReporterPodNamespace, istioAccessLogFieldSet.ReporterPodName)
	},
)

var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](googlecloudlogcsm_contract.LogToTimelineMapperTaskID, &csmAccessLogLogToTimelineMapperSetting{}, inspectioncore_contract.FeatureTaskLabel(
	"CSM Access Log",
	"Gather CSM access logs from Cloud Logging and associate them in client or server Pods on timelines",
	enum.LogTypeCSMAccessLog,
	10000,
	false,
	googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes...,
))

type csmAccessLogLogToTimelineMapperSetting struct{}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (c *csmAccessLogLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (c *csmAccessLogLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogcsm_contract.LogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (c *csmAccessLogLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcsm_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (c *csmAccessLogLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	gcpCommonAccessLog := log.MustGetFieldSet(l, &googlecloudcommon_contract.GCPAccessLogFieldSet{})
	istioAccessLog := log.MustGetFieldSet(l, &googlecloudlogcsm_contract.IstioAccessLogFieldSet{})

	switch istioAccessLog.Type {
	case googlecloudlogcsm_contract.AccessLogTypeServer:
		cs.AddEvent(resourcepath.CSMServerAccess(istioAccessLog.ReporterPodNamespace, istioAccessLog.ReporterPodName, istioAccessLog.ReporterContainerName))
		if istioAccessLog.SourceName != "" && istioAccessLog.SourceNamespace != "" {
			cs.AddEvent(resourcepath.CSMClientAccess(istioAccessLog.SourceNamespace, istioAccessLog.SourceName))
		}
		if istioAccessLog.DestinationServiceName != "" && istioAccessLog.DestinationServiceNamespace != "" {
			cs.AddEvent(resourcepath.CSMServiceServerAccess(istioAccessLog.DestinationServiceNamespace, istioAccessLog.DestinationServiceName))
		}
	case googlecloudlogcsm_contract.AccessLogTypeClient:
		cs.AddEvent(resourcepath.CSMClientAccess(istioAccessLog.ReporterPodNamespace, istioAccessLog.ReporterPodName))
		if istioAccessLog.DestinationName != "" && istioAccessLog.DestinationNamespace != "" {
			cs.AddEvent(resourcepath.CSMServerAccess(istioAccessLog.DestinationNamespace, istioAccessLog.DestinationName, ""))
		}
		if istioAccessLog.DestinationServiceName != "" && istioAccessLog.DestinationServiceNamespace != "" {
			cs.AddEvent(resourcepath.CSMServiceClientAccess(istioAccessLog.DestinationServiceNamespace, istioAccessLog.DestinationServiceName))
		}
	}
	summary := fmt.Sprintf("%d %s %s", gcpCommonAccessLog.Status, gcpCommonAccessLog.Method, gcpCommonAccessLog.RequestURL)
	if istioAccessLog.ResponseFlag != googlecloudlogcsm_contract.ResponseFlagNoError {
		summary = fmt.Sprintf("【%s(%s)】", istioAccessLog.ResponseFlagMessage(), istioAccessLog.ResponseFlag) + summary
	}
	cs.SetLogSummary(summary)
	return struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*csmAccessLogLogToTimelineMapperSetting)(nil)
