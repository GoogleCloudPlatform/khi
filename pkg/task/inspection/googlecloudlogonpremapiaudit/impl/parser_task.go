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

package googlecloudlogonpremapiaudit_impl

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
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogonpremapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogonpremapiaudit/contract"
)

type onpremCloudAuditLogParser struct {
}

// TargetLogType implements parsertask.Parser.
func (o *onpremCloudAuditLogParser) TargetLogType() enum.LogType {
	return enum.LogTypeOnPremAPI
}

// Dependencies implements parsertask.Parser.
func (*onpremCloudAuditLogParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parsertask.Parser.
func (*onpremCloudAuditLogParser) Description() string {
	return `Gather Anthos OnPrem audit log including cluster creation,deletion,enroll,unenroll and upgrades.`
}

// GetParserName implements parsertask.Parser.
func (*onpremCloudAuditLogParser) GetParserName() string {
	return `OnPrem API logs`
}

// LogTask implements parsertask.Parser.
func (*onpremCloudAuditLogParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogonpremapiaudit_contract.OnPremCloudAuditLogQueryTaskID.Ref()
}

func (*onpremCloudAuditLogParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parsertask.Parser.
func (*onpremCloudAuditLogParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	resourceName := l.ReadStringOrDefault("protoPayload.resourceName", "")
	resource := parseResourceNameOfOnPremAPI(resourceName)
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
		if filterMethodNameOperation(methodName, "Enroll", "Cluster") && !isFirst && isSucceedRequest {
			// Cluster info is stored at protoPayload.request.(aws|azure)Cluster
			bodyRaw, err := l.Serialize("protoPayload.response", &structured.YAMLNodeSerializer{})
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
		if filterMethodNameOperation(methodName, "Unenroll", "Cluster") && !isFirst && isSucceedRequest {
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
			bodyRaw, err := l.Serialize("protoPayload.request", &structured.YAMLNodeSerializer{})
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

var _ legacyparser.Parser = (*onpremCloudAuditLogParser)(nil)

var OnPremCloudAuditLogParseTask = legacyparser.NewParserTaskFromParser(googlecloudlogonpremapiaudit_contract.OnPremCloudAuditLogParseTaskID, &onpremCloudAuditLogParser{}, 5000, true, googlecloudinspectiontypegroup_contract.GDCClusterInspectionTypes)

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
