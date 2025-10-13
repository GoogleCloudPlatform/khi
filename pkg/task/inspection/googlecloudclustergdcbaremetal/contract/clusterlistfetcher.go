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

package googlecloudclustergdcbaremetal_contract

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloudv2"
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

	onpremAPI, err := cf.GKEOnPremService(ctx, googlecloudv2.Project(project))
	if err != nil {
		return nil, fmt.Errorf("failed to generate onprem API client: %v", err)
	}

	parent := fmt.Sprintf("projects/%s/locations/-", project)
	resultCh := make(chan []string, 2)
	errGrp, groupCtx := errgroup.WithContext(ctx)

	errGrp.Go(func() error {
		adminClusters, err := getAdminClusters(groupCtx, onpremAPI, parent)
		if err != nil {
			return err
		}
		resultCh <- adminClusters
		return nil
	})

	errGrp.Go(func() error {
		userClusters, err := getUserClusters(groupCtx, onpremAPI, parent)
		if err != nil {
			return err
		}
		resultCh <- userClusters
		return nil
	})

	err = errGrp.Wait()
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

var _ ClusterListFetcher = (*ClusterListFetcherImpl)(nil)

func getAdminClusters(ctx context.Context, client *gkeonprem.Service, parent string) ([]string, error) {
	var nextPageToken string
	var result []string
	for {
		req := client.Projects.Locations.BareMetalAdminClusters.List(parent).PageToken(nextPageToken)
		resp, err := req.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		for _, cluster := range resp.BareMetalAdminClusters {
			result = append(result, toShortClusterName(cluster.Name))
		}
		nextPageToken = resp.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
	return result, nil
}

func getUserClusters(ctx context.Context, client *gkeonprem.Service, parent string) ([]string, error) {
	var nextPageToken string
	var result []string
	for {
		req := client.Projects.Locations.BareMetalClusters.List(parent).PageToken(nextPageToken)
		resp, err := req.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		for _, cluster := range resp.BareMetalClusters {
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
// The original format is /projects/{projectID}/locations/{location}/(baremetalClusters|baremetalAdminClusters)/{clusterName}
func toShortClusterName(longClusterName string) string {
	li := strings.LastIndex(longClusterName, "/")
	return longClusterName[li+1:]
}
