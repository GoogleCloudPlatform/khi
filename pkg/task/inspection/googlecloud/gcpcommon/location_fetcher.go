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

package gcpcommon

import (
	"context"
	"errors"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
)

var _ LocationFetcher = (*locationFetcherImpl)(nil)

// LocationFetcher fetches available Compute Engine regions for a Google Cloud project.
type LocationFetcher interface {
	// FetchRegions returns the list of Compute Engine region names available in the given project.
	FetchRegions(ctx context.Context, projectId string) ([]string, error)
}

type locationFetcherImpl struct {
	client             *compute.RegionsClient
	callOptionInjector *googlecloud.CallOptionInjector
}

// NewLocationFetcher creates a LocationFetcher backed by the given Compute Engine RegionsClient.
func NewLocationFetcher(client *compute.RegionsClient, callOptionInjector *googlecloud.CallOptionInjector) LocationFetcher {
	return &locationFetcherImpl{
		client:             client,
		callOptionInjector: callOptionInjector,
	}
}

// FetchRegions implements LocationFetcher.
func (l *locationFetcherImpl) FetchRegions(ctx context.Context, projectId string) ([]string, error) {
	ctx = l.callOptionInjector.InjectToCallContext(ctx, googlecloud.Project(projectId))
	iter := l.client.List(ctx, &computepb.ListRegionsRequest{
		Project: projectId,
	})

	var result []string
	for {
		region, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, googlecloud.WrapPermissionDenied(err, googlecloud.Project(projectId).Identifier(), "compute.regions.list")
		}
		if region != nil {
			result = append(result, *region.Name)
		}
	}
	return result, nil
}
