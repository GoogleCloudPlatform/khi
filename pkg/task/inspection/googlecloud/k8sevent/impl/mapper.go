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

package k8sevent_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8sevent"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// processK8sEventLog sets the log type, timestamp, severity and summary of a Kubernetes event log.
// The summary shows the resources that finder resolves from the resource UIDs in the event message.
func processK8sEventLog(ctx context.Context, l *log.Log, finder patternfinder.PatternFinder[*k8saudit.ResourceIdentity]) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(k8saudit.LogTypeEvent)
	cs.SetTimestamp(l.Timestamp)

	if severity, err := gcpcommon.ExtractGCPSeverity(l.NodeReader); err == nil && severity != nil {
		cs.SetSeverity(severity)
	}

	event, err := k8sevent.ExtractKubernetesEvent(l.NodeReader)
	if err != nil {
		return nil, fmt.Errorf("failed to extract kubernetes event: %w", err)
	}
	cs.SetSummary(k8saudit.FormatEventSummary(event.Reason, event.Message, finder))

	return cs, nil
}

// logIngesterTask is the log ingester task for GKE Event Logs.
var logIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	k8sevent.LogIngesterTaskID,
	k8sevent.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		finder := coretask.Use(b, k8saudit.ResourceUIDPatternFinderTaskID.Ref())
		return func(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
			return processK8sEventLog(ctx, l, finder.Get(ctx))
		}
	},
)

// logGrouperTask groups logs by the event's resource path so they can be mapped to timelines.
var logGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	k8sevent.LogGrouperTaskID,
	k8sevent.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			event, err := k8sevent.ExtractKubernetesEvent(l.NodeReader)
			if err != nil {
				return "unknown"
			}
			return fmt.Sprintf("cluster=%s,kind=%s,namespace=%s,name=%s", event.ClusterName, event.ResourceKind, event.Namespace, event.Resource)
		}
	},
)

// k8sEventTimelineMapper maps grouped GKE Event Logs to resource timelines.
type k8sEventTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	finder coretask.Input[patternfinder.PatternFinder[*k8saudit.ResourceIdentity]]
}

// ProcessLogByGroup maps a single GKE Event Log to its resource timeline path and matches any resource UIDs in the message.
func (m *k8sEventTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs, err := mapK8sEventLog(ctx, l, m.finder.Get(ctx))
	return cs, struct{}{}, err
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*k8sEventTimelineMapper)(nil)

// mapK8sEventLog adds an event for a Kubernetes event log to the timeline of its involved resource
// and to the timelines of the resources that finder resolves from the resource UIDs in the event message.
func mapK8sEventLog(ctx context.Context, l *log.Log, finder patternfinder.PatternFinder[*k8saudit.ResourceIdentity]) (*khifilev6.TimelineChangeSet, error) {
	event, err := k8sevent.ExtractKubernetesEvent(l.NodeReader)
	if err != nil {
		return nil, fmt.Errorf("failed to extract kubernetes event: %w", err)
	}

	primaryResourcePath := mustResolveK8sResourceTimelinePath(ctx, &event)
	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(primaryResourcePath)

	if event.Message != "" {
		if finder != nil {
			matches := patternfinder.FindAllWithStarterRunes(event.Message, finder, true, k8saudit.EventMessageUIDStarterRunes...)
			for _, match := range matches {
				matchedPath := k8saudit.MustResourceTimeline(ctx, event.ClusterName, match.Value)
				cs.AddEvent(matchedPath)
			}
		}
	}

	return cs, nil
}

// logToTimelineMapperTask is the task to map GKE Event Logs into timeline events.
var logToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	k8sevent.LogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: k8sevent.LogIngesterTaskID.Ref(),
		GroupedLogs: k8sevent.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &k8sEventTimelineMapper{
			finder: coretask.Use(b, k8saudit.ResourceUIDPatternFinderTaskID.Ref()),
		}
	},
	inspectioncore.FeatureTaskLabel(
		"Kubernetes Event Logs",
		"Gather Kubernetes event logs to visualize cluster events on associated resource timelines.",
		2000,
		true,
	),
)

// mustResolveK8sResourceTimelinePath resolves a KubernetesEventFieldSet to a *khifilev6.TimelinePath.
func mustResolveK8sResourceTimelinePath(ctx context.Context, event *k8sevent.KubernetesEventFieldSet) *khifilev6.TimelinePath {
	if event.Resource == "" {
		projectTimeline := gcpcommon.MustGCPProjectTimeline(ctx, event.ProjectID)
		gkeTimeline := gcpcommon.MustGKEClusterTimeline(ctx, projectTimeline, event.ClusterName)
		return k8sevent.MustEventExporterTimeline(ctx, gkeTimeline)
	}

	clusterTimeline := k8saudit.MustK8sClusterTimeline(ctx, event.ClusterName)
	apiVersionPath := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterTimeline, event.APIVersion)
	kindPath := k8saudit.MustK8sKindTimeline(ctx, apiVersionPath, event.ResourceKind)
	if event.Namespace == "cluster-scope" || event.Namespace == "" {
		return k8saudit.MustK8sClusterScopeResourceTimeline(ctx, kindPath, event.Resource)
	}
	namespacePath := k8saudit.MustK8sNamespaceTimeline(ctx, kindPath, event.Namespace)
	return k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespacePath, event.Resource)
}
