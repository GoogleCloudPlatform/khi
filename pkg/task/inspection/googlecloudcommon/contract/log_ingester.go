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

package googlecloudcommon_contract

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GCPOperationLogIngester is a common LogIngesterV2 implementation for GCP Operation audit logs.
type GCPOperationLogIngester struct {
	rawLogTask taskid.TaskReference[[]*log.Log]
	logType    *pb.LogType
}

// NewGCPOperationLogIngester creates a new GCPOperationLogIngester.
func NewGCPOperationLogIngester(rawLogTask taskid.TaskReference[[]*log.Log], logType *pb.LogType) inspectiontaskbase.LogIngesterV2 {
	return &GCPOperationLogIngester{
		rawLogTask: rawLogTask,
		logType:    logType,
	}
}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *GCPOperationLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return i.rawLogTask
}

// Dependencies returns additional task dependencies of the ingester.
func (i *GCPOperationLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog parses raw log entry and populates the LogChangeSet.
func (i *GCPOperationLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
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

	cs.SetLogType(i.logType)

	audit, err := log.GetFieldSet(l, &GCPAuditLogFieldSet{})
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

// Explicit interface compliance assertion.
var _ inspectiontaskbase.LogIngesterV2 = (*GCPOperationLogIngester)(nil)

// NewGCPOperationLogIngesterTask returns a new log ingester task for GCP Operation audit logs.
func NewGCPOperationLogIngesterTask(taskID taskid.TaskImplementationID[[]*log.Log], rawLogTask taskid.TaskReference[[]*log.Log], logType *pb.LogType) coretask.Task[[]*log.Log] {
	return inspectiontaskbase.NewLogIngesterTaskV2(taskID, NewGCPOperationLogIngester(rawLogTask, logType))
}
