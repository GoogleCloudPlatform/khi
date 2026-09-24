// Copyright 2026 Google LLC
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

package caik8s_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func resolveGKEResourceSearchTarget(ctx context.Context, _ inspectioncore.InspectionTaskModeType) (string, gcpcommon.CAIAssetSearchTarget, bool, error) {
	cluster := coretask.GetTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref())
	if !cluster.IsComplete() {
		return "", gcpcommon.CAIAssetSearchTarget{}, true, nil
	}

	scope := fmt.Sprintf("projects/%s", cluster.ProjectID)
	return cluster.ProjectID, gcpcommon.CAIAssetSearchTarget{
		Scope: scope,
		Discover: func(ctx context.Context, fetcher gcpcommon.CAIFetcher) ([]string, error) {
			return discoverGKEResourceAssetNames(ctx, fetcher, cluster)
		},
	}, false, nil
}

// discoverGKEResourceAssetNames searches CAI for the cluster and its nodepools and returns their full resource names.
func discoverGKEResourceAssetNames(
	ctx context.Context,
	fetcher gcpcommon.CAIFetcher,
	cluster k8scommon.GoogleCloudClusterIdentity,
) ([]string, error) {
	scope := fmt.Sprintf("projects/%s", cluster.ProjectID)
	progress.Report(ctx, 0, "[1/2] Searching GKE cluster in Cloud Asset Inventory... (0 resources scanned)")

	candidates := clusterAssetNameCandidates(cluster)
	clusterQuery := fmt.Sprintf("name=%q OR name=%q", candidates[0], candidates[1])
	clusterSearchResults, err := fetcher.SearchResources(
		ctx,
		scope,
		clusterQuery,
		[]string{caik8s.GKEClusterAssetType},
		func(scannedInQuery int) {
			progress.Report(ctx, 0, fmt.Sprintf("[1/2] Searching GKE cluster in Cloud Asset Inventory... (%d resources scanned)", scannedInQuery))
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search GKE cluster from CAI: %w", err)
	}
	if len(clusterSearchResults) == 0 {
		return []string{}, nil
	}

	clusterAssetName := clusterSearchResults[0].Name
	progress.Report(ctx, 0.5, fmt.Sprintf("[2/2] Searching GKE nodepools in Cloud Asset Inventory... (%d resources scanned)", len(clusterSearchResults)))

	nodePoolQuery := fmt.Sprintf("parentFullResourceName=%q", clusterAssetName)
	nodePoolSearchResults, err := fetcher.SearchResources(
		ctx,
		scope,
		nodePoolQuery,
		[]string{caik8s.GKENodePoolAssetType},
		func(scannedInQuery int) {
			progress.Report(ctx, 0.5, fmt.Sprintf("[2/2] Searching GKE nodepools in Cloud Asset Inventory... (%d resources scanned)", len(clusterSearchResults)+scannedInQuery))
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search GKE nodepools from CAI: %w", err)
	}
	progress.Report(ctx, 1.0, fmt.Sprintf("[2/2] Searching GKE nodepools in Cloud Asset Inventory... (%d resources scanned)", len(clusterSearchResults)+len(nodePoolSearchResults)))

	assetNames := make([]string, 0, 1+len(nodePoolSearchResults))
	assetNames = append(assetNames, clusterAssetName)
	for _, res := range nodePoolSearchResults {
		assetNames = append(assetNames, res.Name)
	}
	return assetNames, nil
}
