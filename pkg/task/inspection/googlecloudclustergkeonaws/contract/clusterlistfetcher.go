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

package googlecloudclustergkeonaws_contract

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/gkemulticloud/apiv1/gkemulticloudpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	"google.golang.org/api/iterator"
)

// ClusterListFetcher fetches the list of GKE on AWS cluster in the project.
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

	gkeMultiCloudAwsClient, err := cf.GKEMultiCloudAWSClustersClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return nil, fmt.Errorf("failed to get the GKE on AWS client:%v", err)
	}
	defer gkeMultiCloudAwsClient.Close()

	ctx = injector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	itr := gkeMultiCloudAwsClient.ListAwsClusters(ctx, &gkemulticloudpb.ListAwsClustersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectID),
	})

	var result []string
	for {
		resp, err := itr.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list GKE on AWS clusters: %v", err)
		}
		result = append(result, awsClusterToClusterName(resp))
	}

	return result, nil
}

var _ ClusterListFetcher = (*ClusterListFetcherImpl)(nil)

// awsClusterToClusterName returns the list of cluster names from the API response.
func awsClusterToClusterName(awsCluster *gkemulticloudpb.AwsCluster) string {
	li := strings.LastIndex(awsCluster.Name, "/")
	return awsCluster.Name[li+1:]
}
