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

package caik8s

import (
	"context"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
)

// CAIFetcher defines generic operations to query Google Cloud Asset Inventory.
// It is completely agnostic to Kubernetes-specific concepts.
type CAIFetcher interface {
	// SearchResources searches for assets under the given scope matching query and asset types.
	SearchResources(ctx context.Context, scope, query string, assetTypes []string) ([]*assetpb.ResourceSearchResult, error)

	// BatchGetAssetsHistory retrieves historical temporal snapshots for specified asset names.
	// Implementations must automatically chunk assetNames exceeding Cloud Asset Inventory limits (max 100 per request).
	// onProgress is called after each chunk completes with the number of completed chunks and total chunks.
	BatchGetAssetsHistory(ctx context.Context, parent string, assetNames []string, contentType assetpb.ContentType, timeWindow *assetpb.TimeWindow, onProgress func(completedChunks, totalChunks int)) ([]*assetpb.TemporalAsset, error)
}
