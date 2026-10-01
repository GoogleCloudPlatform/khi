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

package gcpcommon

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// maxResourceNameCountPerRequest is the maximum allowed count of resource names per single entries.list. The default quota is 100.
var maxResourceNameCountPerRequest = 100

func monitorProgress(ctx context.Context, wg *sync.WaitGroup, source <-chan LogFetchProgress, tracker *progress.RatioTracker, baseLogCount int, listCallIndex int, totalListCalls int) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case p, ok := <-source:
				if !ok {
					return
				}
				totalLogCount := baseLogCount + p.LogCount
				completeRatio := (float32(listCallIndex) + p.Progress) / float32(totalListCalls)
				tracker.Update(completeRatio, totalLogCount, progress.WithStep(listCallIndex+1, totalListCalls))
			}
		}
	}()
}

// handleResourceNames retrieves and validates resource names for a given task, updating default values if necessary.
func handleResourceNames(ctx context.Context, taskID taskid.TaskImplementationID[[]*log.Log], resourceNamesInput *ResourceNamesInput, getDefaultResourceNames func(ctx context.Context) ([]string, error)) ([]string, error) {
	queryResourceNamePair := resourceNamesInput.GetResourceNamesForQuery(ctx, taskID.ReferenceIDString())

	defaultResourceNames, err := getDefaultResourceNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("ResourceNames returned an error: %w", err)
	}

	resourceNamesInput.UpdateDefaultResourceNamesForQuery(taskID.ReferenceIDString(), defaultResourceNames)

	return queryResourceNamePair.CurrentResourceNames, nil
}

// setErrorMetadataForFetchLogError extracts error information from a log fetching operation and adds it to the inspection run's error message set metadata.
func setErrorMetadataForFetchLogError(ctx context.Context, err error) error {
	metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
	errorMessageSet, found := typedmap.Get(metadata, inspectionmetadata.ErrorMessageSetMetadataKey)
	if !found {
		return fmt.Errorf("error message set metadata was not found. originalError=%w", err)
	}
	errorMessageSet.AddErrorMessage(&inspectionmetadata.ErrorMessage{
		ErrorId: 0,
		Message: err.Error(),
	})
	return err
}

// resourceContainerLogQueryGroup groups resource names under a common Google Cloud resource container.
type resourceContainerLogQueryGroup struct {
	container     googlecloud.ResourceContainer
	resourceNames []string
}

// groupResourceNamesByContainer groups a list of resource names by their Google Cloud resource container.
// It returns a slice of resourceContainerLogQueryGroup, where each group contains resource names
// belonging to the same container (e.g., project).
func groupResourceNamesByContainer(resourceNames []string) ([]*resourceContainerLogQueryGroup, error) {
	groups := make(map[string]*resourceContainerLogQueryGroup)

	for _, resourceName := range resourceNames {
		var container googlecloud.ResourceContainer
		switch {
		case strings.HasPrefix(resourceName, "projects/"):
			projectID := resourceName[len("projects/"):]
			slashIndex := strings.Index(projectID, "/")
			if slashIndex != -1 {
				projectID = projectID[:slashIndex]
			}
			container = googlecloud.Project(projectID)
		default:
			// TODO: Add support for other resource containers like organizations, folders, and billingAccounts.
			// Unsupported resource container types.
		}
		if container == nil {
			return nil, fmt.Errorf("unsupported resource name %q : %w", resourceName, khierrors.ErrInvalidInput)
		}
		containerIdentifier := container.Identifier()
		if _, ok := groups[containerIdentifier]; !ok {
			groups[containerIdentifier] = &resourceContainerLogQueryGroup{
				container: container,
			}
		}

		group := groups[containerIdentifier]
		group.resourceNames = append(group.resourceNames, resourceName)
	}

	result := slices.Collect(maps.Values(groups))
	slices.SortFunc(result, func(a, b *resourceContainerLogQueryGroup) int {
		return strings.Compare(a.container.Identifier(), b.container.Identifier())
	})
	return result, nil
}

// divideGroupByMaximumResourceName divides resourceContainerLogQueryGroup instances into smaller groups if their resourceNames slice exceeds maxResourceNamePerGroup.
func divideGroupByMaximumResourceName(groups []*resourceContainerLogQueryGroup, maxResourceNamePerGroup int) []*resourceContainerLogQueryGroup {
	var dividedGroups []*resourceContainerLogQueryGroup
	for _, group := range groups {
		for len(group.resourceNames) > maxResourceNamePerGroup {
			dividedGroups = append(dividedGroups, &resourceContainerLogQueryGroup{
				container:     group.container,
				resourceNames: group.resourceNames[:maxResourceNamePerGroup],
			})
			group.resourceNames = group.resourceNames[maxResourceNamePerGroup:]
		}
		dividedGroups = append(dividedGroups, group)
	}
	return dividedGroups
}
