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

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/legacyparser"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/inspectiontype"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/multicloud_api/multicloud_api_taskid"
)

type multiCloudAuditLogParser struct {
}

// TargetLogType implements parsertask.Parser.
func (m *multiCloudAuditLogParser) TargetLogType() enum.LogType {
	return enum.LogTypeMulticloudAPI
}

// Dependencies implements parsertask.Parser.
func (*multiCloudAuditLogParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parsertask.Parser.
func (*multiCloudAuditLogParser) Description() string {
	return `Gather Anthos Multicloud audit log including cluster creation,deletion and upgrades.`
}

// GetParserName implements parsertask.Parser.
func (*multiCloudAuditLogParser) GetParserName() string {
	return `MultiCloud API logs`
}

// LogTask implements parsertask.Parser.
func (*multiCloudAuditLogParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return multicloud_api_taskid.MultiCloudAPIQueryTaskID.Ref()
}

func (*multiCloudAuditLogParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parsertask.Parser.
func (*multiCloudAuditLogParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	resourceName := l.ReadStringOrDefault("protoPayload.resourceName", "")
	resource := parseResourceNameOfMulticloudAPI(resourceName)
	isFirst := l.Has("operation.first")
	isLast := l.Has("operation.last")
	operationId := l.ReadStringOrDefault("operation.id", "unknown")
	methodName := l.ReadStringOrDefault("protoPayload.methodName", "unknown")
	principal := l.ReadStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")
	code := l.ReadStringOrDefault("protoPayload.status.code", "0")
	commonFieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
	isSucceedRequest := code == "0"

	var operationResourcePath resourcepath.ResourcePath

	if resource.NodepoolName == "" {
		// assume this is a cluster operation
		clusterResourcePath := resourcepath.Cluster(resource.ClusterName)
		if filterMethodNameOperation(methodName, "Create", "Cluster") && isFirst && isSucceedRequest {
			// Cluster info is stored at protoPayload.request.(aws|azure)Cluster
			bodyRaw, err := l.Serialize(fmt.Sprintf("protoPayload.request.%sCluster", resource.ClusterType), &structured.YAMLNodeSerializer{})
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Failed to get the cluster info from the log\n%v", err))
			}
			cs.RecordRevision(clusterResourcePath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbCreate,
				State:      enum.RevisionStateExisting,
				Requestor:  principal,
				ChangeTime: commonFieldSet.Timestamp,
				Partial:    false,
				Body:       string(bodyRaw),
			})
		}
		if filterMethodNameOperation(methodName, "Delete", "Cluster") && isFirst && isSucceedRequest {
			cs.RecordRevision(clusterResourcePath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbDelete,
				State:      enum.RevisionStateDeleted,
				Requestor:  principal,
				ChangeTime: commonFieldSet.Timestamp,
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
			bodyRaw, err := l.Serialize(fmt.Sprintf("protoPayload.request.%sNodePool", resource.ClusterType), &structured.YAMLNodeSerializer{})
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Failed to get the nodepool info from the log\n%v", err))
			}
			cs.RecordRevision(nodepoolResourcePath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbCreate,
				State:      enum.RevisionStateExisting,
				Requestor:  principal,
				ChangeTime: commonFieldSet.Timestamp,
				Partial:    false,
				Body:       string(bodyRaw),
			})
		}
		if filterMethodNameOperation(methodName, "Delete", "NodePool") && isFirst && isSucceedRequest {
			cs.RecordRevision(nodepoolResourcePath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbDelete,
				State:      enum.RevisionStateDeleted,
				Requestor:  principal,
				ChangeTime: commonFieldSet.Timestamp,
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
			ChangeTime: commonFieldSet.Timestamp,
			Partial:    false,
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

var _ legacyparser.Parser = (*multiCloudAuditLogParser)(nil)

var MultiCloudAuditLogParseJob = legacyparser.NewParserTaskFromParser(multicloud_api_taskid.MultiCloudAPIParserTaskID, &multiCloudAuditLogParser{}, true, inspectiontype.GKEMultiCloudClusterInspectionTypes)

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
