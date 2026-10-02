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

package gcpcommon

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// GCPOperationLogIngester sets the log type, timestamp, severity and summary of GCP Operation audit logs.
type GCPOperationLogIngester struct {
	logType *pb.LogType
}

// NewGCPOperationLogIngester creates a new GCPOperationLogIngester.
func NewGCPOperationLogIngester(logType *pb.LogType) *GCPOperationLogIngester {
	return &GCPOperationLogIngester{
		logType: logType,
	}
}

// ProcessLog parses raw log entry and populates the LogChangeSet.
func (i *GCPOperationLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetTimestamp(l.Timestamp)

	if severity, err := ExtractGCPSeverity(l.NodeReader); err == nil && severity != nil {
		cs.SetSeverity(severity)
	} else {
		cs.SetSeverity(inspectioncore.SeverityUnknown)
	}

	cs.SetLogType(i.logType)

	audit, err := ExtractGCPAuditLog(l.NodeReader)
	if err != nil {
		return nil, err
	}

	var summary string
	switch {
	// Status defaults to -1 when protoPayload.status.code is omitted in log entries.
	// A value greater than 0 represents an explicit error code.
	case audit.Status > 0:
		summary = fmt.Sprintf("Failed: [%d: %s] %s", audit.Status, audit.StatusMessage, audit.MethodName)
	case audit.OperationLast:
		summary = fmt.Sprintf("Succeeded: %s", audit.MethodName)
	case audit.OperationFirst:
		summary = fmt.Sprintf("Start: %s", audit.MethodName)
	default:
		summary = audit.MethodName
	}
	cs.SetSummary(summary)

	return cs, nil
}

// DefineGCPOperationLogIngesterTask defines a log ingester task for GCP Operation audit logs read from rawLogTask.
func DefineGCPOperationLogIngesterTask(taskID taskid.TaskImplementationID[struct{}], rawLogTask taskid.TaskReference[[]*log.Log], logType *pb.LogType) coretask.DefinedTask[struct{}] {
	ingester := NewGCPOperationLogIngester(logType)
	return inspectiontaskbase.DefineLogIngesterTask(taskID, rawLogTask, func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		return ingester.ProcessLog
	})
}
