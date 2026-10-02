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

package serialport_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/serialport"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// logFilterTask removes logs with empty message.
// This message is mostly just contained escape sequences and stripped by ANSIEscapeSequenceStripper.
var logFilterTask = inspectiontaskbase.DefineLogFilterTask(
	serialport.LogFilterTaskID,
	serialport.LogQueryTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogFilterFunc {
		return func(ctx context.Context, l *log.Log) bool {
			msg, err := serialport.ExtractGCESerialPortMessage(l.NodeReader)
			if err != nil {
				return false
			}
			return msg != ""
		}
	},
)

// processSerialPortLog sets the log type, timestamp, severity and summary of a serial port log.
func processSerialPortLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(serialport.LogTypeSerialPort)
	cs.SetTimestamp(l.Timestamp)

	if severity, err := gcpcommon.ExtractGCPSeverity(l.NodeReader); err == nil {
		cs.SetSeverity(severity)
	}

	if serialFS, err := serialport.ExtractGCESerialPortLog(l.NodeReader); err == nil {
		cs.SetSummary(serialFS.Message)
	}

	return cs, nil
}

// logIngesterTask ingests the metadata of the filtered serial port logs.
var logIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	serialport.LogIngesterTaskID,
	serialport.LogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		return processSerialPortLog
	},
)

// logGrouperTask is the grouper task for GCE serial port logs.
// It groups logs by the node name and port name.
var logGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	serialport.LogGrouperTaskID,
	serialport.LogFilterTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return func(ctx context.Context, l *log.Log) string {
			serialFS, err := serialport.ExtractGCESerialPortLog(l.NodeReader)
			if err != nil {
				return ""
			}
			return fmt.Sprintf("%s#%s", serialFS.NodeName, serialFS.Port)
		}
	},
)

// serialPortLogToTimelineMapper maps logs to hierarchical node serial port timelines.
type serialPortLogToTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
}

// ProcessLogByGroup processes each log inside the group and stages the event on the timeline.
func (s *serialPortLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs, err := mapSerialPortLog(ctx, l, s.clusterIdentity.Get(ctx).ClusterName)
	return cs, struct{}{}, err
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*serialPortLogToTimelineMapper)(nil)

// mapSerialPortLog adds an event for a serial port log to the serial port timeline of its node in the given cluster.
func mapSerialPortLog(ctx context.Context, l *log.Log, clusterName string) (*khifilev6.TimelineChangeSet, error) {
	serialportFieldSet, err := serialport.ExtractGCESerialPortLog(l.NodeReader)
	if err != nil {
		return nil, err
	}

	targetPath := serialport.MustSerialPortTimeline(
		ctx,
		clusterName,
		serialportFieldSet.NodeName,
		serialportFieldSet.Port,
	)

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(targetPath)

	return cs, nil
}

// logToTimelineMapperTask is the timeline mapper task.
var logToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	serialport.LogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: serialport.LogIngesterTaskID.Ref(),
		GroupedLogs: serialport.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &serialPortLogToTimelineMapper{
			clusterIdentity: coretask.Use(b, k8scommon.ClusterIdentityTaskID.Ref()),
		}
	},
	inspectioncore.FeatureTaskLabel(
		"GCE Node Serial Port Logs",
		`Gather serial port logs from GCE instances to troubleshoot VM bootstrapping and OS initialization issues.`,
		10000,
		false,
	),
)
