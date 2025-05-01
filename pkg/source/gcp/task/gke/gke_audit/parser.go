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

package gke_audit

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
	gke_audit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/gke_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type gkeAuditLogParser struct {
}

// TargetLogType implements parser.Parser.
func (p *gkeAuditLogParser) TargetLogType() enum.LogType {
	return enum.LogTypeGkeAudit
}

// Dependencies implements parser.Parser.
func (*gkeAuditLogParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parser.Parser.
func (*gkeAuditLogParser) Description() string {
	return `Gather GKE audit log to show creation/upgrade/deletion of logs cluster/nodepool`
}

// GetParserName implements parser.Parser.
func (*gkeAuditLogParser) GetParserName() string {
	return `GKE Audit logs`
}

// LogTask implements parser.Parser.
func (*gkeAuditLogParser) LogTask() taskid.TaskReference[[]*log.LogEntity] {
	return gke_audit_taskid.GKEAuditLogQueryTaskID.Ref()
}

func (*gkeAuditLogParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parser.Parser.
func (p *gkeAuditLogParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {
	clusterName := l.GetStringOrDefault("resource.labels.cluster_name", "unknown")
	isFirst := l.Has("operation.first")
	isLast := l.Has("operation.last")
	operationId := l.GetStringOrDefault("operation.id", "unknown")
	methodName := l.GetStringOrDefault("protoPayload.methodName", "unknown")
	principal := l.GetStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")
	statusCode := l.GetIntOrDefault("protoPayload.status.code", 0)
	shouldRecordResourceRevision := statusCode == 0
	var operationResourcePath resourcepath.ResourcePath

	nodepoolName, err := getRelatedNodepool(l)
	if err != nil {
		// assume this is a cluster operation
		clusterResourcePath := resourcepath.Cluster(clusterName)
		if shouldRecordResourceRevision {
			if strings.HasSuffix(methodName, "CreateCluster") {
				body, _ := l.GetChildYamlOf("protoPayload.request.cluster") // Ignore the error and use "" as the body of the cluster setting when the field is not available.
				state := enum.RevisionStateExisting
				if isFirst {
					state = enum.RevisionStateProvisioning
				}
				cs.RecordRevision(clusterResourcePath, &history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      state,
					Requestor:  principal,
					ChangeTime: l.Timestamp(),
					Partial:    false,
					Body:       body,
				})
			}
			if strings.HasSuffix(methodName, "DeleteCluster") {
				state := enum.RevisionStateDeleted
				if isFirst {
					state = enum.RevisionStateDeleting
				}
				cs.RecordRevision(clusterResourcePath, &history.StagingResourceRevision{
					Verb:       enum.RevisionVerbDelete,
					State:      state,
					Requestor:  principal,
					ChangeTime: l.Timestamp(),
					Partial:    false,
					Body:       "",
				})
			}
		}

		methodNameSplitted := strings.Split(methodName, ".")
		methodVerb := methodNameSplitted[len(methodNameSplitted)-1]
		operationResourcePath = resourcepath.Operation(clusterResourcePath, methodVerb, operationId)

		cs.RecordEvent(clusterResourcePath)
	} else {
		nodepoolResourcePath := resourcepath.Nodepool(clusterName, nodepoolName)
		if shouldRecordResourceRevision {
			if strings.HasSuffix(methodName, "CreateNodePool") {
				body, _ := l.GetChildYamlOf("protoPayload.request.nodePool") // Ignore the error and use "" as the body of the nodepool setting when the field is not available.
				state := enum.RevisionStateExisting
				if isFirst {
					state = enum.RevisionStateProvisioning
				}
				cs.RecordRevision(nodepoolResourcePath, &history.StagingResourceRevision{
					Verb:       enum.RevisionVerbCreate,
					State:      state,
					Requestor:  principal,
					ChangeTime: l.Timestamp(),
					Partial:    false,
					Body:       body,
				})
			}
			if strings.HasSuffix(methodName, "DeleteNodePool") {
				state := enum.RevisionStateDeleted
				if isFirst {
					state = enum.RevisionStateDeleting
				}
				cs.RecordRevision(nodepoolResourcePath, &history.StagingResourceRevision{
					Verb:       enum.RevisionVerbDelete,
					State:      state,
					Requestor:  principal,
					ChangeTime: l.Timestamp(),
					Partial:    false,
					Body:       "",
				})
			}
		}
		cs.RecordEvent(nodepoolResourcePath)
		methodNameSplitted := strings.Split(methodName, ".")
		methodVerb := methodNameSplitted[len(methodNameSplitted)-1]
		operationResourcePath = resourcepath.Operation(nodepoolResourcePath, methodVerb, operationId)
	}

	// If this was an operation, it will be recorded as operation data
	if !(isLast && isFirst) && (isLast || isFirst) && shouldRecordResourceRevision {
		requestBody, _ := l.GetChildYamlOf("protoPayload.request") // ignore the error to set the empty body when the field is not available in the log.
		state := enum.RevisionStateOperationStarted
		verb := enum.RevisionVerbOperationStart
		if isLast {
			state = enum.RevisionStateOperationFinished
			verb = enum.RevisionVerbOperationFinish
		}
		cs.RecordRevision(operationResourcePath, &history.StagingResourceRevision{
			Verb:       verb,
			State:      state,
			Requestor:  principal,
			ChangeTime: l.Timestamp(),
			Partial:    false,
			Body:       requestBody,
		})
	}

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

func getRelatedNodepool(l *log.LogEntity) (string, error) {
	nodepoolName, err := l.GetString("resource.labels.nodepool_name")
	if err == nil {
		return nodepoolName, nil
	}
	return l.GetString("protoPayload.request.update.desiredNodePoolId")
}

var _ parser.Parser = (*gkeAuditLogParser)(nil)

var GKEAuditLogParseJob = parser.NewParserTaskFromParser(gke_audit_taskid.GKEAuditParserTaskID, &gkeAuditLogParser{}, true, inspectiontype.GKEBasedClusterInspectionTypes)
