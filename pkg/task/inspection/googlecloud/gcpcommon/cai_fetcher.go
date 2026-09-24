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

package gcpcommon

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	asset "cloud.google.com/go/asset/apiv1"
	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// MaxCAIBatchHistorySize is the maximum number of asset names allowed in a single BatchGetAssetsHistory request.
	MaxCAIBatchHistorySize = 100

	// MaxCAISearchQueryComparisons is the maximum number of comparison clauses per CAI search query.
	MaxCAISearchQueryComparisons = 10
	// MaxCAISearchQueryCharacters is the maximum number of characters per CAI search query.
	MaxCAISearchQueryCharacters = 2048
)

// CAISearchProgressCallback is invoked each time a resource is scanned during a CAI search.
// scannedInQuery is the cumulative count of resources scanned so far within the current SearchResources call.
type CAISearchProgressCallback func(scannedInQuery int)

// CAIFetcher defines operations to query Google Cloud Asset Inventory.
type CAIFetcher interface {
	// SearchResources searches for assets under the given scope matching query and asset types.
	// If onProgress is non-nil, it is invoked each time a resource is yielded from the CAI iterator.
	SearchResources(ctx context.Context, scope, query string, assetTypes []string, onProgress CAISearchProgressCallback) ([]*assetpb.ResourceSearchResult, error)

	// BatchGetAssetsHistory retrieves historical temporal snapshots for specified asset names.
	// Implementations automatically chunk assetNames exceeding Cloud Asset Inventory limits (max 100 per request).
	BatchGetAssetsHistory(ctx context.Context, parent string, assetNames []string, contentType assetpb.ContentType, timeWindow *assetpb.TimeWindow) ([]*assetpb.TemporalAsset, error)
}

// caiFetcher implements CAIFetcher to query Google Cloud Asset Inventory.
type caiFetcher struct {
	factory            *googlecloud.ClientFactory
	callOptionInjector *googlecloud.CallOptionInjector
	defaultProjectID   string
}

var _ CAIFetcher = (*caiFetcher)(nil)

// NewCAIFetcher creates a new CAIFetcher with the given client factory and call option injector.
func NewCAIFetcher(factory *googlecloud.ClientFactory, callOptionInjector *googlecloud.CallOptionInjector, defaultProjectID string) CAIFetcher {
	return &caiFetcher{
		factory:            factory,
		callOptionInjector: callOptionInjector,
		defaultProjectID:   defaultProjectID,
	}
}

// SearchResources searches for assets under the given scope matching query and asset types.
func (f *caiFetcher) SearchResources(ctx context.Context, scope, query string, assetTypes []string, onProgress CAISearchProgressCallback) ([]*assetpb.ResourceSearchResult, error) {
	container := f.resourceContainer()
	callCtx := ctx
	if f.callOptionInjector != nil {
		callCtx = f.callOptionInjector.InjectToCallContext(ctx, container)
	}

	client, err := f.factory.AssetClient(callCtx, container, f.clientOptions()...)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset client: %w", err)
	}

	req := &assetpb.SearchAllResourcesRequest{
		Scope:      scope,
		Query:      query,
		AssetTypes: assetTypes,
	}
	it := client.SearchAllResources(callCtx, req)
	var results []*assetpb.ResourceSearchResult
	for {
		res, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if callCtx.Err() != nil {
				return nil, callCtx.Err()
			}
			return nil, fmt.Errorf("error iterating search resources: %w", err)
		}
		results = append(results, res)
		if onProgress != nil {
			onProgress(len(results))
		}
	}
	return results, nil
}

// BatchGetAssetsHistory retrieves historical temporal snapshots for specified asset names.
func (f *caiFetcher) BatchGetAssetsHistory(ctx context.Context, parent string, assetNames []string, contentType assetpb.ContentType, timeWindow *assetpb.TimeWindow) ([]*assetpb.TemporalAsset, error) {
	if len(assetNames) == 0 {
		return nil, nil
	}

	container := f.resourceContainer()
	callCtx := ctx
	if f.callOptionInjector != nil {
		callCtx = f.callOptionInjector.InjectToCallContext(ctx, container)
	}

	client, err := f.factory.AssetClient(callCtx, container, f.clientOptions()...)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset client: %w", err)
	}

	var allAssets []*assetpb.TemporalAsset
	totalAssets := len(assetNames)
	startTime := time.Now()
	progress.Report(ctx, 0, formatAssetHistoryProgress(0, totalAssets, 0, 0))

	for i := 0; i < totalAssets; i += MaxCAIBatchHistorySize {
		end := min(i+MaxCAIBatchHistorySize, totalAssets)
		assets, err := f.fetchAssetHistoryChunk(callCtx, client, parent, assetNames[i:end], contentType, timeWindow)
		if err != nil {
			return nil, err
		}
		allAssets = append(allAssets, assets...)
		ratio := float32(end) / float32(totalAssets)
		progress.Report(ctx, ratio, formatAssetHistoryProgress(end, totalAssets, len(allAssets), time.Since(startTime)))
	}
	return allAssets, nil
}

