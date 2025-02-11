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

package multicloud_api

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	aws "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke-on-aws"
	azure "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke-on-azure"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

type multiCloudAuditLogParser struct {
}

// TargetLogType implements parser.Parser.
func (m *multiCloudAuditLogParser) TargetLogType() enum.LogType {
	return enum.LogTypeMulticloudAPI
}

// Dependencies implements parser.Parser.
func (*multiCloudAuditLogParser) Dependencies() []string {
	return []string{}
}

// Description implements parser.Parser.
func (*multiCloudAuditLogParser) Description() string {
	return `Gather Anthos Multicloud audit log including cluster creation,deletion and upgrades.`
}

// GetParserName implements parser.Parser.
func (*multiCloudAuditLogParser) GetParserName() string {
	return `MultiCloud API logs`
}

// LogTask implements parser.Parser.
func (*multiCloudAuditLogParser) LogTask() string {
	return MultiCloudAPIQueryTaskID
}

func (*multiCloudAuditLogParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parser.Parser.
func (*multiCloudAuditLogParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder, variables *task.VariableSet) error {
	resourceName := l.GetStringOrDefault("protoPayload.resourceName", "")
	resource := parseResourceNameOfMulticloudAPI(resourceName)
	isFirst := l.Has("operation.first")
	isLast := l.Has("operation.last")
	operationId := l.GetStringOrDefault("operation.id", "unknown")
	methodName := l.GetStringOrDefault("protoPayload.methodName", "unknown")
	principal := l.GetStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")
	code := l.GetStringOrDefault("protoPayload.status.code", "0")
	isSucceedRequest := code == "0"
	operationResourcePath := resourcepath.ResourcePath{}
	if resource.NodepoolName == "" {
		// assume this is a cluster operation
		clusterResourcePath := resourcepath.Cluster(resource.ClusterName)
		if filterMethodNameOperation(methodName, "Create", "Cluster") && isFirst && isSucceedRequest {
			// Cluster info is stored at protoPayload.request.(aws|azure)Cluster
			body, err := l.GetChildYamlOf(fmt.Sprintf("protoPayload.request.%sCluster", resource.ClusterType))
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Failed to get the cluster info from the log\n%v", err))
			}
			cs.RecordRevision(clusterResourcePath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbCreate,
				State:      enum.RevisionStateExisting,
				Requestor:  principal,
				ChangeTime: l.Timestamp(),
				Partial:    false,
				Body:       body,
			})
		}
		if filterMethodNameOperation(methodName, "Delete", "Cluster") && isFirst && isSucceedRequest {
			cs.RecordRevision(clusterResourcePath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbDelete,
				State:      enum.RevisionStateDeleted,
				Requestor:  principal,
				ChangeTime: l.Timestamp(),
				Partial:    false,
				Body:       "",
			})
		}
		methodNameSplitted := strings.Split(methodName, ".")
		methodVerb := methodNameSplitted[len(methodNameSplitted)-1]
		operationResourcePath = resourcepath.Operation(clusterResourcePath, methodVerb, operationId)
		cs.RecordEvent(clusterResourcePath)
	} else {
		nodepoolResourcePath := resourcepath.Nodepool(resource.ClusterName, resource.NodepoolName)
		if filterMethodNameOperation(methodName, "Create", "NodePool") && isFirst && isSucceedRequest {
			// NodePool info is stored at protoPayload.request.(aws|azure)NodePool
			body, err := l.GetChildYamlOf(fmt.Sprintf("protoPayload.request.%sNodePool", resource.ClusterType))
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Failed to get the nodepool info from the log\n%v", err))
			}
			cs.RecordRevision(nodepoolResourcePath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbCreate,
				State:      enum.RevisionStateExisting,
				Requestor:  principal,
				ChangeTime: l.Timestamp(),
				Partial:    false,
				Body:       body,
			})
		}
		if filterMethodNameOperation(methodName, "Delete", "NodePool") && isFirst && isSucceedRequest {
			cs.RecordRevision(nodepoolResourcePath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbDelete,
				State:      enum.RevisionStateDeleted,
				Requestor:  principal,
				ChangeTime: l.Timestamp(),
				Partial:    false,
				Body:       "",
			})
		}
		cs.RecordEvent(nodepoolResourcePath)
		methodNameSplitted := strings.Split(methodName, ".")
		methodVerb := methodNameSplitted[len(methodNameSplitted)-1]
		operationResourcePath = resourcepath.Operation(nodepoolResourcePath, methodVerb, operationId)
	}

	// If this was an operation, it will be recorded as operation data
	if !(isLast && isFirst) && (isLast || isFirst) {
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
		})
	}

	if isFirst && !isLast {
		cs.RecordLogSummary(fmt.Sprintf("%s Started", methodName))
	} else if !isFirst && isLast {
		cs.RecordLogSummary(fmt.Sprintf("%s Finished", methodName))
	} else {
		cs.RecordLogSummary(methodName)
	}
	return nil
}

var _ parser.Parser = (*multiCloudAuditLogParser)(nil)

var MultiCloudAuditLogParseJob = parser.NewParserTaskFromParser(gcp_task.GCPPrefix+"feature/multicloud-audit-parser", &multiCloudAuditLogParser{}, true, inspection_task.InspectionTypeLabel(aws.InspectionTypeId, azure.InspectionTypeId))

type multiCloudResource struct {
	ClusterType  string // aws or azure
	ClusterName  string
	NodepoolName string
}

func parseResourceNameOfMulticloudAPI(resourceName string) *multiCloudResource {
	// resourceName should be in the format of
	// projects/<PROJECT_NUMBER>/locations/<LOCATION>/(aws|azure)Clusters/<CLUSTER_NAME>(/(aws|azure)NodePools/<NODEPOOL_NAME>)
	splited := strings.Split(resourceName, "/")
	clusterName := "unknown"
	nodepoolName := ""
	clusterType := "unknown"
	if len(splited) > 5 {
		clusterName = splited[5]
	}
	if len(splited) > 7 {
		nodepoolName = splited[7]
	}
	if len(splited) > 4 {
		clusterType = strings.TrimSuffix(splited[4], "Clusters")
	}
	return &multiCloudResource{
		ClusterName:  clusterName,
		NodepoolName: nodepoolName,
		ClusterType:  clusterType,
	}
}

func filterMethodNameOperation(methodName string, operation string, operand string) bool {
	return strings.Contains(methodName, fmt.Sprintf("%sAws%s", operation, operand)) || strings.Contains(methodName, fmt.Sprintf("%sAzure%s", operation, operand))
}
