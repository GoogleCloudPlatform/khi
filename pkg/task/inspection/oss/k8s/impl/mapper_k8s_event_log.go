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
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	ossk8s "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/oss/k8s"
)

// OSSK8sEventLogIngester handles event log metadata ingestion.
type OSSK8sEventLogIngester struct{}

// RawLogTask returns the task reference providing raw event logs.
func (i *OSSK8sEventLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return ossk8s.EventAuditLogFilterTaskID.Ref()
}

// Dependencies returns additional dependencies of the ingester.
func (i *OSSK8sEventLogIngester) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8saudit.ResourceUIDPatternFinderTaskID.Ref(),
	}
}

// ProcessLog populates metadata into the LogChangeSet.
func (i *OSSK8sEventLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
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
	finder := coretask.GetTaskResult(ctx, k8saudit.ResourceUIDPatternFinderTaskID.Ref())
	cs.SetSummary(k8saudit.FormatEventSummary(event.Reason, event.Message, finder))
	cs.SetSeverity(inspectioncore.SeverityUnknown)

	return cs, nil
}

var _ inspectiontaskbase.LogIngester = (*OSSK8sEventLogIngester)(nil)

// OSSK8sEventLogIngesterTask is the log ingester task.
var OSSK8sEventLogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	ossk8s.OSSK8sEventLogIngesterTaskID,
	&OSSK8sEventLogIngester{},
)

// OSSK8sEventLogGrouperTask groups event logs by their resource path.
var OSSK8sEventLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	ossk8s.OSSK8sEventLogGrouperTaskID,
	ossk8s.EventAuditLogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		event, err := ossk8s.ExtractOSSK8sEvent(l.NodeReader)
		if err != nil {
			return "unknown"
		}
		return event.ResourceIdentity().String()
	},
)

// OSSK8sEventTimelineMapper maps grouped events to timeline paths.
type OSSK8sEventTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns the prerequisite log ingester task.
func (m *OSSK8sEventTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return ossk8s.OSSK8sEventLogIngesterTaskID.Ref()
}

// Dependencies returns additional mapper dependencies.
func (m *OSSK8sEventTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8saudit.ResourceUIDPatternFinderTaskID.Ref(),
	}
}

// GroupedLogTask returns the task providing grouped logs.
func (m *OSSK8sEventTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return ossk8s.OSSK8sEventLogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps a single event log to its resource timeline and matches any resource UIDs in the message.
func (m *OSSK8sEventTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	event, err := ossk8s.ExtractOSSK8sEvent(l.NodeReader)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to get OSS k8s event fieldset: %w", err)
	}

	primaryResourcePath := k8saudit.MustResourceTimeline(ctx, "cluster", event.ResourceIdentity())
	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(primaryResourcePath)

	if event.Message != "" {
		finder := coretask.GetTaskResult(ctx, k8saudit.ResourceUIDPatternFinderTaskID.Ref())
		if finder != nil {
			matches := patternfinder.FindAllWithStarterRunes(event.Message, finder, true, k8saudit.EventMessageUIDStarterRunes...)
			for _, match := range matches {
				matchedPath := k8saudit.MustResourceTimeline(ctx, "cluster", match.Value)
				cs.AddEvent(matchedPath)
			}
		}
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*OSSK8sEventTimelineMapper)(nil)

// OSSK8sEventLogToTimelineMapperTask is the log to timeline mapper task.
var OSSK8sEventLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	ossk8s.OSSK8sEventLogToTimelineMapperTaskID,
	&OSSK8sEventTimelineMapper{},
	inspectioncore.FeatureTaskLabel(
		"OSS Kubernetes Event Logs",
		"Gather and parse Kubernetes event logs from OSS Kubernetes JSONL audit logs to visualize resource lifecycle and operational events.",
		2000,
		true,
	),
)
