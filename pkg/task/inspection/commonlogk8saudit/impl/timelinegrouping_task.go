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

package commonlogk8saudit_impl

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/grouper"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// resourceGroupDecider decides the resource path of the given logs written to.
type resourceGroupDecider interface {
	// Name returns the name of this decider implementation.
	Name() string
	// GetResourceGroup returns the resource path of the logs written to. This will return empty string when this decider won't decide the decision and delegate the decision to the later decider.
	GetResourceGroup(log *commonlogk8saudit_contract.AuditLogParserInput) (string, error)
}

// defaultResourceGroupDecider is a resourceGroupDecider that uses the default
// resource path derived from the KubernetesObjectOperation.
type defaultResourceGroupDecider struct{}

// Name implements resourceGroupDecider.
func (d *defaultResourceGroupDecider) Name() string {
	return "default"
}

// GetResourceGroup implements resourceGroupDecider. It returns the resource path
// by converting the log's Operation to a resource path.
func (d *defaultResourceGroupDecider) GetResourceGroup(log *commonlogk8saudit_contract.AuditLogParserInput) (string, error) {
	return log.Operation.CovertToResourcePath(), nil
}

// Ensure defaultResourceGroupDecider implements resourceGroupDecider.
var _ resourceGroupDecider = (*defaultResourceGroupDecider)(nil)

// subresourceDefaultBehavior defines how a subresource should be treated by default
// if its associated resource type cannot be determined from the log's request or response.
type subresourceDefaultBehavior int

const (
	// Subresource means the subresourceResourceGroupDecider must treat it as subresource by default. This is the default value.
	Subresource = 0
	// Parent means the subresourceResourceGroupDecider must treat it as its parent by default.
	Parent = 1
)

// subresourceResourceGroupDecider is a resourceGroupDecider that specifically
// handles subresources. It attempts to determine the associated resource type
// from the log's response or request, and if that fails, it falls back to
// a default behavior defined by `defaultBehaviorOverrides`.
type subresourceResourceGroupDecider struct {
	defaultBehaviorOverrides map[string]subresourceDefaultBehavior
}

// Name implements resourceGroupDecider.
func (s *subresourceResourceGroupDecider) Name() string {
	return "subresource"
}

// GetResourceGroup implements resourceGroupDecider.
func (s *subresourceResourceGroupDecider) GetResourceGroup(log *commonlogk8saudit_contract.AuditLogParserInput) (string, error) {
	if log.Operation.SubResourceName == "" {
		return "", nil // delegate this decision to the later deciders.
	}

	// Attempting to get the associated resource type from its response.
	if log.Response != nil {
		apiVersion, err := log.Response.ReadString("apiVersion")
		if err == nil {
			kind, err := log.Response.ReadString("kind")
			if err == nil {
				// If the response object is v1/Status, then use the request as group name source instead.
				if apiVersion != "v1" || kind != "Status" {
					parentKind := log.Operation.GetSingularKindName()
					if strings.ToLower(kind) == parentKind {
						// This response represents its parent resource. Return its parent resource path instead.
						return resourcepath.NameLayerGeneralItem(log.Operation.APIVersion, log.Operation.GetSingularKindName(), log.Operation.Namespace, log.Operation.Name).Path, nil
					} else {
						// Response contains not Status and not its parent kind resource, it must be the subresource itself.
						return resourcepath.SubresourceLayerGeneralItem(log.Operation.APIVersion, log.Operation.GetSingularKindName(), log.Operation.Namespace, log.Operation.Name, log.Operation.SubResourceName).Path, nil
					}
				}
			}
		}
	}

	// Attempting to get the associated resource type from its request.
	if log.Request != nil {
		_, err := log.Request.ReadString("apiVersion")
		if err == nil {
			kind, err := log.Request.ReadString("kind")
			if err == nil {
				parentKind := log.Operation.GetSingularKindName()
				if strings.ToLower(kind) == parentKind {
					// This request represents its parent resource. Return its parent resource path instead.
					return resourcepath.NameLayerGeneralItem(log.Operation.APIVersion, log.Operation.GetSingularKindName(), log.Operation.Namespace, log.Operation.Name).Path, nil
				} else {
					// Request contains non parent kind resource, it must be the subresource itself.
					return resourcepath.SubresourceLayerGeneralItem(log.Operation.APIVersion, log.Operation.GetSingularKindName(), log.Operation.Namespace, log.Operation.Name, log.Operation.SubResourceName).Path, nil
				}
			}
		}
	}

	// If finally the logic couldn't determine the associated resource type, then use the strategy defined for the name of subresource.
	if s.defaultBehaviorOverrides[log.Operation.SubResourceName] == Parent {
		return resourcepath.NameLayerGeneralItem(log.Operation.APIVersion, log.Operation.GetSingularKindName(), log.Operation.Namespace, log.Operation.Name).Path, nil
	} else {
		return log.Operation.CovertToResourcePath(), nil
	}
}

// Ensure subresourceResourceGroupDecider implements resourceGroupDecider.
var _ resourceGroupDecider = (*subresourceResourceGroupDecider)(nil)

// defaultTimelineGroupDeciders is the list of group deciders used in TimelineGroupingTask.
var defaultTimelineGroupDeciders []resourceGroupDecider = []resourceGroupDecider{
	&subresourceResourceGroupDecider{
		defaultBehaviorOverrides: map[string]subresourceDefaultBehavior{
			"status": Parent, // the status subresource is usually used with PATCH request and its response is its parent.
		},
	},
	&defaultResourceGroupDecider{},
}

