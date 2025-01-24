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

package network_api

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
	composer_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"gopkg.in/yaml.v3"
)

type gceNetworkParser struct{}

// Dependencies implements parser.Parser.
func (*gceNetworkParser) Dependencies() []string {
	return []string{}
}

// Description implements parser.Parser.
func (*gceNetworkParser) Description() string {
	return `GCE network API audit log including NEG related audit logs to identify when the associated NEG was attached/detached.`
}

// GetParserName implements parser.Parser.
func (*gceNetworkParser) GetParserName() string {
	return "GCE Network Logs"
}

// LogTask implements parser.Parser.
func (*gceNetworkParser) LogTask() string {
	return GCPNetworkLogQueryTaskID
}

func (*gceNetworkParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parser.Parser.
func (*gceNetworkParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder, variables *task.VariableSet) error {
	isFirst := l.Has("operation.first")
	isLast := l.Has("operation.last")
	operationId := l.GetStringOrDefault("operation.id", "unknown")
	methodName := l.GetStringOrDefault("protoPayload.methodName", "unknown")
	methodNameSplitted := strings.Split(methodName, ".")
	resourceName := l.GetStringOrDefault("protoPayload.resourceName", "unknown")
	resourceNameSplitted := strings.Split(resourceName, "/")
	negName := resourceNameSplitted[len(resourceNameSplitted)-1]
	principal := l.GetStringOrDefault("protoPayload.authenticationInfo.principalEmail", "unknown")
	var negResourcePath resourcepath.ResourcePath
	lease, err := builder.ClusterResource.NEGs.GetResourceLeaseHolderAt(negName, l.Timestamp())
	if err == nil {
		negResourcePath = resourcepath.NetworkEndpointGroup(lease.Holder.Namespace, negName)
	} else {
		negResourcePath = resourcepath.NetworkEndpointGroup("unknown", negName)
	}
	if !(isLast && isFirst) && (isLast || isFirst) {
		state := enum.RevisionStateOperationStarted
		if isLast {
			state = enum.RevisionStateOperationFinished
		}
		operationPath := resourcepath.Operation(negResourcePath, methodNameSplitted[len(methodNameSplitted)-1], operationId)
		cs.RecordRevision(operationPath, &history.StagingResourceRevision{
			Verb:       enum.RevisionVerbCreate,
			State:      state,
			Requestor:  principal,
			ChangeTime: l.Timestamp(),
			Partial:    false,
		})
	} else {
		cs.RecordEvent(negResourcePath)
	}
	if isFirst {
		method := methodNameSplitted[len(methodNameSplitted)-1]
		if method == "detachNetworkEndpoints" || method == "attachNetworkEndpoints" {
			isDetach := strings.HasPrefix(method, "detach")
			requestBody, err := l.GetChildYamlOf("protoPayload.request")
			if err != nil {
				return err
			}
			var negRequest NegAttachOrDetachRequest
			err = yaml.Unmarshal([]byte(requestBody), &negRequest)
			if err != nil {
				return err
			}
			for _, endpoint := range negRequest.NetworkEndpoints {
				lease, err := builder.ClusterResource.IPs.GetResourceLeaseHolderAt(endpoint.IpAddress, l.Timestamp())
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
						ChangeTime: l.Timestamp(),
						Partial:    false,
					})
				}
			}
		}
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

var _ parser.Parser = (*gceNetworkParser)(nil)

var NetowrkAPIParserTask = parser.NewParserTaskFromParser(gcp_task.GCPPrefix+"feature/network-api-parser", &gceNetworkParser{}, true, inspection_task.InspectionTypeLabel(gke.InspectionTypeId, composer_task.InspectionTypeId))
