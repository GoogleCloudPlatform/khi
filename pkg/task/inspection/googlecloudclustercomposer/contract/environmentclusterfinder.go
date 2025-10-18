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
	"errors"
	"fmt"

	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

var ErrEnvironmentClusterNotFound = errors.New("not found")

type ComposerEnvironmentClusterFinder interface {
	GetGKEClusterName(ctx context.Context, projectID, environment string) (string, error)
}

type EnvironmentClusterFinderImpl struct{}

// GetGKEClusterName implements EnvironmentClusterFinder.
func (e *EnvironmentClusterFinderImpl) GetGKEClusterName(ctx context.Context, projectID string, environment string) (string, error) {
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	injector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	containerClusterManagerClient, err := cf.ContainerClusterManagerClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return "", err
	}
	defer containerClusterManagerClient.Close()

	ctx = injector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	cluster, err := containerClusterManagerClient.ListClusters(ctx, &containerpb.ListClustersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectID),
	})
	if err != nil {
		return "", err
	}

	for _, c := range cluster.Clusters {
		if c.ResourceLabels["goog-composer-environment"] == environment {
			return c.Name, nil
		}
	}
	return "", ErrEnvironmentClusterNotFound
}

var _ ComposerEnvironmentClusterFinder = (*EnvironmentClusterFinderImpl)(nil)