// defaultUnknownGroupResourcePath is the default resource path used when
// the timeline grouper cannot determine a specific group for a log.
var defaultUnknownGroupResourcePath = resourcepath.NameLayerGeneralItem("unknown", "unknown", "unknown", "unknown")

// TimelineGroupingTask is an inspection task that groups audit logs into
// timelines based on the Kubernetes resource they relate to.
var TimelineGroupingTask = inspectiontaskbase.NewProgressReportableInspectionTask(commonlogk8saudit_contract.TimelineGroupingTaskID, []taskid.UntypedTaskReference{
	commonlogk8saudit_contract.CommonLogParseTaskID.Ref(),
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) ([]*commonlogk8saudit_contract.TimelineGrouperResult, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return nil, nil
	}
	preStepParseResult := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.CommonLogParseTaskID.Ref())
	progressUpdater := progressutil.NewIndeterminateUpdator(tp, time.Second)
	err := progressUpdater.Start("Grouping logs by timeline")
	if err != nil {
		return nil, err
	}
	defer progressUpdater.Done()

	timelineGrouper := grouper.NewBasicGrouper(func(input *commonlogk8saudit_contract.AuditLogParserInput) string {
		for _, decider := range defaultTimelineGroupDeciders {
			resourcePath, err := decider.GetResourceGroup(input)
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("group decider %s returned an error %v. Group decision will be handled by the later deciders instead", decider.Name(), err))
			}
			if resourcePath != "" {
				return resourcePath
			}
		}
		slog.WarnContext(ctx, fmt.Sprintf("failed to decide the group of log %s. Using a default group %s", input.Log.ID, defaultUnknownGroupResourcePath.Path))
		return defaultUnknownGroupResourcePath.Path
	})

	groups := timelineGrouper.Group(preStepParseResult)
	result := []*commonlogk8saudit_contract.TimelineGrouperResult{}
	for key, group := range groups {
		result = append(result, &commonlogk8saudit_contract.TimelineGrouperResult{
			TimelineResourcePath: key,
			PreParsedLogs:        group,
		})
	}
	createDeletionRequestsByDeleteColection(result)
	return result, nil
})

// TODO(#278): move this logic to outside of the grouping task with refactoring.
func createDeletionRequestsByDeleteColection(groups []*commonlogk8saudit_contract.TimelineGrouperResult) {
	requireSortTimelinePaths := map[string]struct{}{}
	for _, group := range groups {
		// delete collection only happens for namespace layer
		if strings.HasSuffix(group.TimelineResourcePath, "#") {
			for _, l := range group.PreParsedLogs {
				if l.Operation.Verb == enum.RevisionVerbDeleteCollection {
					for _, childGroup := range groups {
						// find any timelines under current timeline
						// Example: current timeline v1/core#pods#default
						// Example childGroups:
						// * v1/core#pods#default#foo -> match
						// * v1/core#pods#default#foo#binding -> not match. Subresource deletions are handled in each parsers for resource.
						if childGroup.TimelineResourcePath != group.TimelineResourcePath && strings.HasPrefix(childGroup.TimelineResourcePath, group.TimelineResourcePath) && strings.Count(childGroup.TimelineResourcePath, "#") == 3 {
							refLog := childGroup.PreParsedLogs[0]
							k8sOp := model.KubernetesObjectOperation{
								APIVersion: refLog.Operation.APIVersion,
								PluralKind: refLog.Operation.PluralKind,
								Namespace:  refLog.Operation.Namespace,
								Name:       refLog.Operation.Name,
								Verb:       enum.RevisionVerbDelete,
							}
							refLogCommonField := log.MustGetFieldSet(refLog.Log, &log.CommonFieldSet{})
							logCommonField := log.MustGetFieldSet(l.Log, &log.CommonFieldSet{})
							if refLogCommonField.Timestamp.Sub(logCommonField.Timestamp) > 0 {
								// This delete collection happened before the resource existing. ignore the delete collection request.
								continue
							}
							childGroup.PreParsedLogs = append(childGroup.PreParsedLogs, &commonlogk8saudit_contract.AuditLogParserInput{
								Log:                                    l.Log,
								Requestor:                              l.Requestor,
								Operation:                              &k8sOp,
								ResponseErrorCode:                      l.ResponseErrorCode,
								ResponseErrorMessage:                   l.ResponseErrorMessage,
								IsErrorResponse:                        l.IsErrorResponse,
								Request:                                nil,
								RequestType:                            commonlogk8saudit_contract.RTypeUnknown,
								Response:                               nil,
								ResponseType:                           commonlogk8saudit_contract.RTypeUnknown,
								GeneratedFromDeleteCollectionOperation: true,
							})
							requireSortTimelinePaths[childGroup.TimelineResourcePath] = struct{}{}
						}
					}
				}
			}
		}
	}
	// sort logs with additional deletion logs in timeline
	for _, group := range groups {
		if _, found := requireSortTimelinePaths[group.TimelineResourcePath]; found {
			sort.Slice(group.PreParsedLogs, func(i, j int) bool {
				logICommonField := log.MustGetFieldSet(group.PreParsedLogs[i].Log, &log.CommonFieldSet{})
				logJCommonField := log.MustGetFieldSet(group.PreParsedLogs[j].Log, &log.CommonFieldSet{})
				return logICommonField.Timestamp.Sub(logJCommonField.Timestamp) <= 0
			})
		}
	}
}
