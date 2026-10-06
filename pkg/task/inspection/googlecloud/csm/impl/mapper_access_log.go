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
package csm_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csm"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// processCSMTrafficLog sets the timestamp, summary, log type and severity of a CSM traffic log.
func processCSMTrafficLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetTimestamp(l.Timestamp)

	gcpCommonAccessLog, err := gcpcommon.ExtractGCPAccessLog(l.NodeReader)
	if err != nil {
		return nil, err
	}
	istioAccessLog, err := csm.ExtractIstioAccessLog(l.NodeReader)
	if err != nil {
		return nil, err
	}

	summary := logutil.FormatEnvoySummary(gcpCommonAccessLog.Status, gcpCommonAccessLog.Method, gcpCommonAccessLog.RequestURL, istioAccessLog.ResponseFlags)
	cs.SetSummary(summary)
	cs.SetLogType(csm.LogTypeCSMTrafficLog)

	if severity, err := gcpcommon.ExtractGCPSeverity(l.NodeReader); err == nil && severity != nil {
		cs.SetSeverity(severity)
	}

	return cs, nil
}

// logIngesterTask ingests the metadata of CSM traffic logs.
var logIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	csm.LogIngesterTaskID,
	csm.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		return processCSMTrafficLog
	},
)

// logGrouperTask groups CSM traffic logs by their reporter pod.
var logGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	csm.LogGrouperTaskID,
	csm.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			istioAccessLogFieldSet, err := csm.ExtractIstioAccessLog(l.NodeReader)
			if err != nil {
				return "unknown"
			}
			return fmt.Sprintf("%s-%s", istioAccessLogFieldSet.ReporterPodNamespace, istioAccessLogFieldSet.ReporterPodName)
		}
	},
)

// csmTrafficLogTimelineMapper maps CSM traffic logs to resource timelines.
type csmTrafficLogTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
}

// ProcessLogByGroup maps each log inside a group to one or more timeline events.
func (m *csmTrafficLogTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs, err := mapCSMTrafficLog(ctx, l, m.clusterIdentity.Get(ctx).ClusterName)
	return cs, struct{}{}, err
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*csmTrafficLogTimelineMapper)(nil)

// mapCSMTrafficLog adds events for a CSM traffic log to the access timelines of the pods and the service it involves in the given cluster.
func mapCSMTrafficLog(ctx context.Context, l *log.Log, clusterName string) (*khifilev6.TimelineChangeSet, error) {
	istioAccessLog, err := csm.ExtractIstioAccessLog(l.NodeReader)
	if err != nil {
		return nil, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	switch istioAccessLog.Type {
	case csm.AccessLogTypeServer:
		cs.AddEvent(csm.MustCSMServerAccessTimeline(ctx, clusterName, istioAccessLog.ReporterPodNamespace, istioAccessLog.ReporterPodName, istioAccessLog.ReporterContainerName))
		if istioAccessLog.SourceName != "" && istioAccessLog.SourceNamespace != "" {
			cs.AddEvent(csm.MustCSMClientAccessTimeline(ctx, clusterName, istioAccessLog.SourceNamespace, istioAccessLog.SourceName))
		}
		if istioAccessLog.DestinationServiceName != "" && istioAccessLog.DestinationServiceNamespace != "" {
			cs.AddEvent(csm.MustCSMServiceServerAccessTimeline(ctx, clusterName, istioAccessLog.DestinationServiceNamespace, istioAccessLog.DestinationServiceName))
		}
	case csm.AccessLogTypeClient:
		cs.AddEvent(csm.MustCSMClientAccessTimeline(ctx, clusterName, istioAccessLog.ReporterPodNamespace, istioAccessLog.ReporterPodName))
		if istioAccessLog.DestinationName != "" && istioAccessLog.DestinationNamespace != "" {
			cs.AddEvent(csm.MustCSMServerAccessTimeline(ctx, clusterName, istioAccessLog.DestinationNamespace, istioAccessLog.DestinationName, ""))
		}
		if istioAccessLog.DestinationServiceName != "" && istioAccessLog.DestinationServiceNamespace != "" {
			cs.AddEvent(csm.MustCSMServiceClientAccessTimeline(ctx, clusterName, istioAccessLog.DestinationServiceNamespace, istioAccessLog.DestinationServiceName))
		}
	}

	return cs, nil
}

// logToTimelineMapperTask maps CSM traffic logs to timelines.
var logToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	csm.LogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: csm.LogIngesterTaskID.Ref(),
		GroupedLogs: csm.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &csmTrafficLogTimelineMapper{
			clusterIdentity: coretask.Use(b, csm.ClusterIdentityTaskID.Ref()),
		}
	},
	inspectioncore.FeatureTaskLabel(
		"CSM Traffic Logs",
		"Gather CSM traffic logs to visualize network traffic flows and latency under client or server Pod timelines.",
		10000,
		false,
	),
)
