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

package googlecloudclustercomposer_contract

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	"google.golang.org/api/composer/v1"
)

// ComposerEnvironmentListFetcher fetches the list of Cloud Composer environment names in project and the location.
type ComposerEnvironmentListFetcher interface {
	GetEnvironmentNames(ctx context.Context, projectID, location string) ([]string, error)
}

type ComposerEnvironmentListFetcherImpl struct{}

// GetEnvironmentNames implements ComposerEnvironmentListFetcher.
func (c *ComposerEnvironmentListFetcherImpl) GetEnvironmentNames(ctx context.Context, projectID string, location string) ([]string, error) {
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	injector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	composerClient, err := cf.ComposerService(ctx, googlecloud.Project(projectID))
	if err != nil {
		return nil, fmt.Errorf("failed to get the composer api client:%v", err)
	}

	var result []string
	var nextPageToken string
	for {
		req := composerClient.Projects.Locations.Environments.List(fmt.Sprintf("projects/%s/locations/%s", projectID, location)).PageToken(nextPageToken)
		injector.InjectToCall(req, googlecloud.Project(projectID))
		resp, err := req.Do()
		if err != nil {
			return nil, err
		}
		for _, cluster := range resp.Environments {
			result = append(result, apiEnvironmentToClusterName(cluster))
		}
		nextPageToken = resp.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	return result, nil
}

var _ ComposerEnvironmentListFetcher = (*ComposerEnvironmentListFetcherImpl)(nil)

// apiEnvironmentToClusterName convert the API response to cluster name.
func apiEnvironmentToClusterName(env *composer.Environment) string {
	// The name is in the form "projects/{projectId}/locations/{locationId}/environments/{environmentId}"
	li := strings.LastIndex(env.Name, "/")
	return env.Name[li+1:]
}
