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
	"errors"
	"fmt"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const maxBatchHistorySize = 100

// caiFetcher implements googlecloudcaik8s_contract.CAIFetcher to query Google Cloud Asset Inventory.
type caiFetcher struct {
	factory            *googlecloud.ClientFactory
	callOptionInjector *googlecloud.CallOptionInjector
	defaultProjectID   string
}

var _ caik8s.CAIFetcher = (*caiFetcher)(nil)

// NewCAIFetcher creates a new generic CAIFetcher with the given client factory and call option injector.
func NewCAIFetcher(factory *googlecloud.ClientFactory, callOptionInjector *googlecloud.CallOptionInjector, defaultProjectID string) caik8s.CAIFetcher {
	return &caiFetcher{
		factory:            factory,
		callOptionInjector: callOptionInjector,
		defaultProjectID:   defaultProjectID,
	}
}

// SearchResources searches for assets under the given scope matching query and asset types.
func (f *caiFetcher) SearchResources(ctx context.Context, scope, query string, assetTypes []string) ([]*assetpb.ResourceSearchResult, error) {
	resourceContainer := f.resourceContainer()
	client, err := f.factory.AssetClient(ctx, resourceContainer, f.clientOptions()...)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset client: %w", err)
	}

	callCtx := f.callOptionInjector.InjectToCallContext(ctx, resourceContainer)

	req := &assetpb.SearchAllResourcesRequest{
		Scope:      scope,
		Query:      query,
		AssetTypes: assetTypes,
	}

	it := client.SearchAllResources(callCtx, req)
	var results []*assetpb.ResourceSearchResult
	for {
		select {
		case <-callCtx.Done():
			return nil, callCtx.Err()
		default:
		}

		result, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if callCtx.Err() != nil {
				return nil, callCtx.Err()
			}
			return nil, fmt.Errorf("failed to search all resources: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// BatchGetAssetsHistory retrieves historical temporal snapshots for specified asset names.
// It automatically chunks assetNames exceeding Cloud Asset Inventory limits (max 100 per request).
func (f *caiFetcher) BatchGetAssetsHistory(ctx context.Context, parent string, assetNames []string, contentType assetpb.ContentType, timeWindow *assetpb.TimeWindow, onProgress func(completedChunks, totalChunks int)) ([]*assetpb.TemporalAsset, error) {
	if len(assetNames) == 0 {
		return nil, nil
	}

	resourceContainer := f.resourceContainer()
	client, err := f.factory.AssetClient(ctx, resourceContainer, f.clientOptions()...)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset client: %w", err)
	}

	callCtx := f.callOptionInjector.InjectToCallContext(ctx, resourceContainer)

	totalChunks := (len(assetNames) + maxBatchHistorySize - 1) / maxBatchHistorySize
	var allAssets []*assetpb.TemporalAsset
	for i := 0; i < len(assetNames); i += maxBatchHistorySize {
		select {
		case <-callCtx.Done():
			return nil, callCtx.Err()
		default:
		}

		chunk := assetNames[i:min(i+maxBatchHistorySize, len(assetNames))]

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
		allAssets = append(allAssets, resp.Assets...)
		if onProgress != nil {
			onProgress(i/maxBatchHistorySize+1, totalChunks)
		}
	}
	return allAssets, nil
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
