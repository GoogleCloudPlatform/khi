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
	"net"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var defaultCAISearchResults = []*assetpb.ResourceSearchResult{
	{Name: "res-1", AssetType: "k8s.io/Pod"},
	{Name: "res-2", AssetType: "k8s.io/Pod"},
}

type mockCAIAssetServer struct {
	assetpb.UnimplementedAssetServiceServer

	searchResultsByQuery map[string][]*assetpb.ResourceSearchResult
	searchErr            error
	batchAssets          []*assetpb.TemporalAsset
	batchErr             error

	mu               sync.Mutex
	batchRequests    []*assetpb.BatchGetAssetsHistoryRequest
	incomingMetadata []metadata.MD
	searchRequests   []*assetpb.SearchAllResourcesRequest
	onBatchRequest   func(req *assetpb.BatchGetAssetsHistoryRequest)
}

func (m *mockCAIAssetServer) recordedBatchRequests() []*assetpb.BatchGetAssetsHistoryRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.batchRequests)
}

func (m *mockCAIAssetServer) recordedSearchRequests() []*assetpb.SearchAllResourcesRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.searchRequests)
}

func (m *mockCAIAssetServer) recordedMetadata() []metadata.MD {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.incomingMetadata)
}

func (m *mockCAIAssetServer) BatchGetAssetsHistory(ctx context.Context, req *assetpb.BatchGetAssetsHistoryRequest) (*assetpb.BatchGetAssetsHistoryResponse, error) {
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

func (m *mockCAIAssetServer) SearchAllResources(ctx context.Context, req *assetpb.SearchAllResourcesRequest) (*assetpb.SearchAllResourcesResponse, error) {
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
	return &assetpb.SearchAllResourcesResponse{Results: defaultCAISearchResults}, nil
}

func setupMockCAIServer(t *testing.T, srv *mockCAIAssetServer) *googlecloud.ClientFactory {
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

func generateTestCAIAssetNames(count int) []string {
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
			mockServer := &mockCAIAssetServer{}
			factory := setupMockCAIServer(t, mockServer)

			fetcher := NewCAIFetcher(factory, googlecloud.NewCallOptionInjector(), "test-project")
			inputNames := generateTestCAIAssetNames(tc.assetCount)

			tp := inspectionmetadata.NewTaskProgressMetadata("test-task")
			ctx := progress.WithContext(t.Context(), tp)

			assets, err := fetcher.BatchGetAssetsHistory(ctx, "projects/test-project", inputNames, assetpb.ContentType_RESOURCE, nil)
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

			var wantProgressRatio float32
			if tc.wantRequestsCount > 0 {
				wantProgressRatio = 1.0
			}
			gotRatio := tp.Snapshot().Ratio
			if gotRatio != wantProgressRatio {
				t.Errorf("progress ratio = %v, want %v", gotRatio, wantProgressRatio)
			}
			if tc.wantRequestsCount > 0 {
				wantMsg := fmt.Sprintf("Fetching asset history: %d/%d assets, %d snapshots", tc.assetCount, tc.assetCount, tc.assetCount)
				if !strings.Contains(tp.Snapshot().Message, wantMsg) {
					t.Errorf("progress message %q does not contain %q", tp.Snapshot().Message, wantMsg)
				}
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

			mockServer := &mockCAIAssetServer{}
			factory := setupMockCAIServer(t, mockServer)

			fetcher := NewCAIFetcher(factory, googlecloud.NewCallOptionInjector(), tc.defaultProjectID)
			_, err := fetcher.BatchGetAssetsHistory(t.Context(), fmt.Sprintf("projects/%s", tc.defaultProjectID), []string{"asset-1"}, assetpb.ContentType_RESOURCE, nil)
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
		test func(t *testing.T, fetcher CAIFetcher, mockServer *mockCAIAssetServer)
	}{
		{
			name: "BatchGetAssetsHistory returns error immediately on pre-canceled context",
			test: func(t *testing.T, fetcher CAIFetcher, mockServer *mockCAIAssetServer) {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				_, err := fetcher.BatchGetAssetsHistory(ctx, "projects/test-project", generateTestCAIAssetNames(50), assetpb.ContentType_RESOURCE, nil)
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
			test: func(t *testing.T, fetcher CAIFetcher, mockServer *mockCAIAssetServer) {
				ctx, cancel := context.WithCancel(t.Context())

				mockServer.mu.Lock()
				mockServer.onBatchRequest = func(req *assetpb.BatchGetAssetsHistoryRequest) {
					cancel()
				}
				mockServer.mu.Unlock()

				_, err := fetcher.BatchGetAssetsHistory(ctx, "projects/test-project", generateTestCAIAssetNames(250), assetpb.ContentType_RESOURCE, nil)
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled error, got: %v", err)
				}
				if got := len(mockServer.recordedBatchRequests()); got != 1 {
					t.Errorf("expected exactly 1 chunk request before cancellation, got: %d", got)
				}
			},
		},
		{
			name: "SearchResources returns error immediately on pre-canceled context",
			test: func(t *testing.T, fetcher CAIFetcher, mockServer *mockCAIAssetServer) {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				_, err := fetcher.SearchResources(ctx, "projects/test-project", "name:*", []string{"k8s.io/Pod"}, nil)
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled error, got: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := &mockCAIAssetServer{}
			factory := setupMockCAIServer(t, mockServer)
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
			mockServer := &mockCAIAssetServer{}
			factory := setupMockCAIServer(t, mockServer)
			fetcher := NewCAIFetcher(factory, googlecloud.NewCallOptionInjector(), "test-project")

			var gotProgress []int
			results, err := fetcher.SearchResources(t.Context(), tc.scope, tc.query, tc.assetTypes, func(scannedInQuery int) {
				gotProgress = append(gotProgress, scannedInQuery)
			})
			if err != nil {
				t.Fatalf("SearchResources() unexpected error: %v", err)
			}

			if len(results) != tc.wantCount {
				t.Errorf("SearchResources() result count = %d, want %d", len(results), tc.wantCount)
			}
			wantProgress := []int{1, 2}
			if diff := cmp.Diff(wantProgress, gotProgress); diff != "" {
				t.Errorf("onProgress calls mismatch (-want +got):\n%s", diff)
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

func TestCAIFetcher_Errors(t *testing.T) {
	testCases := []struct {
		name      string
		setupMock func(m *mockCAIAssetServer)
		call      func(fetcher CAIFetcher) error
		wantCode  codes.Code
	}{
		{
			name: "SearchResources returns permission denied error",
			setupMock: func(m *mockCAIAssetServer) {
				m.searchErr = status.Error(codes.PermissionDenied, "permission denied")
			},
			call: func(fetcher CAIFetcher) error {
				_, err := fetcher.SearchResources(t.Context(), "projects/test-project", "name:*", []string{"k8s.io/Pod"}, nil)
				return err
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name: "BatchGetAssetsHistory returns internal error",
			setupMock: func(m *mockCAIAssetServer) {
				m.batchErr = status.Error(codes.Internal, "internal error")
			},
			call: func(fetcher CAIFetcher) error {
				_, err := fetcher.BatchGetAssetsHistory(t.Context(), "projects/test-project", []string{"res-1"}, assetpb.ContentType_RESOURCE, nil)
				return err
			},
			wantCode: codes.Internal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := &mockCAIAssetServer{}
			tc.setupMock(mockServer)
			factory := setupMockCAIServer(t, mockServer)
			fetcher := NewCAIFetcher(factory, googlecloud.NewCallOptionInjector(), "test-project")

			err := tc.call(fetcher)
			if err == nil {
				t.Fatalf("expected error with code %v, got nil", tc.wantCode)
			}
			var s interface{ GRPCStatus() *status.Status }
			if !errors.As(err, &s) {
				t.Fatalf("expected gRPC status error, got: %v", err)
			}
			if s.GRPCStatus().Code() != tc.wantCode {
				t.Errorf("error code = %v, want %v", s.GRPCStatus().Code(), tc.wantCode)
			}
		})
	}
}

func TestBuildCAIParentSearchQueries(t *testing.T) {
	testCases := []struct {
		name             string
		parentAssetNames []string
		wantQueries      []CAIParentSearchQuery
	}{
		{
			name:             "empty parents returns empty queries",
			parentAssetNames: nil,
			wantQueries:      nil,
		},
		{
			name: "fewer than 10 parents joins into one query",
			parentAssetNames: []string{
				"//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/ns1",
				"//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/ns2",
			},
			wantQueries: []CAIParentSearchQuery{
				{
					Query:       `parentFullResourceName="//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/ns1" OR parentFullResourceName="//container.googleapis.com/projects/p/locations/l/clusters/c/k8s/namespaces/ns2"`,
					ParentCount: 2,
				},
			},
		},
		{
			name: "11 parents splits at 10 comparison limit",
			parentAssetNames: func() []string {
				p := make([]string, 11)
				for i := range p {
					p[i] = fmt.Sprintf("parent-%d", i)
				}
				return p
			}(),
			wantQueries: func() []CAIParentSearchQuery {
				var q1 []string
				for i := 0; i < 10; i++ {
					q1 = append(q1, fmt.Sprintf(`parentFullResourceName="parent-%d"`, i))
				}
				return []CAIParentSearchQuery{
					{
						Query:       strings.Join(q1, " OR "),
						ParentCount: 10,
					},
					{
						Query:       `parentFullResourceName="parent-10"`,
						ParentCount: 1,
					},
				}
			}(),
		},
		{
			name: "splits before the character limit is exceeded",
			parentAssetNames: []string{
				strings.Repeat("a", 800),
				strings.Repeat("b", 800),
				strings.Repeat("c", 800),
			},
			wantQueries: []CAIParentSearchQuery{
				{
					Query:       fmt.Sprintf(`parentFullResourceName=%q OR parentFullResourceName=%q`, strings.Repeat("a", 800), strings.Repeat("b", 800)),
					ParentCount: 2,
				},
				{
					Query:       fmt.Sprintf(`parentFullResourceName=%q`, strings.Repeat("c", 800)),
					ParentCount: 1,
				},
			},
		},
		{
			name: "keeps a parent longer than the character limit in a query of its own",
			parentAssetNames: []string{
				strings.Repeat("a", 100),
				strings.Repeat("b", 2200),
				strings.Repeat("c", 100),
			},
			wantQueries: []CAIParentSearchQuery{
				{
					Query:       fmt.Sprintf(`parentFullResourceName=%q`, strings.Repeat("a", 100)),
					ParentCount: 1,
				},
				{
					Query:       fmt.Sprintf(`parentFullResourceName=%q`, strings.Repeat("b", 2200)),
					ParentCount: 1,
				},
				{
					Query:       fmt.Sprintf(`parentFullResourceName=%q`, strings.Repeat("c", 100)),
					ParentCount: 1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildCAIParentSearchQueries(tc.parentAssetNames)
			if diff := cmp.Diff(tc.wantQueries, got); diff != "" {
				t.Errorf("BuildCAIParentSearchQueries() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConvertTemporalAssetsToCAISnapshots(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	testCases := []struct {
		name      string
		assets    []*assetpb.TemporalAsset
		wantTimes []time.Time
	}{
		{
			name: "filters nil items and sorts stably by start time ascending including zero time",
			assets: []*assetpb.TemporalAsset{
				nil,
				{Window: &assetpb.TimeWindow{StartTime: timestamppb.New(t2)}, Asset: &assetpb.Asset{Name: "a2"}},
				{Asset: nil},
				{Window: nil, Asset: &assetpb.Asset{Name: "a0"}},
				{Window: &assetpb.TimeWindow{StartTime: timestamppb.New(t1)}, Asset: &assetpb.Asset{Name: "a1"}},
			},
			wantTimes: []time.Time{{}, t1, t2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ConvertTemporalAssetsToCAISnapshots(tc.assets)
			var gotTimes []time.Time
			for _, s := range got {
				gotTimes = append(gotTimes, s.StartTime())
			}
			if diff := cmp.Diff(tc.wantTimes, gotTimes); diff != "" {
				t.Errorf("ConvertTemporalAssetsToCAISnapshots() times mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type stubCAIFetcher struct {
	searchResults map[string][]*assetpb.ResourceSearchResult
	searchErr     error
	batchAssets   []*assetpb.TemporalAsset
	batchErr      error
	gotQueries    []string
}

var _ CAIFetcher = (*stubCAIFetcher)(nil)

func (s *stubCAIFetcher) SearchResources(_ context.Context, _ string, query string, _ []string, onProgress CAISearchProgressCallback) ([]*assetpb.ResourceSearchResult, error) {
	s.gotQueries = append(s.gotQueries, query)
	if s.searchErr != nil {
		return nil, s.searchErr
	}
	results := s.searchResults[query]
	for i := range results {
		if onProgress != nil {
			onProgress(i + 1)
		}
	}
	return results, nil
}

func (s *stubCAIFetcher) BatchGetAssetsHistory(ctx context.Context, _ string, assetNames []string, _ assetpb.ContentType, _ *assetpb.TimeWindow) ([]*assetpb.TemporalAsset, error) {
	if s.batchErr != nil {
		return nil, s.batchErr
	}
	if len(assetNames) > 0 {
		totalChunks := (len(assetNames) + MaxCAIBatchHistorySize - 1) / MaxCAIBatchHistorySize
		tracker := progress.NewTracker(ctx, totalChunks, progress.WithUnit("chunks"))
		defer tracker.Done()
		for i := 1; i <= totalChunks; i++ {
			tracker.Add(1)
		}
	}
	return s.batchAssets, nil
}

func TestSearchCAIAssetsByParents(t *testing.T) {
	testCases := []struct {
		name              string
		parentAssetNames  []string
		progressCfg       CAIParentSearchProgressConfig
		fetcher           *stubCAIFetcher
		wantQueries       []string
		wantNames         []string
		wantIndeterminate bool
		wantRatio         float32
		wantMessageSubstr string
		wantErr           bool
	}{
		{
			name:             "indeterminate progress reporting when parent unit is empty",
			parentAssetNames: []string{"parent-a", "parent-b"},
			progressCfg: CAIParentSearchProgressConfig{
				Label: "Searching cluster resources",
			},
			fetcher: &stubCAIFetcher{
				searchResults: map[string][]*assetpb.ResourceSearchResult{
					`parentFullResourceName="parent-a" OR parentFullResourceName="parent-b"`: {
						{Name: "asset-1"},
						{Name: "asset-2"},
					},
				},
			},
			wantQueries:       []string{`parentFullResourceName="parent-a" OR parentFullResourceName="parent-b"`},
			wantNames:         []string{"asset-1", "asset-2"},
			wantIndeterminate: true,
			wantRatio:         0,
			wantMessageSubstr: "Searching cluster resources... (2 resources scanned)",
		},
		{
			name:             "determinate progress reporting when parent unit is specified",
			parentAssetNames: []string{"parent-a", "parent-b"},
			progressCfg: CAIParentSearchProgressConfig{
				Label:      "Searching namespaced resources",
				ParentUnit: "namespaces",
			},
			fetcher: &stubCAIFetcher{
				searchResults: map[string][]*assetpb.ResourceSearchResult{
					`parentFullResourceName="parent-a" OR parentFullResourceName="parent-b"`: {
						{Name: "asset-1"},
						{Name: "asset-2"},
					},
				},
			},
			wantQueries:       []string{`parentFullResourceName="parent-a" OR parentFullResourceName="parent-b"`},
			wantNames:         []string{"asset-1", "asset-2"},
			wantIndeterminate: false,
			wantRatio:         1.0,
			wantMessageSubstr: "Searching namespaced resources: 2/2 namespaces, 2 resources scanned",
		},
		{
			name:             "returns error when search fails",
			parentAssetNames: []string{"parent-a"},
			progressCfg: CAIParentSearchProgressConfig{
				Label: "Searching cluster resources",
			},
			fetcher: &stubCAIFetcher{
				searchErr: fmt.Errorf("search failed"),
			},
			wantQueries: []string{`parentFullResourceName="parent-a"`},
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			progressMeta := inspectionmetadata.NewTaskProgressMetadata("test-task")
			ctx := progress.WithContext(t.Context(), progressMeta)
			results, err := SearchCAIAssetsByParents(ctx, tc.fetcher, "projects/p", []string{"k8s.io/Pod"}, tc.parentAssetNames, tc.progressCfg)
			if (err != nil) != tc.wantErr {
				t.Fatalf("SearchCAIAssetsByParents() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.wantQueries, tc.fetcher.gotQueries); diff != "" {
				t.Errorf("gotQueries mismatch (-want +got):\n%s", diff)
			}
			if !tc.wantErr {
				var gotNames []string
				for _, r := range results {
					gotNames = append(gotNames, r.Name)
				}
				if diff := cmp.Diff(tc.wantNames, gotNames); diff != "" {
					t.Errorf("gotNames mismatch (-want +got):\n%s", diff)
				}
				snapshot := progressMeta.Snapshot()
				if snapshot.Indeterminate != tc.wantIndeterminate {
					t.Errorf("snapshot.Indeterminate = %v, want %v", snapshot.Indeterminate, tc.wantIndeterminate)
				}
				if snapshot.Ratio != tc.wantRatio {
					t.Errorf("snapshot.Ratio = %v, want %v", snapshot.Ratio, tc.wantRatio)
				}
				if !strings.Contains(snapshot.Message, tc.wantMessageSubstr) {
					t.Errorf("snapshot.Message %q does not contain %q", snapshot.Message, tc.wantMessageSubstr)
				}
			}
		})
	}
}

func TestFetchCAIAssetSnapshots(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	testCases := []struct {
		name              string
		discoverAssets    []string
		discoverErr       error
		fetcher           *stubCAIFetcher
		wantCount         int
		wantProgressRatio float32
		wantErr           bool
	}{
		{
			name:              "returns empty slice when no assets are discovered",
			discoverAssets:    nil,
			fetcher:           &stubCAIFetcher{},
			wantCount:         0,
			wantProgressRatio: 0,
		},
		{
			name:        "returns error when discovery fails",
			discoverErr: fmt.Errorf("discovery failed"),
			fetcher:     &stubCAIFetcher{},
			wantErr:     true,
		},
		{
			name:           "fetches batch history, updates progress, and returns sorted snapshots",
			discoverAssets: []string{"asset-1", "asset-2"},
			fetcher: &stubCAIFetcher{
				batchAssets: []*assetpb.TemporalAsset{
					{Window: &assetpb.TimeWindow{StartTime: timestamppb.New(t2)}, Asset: &assetpb.Asset{Name: "asset-2"}},
					{Window: &assetpb.TimeWindow{StartTime: timestamppb.New(t1)}, Asset: &assetpb.Asset{Name: "asset-1"}},
				},
			},
			wantCount:         2,
			wantProgressRatio: 1.0,
		},
		{
			name:           "returns error when batch history fails",
			discoverAssets: []string{"asset-1"},
			fetcher: &stubCAIFetcher{
				batchErr: fmt.Errorf("batch history failed"),
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			progressMeta := inspectionmetadata.NewTaskProgressMetadata("test")
			ctx := progress.WithContext(t.Context(), progressMeta)
			got, err := FetchCAIAssetSnapshots(
				ctx,
				tc.fetcher,
				"projects/p",
				t1,
				t2,
				func(ctx context.Context, f CAIFetcher) ([]string, error) {
					return tc.discoverAssets, tc.discoverErr
				},
			)
			if (err != nil) != tc.wantErr {
				t.Fatalf("FetchCAIAssetSnapshots() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if len(got) != tc.wantCount {
					t.Errorf("len(got) = %d, want %d", len(got), tc.wantCount)
				}
				gotRatio := progressMeta.Snapshot().Ratio
				if gotRatio != tc.wantProgressRatio {
					t.Errorf("progress ratio = %v, want %v", gotRatio, tc.wantProgressRatio)
				}
			}
		})
	}
}

func TestFormatAssetHistoryProgress(t *testing.T) {
	testCases := []struct {
		name            string
		completedAssets int
		totalAssets     int
		snapshotCount   int
		elapsed         time.Duration
		want            string
	}{
		{
			name:            "omits rate and ETA when elapsed is less than 1 second",
			completedAssets: 10,
			totalAssets:     100,
			snapshotCount:   5,
			elapsed:         500 * time.Millisecond,
			want:            "Fetching asset history: 10/100 assets, 5 snapshots",
		},
		{
			name:            "computes rate and ETA when elapsed is at least 1 second and 0.005 < ratio < 1.0",
			completedAssets: 50,
			totalAssets:     100,
			snapshotCount:   25,
			elapsed:         2 * time.Second,
			want:            "Fetching asset history: 50/100 assets, 25 snapshots (25.0 assets/s, ETA 2s)",
		},
		{
			name:            "computes rate and omits ETA when ratio is less than or equal to 0.005",
			completedAssets: 1,
			totalAssets:     1000,
			snapshotCount:   1,
			elapsed:         2 * time.Second,
			want:            "Fetching asset history: 1/1000 assets, 1 snapshots (0.5 assets/s)",
		},
		{
			name:            "computes rate and omits ETA when ratio is greater than or equal to 1.0",
			completedAssets: 100,
			totalAssets:     100,
			snapshotCount:   50,
			elapsed:         2 * time.Second,
			want:            "Fetching asset history: 100/100 assets, 50 snapshots (50.0 assets/s)",
		},
		{
			name:            "omits rate and ETA when completedAssets is 0 even if elapsed is at least 1 second",
			completedAssets: 0,
			totalAssets:     100,
			snapshotCount:   0,
			elapsed:         2 * time.Second,
			want:            "Fetching asset history: 0/100 assets, 0 snapshots",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatAssetHistoryProgress(tc.completedAssets, tc.totalAssets, tc.snapshotCount, tc.elapsed)
			if got != tc.want {
				t.Errorf("formatAssetHistoryProgress() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatDeterminateParentSearchProgress(t *testing.T) {
	testCases := []struct {
		name             string
		label            string
		completedParents int
		totalParents     int
		parentUnit       string
		scannedCount     int
		elapsed          time.Duration
		want             string
	}{
		{
			name:             "omits rate when elapsed is less than 1 second",
			label:            "Searching namespaced resources",
			completedParents: 2,
			totalParents:     10,
			parentUnit:       "namespaces",
			scannedCount:     5,
			elapsed:          500 * time.Millisecond,
			want:             "Searching namespaced resources: 2/10 namespaces, 5 resources scanned",
		},
		{
			name:             "computes rate when elapsed is at least 1 second and scannedCount is positive",
			label:            "Searching namespaced resources",
			completedParents: 5,
			totalParents:     10,
			parentUnit:       "namespaces",
			scannedCount:     20,
			elapsed:          2 * time.Second,
			want:             "Searching namespaced resources: 5/10 namespaces, 20 resources scanned (10.0 resources/s)",
		},
		{
			name:             "omits rate when scannedCount is 0 even if elapsed is at least 1 second",
			label:            "Searching namespaced resources",
			completedParents: 0,
			totalParents:     10,
			parentUnit:       "namespaces",
			scannedCount:     0,
			elapsed:          2 * time.Second,
			want:             "Searching namespaced resources: 0/10 namespaces, 0 resources scanned",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatDeterminateParentSearchProgress(tc.label, tc.completedParents, tc.totalParents, tc.parentUnit, tc.scannedCount, tc.elapsed)
			if got != tc.want {
				t.Errorf("formatDeterminateParentSearchProgress() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatIndeterminateSearchProgress(t *testing.T) {
	testCases := []struct {
		name         string
		label        string
		scannedCount int
		elapsed      time.Duration
		want         string
	}{
		{
			name:         "omits rate when elapsed is less than 1 second",
			label:        "Searching cluster resources",
			scannedCount: 10,
			elapsed:      500 * time.Millisecond,
			want:         "Searching cluster resources... (10 resources scanned)",
		},
		{
			name:         "computes rate when elapsed is at least 1 second and scannedCount is positive",
			label:        "Searching cluster resources",
			scannedCount: 30,
			elapsed:      2 * time.Second,
			want:         "Searching cluster resources... (30 resources scanned, 15.0 resources/s)",
		},
		{
			name:         "omits rate when scannedCount is 0 even if elapsed is at least 1 second",
			label:        "Searching cluster resources",
			scannedCount: 0,
			elapsed:      2 * time.Second,
			want:         "Searching cluster resources... (0 resources scanned)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatIndeterminateSearchProgress(tc.label, tc.scannedCount, tc.elapsed)
			if got != tc.want {
				t.Errorf("formatIndeterminateSearchProgress() = %q, want %q", got, tc.want)
			}
		})
	}
}