func (f *caiFetcher) fetchAssetHistoryChunk(callCtx context.Context, client *asset.Client, parent string, chunk []string, contentType assetpb.ContentType, timeWindow *assetpb.TimeWindow) ([]*assetpb.TemporalAsset, error) {
	req := &assetpb.BatchGetAssetsHistoryRequest{
		Parent:         parent,
		AssetNames:     chunk,
		ContentType:    contentType,
		ReadTimeWindow: timeWindow,
	}
	resp, err := client.BatchGetAssetsHistory(callCtx, req)
	if err != nil {
		if callCtx.Err() != nil {
			return nil, callCtx.Err()
		}
		return nil, fmt.Errorf("failed during batch get assets history: %w", err)
	}
	return resp.Assets, nil
}

// resolveQuotaProjectID determines the quota project to use, prioritizing parameters.Auth.QuotaProjectID.
func (f *caiFetcher) resolveQuotaProjectID() string {
	if parameters.Auth.QuotaProjectID != nil && *parameters.Auth.QuotaProjectID != "" {
		return *parameters.Auth.QuotaProjectID
	}
	return f.defaultProjectID
}

// clientOptions generates the client options for CAI API, including quota project settings.
func (f *caiFetcher) clientOptions() []option.ClientOption {
	quotaProject := f.resolveQuotaProjectID()
	if quotaProject != "" {
		return []option.ClientOption{option.WithQuotaProject(quotaProject)}
	}
	return nil
}

// resourceContainer returns a ResourceContainer representing the project for the call.
func (f *caiFetcher) resourceContainer() googlecloud.ResourceContainer {
	return googlecloud.Project(f.defaultProjectID)
}

// CAIParentSearchQuery represents a single CAI search query expression and the number of parent resources it covers.
type CAIParentSearchQuery struct {
	Query       string
	ParentCount int
}

// BuildCAIParentSearchQueries builds the CAI search expressions matching assets whose parent is
// exactly one of the given full resource names, splitting them to stay within the CAI query limits.
func BuildCAIParentSearchQueries(parentAssetNames []string) []CAIParentSearchQuery {
	var queries []CAIParentSearchQuery
	var comparisons []string
	characterCount := 0
	for _, parent := range parentAssetNames {
		comparison := fmt.Sprintf("parentFullResourceName=%q", parent)
		addedCharacters := len(comparison)
		if len(comparisons) > 0 {
			addedCharacters += len(" OR ")
		}
		if len(comparisons) > 0 && (len(comparisons) == MaxCAISearchQueryComparisons || characterCount+addedCharacters > MaxCAISearchQueryCharacters) {
			queries = append(queries, CAIParentSearchQuery{
				Query:       strings.Join(comparisons, " OR "),
				ParentCount: len(comparisons),
			})
			comparisons = nil
			characterCount = 0
			addedCharacters = len(comparison)
		}
		comparisons = append(comparisons, comparison)
		characterCount += addedCharacters
	}
	if len(comparisons) > 0 {
		queries = append(queries, CAIParentSearchQuery{
			Query:       strings.Join(comparisons, " OR "),
			ParentCount: len(comparisons),
		})
	}
	return queries
}

// CAIParentSearchProgressConfig configures progress reporting for SearchCAIAssetsByParents.
type CAIParentSearchProgressConfig struct {
	// Label is the human-readable description of the search phase.
	Label string
	// ParentUnit is the unit label for parentAssetNames when reporting determinate progress (e.g., "namespaces").
	// When empty, progress is reported as indeterminate with a live scanned resource counter.
	ParentUnit string
}

