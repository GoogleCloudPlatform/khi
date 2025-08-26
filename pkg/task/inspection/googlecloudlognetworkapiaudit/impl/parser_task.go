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

package googlecloudlognetworkapiaudit_impl

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
	googlecloudlognetworkapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlognetworkapiaudit/contract"
	"gopkg.in/yaml.v3"
)

type negAttachOrDetachRequestEndpoint struct {
	Instance  string `yaml:"instance"`
	IpAddress string `yaml:"ipAddress"`
	Port      string `yaml:"port"`
}

type negAttachOrDetachRequest struct {
	NetworkEndpoints []*negAttachOrDetachRequestEndpoint `yaml:"networkEndpoints"`
}

type gceNetworkParser struct{}

// TargetLogType implements parsertask.Parser.
func (g *gceNetworkParser) TargetLogType() enum.LogType {
	return enum.LogTypeNetworkAPI
}

// Dependencies implements parsertask.Parser.
func (*gceNetworkParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parsertask.Parser.
func (*gceNetworkParser) Description() string {
	return `Gather GCE Network API logs to visualize statuses of Network Endpoint Groups(NEG)`
}

// GetParserName implements parsertask.Parser.
func (*gceNetworkParser) GetParserName() string {
	return "GCE Network Logs"
}

// LogTask implements parsertask.Parser.
func (*gceNetworkParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlognetworkapiaudit_contract.NetworkAPIQueryTaskID.Ref()
}

func (*gceNetworkParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parsertask.Parser.
func (*gceNetworkParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	commonFieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
	isFirst := l.Has("operation.first")
	isLast := l.Has("operation.last")
	operationId := l.ReadStringOrDefault("operation.id", "unknown")
	methodName := l.ReadStringOrDefault("protoPayload.methodName", "unknown")
	methodNameSplitted := strings.Split(methodName, ".")
	resourceName := l.ReadStringOrDefault("protoPayload.resourceName", "unknown")
	resourceNameSplitted := strings.Split(resourceName, "/")
	negName := resourceNameSplitted[len(resourceNameSplitted)-1]
	principal := l.ReadStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")
	var negResourcePath resourcepath.ResourcePath
	lease, err := builder.ClusterResource.NEGs.GetResourceLeaseHolderAt(negName, commonFieldSet.Timestamp)
	if err == nil {
		negResourcePath = resourcepath.NetworkEndpointGroup(lease.Holder.Namespace, negName)
	} else {
		negResourcePath = resourcepath.NetworkEndpointGroup("unknown", negName)
	}

	switch {
	case !(isLast && isFirst) && (isLast || isFirst):
		state := enum.RevisionStateOperationStarted
		if isLast {
			state = enum.RevisionStateOperationFinished
		}
		operationPath := resourcepath.Operation(negResourcePath, methodNameSplitted[len(methodNameSplitted)-1], operationId)
		cs.RecordRevision(operationPath, &history.StagingResourceRevision{
			Verb:       enum.RevisionVerbCreate,
			State:      state,
			Requestor:  principal,
			ChangeTime: commonFieldSet.Timestamp,
			Partial:    false,
		})
	default:
		cs.RecordEvent(negResourcePath)
	}
	if isFirst {
		method := methodNameSplitted[len(methodNameSplitted)-1]
		if method == "detachNetworkEndpoints" || method == "attachNetworkEndpoints" {
			isDetach := strings.HasPrefix(method, "detach")
			requestBody, err := l.Serialize("protoPayload.request", &structured.YAMLNodeSerializer{})
			if err != nil {
				return err
			}
			var negRequest negAttachOrDetachRequest
			err = yaml.Unmarshal([]byte(requestBody), &negRequest)
			if err != nil {
				return err
			}
			for _, endpoint := range negRequest.NetworkEndpoints {
				lease, err := builder.ClusterResource.IPs.GetResourceLeaseHolderAt(endpoint.IpAddress, commonFieldSet.Timestamp)
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("Failed to identify the holder of the IP %s.\n This might be because the IP holder resource wasn't updated during the log period ", endpoint.IpAddress))
					continue
				}
				holder := lease.Holder
				if holder.Kind == "pod" {
					podPath := resourcepath.Pod(holder.Namespace, holder.Name)
					negSubresourcePath := resourcepath.NetworkEndpointGroupUnderResource(podPath, holder.Namespace, negName)
					state := enum.RevisionStateConditionTrue
					verb := enum.RevisionVerbReady
					if isDetach {
						state = enum.RevisionStateConditionFalse
						verb = enum.RevisionVerbNonReady
					}
					cs.RecordRevision(negSubresourcePath, &history.StagingResourceRevision{
						Verb:       verb,
						State:      state,
						Requestor:  principal,
						ChangeTime: commonFieldSet.Timestamp,
						Partial:    false,
					})
				}
			}
		}
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

var _ legacyparser.Parser = (*gceNetworkParser)(nil)

// NetworkAPIParserTask is the parser task for network API audit logs.
var NetworkAPIParserTask = legacyparser.NewParserTaskFromParser(googlecloudlognetworkapiaudit_contract.NetworkAPIParserTaskID, &gceNetworkParser{}, 7000, true, googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes)
