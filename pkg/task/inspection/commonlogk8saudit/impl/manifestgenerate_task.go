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
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var bodyPlaceholderForMetadataLevelAuditLog = "# Resource data is unavailable. Audit logs for this resource is recorded at metadata level."

var ManifestGenerateTask = inspectiontaskbase.NewProgressReportableInspectionTask(commonlogk8saudit_contract.ManifestGenerateTaskID, []taskid.UntypedTaskReference{
	commonlogk8saudit_contract.TimelineGroupingTaskID.Ref(),
	googlecloudk8scommon_contract.K8sResourceMergeConfigTaskID.Ref(),
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) ([]*commonlogk8saudit_contract.TimelineGrouperResult, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return nil, nil
	}
	groups := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.TimelineGroupingTaskID.Ref())
	mergeConfigRegistry := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.K8sResourceMergeConfigTaskID.Ref())

	totalLogCount := 0
	for _, group := range groups {
		totalLogCount += len(group.PreParsedLogs)
	}
	processedCount := atomic.Int32{}
	updator := progressutil.NewProgressUpdator(tp, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
		current := processedCount.Load()
		tp.Percentage = float32(current) / float32(totalLogCount)
		tp.Message = fmt.Sprintf("%d/%d", current, totalLogCount)
	})
	err := updator.Start(ctx)
	if err != nil {
		return nil, err
	}
	defer updator.Done()
	workerPool := worker.NewPool(16)
	for _, group := range groups {
		currentGroup := group
		workerPool.Run(func() {
			var prevRevisionBody string
			var prevResourceUID string
			prevRevisionReader := structured.NewNodeReader(structured.NewEmptyMapNode())
			for _, log := range currentGroup.PreParsedLogs {
				var currentRevisionBodyType commonlogk8saudit_contract.RequestResponseType
				if log.IsErrorResponse || log.GeneratedFromDeleteCollectionOperation {
					log.ResourceBodyYaml = prevRevisionBody
					log.ResourceBodyReader = prevRevisionReader
					log.ResourceUID = prevResourceUID
					processedCount.Add(1)
					continue
				}
				currentRevisionReader := log.Response
				currentRevisionBodyType = log.ResponseType
				if currentRevisionReader == nil || log.ResponseType != commonlogk8saudit_contract.RTypeUnknown {
					currentRevisionReader = log.Request
					currentRevisionBodyType = log.RequestType
				}

				// Manifest is unknown because it doesn't contain request or response in the body.
				if currentRevisionReader == nil {
					log.ResourceBodyYaml = bodyPlaceholderForMetadataLevelAuditLog
					log.ResourceUID = prevResourceUID
					processedCount.Add(1)
					continue
				} else {
					uid, err := currentRevisionReader.ReadString("metadata.uid")
					if err == nil {
						// uid is different from the past. The resource must be newly created.
						if uid != prevResourceUID {
							prevRevisionBody = ""
							prevRevisionReader = structured.NewNodeReader(structured.NewEmptyMapNode())
							prevResourceUID = uid
						}
					}
				}
				log.ResourceUID = prevResourceUID

				isPartial := currentRevisionBodyType == commonlogk8saudit_contract.RTypePatch
				currentRevisionBodyRaw, err := currentRevisionReader.Serialize("", &structured.YAMLNodeSerializer{})
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to serialize resource body to yaml\n%s", err.Error()))
					processedCount.Add(1)
					continue
				}
				currentRevisionBody := string(currentRevisionBodyRaw)
				currentRevisionBody = removeAtType(currentRevisionBody)

				if isPartial {
					mergeConfigResolver := mergeConfigRegistry.Get(log.Operation.APIVersion, log.Operation.GetSingularKindName())
					mergedNode, err := structured.MergeNode(prevRevisionReader.Node, currentRevisionReader.Node, structured.MergeConfiguration{
						MergeMapOrderStrategy:    &structured.DefaultMergeMapOrderStrategy{},
						ArrayMergeConfigResolver: mergeConfigResolver,
					})
					if err != nil {
						slog.WarnContext(ctx, fmt.Sprintf("failed to merge resource body\n%s", err.Error()))
						processedCount.Add(1)
						continue
					}
					mergedNodeReader := structured.NewNodeReader(mergedNode)
					mergedYaml, err := mergedNodeReader.Serialize("", &structured.YAMLNodeSerializer{})
					if err != nil {
						slog.WarnContext(ctx, fmt.Sprintf("failed to read the merged resource body\n%s", err.Error()))
						processedCount.Add(1)
						continue
					}
					log.ResourceBodyYaml = removeAtType(string(mergedYaml))
					log.ResourceBodyReader = mergedNodeReader
				} else {
					if currentRevisionBodyType == commonlogk8saudit_contract.RTypeDeleteOptions {
						log.ResourceBodyYaml = prevRevisionBody
						log.ResourceBodyReader = prevRevisionReader
						processedCount.Add(1)
						continue
					}
					log.ResourceBodyYaml = currentRevisionBody
					log.ResourceBodyReader = currentRevisionReader
				}
				prevRevisionBody = log.ResourceBodyYaml
				prevRevisionReader = log.ResourceBodyReader
				processedCount.Add(1)
			}
		})
	}
	workerPool.Wait()

	slices.SortFunc(groups, func(a, b *commonlogk8saudit_contract.TimelineGrouperResult) int {
		return strings.Compare(a.TimelineResourcePath, b.TimelineResourcePath)
	})

	return groups, nil
})

// Remove @type in response or request payload
func removeAtType(yamlString string) string {
	if strings.Contains(yamlString, "'@type'") {
		index := strings.Index(yamlString, "\n")
		return yamlString[index+1:]
	}
	return yamlString
}