// SearchCAIAssetsByParents builds parent search queries for the given parent full resource names
// and runs one CAI search per query, returning the concatenated results.
func SearchCAIAssetsByParents(
	ctx context.Context,
	fetcher CAIFetcher,
	scope string,
	assetTypes []string,
	parentAssetNames []string,
	progressCfg CAIParentSearchProgressConfig,
) ([]*assetpb.ResourceSearchResult, error) {
	if len(parentAssetNames) == 0 {
		return nil, nil
	}
	queries := BuildCAIParentSearchQueries(parentAssetNames)
	totalParents := len(parentAssetNames)
	completedParents := 0
	startTime := time.Now()

	reportSearchProgress := func(scannedCount int) {
		if progressCfg.Label == "" {
			return
		}
		elapsed := time.Since(startTime)
		if progressCfg.ParentUnit != "" && totalParents > 0 {
			ratio := float32(completedParents) / float32(totalParents)
			progress.Report(ctx, ratio, formatDeterminateParentSearchProgress(
				progressCfg.Label,
				completedParents,
				totalParents,
				progressCfg.ParentUnit,
				scannedCount,
				elapsed,
			))
			return
		}
		progress.ReportIndeterminate(ctx, formatIndeterminateSearchProgress(progressCfg.Label, scannedCount, elapsed))
	}

	reportSearchProgress(0)

	var results []*assetpb.ResourceSearchResult
	for _, q := range queries {
		baseCount := len(results)
		searchResults, err := fetcher.SearchResources(ctx, scope, q.Query, assetTypes, func(scannedInQuery int) {
			reportSearchProgress(baseCount + scannedInQuery)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search assets with query %q: %w", q.Query, err)
		}
		results = append(results, searchResults...)
		completedParents += q.ParentCount
		reportSearchProgress(len(results))
	}
	return results, nil
}

// formatIndeterminateSearchProgress formats progress message for an indeterminate search phase.
func formatIndeterminateSearchProgress(label string, scannedCount int, elapsed time.Duration) string {
	if elapsed >= time.Second && scannedCount > 0 {
		rate := float64(scannedCount) / elapsed.Seconds()
		return fmt.Sprintf("%s... (%d resources scanned, %.1f resources/s)", label, scannedCount, rate)
	}
	return fmt.Sprintf("%s... (%d resources scanned)", label, scannedCount)
}

// formatDeterminateParentSearchProgress formats progress message for a parent search phase with known total parents.
func formatDeterminateParentSearchProgress(label string, completedParents, totalParents int, parentUnit string, scannedCount int, elapsed time.Duration) string {
	base := fmt.Sprintf("%s: %d/%d %s, %d resources scanned", label, completedParents, totalParents, parentUnit, scannedCount)
	if elapsed >= time.Second && scannedCount > 0 {
		rate := float64(scannedCount) / elapsed.Seconds()
		return fmt.Sprintf("%s (%.1f resources/s)", base, rate)
	}
	return base
}

// formatAssetHistoryProgress formats progress message for BatchGetAssetsHistory chunk processing.
func formatAssetHistoryProgress(completedAssets, totalAssets, snapshotCount int, elapsed time.Duration) string {
	base := fmt.Sprintf("Fetching asset history: %d/%d assets, %d snapshots", completedAssets, totalAssets, snapshotCount)
	if elapsed >= time.Second && completedAssets > 0 {
		rate := float64(completedAssets) / elapsed.Seconds()
		var ratio float32
		if totalAssets > 0 {
			ratio = float32(completedAssets) / float32(totalAssets)
		}
		if ratio > 0.005 && ratio < 1.0 {
			eta := time.Duration(elapsed.Seconds() * float64(1.0-ratio) / float64(ratio) * float64(time.Second)).Round(time.Second)
			return fmt.Sprintf("%s (%.1f assets/s, ETA %s)", base, rate, eta)
		}
		return fmt.Sprintf("%s (%.1f assets/s)", base, rate)
	}
	return base
}

// ConvertTemporalAssetsToCAISnapshots converts a slice of TemporalAsset into sorted CAIAssetSnapshot entries.
func ConvertTemporalAssetsToCAISnapshots(temporalAssets []*assetpb.TemporalAsset) []*CAIAssetSnapshot {
	snapshots := make([]*CAIAssetSnapshot, 0, len(temporalAssets))
	for _, ta := range temporalAssets {
		if ta == nil || ta.Asset == nil {
			continue
		}
		snapshots = append(snapshots, &CAIAssetSnapshot{
			TemporalAsset: ta,
		})
	}
	slices.SortStableFunc(snapshots, func(a, b *CAIAssetSnapshot) int {
		return a.StartTime().Compare(b.StartTime())
	})
	return snapshots
}

// FetchCAIAssetSnapshots discovers asset names using discover, fetches their temporal history,
// and returns sorted CAIAssetSnapshot entries.
func FetchCAIAssetSnapshots(
	ctx context.Context,
	fetcher CAIFetcher,
	scope string,
	startTime, endTime time.Time,
	discover func(ctx context.Context, fetcher CAIFetcher) ([]string, error),
) ([]*CAIAssetSnapshot, error) {
	assetNames, err := discover(ctx, fetcher)
	if err != nil {
		return nil, err
	}
	if len(assetNames) == 0 {
		return []*CAIAssetSnapshot{}, nil
	}

	timeWindow := &assetpb.TimeWindow{
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
	}

	temporalAssets, err := fetcher.BatchGetAssetsHistory(ctx, scope, assetNames, assetpb.ContentType_RESOURCE, timeWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to batch get assets history from CAI: %w", err)
	}

	return ConvertTemporalAssetsToCAISnapshots(temporalAssets), nil
}
