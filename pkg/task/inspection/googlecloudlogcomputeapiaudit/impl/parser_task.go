// Copyright 2024 Google LLC
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

// Package googlecloudlogcomputeapiaudit_impl defines the implementation of the googlecloudlogcomputeapiaudit task.
package googlecloudlogcomputeapiaudit_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/legacyparser"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogcomputeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomputeapiaudit/contract"
)

// computeAPIParser is a parser for Compute API audit logs.
type computeAPIParser struct {
}

// TargetLogType implements parsertask.Parser.
func (c *computeAPIParser) TargetLogType() enum.LogType {
	return enum.LogTypeComputeApi
}

// Dependencies implements parsertask.Parser.
func (*computeAPIParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parsertask.Parser.
func (*computeAPIParser) Description() string {
	return `Gather Compute API audit logs to show the timings of the provisioning of resources(e.g creating/deleting GCE VM,mounting Persistent Disk...etc) on associated timelines.`
}

// GetParserName implements parsertask.Parser.
func (*computeAPIParser) GetParserName() string {
	return `Compute API Logs`
}

// LogTask implements parsertask.Parser.
func (*computeAPIParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogcomputeapiaudit_contract.ComputeAPIQueryTaskID.Ref()
}
func (*computeAPIParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parsertask.Parser.
func (*computeAPIParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	commonLogFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return err
	}
	isFirst := l.Has("operation.first")
	isLast := l.Has("operation.last")
	operationId := l.ReadStringOrDefault("operation.id", "unknown")
	methodName := l.ReadStringOrDefault("protoPayload.methodName", "unknown")
	methodNameSplitted := strings.Split(methodName, ".")
	resourceName := l.ReadStringOrDefault("protoPayload.resourceName", "unknown")
	resourceNameSplitted := strings.Split(resourceName, "/")
	instanceName := resourceNameSplitted[len(resourceNameSplitted)-1]
	principal := l.ReadStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")
	nodeResourcePath := resourcepath.Node(instanceName)
	// If this was an operation, it will be recorded as operation data
	if !(isLast && isFirst) && (isLast || isFirst) {
		state := enum.RevisionStateOperationStarted
		verb := enum.RevisionVerbOperationStart
		if isLast {
			state = enum.RevisionStateOperationFinished
			verb = enum.RevisionVerbOperationFinish
		}
		requestBodyRaw, _ := l.Serialize("protoPayload.request", &structured.YAMLNodeSerializer{}) // ignore the error to set the empty body when the field is not available in the log.
		operationPath := resourcepath.Operation(nodeResourcePath, methodNameSplitted[len(methodNameSplitted)-1], operationId)
		cs.RecordRevision(operationPath, &history.StagingResourceRevision{
			Body:       string(requestBodyRaw),
			Verb:       verb,
			State:      state,
			Requestor:  principal,
			ChangeTime: commonLogFieldSet.Timestamp,
			Partial:    false,
		})
	}
	cs.RecordEvent(nodeResourcePath)

	switch {
	case isFirst && !isLast:
		cs.RecordLogSummary(fmt.Sprintf("%s Started", methodName))
	case !isFirst && isLast:
		cs.RecordLogSummary(fmt.Sprintf("%s Finished", methodName))
	default:
		cs.RecordLogSummary(methodName)
	}

	return nil
}

var _ legacyparser.Parser = (*computeAPIParser)(nil)

// ComputeAPIParserTask is the parser task for compute API logs.
var ComputeAPIParserTask = legacyparser.NewParserTaskFromParser(googlecloudlogcomputeapiaudit_contract.ComputeAPIParserTaskID, &computeAPIParser{}, 6000, true, googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes)
