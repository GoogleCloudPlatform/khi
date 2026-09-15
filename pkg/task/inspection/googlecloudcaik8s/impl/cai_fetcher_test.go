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
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"
	"testing"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// defaultSearchResults is the canned SearchAllResources page used when a test does not configure its own.
var defaultSearchResults = []*assetpb.ResourceSearchResult{
	{Name: "res-1", AssetType: "k8s.io/Pod"},
	{Name: "res-2", AssetType: "k8s.io/Pod"},
}

// mockAssetServer is an in-process AssetService serving canned responses.
// The response fields must be set before the first RPC reaches the server.
type mockAssetServer struct {
	assetpb.UnimplementedAssetServiceServer

	// searchResultsByQuery returns the results of SearchAllResources per search query.
	// When nil, defaultSearchResults is returned, and an unlisted query yields no result.
	searchResultsByQuery map[string][]*assetpb.ResourceSearchResult
	// searchErr is returned by SearchAllResources instead of any result when set.
	searchErr error
	// batchAssets is returned by BatchGetAssetsHistory. When nil, the requested asset names are echoed back.
	batchAssets []*assetpb.TemporalAsset
	// batchErr is returned by BatchGetAssetsHistory instead of any asset when set.
	batchErr error

	mu               sync.Mutex
	batchRequests    []*assetpb.BatchGetAssetsHistoryRequest
	incomingMetadata []metadata.MD
	searchRequests   []*assetpb.SearchAllResourcesRequest
	onBatchRequest   func(req *assetpb.BatchGetAssetsHistoryRequest)
}

// recordedBatchRequests returns a snapshot of the BatchGetAssetsHistory requests observed so far.
func (m *mockAssetServer) recordedBatchRequests() []*assetpb.BatchGetAssetsHistoryRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.batchRequests)
}

// recordedSearchRequests returns a snapshot of the SearchAllResources requests observed so far.
func (m *mockAssetServer) recordedSearchRequests() []*assetpb.SearchAllResourcesRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.searchRequests)
}

// recordedMetadata returns a snapshot of the gRPC metadata observed so far.
func (m *mockAssetServer) recordedMetadata() []metadata.MD {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.incomingMetadata)
}

func (m *mockAssetServer) BatchGetAssetsHistory(ctx context.Context, req *assetpb.BatchGetAssetsHistoryRequest) (*assetpb.BatchGetAssetsHistoryResponse, error) {
	m.mu.Lock()
	m.batchRequests = append(m.batchRequests, req)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		m.incomingMetadata = append(m.incomingMetadata, md)
	}
	callback := m.onBatchRequest
	m.mu.Unlock()

	if callback != nil {
		callback(req)
	}

	if m.batchErr != nil {
		return nil, m.batchErr
	}
	if m.batchAssets != nil {
		return &assetpb.BatchGetAssetsHistoryResponse{Assets: m.batchAssets}, nil
	}

	resp := &assetpb.BatchGetAssetsHistoryResponse{}
	for _, name := range req.AssetNames {
		resp.Assets = append(resp.Assets, &assetpb.TemporalAsset{
			Asset: &assetpb.Asset{
				Name: name,
			},
		})
	}
	return resp, nil
}

func (m *mockAssetServer) SearchAllResources(ctx context.Context, req *assetpb.SearchAllResourcesRequest) (*assetpb.SearchAllResourcesResponse, error) {
	m.mu.Lock()
	m.searchRequests = append(m.searchRequests, req)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		m.incomingMetadata = append(m.incomingMetadata, md)
	}
	m.mu.Unlock()

	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if m.searchResultsByQuery != nil {
		return &assetpb.SearchAllResourcesResponse{Results: m.searchResultsByQuery[req.Query]}, nil
	}
	return &assetpb.SearchAllResourcesResponse{Results: defaultSearchResults}, nil
}

func setupMockServer(t *testing.T, srv *mockAssetServer) *googlecloud.ClientFactory {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	grpcServer := grpc.NewServer()
	assetpb.RegisterAssetServiceServer(grpcServer, srv)
	go func() {
		_ = grpcServer.Serve(listener)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	factory, err := googlecloud.NewClientFactory(func(f *googlecloud.ClientFactory) error {
		f.ClientOptions = append(f.ClientOptions, func(opts []option.ClientOption, c googlecloud.ResourceContainer) ([]option.ClientOption, error) {
			return append(opts,
				option.WithEndpoint(listener.Addr().String()),
				option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "dummy"})),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			), nil
		})
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create client factory: %v", err)
	}
	t.Cleanup(func() {
		_ = factory.Close()
	})

	return factory
}

func generateAssetNames(count int) []string {
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = fmt.Sprintf("//container.googleapis.com/projects/p/zones/z/clusters/c/k8s/namespaces/default/pods/pod-%d", i)
	}
	return names
}

