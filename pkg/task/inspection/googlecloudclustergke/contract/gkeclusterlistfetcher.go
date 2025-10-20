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

package googlecloudclustergke_contract

import (
	"context"
	"fmt"

	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

// ClusterListFetcher fetches the list of GKE cluster in the project.
type ClusterListFetcher interface {
	GetClusterNames(ctx context.Context, projectID string) ([]string, error)
}

// ClusterListFetcherImpl is the default implementation of ClusterListFetcher.
type ClusterListFetcherImpl struct{}

// GetClusterNames implements ClusterListFetcher.
// This expects the task googlecloudcommon_contract.APIClientFactoryTaskID is already resolved.
func (g *ClusterListFetcherImpl) GetClusterNames(ctx context.Context, projectID string) ([]string, error) {
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	injector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	ccmc, err := cf.ContainerClusterManagerClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create container cluster manager client: %w", err)
	}
	defer ccmc.Close()

	ctx = injector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	clusters, err := ccmc.ListClusters(ctx, &containerpb.ListClustersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read the cluster names in the project %s: %w", projectID, err)
	}

	return apiResponseToClusterNameList(clusters), nil
}

var _ ClusterListFetcher = (*ClusterListFetcherImpl)(nil)

// apiResponseToClusterNameList returns the list of cluster names from the API response.
func apiResponseToClusterNameList(response *containerpb.ListClustersResponse) []string {
	if response == nil {
		return []string{}
	}
	result := make([]string, 0, len(response.Clusters))
	for _, cluster := range response.Clusters {
		result = append(result, cluster.Name)
	}
	return result
}
