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

package commonlogk8sauditv2_impl

import (
	"context"
	"fmt"
	"strings"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// NonSuccessLogGrouperTask groups logs by resource path.
// K8s audit error logs are simply associated with timelines as events. They don't require any special grouping, so they use the resource associated with the original resource name modified by the request.
var NonSuccessLogGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	commonlogk8sauditv2_contract.NonSuccessLogGrouperTaskID,
	commonlogk8sauditv2_contract.NonSuccessLogFilterTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		fieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
		return fieldSet.K8sOperation.ResourcePath()
	},
)

// ChangeTargetGrouperTask groups logs by resource that is modified by the operation in the log.
// This task determines the group, specifically handling the following cases:
// 1. When multiple resources are modified by the operation, the log entry is duplicated and assigned to each group.
// 2. When a subresource is modified by the operation and its result contains its parent manifest, it uses the parent resource as the group key.
var ChangeTargetGrouperTask = inspectiontaskbase.NewProgressReportableInspectionTask[commonlogk8sauditv2_contract.ResourceLogGroupMap](
	commonlogk8sauditv2_contract.ChangeTargetGrouperTaskID,
	[]taskid.UntypedTaskReference{commonlogk8sauditv2_contract.LogSorterTaskID.Ref()},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (commonlogk8sauditv2_contract.ResourceLogGroupMap, error) {
		if taskMode != inspectioncore_contract.TaskModeRun {
			return commonlogk8sauditv2_contract.ResourceLogGroupMap{}, nil
		}

		progress.MarkIndeterminate()

		logs := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.LogSorterTaskID.Ref())
		result := commonlogk8sauditv2_contract.ResourceLogGroupMap{}
		scanner := targetResourceScanner{
			resourcesByNamespaceKindAPIVersions: map[string]map[string]struct{}{},
			subresourceDefaultBehaviorOverrides: defaultSubresourceDefaultBehaviorOverrides,
		}

		for _, l := range logs {
			ops := scanner.scanTargetResource(l)
			for _, op := range ops {
				resource := commonlogk8sauditv2_contract.ResourceIdentityFromKubernetesOperation(op)
				path := resource.ResourcePathString()
				if result[path] == nil {
					result[path] = &commonlogk8sauditv2_contract.ResourceLogGroup{
						Logs:     []*log.Log{},
						Resource: resource,
					}
				}
				result[path].Logs = append(result[path].Logs, l)
			}
		}

		return result, nil
	},
)

// subresourceDefaultBehavior defines how a subresource should be treated by default
// if its associated resource type cannot be determined from the log's request or response.
type subresourceDefaultBehavior int

const (
	// Subresource means the subresourceResourceGroupDecider must treat it as subresource by default. This is the default value.
	Subresource = 0
	// Parent means the subresourceResourceGroupDecider must treat it as its parent by default.
	Parent = 1
)

var defaultSubresourceDefaultBehaviorOverrides = map[string]subresourceDefaultBehavior{
	"status": Parent,
}

type targetResourceScanner struct {
	resourcesByNamespaceKindAPIVersions map[string]map[string]struct{}
	subresourceDefaultBehaviorOverrides map[string]subresourceDefaultBehavior
}

func (s *targetResourceScanner) scanTargetResource(l *log.Log) []*model.KubernetesObjectOperation {
	targetResource := s.scanTargetResourceInternal(l)

	// Memorize all resources modified up to this point to handle delete collection methods
	for _, resource := range targetResource {
		if resource.Namespace == "cluster-scope" {
			continue
		}
		namespaceKindAPIVersions := fmt.Sprintf("%s/%s/%s", resource.APIVersion, resource.PluralKind, resource.Namespace)
		if s.resourcesByNamespaceKindAPIVersions[namespaceKindAPIVersions] == nil {
			s.resourcesByNamespaceKindAPIVersions[namespaceKindAPIVersions] = map[string]struct{}{}
		}
		s.resourcesByNamespaceKindAPIVersions[namespaceKindAPIVersions][resource.Name] = struct{}{}
	}
	return targetResource
}

func (s *targetResourceScanner) scanTargetResourceInternal(l *log.Log) []*model.KubernetesObjectOperation {
	fieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
	op := fieldSet.K8sOperation
	if fieldSet.K8sOperation.Verb == enum.RevisionVerbDeleteCollection {
		removedResourceNames := []*model.KubernetesObjectOperation{}
		foundItemSource := false
		if fieldSet.Response != nil {
			reader, err := fieldSet.Response.GetReader("items")
			if err == nil {
				foundItemSource = true
				for _, resource := range reader.Children() {
					name, err := resource.ReadString("metadata.name")
					if err == nil {
						itemOperation := op.Clone()
						itemOperation.Name = name
						removedResourceNames = append(removedResourceNames, itemOperation)
					}
				}
			}
		}
		// If no items found in its response fields, then the request is to delete all the resources under a namespace.
		if !foundItemSource {
			namespaceKindAPIVersions := fmt.Sprintf("%s/%s/%s", op.APIVersion, op.PluralKind, op.Namespace)
			knownResources := s.resourcesByNamespaceKindAPIVersions[namespaceKindAPIVersions]
			for resource := range knownResources {
				itemOperation := op.Clone()
				itemOperation.Name = resource
				removedResourceNames = append(removedResourceNames, itemOperation)
			}
			namespaceOperation := op.Clone()
			removedResourceNames = append(removedResourceNames, namespaceOperation)
		}
		return removedResourceNames
	} else {
		if op.SubResourceName == "" {
			return []*model.KubernetesObjectOperation{op}
		} else {
			// An audit log for subresource may contain a response for its parent.
			// Response to a subresource may contain its parent resource, so we need to check the response kind.
			opForSubresource := op.Clone()
			if fieldSet.Response != nil {
				apiVersion, err := fieldSet.Response.ReadString("apiVersion")
				if err == nil {
					kind, err := fieldSet.Response.ReadString("kind")
					if err == nil {
						// If the response object is v1/Status, then use the request as group name source instead.
						if apiVersion != "v1" || kind != "Status" {
							if strings.ToLower(kind) == op.GetSingularKindName() {
								opForSubresource.SubResourceName = ""
							}
						}
					}
				}
			}

			// All logs with non-Metadata audit log levels should be captured above, but for logs with Metadata level, we have no way to identify the content.
			// So just use a predefined map to determine if the log should be annotated on its parent timeline or a subresource timeline.
			if s.subresourceDefaultBehaviorOverrides[op.SubResourceName] == Parent {
				opForSubresource.SubResourceName = ""
			}
			return []*model.KubernetesObjectOperation{opForSubresource}
		}
	}
}
