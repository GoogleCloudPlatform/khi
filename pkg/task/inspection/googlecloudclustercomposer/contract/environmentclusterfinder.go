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
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

var ErrEnvironmentClusterNotFound = errors.New("not found")

type ComposerEnvironmentClusterFinder interface {
	GetGKEClusterNames(ctx context.Context, projectID, location, environment string, startTime, endTime time.Time) ([]string, error)
}

type EnvironmentClusterFinderImpl struct{}

// GetGKEClusterNames implements ComposerEnvironmentClusterFinder.
func (e *EnvironmentClusterFinderImpl) GetGKEClusterNames(ctx context.Context, projectID, location, environment string, startTime, endTime time.Time) ([]string, error) {
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	injector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	client, err := cf.MonitoringMetricClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return nil, err
	}

	ctx = injector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	filter := `metric.type="kubernetes.io/container/uptime" AND resource.type="k8s_container"`
	metricsLabels, err := googlecloud.QueryResourceLabelsFromMetrics(ctx, client, projectID, filter, startTime, endTime, []string{"resource.label.cluster_name", "resource.label.location"})
	if err != nil {
		return nil, err
	}

	matchedClusters := filterAndMatchComposerGKEClusterNames(metricsLabels, location, environment)
	if len(matchedClusters) == 0 {
		return nil, ErrEnvironmentClusterNotFound
	}
	return matchedClusters, nil
}

// filterAndMatchComposerGKEClusterNames extracts and deduplicates GKE cluster names matching the Cloud Composer naming convention.
// A Composer GKE cluster name follows the pattern: <location>-<environment>-<hash (8 chars)>-gke.
func filterAndMatchComposerGKEClusterNames(metricsLabels []map[string]string, location, environment string) []string {
	if environment == "" {
		return nil
	}
	expectedPrefix := fmt.Sprintf("%s-%s-", location, environment)
	seen := make(map[string]struct{})
	var result []string

	for _, labels := range metricsLabels {
		cName := labels["cluster_name"]
		cLoc := labels["location"]
		if location != "" && cLoc != "" && cLoc != location {
			continue
		}
		if !strings.HasPrefix(cName, expectedPrefix) || !strings.HasSuffix(cName, "-gke") {
			continue
		}
		body := strings.TrimSuffix(cName, "-gke")
		hashPart := strings.TrimPrefix(body, expectedPrefix)
		if len(hashPart) == 8 {
			if _, exists := seen[cName]; !exists {
				seen[cName] = struct{}{}
				result = append(result, cName)
			}
		}
	}
	sort.Strings(result)
	return result
}

var _ ComposerEnvironmentClusterFinder = (*EnvironmentClusterFinderImpl)(nil)
