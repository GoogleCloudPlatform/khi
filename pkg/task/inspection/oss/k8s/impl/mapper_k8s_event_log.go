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

package ossk8s_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	ossk8s "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/oss/k8s"
)

// processOSSK8sEventLog sets the log type, timestamp, severity and summary of a Kubernetes event in an OSS audit log.
// The summary shows the resources that finder resolves from the resource UIDs in the event message.
func processOSSK8sEventLog(ctx context.Context, l *log.Log, finder patternfinder.PatternFinder[*k8saudit.ResourceIdentity]) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(k8saudit.LogTypeEvent)
	cs.SetTimestamp(l.Timestamp)

	event, err := ossk8s.ExtractOSSK8sEvent(l.NodeReader)
	if err != nil {
		return nil, fmt.Errorf("failed to get OSS k8s event fieldset: %w", err)
	}
	cs.SetSummary(k8saudit.FormatEventSummary(event.Reason, event.Message, finder))
	cs.SetSeverity(inspectioncore.SeverityUnknown)

	return cs, nil
}

// ossK8sEventLogIngesterTask is the log ingester task.
var ossK8sEventLogIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	ossk8s.OSSK8sEventLogIngesterTaskID,
	ossk8s.EventAuditLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		finder := coretask.Use(b, k8saudit.ResourceUIDPatternFinderTaskID.Ref())
		return func(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
			return processOSSK8sEventLog(ctx, l, finder.Get(ctx))
		}
	},
)

// ossK8sEventLogGrouperTask groups event logs by their resource path.
var ossK8sEventLogGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	ossk8s.OSSK8sEventLogGrouperTaskID,
	ossk8s.EventAuditLogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			event, err := ossk8s.ExtractOSSK8sEvent(l.NodeReader)
			if err != nil {
				return "unknown"
			}
			return event.ResourceIdentity().String()
		}
	},
)

// ossK8sEventTimelineMapper maps grouped events to timeline paths.
type ossK8sEventTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	finder coretask.Input[patternfinder.PatternFinder[*k8saudit.ResourceIdentity]]
}

// ProcessLogByGroup maps a single event log to its resource timeline and matches any resource UIDs in the message.
func (m *ossK8sEventTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs, err := mapOSSK8sEventLog(ctx, l, m.finder.Get(ctx))
	return cs, struct{}{}, err
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*ossK8sEventTimelineMapper)(nil)

// mapOSSK8sEventLog adds an event for a Kubernetes event in an OSS audit log to the timeline of its involved resource
// and to the timelines of the resources that finder resolves from the resource UIDs in the event message.
func mapOSSK8sEventLog(ctx context.Context, l *log.Log, finder patternfinder.PatternFinder[*k8saudit.ResourceIdentity]) (*khifilev6.TimelineChangeSet, error) {
	event, err := ossk8s.ExtractOSSK8sEvent(l.NodeReader)
	if err != nil {
		return nil, fmt.Errorf("failed to get OSS k8s event fieldset: %w", err)
	}

	primaryResourcePath := k8saudit.MustResourceTimeline(ctx, "cluster", event.ResourceIdentity())
	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(primaryResourcePath)

	if event.Message != "" {
		if finder != nil {
			matches := patternfinder.FindAllWithStarterRunes(event.Message, finder, true, k8saudit.EventMessageUIDStarterRunes...)
			for _, match := range matches {
				matchedPath := k8saudit.MustResourceTimeline(ctx, "cluster", match.Value)
				cs.AddEvent(matchedPath)
			}
		}
	}

	return cs, nil
}

// ossK8sEventLogToTimelineMapperTask is the log to timeline mapper task.
var ossK8sEventLogToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	ossk8s.OSSK8sEventLogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: ossk8s.OSSK8sEventLogIngesterTaskID.Ref(),
		GroupedLogs: ossk8s.OSSK8sEventLogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &ossK8sEventTimelineMapper{finder: coretask.Use(b, k8saudit.ResourceUIDPatternFinderTaskID.Ref())}
	},
	inspectioncore.FeatureTaskLabel(
		"OSS Kubernetes Event Logs",
		"Gather and parse Kubernetes event logs from OSS Kubernetes JSONL audit logs to visualize resource lifecycle and operational events.",
		2000,
		true,
	),
)
