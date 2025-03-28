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

package onprem_api

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
	baremetal "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gdcv-for-baremetal"
	vmware "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gdcv-for-vmware"
	multicloud_api_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/onprem_api/taskid"
	onprem_api_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/onprem_api/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type onpremCloudAuditLogParser struct {
}

// TargetLogType implements parser.Parser.
func (o *onpremCloudAuditLogParser) TargetLogType() enum.LogType {
	return enum.LogTypeOnPremAPI
}

// Dependencies implements parser.Parser.
func (*onpremCloudAuditLogParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parser.Parser.
func (*onpremCloudAuditLogParser) Description() string {
	return `Gather Anthos OnPrem audit log including cluster creation,deletion,enroll,unenroll and upgrades.`
}

// GetParserName implements parser.Parser.
func (*onpremCloudAuditLogParser) GetParserName() string {
	return `OnPrem API logs`
}

// LogTask implements parser.Parser.
func (*onpremCloudAuditLogParser) LogTask() taskid.TaskReference[[]*log.LogEntity] {
	return multicloud_api_taskid.OnPremCloudAPIQueryTaskID.GetTaskReference()
}

func (*onpremCloudAuditLogParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parser.Parser.
func (*onpremCloudAuditLogParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {
	resourceName := l.GetStringOrDefault("protoPayload.resourceName", "")
	resource := parseResourceNameOfOnPremAPI(resourceName)
	isFirst := l.Has("operation.first")
	isLast := l.Has("operation.last")
	operationId := l.GetStringOrDefault("operation.id", "unknown")
	methodName := l.GetStringOrDefault("protoPayload.methodName", "unknown")
	principal := l.GetStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")
	code := l.GetStringOrDefault("protoPayload.status.code", "0")
	isSucceedRequest := code == "0"
	var operationResourcePath resourcepath.ResourcePath
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
		if filterMethodNameOperation(methodName, "Enroll", "Cluster") && !isFirst && isSucceedRequest {
			// Cluster info is stored at protoPayload.request.(aws|azure)Cluster
			body, err := l.GetChildYamlOf("protoPayload.response")
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
		if filterMethodNameOperation(methodName, "Unenroll", "Cluster") && !isFirst && isSucceedRequest {
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
			body, err := l.GetChildYamlOf("protoPayload.request")
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

var _ parser.Parser = (*onpremCloudAuditLogParser)(nil)

var OnPremCloudAuditLogParseTask = parser.NewParserTaskFromParser(onprem_api_taskid.OnPremCloudAPIParserTaskID, &onpremCloudAuditLogParser{}, true, inspection_task.InspectionTypeLabel(baremetal.InspectionTypeId, vmware.InspectionTypeId))

type onpremResource struct {
	ClusterType  string // aws or azure
	ClusterName  string
	NodepoolName string
}

func parseResourceNameOfOnPremAPI(resourceName string) *onpremResource {
	// resourceName should be in the format of
	// projects/<PROJECT_NUMBER>/locations/<LOCATION>/(baremetalAdmin|baremetalStandalone|baremetal|vmware|vmwareAdmin)Clusters/<CLUSTER_NAME>(/(baremetalAdmin|baremetalStandalone|baremetal|vmware|vmwareAdmin)NodePools/<NODEPOOL_NAME>)
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
	return &onpremResource{
		ClusterName:  clusterName,
		NodepoolName: nodepoolName,
		ClusterType:  clusterType,
	}
}

func filterMethodNameOperation(methodName string, operation string, operand string) bool {
	clusterTypes := []string{
		"baremetalAdmin",
		"baremetalStandalone",
		"baremetal",
		"vmware",
		"vmwareAdmin",
	}
	methodNameLower := strings.ToLower(methodName)
	operationLower := strings.ToLower(operation)
	operandLower := strings.ToLower(operand)
	for _, clusterType := range clusterTypes {
		if strings.Contains(methodNameLower, fmt.Sprintf("%s%s%s", operationLower, strings.ToLower(clusterType), operandLower)) {
			return true
		}
	}
	return false
}
