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
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// CSMTrafficDirectorFieldSetReaderTask is a task that reads and parses field sets from CSM Traffic Director logs.
var CSMTrafficDirectorFieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudlogcsm_contract.CSMTrafficDirectorFieldSetReaderTaskID,
	googlecloudlogcsm_contract.ListCSMTrafficDirectorLogEntriesTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
	},
)

// CSMTrafficDirectorLogIngesterTask is a task that ingests CSM Traffic Director logs into the history builder.
var CSMTrafficDirectorLogIngesterTask = inspectiontaskbase.NewLogIngesterTask(
	googlecloudlogcsm_contract.CSMTrafficDirectorLogIngesterTaskID,
	googlecloudlogcsm_contract.ListCSMTrafficDirectorLogEntriesTaskID.Ref(),
)

// CSMTrafficDirectorLogGrouperTask is a task that groups CSM Traffic Director logs by their resource path.
var CSMTrafficDirectorLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogcsm_contract.CSMTrafficDirectorLogGrouperTaskID,
	googlecloudlogcsm_contract.CSMTrafficDirectorFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
		if err != nil {
			return "unknown"
		}
		return resourcepath.GCPResource(audit.ResourceName).Path
	},
)

// LogToTimelineMapperTask is a task that maps CSM Traffic Director logs to resource timelines.
var CSMTrafficDirectorLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[*googlecloudcommon_contract.GCPOperationStateTracker](
	googlecloudlogcsm_contract.CSMTrafficDirectorLogToTimelineMapperTaskID,
	&csmTrafficDirectorLogToTimelineMapperSetting{},
	inspectioncore_contract.FeatureTaskLabel(
		"CSM Resource Audit Logs",
		"Gather audit logs related to TrafficDirector resources created by the TD based CSM. Map them into pseudo timelines with other Kubernetes logs.",
		enum.LogTypeCSMAccessLog, // TODO: Using the access log type here without defining a new log type. No new log type will be added before merging the file schema v6 change not to cause unnecessary file incompatibility issue.
		10100,
		false,
	),
)

type csmTrafficDirectorLogToTimelineMapperSetting struct{}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (s *csmTrafficDirectorLogToTimelineMapperSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (s *csmTrafficDirectorLogToTimelineMapperSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogcsm_contract.CSMTrafficDirectorLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (s *csmTrafficDirectorLogToTimelineMapperSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcsm_contract.CSMTrafficDirectorLogIngesterTaskID.Ref()
}

// InitializeGroupData implements inspectiontaskbase.LogToTimelineMapper.
func (s *csmTrafficDirectorLogToTimelineMapperSetting) InitializeGroupData(ctx context.Context, groupName string) (*googlecloudcommon_contract.GCPOperationStateTracker, error) {
	return googlecloudcommon_contract.NewGCPOperationStateTracker(), nil
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (s *csmTrafficDirectorLogToTimelineMapperSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, tracker *googlecloudcommon_contract.GCPOperationStateTracker) (*googlecloudcommon_contract.GCPOperationStateTracker, error) {
	if tracker == nil {
		tracker = googlecloudcommon_contract.NewGCPOperationStateTracker()
	}
	commonLogFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return tracker, err
	}
	audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if err != nil {
		return tracker, err
	}

	methodNameParts := strings.Split(audit.MethodName, ".")
	shortMethodName := methodNameParts[len(methodNameParts)-1]
	verb := audit.GuessRevisionVerb()

	resourcePath := resourcepath.GCPResource(audit.ResourceName)
	operationPath := audit.OperationPath(resourcePath)

	if !audit.ImmediateOperation() {
		state := enum.RevisionStateOperationStarted
		opVerb := enum.RevisionVerbOperationStart
		if audit.Ending() {
			state = enum.RevisionStateOperationFinished
			opVerb = enum.RevisionVerbOperationFinish
		}
		requestBody, _ := audit.RequestString()
		cs.AddRevision(operationPath, &history.StagingResourceRevision{
			Body:       requestBody,
			Verb:       opVerb,
			State:      state,
			Requestor:  audit.PrincipalEmail,
			ChangeTime: commonLogFieldSet.Timestamp,
			Partial:    false,
		})
	}

	manifest, shouldUpdate := tracker.TrackAndGetManifest(audit)
	if shouldUpdate {
		switch {
		case verb == enum.RevisionVerbDelete:
			cs.AddRevision(resourcePath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbDelete,
				State:      enum.RevisionStateDeleted,
				Requestor:  audit.PrincipalEmail,
				ChangeTime: commonLogFieldSet.Timestamp,
				Partial:    false,
			})
		case audit.ImmediateOperation():
			cs.AddEvent(resourcePath)
		default:
			cs.AddRevision(resourcePath, &history.StagingResourceRevision{
				Body:       manifest,
				Verb:       verb,
				State:      enum.RevisionStateExisting,
				Requestor:  audit.PrincipalEmail,
				ChangeTime: commonLogFieldSet.Timestamp,
				Partial:    false,
			})
		}
	}

	switch {
	case audit.Starting():
		cs.SetLogSummary(fmt.Sprintf("%s Started", shortMethodName))
	case audit.Ending():
		cs.SetLogSummary(fmt.Sprintf("%s Finished", shortMethodName))
	default:
		cs.SetLogSummary(shortMethodName)
	}

	return tracker, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[*googlecloudcommon_contract.GCPOperationStateTracker] = (*csmTrafficDirectorLogToTimelineMapperSetting)(nil)
