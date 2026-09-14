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

package googlecloudcaik8s_impl

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GKEResourceFetcherTask queries CAI for GKE Cluster and NodePool temporal asset snapshots.
var GKEResourceFetcherTask = inspectiontaskbase.NewProgressReportableInspectionTask(
	googlecloudcaik8s_contract.GKEResourceFetcherTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(),
		googlecloudcommon_contract.APIClientFactoryTaskID.Ref(),
		googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(),
		googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
		googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*googlecloudcaik8s_contract.GKEResourceSnapshot, error) {
		cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref())
		factory := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
		injector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())
		startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
		endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())

		if taskMode == inspectioncore_contract.TaskModeDryRun || !cluster.IsComplete() {
			return []*googlecloudcaik8s_contract.GKEResourceSnapshot{}, nil
		}

		fetcher := NewCAIFetcher(factory, injector, cluster.ProjectID)

		snapshots, err := fetchGKEResourceSnapshots(ctx, fetcher, cluster, startTime, endTime, progress)
		if err != nil {
			slog.WarnContext(ctx, "failed to fetch GKE resource snapshots from CAI", "error", err)
			return []*googlecloudcaik8s_contract.GKEResourceSnapshot{}, nil
		}
		return snapshots, nil
	},
)

// fetchGKEResourceSnapshots searches CAI for the cluster and its nodepools, then retrieves their history.
func fetchGKEResourceSnapshots(
	ctx context.Context,
	fetcher googlecloudcaik8s_contract.CAIFetcher,
	cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity,
	startTime, endTime time.Time,
	progress *inspectionmetadata.TaskProgressMetadata,
) ([]*googlecloudcaik8s_contract.GKEResourceSnapshot, error) {
	scope := fmt.Sprintf("projects/%s", cluster.ProjectID)
	progress.Indeterminate = true
	progress.Message = "Searching GKE cluster in Cloud Asset Inventory..."

	candidates := clusterParentCandidates(cluster)
	clusterQuery := fmt.Sprintf("name=%q OR name=%q", candidates[0], candidates[1])
	clusterSearchResults, err := fetcher.SearchResources(ctx, scope, clusterQuery, []string{googlecloudcaik8s_contract.GKEClusterAssetType})
	if err != nil {
		return nil, fmt.Errorf("failed to search GKE cluster from CAI: %w", err)
	}
	if len(clusterSearchResults) == 0 {
		return []*googlecloudcaik8s_contract.GKEResourceSnapshot{}, nil
	}

	clusterAssetName := clusterSearchResults[0].Name
	progress.Message = "Searching GKE nodepools in Cloud Asset Inventory..."

	nodePoolQuery := fmt.Sprintf("parentFullResourceName=%q", clusterAssetName)
	nodePoolSearchResults, err := fetcher.SearchResources(ctx, scope, nodePoolQuery, []string{googlecloudcaik8s_contract.GKENodePoolAssetType})
	if err != nil {
		return nil, fmt.Errorf("failed to search GKE nodepools from CAI: %w", err)
	}

	assetNames := make([]string, 0, 1+len(nodePoolSearchResults))
	assetNames = append(assetNames, clusterAssetName)
	for _, res := range nodePoolSearchResults {
		assetNames = append(assetNames, res.Name)
	}

	totalChunks := (len(assetNames) + maxBatchHistorySize - 1) / maxBatchHistorySize
	progress.Indeterminate = false
	progress.Percentage = 0.0
	progress.Message = fmt.Sprintf("Fetching GKE asset history (0/%d chunks, %d assets)...", totalChunks, len(assetNames))

	timeWindow := &assetpb.TimeWindow{
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
	}

	temporalAssets, err := fetcher.BatchGetAssetsHistory(ctx, scope, assetNames, assetpb.ContentType_RESOURCE, timeWindow, func(completedChunks, totalChunks int) {
		progress.Percentage = float32(completedChunks) / float32(totalChunks)
		progress.Message = fmt.Sprintf("Fetching GKE asset history (%d/%d chunks, %d assets)...", completedChunks, totalChunks, len(assetNames))
	})
	if err != nil {
		return nil, fmt.Errorf("failed to batch get GKE assets history from CAI: %w", err)
	}

	return convertTemporalAssetsToGKEResourceSnapshots(temporalAssets), nil
}

// convertTemporalAssetsToGKEResourceSnapshots converts TemporalAsset slice into GKEResourceSnapshot slice.
func convertTemporalAssetsToGKEResourceSnapshots(temporalAssets []*assetpb.TemporalAsset) []*googlecloudcaik8s_contract.GKEResourceSnapshot {
	snapshots := make([]*googlecloudcaik8s_contract.GKEResourceSnapshot, 0, len(temporalAssets))
	for _, ta := range temporalAssets {
		if ta == nil || ta.Asset == nil {
			continue
		}
		var startTime time.Time
		if st := ta.GetWindow().GetStartTime(); st != nil {
			startTime = st.AsTime()
		}
		snapshots = append(snapshots, &googlecloudcaik8s_contract.GKEResourceSnapshot{
			TemporalAsset: ta,
			StartTime:     startTime,
		})
	}
	return snapshots
}
