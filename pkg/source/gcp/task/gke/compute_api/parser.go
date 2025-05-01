// Copyright 2024 Google LLC
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

package compute_api

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/inspectiontype"
	gke_compute_api_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/compute_api/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type computeAPIParser struct {
}

// TargetLogType implements parser.Parser.
func (c *computeAPIParser) TargetLogType() enum.LogType {
	return enum.LogTypeComputeApi
}

// Dependencies implements parser.Parser.
func (*computeAPIParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parser.Parser.
func (*computeAPIParser) Description() string {
	return `Gather Compute API audit logs to show the timings of the provisioning of resources(e.g creating/deleting GCE VM,mounting Persistent Disk...etc) on associated timelines.`
}

// GetParserName implements parser.Parser.
func (*computeAPIParser) GetParserName() string {
	return `Compute API Logs`
}

// LogTask implements parser.Parser.
func (*computeAPIParser) LogTask() taskid.TaskReference[[]*log.LogEntity] {
	return gke_compute_api_taskid.ComputeAPIQueryTaskID.Ref()
}
func (*computeAPIParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parser.Parser.
func (*computeAPIParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {
	isFirst := l.Has("operation.first")
	isLast := l.Has("operation.last")
	operationId := l.GetStringOrDefault("operation.id", "unknown")
	methodName := l.GetStringOrDefault("protoPayload.methodName", "unknown")
	methodNameSplitted := strings.Split(methodName, ".")
	resourceName := l.GetStringOrDefault("protoPayload.resourceName", "unknown")
	resourceNameSplitted := strings.Split(resourceName, "/")
	instanceName := resourceNameSplitted[len(resourceNameSplitted)-1]
	principal := l.GetStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")
	nodeResourcePath := resourcepath.Node(instanceName)
	// If this was an operation, it will be recorded as operation data
	if !(isLast && isFirst) && (isLast || isFirst) {
		state := enum.RevisionStateOperationStarted
		verb := enum.RevisionVerbOperationStart
		if isLast {
			state = enum.RevisionStateOperationFinished
			verb = enum.RevisionVerbOperationFinish
		}
		requestBody, _ := l.GetChildYamlOf("protoPayload.request") // ignore the error to set the empty body when the field is not available in the log.
		operationPath := resourcepath.Operation(nodeResourcePath, methodNameSplitted[len(methodNameSplitted)-1], operationId)
		cs.RecordRevision(operationPath, &history.StagingResourceRevision{
			Body:       requestBody,
			Verb:       verb,
			State:      state,
			Requestor:  principal,
			ChangeTime: l.Timestamp(),
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

var _ parser.Parser = (*computeAPIParser)(nil)

var ComputeAPIParserTask = parser.NewParserTaskFromParser(gke_compute_api_taskid.ComputeAPIParserTaskID, &computeAPIParser{}, true, inspectiontype.GKEBasedClusterInspectionTypes)
