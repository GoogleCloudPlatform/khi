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

package googlecloudclustergdcvmware_contract

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/gkeonprem/v1"
)

type ClusterListFetcher interface {
	GetClusters(ctx context.Context, project string) ([]string, error)
}

type ClusterListFetcherImpl struct{}

// GetClusters implements ClusterListFetcher.
func (c *ClusterListFetcherImpl) GetClusters(ctx context.Context, project string) ([]string, error) {
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	injector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	onpremAPI, err := cf.GKEOnPremService(ctx, googlecloud.Project(project))
	if err != nil {
		return nil, fmt.Errorf("failed to generate onprem API client: %v", err)
	}

	return getAdminAndUserClusters(ctx, project, func(ctx context.Context, parent string) ([]string, error) {
		return getAdminClustersFromAPI(ctx, injector, onpremAPI, parent)
	}, func(ctx context.Context, parent string) ([]string, error) {
		return getUserClustersFromAPI(ctx, injector, onpremAPI, parent)
	})
}

var _ ClusterListFetcher = (*ClusterListFetcherImpl)(nil)

type fetchClusterFunc = func(ctx context.Context, parent string) ([]string, error)

// getAdminAndUserClusters returns the list of clusters obtained from APIs.
func getAdminAndUserClusters(ctx context.Context, project string, fetchAdminCluster, fetchUserCluster fetchClusterFunc) ([]string, error) {
	resultCh := make(chan []string, 2)
	errGrp, groupCtx := errgroup.WithContext(ctx)

	errGrp.Go(func() error {
		adminClusters, err := fetchAdminCluster(groupCtx, project)
		if err != nil {
			return err
		}
		resultCh <- adminClusters
		return nil
	})

	errGrp.Go(func() error {
		userClusters, err := fetchUserCluster(groupCtx, project)
		if err != nil {
			return err
		}
		resultCh <- userClusters
		return nil
	})

	err := errGrp.Wait()
	close(resultCh)
	if err != nil {
		return nil, err
	}

	var result []string
	for clusters := range resultCh {
		result = append(result, clusters...)
	}
	return result, nil
}

func getAdminClustersFromAPI(ctx context.Context, injector *googlecloud.CallOptionInjector, client *gkeonprem.Service, project string) ([]string, error) {
	parent := fmt.Sprintf("projects/%s/locations/-", project)
	var nextPageToken string
	var result []string
	for {
		req := client.Projects.Locations.VmwareAdminClusters.List(parent).PageToken(nextPageToken)
		injector.InjectToCall(req, googlecloud.Project(project))
		resp, err := req.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		for _, cluster := range resp.VmwareAdminClusters {
			result = append(result, toShortClusterName(cluster.Name))
		}
		nextPageToken = resp.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
	return result, nil
}

func getUserClustersFromAPI(ctx context.Context, injector *googlecloud.CallOptionInjector, client *gkeonprem.Service, project string) ([]string, error) {
	parent := fmt.Sprintf("projects/%s/locations/-", project)
	var nextPageToken string
	var result []string
	for {
		req := client.Projects.Locations.VmwareClusters.List(parent).PageToken(nextPageToken)
		injector.InjectToCall(req, googlecloud.Project(project))
		resp, err := req.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		for _, cluster := range resp.VmwareClusters {
			result = append(result, toShortClusterName(cluster.Name))
		}
		nextPageToken = resp.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
	return result, nil
}

// toShortClusterName converts the cluster name included in the api response to the name used in form field.
// The original format is /projects/{projectID}/locations/{location}/(vmwareClusters|vmwareAdminClusters)/{clusterName}
func toShortClusterName(longClusterName string) string {
	li := strings.LastIndex(longClusterName, "/")
	return longClusterName[li+1:]
}