func TestCAIFetcher_BatchGetAssetsHistory_Chunking(t *testing.T) {
	testCases := []struct {
		name              string
		assetCount        int
		wantRequestsCount int
		wantChunkSizes    []int
	}{
		{
			name:              "0 assets produces no requests",
			assetCount:        0,
			wantRequestsCount: 0,
			wantChunkSizes:    nil,
		},
		{
			name:              "50 assets produces single request",
			assetCount:        50,
			wantRequestsCount: 1,
			wantChunkSizes:    []int{50},
		},
		{
			name:              "100 assets produces single request",
			assetCount:        100,
			wantRequestsCount: 1,
			wantChunkSizes:    []int{100},
		},
		{
			name:              "101 assets chunks into 100 and 1",
			assetCount:        101,
			wantRequestsCount: 2,
			wantChunkSizes:    []int{100, 1},
		},
		{
			name:              "250 assets chunks into 100, 100, and 50",
			assetCount:        250,
			wantRequestsCount: 3,
			wantChunkSizes:    []int{100, 100, 50},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := &mockAssetServer{}
			factory := setupMockServer(t, mockServer)

			fetcher := NewCAIFetcher(factory, googlecloud.NewCallOptionInjector(), "test-project")
			inputNames := generateAssetNames(tc.assetCount)

			var gotProgress [][2]int
			onProgress := func(completedChunks, totalChunks int) {
				gotProgress = append(gotProgress, [2]int{completedChunks, totalChunks})
			}

			assets, err := fetcher.BatchGetAssetsHistory(t.Context(), "projects/test-project", inputNames, assetpb.ContentType_RESOURCE, nil, onProgress)
			if err != nil {
				t.Fatalf("BatchGetAssetsHistory() unexpected error: %v", err)
			}

			gotBatchRequests := mockServer.recordedBatchRequests()
			if len(gotBatchRequests) != tc.wantRequestsCount {
				t.Errorf("request count mismatch: got %d, want %d", len(gotBatchRequests), tc.wantRequestsCount)
			}

			var gotChunkSizes []int
			for _, req := range gotBatchRequests {
				gotChunkSizes = append(gotChunkSizes, len(req.AssetNames))
				if len(req.AssetNames) > 100 {
					t.Errorf("chunk exceeded max 100 items: got %d", len(req.AssetNames))
				}
			}

			if diff := cmp.Diff(tc.wantChunkSizes, gotChunkSizes); diff != "" {
				t.Errorf("chunk sizes mismatch (-want +got):\n%s", diff)
			}

			if len(assets) != tc.assetCount {
				t.Errorf("returned assets count mismatch: got %d, want %d", len(assets), tc.assetCount)
			}

			if tc.wantRequestsCount > 0 {
				var wantProgress [][2]int
				for i := 1; i <= tc.wantRequestsCount; i++ {
					wantProgress = append(wantProgress, [2]int{i, tc.wantRequestsCount})
				}
				if diff := cmp.Diff(wantProgress, gotProgress); diff != "" {
					t.Errorf("progress mismatch (-want +got):\n%s", diff)
				}
			} else if len(gotProgress) != 0 {
				t.Errorf("expected no progress calls, got: %v", gotProgress)
			}
		})
	}
}

func TestCAIFetcher_QuotaProjectResolution(t *testing.T) {
	customQuotaProject := "custom-quota-project"
	emptyQuotaProject := ""

	testCases := []struct {
		name               string
		authQuotaProjectID *string
		defaultProjectID   string
		wantUserProject    string
	}{
		{
			name:               "prioritizes parameters.Auth.QuotaProjectID when set",
			authQuotaProjectID: &customQuotaProject,
			defaultProjectID:   "cluster-project",
			wantUserProject:    "custom-quota-project",
		},
		{
			name:               "falls back to cluster project when Auth.QuotaProjectID is nil",
			authQuotaProjectID: nil,
			defaultProjectID:   "cluster-project",
			wantUserProject:    "cluster-project",
		},
		{
			name:               "falls back to cluster project when Auth.QuotaProjectID is empty",
			authQuotaProjectID: &emptyQuotaProject,
			defaultProjectID:   "cluster-project",
			wantUserProject:    "cluster-project",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			origQuota := parameters.Auth.QuotaProjectID
			parameters.Auth.QuotaProjectID = tc.authQuotaProjectID
			t.Cleanup(func() {
				parameters.Auth.QuotaProjectID = origQuota
			})

			mockServer := &mockAssetServer{}
			factory := setupMockServer(t, mockServer)

			fetcher := NewCAIFetcher(factory, googlecloud.NewCallOptionInjector(), tc.defaultProjectID)
			_, err := fetcher.BatchGetAssetsHistory(t.Context(), fmt.Sprintf("projects/%s", tc.defaultProjectID), []string{"asset-1"}, assetpb.ContentType_RESOURCE, nil, nil)
			if err != nil {
				t.Fatalf("BatchGetAssetsHistory() unexpected error: %v", err)
			}

			gotMetadata := mockServer.recordedMetadata()
			if len(gotMetadata) == 0 {
				t.Fatalf("expected incoming metadata to be captured")
			}

			md := gotMetadata[0]
			gotUserProjects := md.Get("x-goog-user-project")
			if len(gotUserProjects) == 0 {
				t.Fatalf("missing x-goog-user-project header in metadata")
			}

			if gotUserProjects[0] != tc.wantUserProject {
				t.Errorf("x-goog-user-project mismatch: got %s, want %s", gotUserProjects[0], tc.wantUserProject)
			}
		})
	}
}

func TestCAIFetcher_ContextCancellation(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T, fetcher googlecloudcaik8s_contract.CAIFetcher, mockServer *mockAssetServer)
	}{
		{
			name: "BatchGetAssetsHistory returns error immediately on pre-canceled context",
			test: func(t *testing.T, fetcher googlecloudcaik8s_contract.CAIFetcher, mockServer *mockAssetServer) {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				_, err := fetcher.BatchGetAssetsHistory(ctx, "projects/test-project", generateAssetNames(50), assetpb.ContentType_RESOURCE, nil, nil)
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled error, got: %v", err)
				}
				if got := len(mockServer.recordedBatchRequests()); got != 0 {
					t.Errorf("expected 0 requests to reach server, got: %d", got)
				}
			},
		},
		{
			name: "BatchGetAssetsHistory stops subsequent chunks when canceled mid-flight",
			test: func(t *testing.T, fetcher googlecloudcaik8s_contract.CAIFetcher, mockServer *mockAssetServer) {
				ctx, cancel := context.WithCancel(t.Context())

				mockServer.mu.Lock()
				mockServer.onBatchRequest = func(req *assetpb.BatchGetAssetsHistoryRequest) {
					// Cancel the context after the first chunk arrives at the server.
					cancel()
				}
				mockServer.mu.Unlock()

				_, err := fetcher.BatchGetAssetsHistory(ctx, "projects/test-project", generateAssetNames(250), assetpb.ContentType_RESOURCE, nil, nil)
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled error, got: %v", err)
				}
				// It should have halted after the first chunk.
				if got := len(mockServer.recordedBatchRequests()); got != 1 {
					t.Errorf("expected exactly 1 chunk request before cancellation, got: %d", got)
				}
			},
		},
		{
			name: "SearchResources returns error immediately on pre-canceled context",
			test: func(t *testing.T, fetcher googlecloudcaik8s_contract.CAIFetcher, mockServer *mockAssetServer) {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				_, err := fetcher.SearchResources(ctx, "projects/test-project", "name:*", []string{"k8s.io/Pod"})
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled error, got: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := &mockAssetServer{}
			factory := setupMockServer(t, mockServer)
			fetcher := NewCAIFetcher(factory, googlecloud.NewCallOptionInjector(), "test-project")
			tc.test(t, fetcher, mockServer)
		})
	}
}

func TestCAIFetcher_SearchResources(t *testing.T) {
	testCases := []struct {
		name       string
		scope      string
		query      string
		assetTypes []string
		wantCount  int
		wantScope  string
		wantQuery  string
	}{
		{
			name:       "searches resources with given scope, query, and asset types",
			scope:      "projects/test-project",
			query:      "name:*",
			assetTypes: []string{"k8s.io/Pod"},
			wantCount:  2,
			wantScope:  "projects/test-project",
			wantQuery:  "name:*",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := &mockAssetServer{}
			factory := setupMockServer(t, mockServer)
			fetcher := NewCAIFetcher(factory, googlecloud.NewCallOptionInjector(), "test-project")

			results, err := fetcher.SearchResources(t.Context(), tc.scope, tc.query, tc.assetTypes)
			if err != nil {
				t.Fatalf("SearchResources() unexpected error: %v", err)
			}

			if len(results) != tc.wantCount {
				t.Errorf("SearchResources() result count = %d, want %d", len(results), tc.wantCount)
			}
			gotSearchRequests := mockServer.recordedSearchRequests()
			if len(gotSearchRequests) != 1 {
				t.Errorf("search request count = %d, want 1", len(gotSearchRequests))
			}
			req := gotSearchRequests[0]
			if req.Scope != tc.wantScope {
				t.Errorf("request scope = %s, want %s", req.Scope, tc.wantScope)
			}
			if req.Query != tc.wantQuery {
				t.Errorf("request query = %s, want %s", req.Query, tc.wantQuery)
			}
			if diff := cmp.Diff(tc.assetTypes, req.AssetTypes); diff != "" {
				t.Errorf("request assetTypes mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
