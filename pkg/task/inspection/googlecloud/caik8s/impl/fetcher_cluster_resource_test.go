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
	"net"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	regionalClusterAssetName     = "//container.googleapis.com/projects/test-project/locations/us-central1-a/clusters/test-cluster"
	zonalClusterAssetName        = "//container.googleapis.com/projects/test-project/zones/us-central1-a/clusters/test-cluster"
	defaultNamespaceAssetName    = regionalClusterAssetName + "/k8s/namespaces/default"
	kubeSystemNamespaceAssetName = regionalClusterAssetName + "/k8s/namespaces/kube-system"
	podAssetName                 = defaultNamespaceAssetName + "/pods/pod-1"
	nodeAssetName                = regionalClusterAssetName + "/k8s/nodes/node-1"

	clusterParentQuery          = `parentFullResourceName="` + regionalClusterAssetName + `" OR parentFullResourceName="` + zonalClusterAssetName + `"`
	defaultNamespaceParentQuery = `parentFullResourceName="` + defaultNamespaceAssetName + `"`
)

type mockAssetServer struct {
	assetpb.UnimplementedAssetServiceServer

	searchResultsByQuery map[string][]*assetpb.ResourceSearchResult
	searchErr            error
	batchAssets          []*assetpb.TemporalAsset

	mu             sync.Mutex
	batchRequests  []*assetpb.BatchGetAssetsHistoryRequest
	searchRequests []*assetpb.SearchAllResourcesRequest
}

func (m *mockAssetServer) recordedBatchRequests() []*assetpb.BatchGetAssetsHistoryRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.batchRequests)
}

func (m *mockAssetServer) recordedSearchRequests() []*assetpb.SearchAllResourcesRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.searchRequests)
}

func (m *mockAssetServer) BatchGetAssetsHistory(_ context.Context, req *assetpb.BatchGetAssetsHistoryRequest) (*assetpb.BatchGetAssetsHistoryResponse, error) {
	m.mu.Lock()
	m.batchRequests = append(m.batchRequests, req)
	m.mu.Unlock()

	return &assetpb.BatchGetAssetsHistoryResponse{Assets: m.batchAssets}, nil
}

func (m *mockAssetServer) SearchAllResources(_ context.Context, req *assetpb.SearchAllResourcesRequest) (*assetpb.SearchAllResourcesResponse, error) {
	m.mu.Lock()
	m.searchRequests = append(m.searchRequests, req)
	m.mu.Unlock()

	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return &assetpb.SearchAllResourcesResponse{Results: m.searchResultsByQuery[req.Query]}, nil
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

// mockCAIFetcher returns canned CAI responses and records the arguments it was called with.
type mockCAIFetcher struct {
	searchResultsPerCall [][]*assetpb.ResourceSearchResult
	searchErr            error
	searchErrPerCall     []error
	batchAssets          []*assetpb.TemporalAsset
	batchErr             error

	searchCalls []recordedSearchCall

	batchCallCount     int
	gotBatchParent     string
	gotBatchAssetNames []string
}

// recordedSearchCall captures the arguments of a single SearchResources call.
type recordedSearchCall struct {
	scope      string
	query      string
	assetTypes []string
}

var _ gcpcommon.CAIFetcher = (*mockCAIFetcher)(nil)

func (m *mockCAIFetcher) SearchResources(ctx context.Context, scope, query string, assetTypes []string) ([]*assetpb.ResourceSearchResult, error) {
	callIndex := len(m.searchCalls)
	m.searchCalls = append(m.searchCalls, recordedSearchCall{scope: scope, query: query, assetTypes: assetTypes})
	if callIndex < len(m.searchErrPerCall) && m.searchErrPerCall[callIndex] != nil {
		return nil, m.searchErrPerCall[callIndex]
	}
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if callIndex >= len(m.searchResultsPerCall) {
		return nil, nil
	}
	return m.searchResultsPerCall[callIndex], nil
}

// recordedSearchQueries returns the queries of the recorded SearchResources calls in call order.
func (m *mockCAIFetcher) recordedSearchQueries() []string {
	var queries []string
	for _, call := range m.searchCalls {
		queries = append(queries, call.query)
	}
	return queries
}

// recordedSearchAssetTypes returns the asset types of the recorded SearchResources calls in call order.
func (m *mockCAIFetcher) recordedSearchAssetTypes() [][]string {
	var assetTypes [][]string
	for _, call := range m.searchCalls {
		assetTypes = append(assetTypes, call.assetTypes)
	}
	return assetTypes
}

func (m *mockCAIFetcher) BatchGetAssetsHistory(ctx context.Context, parent string, assetNames []string, contentType assetpb.ContentType, timeWindow *assetpb.TimeWindow) ([]*assetpb.TemporalAsset, error) {
	m.batchCallCount++
	m.gotBatchParent = parent
	m.gotBatchAssetNames = assetNames
	if len(assetNames) > 0 {
		totalChunks := (len(assetNames) + gcpcommon.MaxCAIBatchHistorySize - 1) / gcpcommon.MaxCAIBatchHistorySize
		tracker := progress.NewTracker(ctx, totalChunks, progress.WithUnit("chunks"))
		defer tracker.Done()
		for i := 1; i <= totalChunks; i++ {
			tracker.Add(1)
		}
	}
	return m.batchAssets, m.batchErr
}

// newPodTemporalAsset builds a temporal asset carrying a minimal pod manifest.
func newPodTemporalAsset(t *testing.T, startTime, endTime time.Time) *assetpb.TemporalAsset {
	t.Helper()
	podData, err := structpb.NewStruct(map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":      "pod-1",
			"namespace": "default",
		},
	})
	if err != nil {
		t.Fatalf("failed to create structpb: %v", err)
	}
	return &assetpb.TemporalAsset{
		Window: &assetpb.TimeWindow{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(endTime),
		},
		Asset: &assetpb.Asset{
			Name:      podAssetName,
			AssetType: "k8s.io/Pod",
			Resource: &assetpb.Resource{
				Data: podData,
			},
		},
	}
}

func TestClusterAssetNameCandidates(t *testing.T) {
	testCases := []struct {
		name    string
		cluster k8scommon.GoogleCloudClusterIdentity
		want    []string
	}{
		{
			name: "emits the regional and the zonal naming form of the cluster",
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				ClusterName: "test-cluster",
				Location:    "us-central1-a",
			},
			want: []string{regionalClusterAssetName, zonalClusterAssetName},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := clusterAssetNameCandidates(tc.cluster)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("clusterAssetNameCandidates() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// makeNamespaceParents builds count namespace parents under clusterParent, each carrying a namespace
// name of namespaceNameLength characters so that the query length can be controlled from the test.
func makeNamespaceParents(clusterParent string, count, namespaceNameLength int) []string {
	var parents []string
	for i := range count {
		parents = append(parents, fmt.Sprintf("%s/k8s/namespaces/%s%02d", clusterParent, strings.Repeat("n", namespaceNameLength-2), i))
	}
	return parents
}

// parentSearchQuery renders the CAI query that matches exactly the given parent asset names.
func parentSearchQuery(parentAssetNames ...string) string {
	var comparisons []string
	for _, parent := range parentAssetNames {
		comparisons = append(comparisons, fmt.Sprintf("parentFullResourceName=%q", parent))
	}
	return strings.Join(comparisons, " OR ")
}

func TestAssetTypesWithNamespace(t *testing.T) {
	testCases := []struct {
		name       string
		assetTypes []string
		want       []string
	}{
		{
			name:       "appends the Namespace asset type when the kind filter omits it",
			assetTypes: []string{"k8s.io/Pod"},
			want:       []string{"k8s.io/Pod", "k8s.io/Namespace"},
		},
		{
			name:       "leaves the asset types unchanged when the Namespace asset type is present",
			assetTypes: []string{"k8s.io/Namespace", "k8s.io/Pod"},
			want:       []string{"k8s.io/Namespace", "k8s.io/Pod"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := assetTypesWithNamespace(tc.assetTypes)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("assetTypesWithNamespace() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTargetNamespaceParents(t *testing.T) {
	testCases := []struct {
		name          string
		searchResults []*assetpb.ResourceSearchResult
		filter        *gcpqueryutil.SetFilterParseResult
		want          []string
	}{
		{
			name: "selects the Namespace assets the additive filter accepts",
			searchResults: []*assetpb.ResourceSearchResult{
				{Name: defaultNamespaceAssetName, AssetType: "k8s.io/Namespace"},
				{Name: kubeSystemNamespaceAssetName, AssetType: "k8s.io/Namespace"},
				{Name: nodeAssetName, AssetType: "k8s.io/Node"},
			},
			filter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}},
			want:   []string{defaultNamespaceAssetName},
		},
		{
			name: "selects every Namespace asset the subtractive filter keeps",
			searchResults: []*assetpb.ResourceSearchResult{
				{Name: defaultNamespaceAssetName, AssetType: "k8s.io/Namespace"},
				{Name: kubeSystemNamespaceAssetName, AssetType: "k8s.io/Namespace"},
			},
			filter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{"kube-system"}},
			want:   []string{defaultNamespaceAssetName},
		},
		{
			name: "selects no parent when only cluster scoped resources are requested",
			searchResults: []*assetpb.ResourceSearchResult{
				{Name: defaultNamespaceAssetName, AssetType: "k8s.io/Namespace"},
			},
			filter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped"}},
			want:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := targetNamespaceParents(tc.searchResults, tc.filter)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("targetNamespaceParents() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFilterSearchResultsByAssetType(t *testing.T) {
	searchResults := []*assetpb.ResourceSearchResult{
		{Name: nodeAssetName, AssetType: "k8s.io/Node"},
		{Name: defaultNamespaceAssetName, AssetType: "k8s.io/Namespace"},
	}

	testCases := []struct {
		name       string
		assetTypes []string
		want       []string
	}{
		{
			name:       "drops the Namespace assets the kind filter did not request",
			assetTypes: []string{"k8s.io/Node"},
			want:       []string{nodeAssetName},
		},
		{
			name:       "keeps the Namespace assets the kind filter requested",
			assetTypes: []string{"k8s.io/Node", "k8s.io/Namespace"},
			want:       []string{nodeAssetName, defaultNamespaceAssetName},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, searchResult := range filterSearchResultsByAssetType(searchResults, tc.assetTypes) {
				got = append(got, searchResult.Name)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("filterSearchResultsByAssetType() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDiscoverClusterResourceAssetNames(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	splitNamespaceParents := makeNamespaceParents(regionalClusterAssetName, gcpcommon.MaxCAISearchQueryComparisons+1, 10)
	splitNamespaceResults := make([]*assetpb.ResourceSearchResult, 0, len(splitNamespaceParents))
	for _, parent := range splitNamespaceParents {
		splitNamespaceResults = append(splitNamespaceResults, &assetpb.ResourceSearchResult{Name: parent, AssetType: "k8s.io/Namespace"})
	}

	testCases := []struct {
		name                 string
		assetTypes           []string
		fetcher              *mockCAIFetcher
		namespaceFilter      *gcpqueryutil.SetFilterParseResult
		wantSearchQueries    []string
		wantSearchAssetTypes [][]string
		wantCount            int
		wantErr              bool
		wantBatchCallCount   int
		wantBatchAssets      []string
	}{
		{
			name:       "searches the namespaces discovered by the cluster parented phase",
			assetTypes: []string{"k8s.io/Pod"},
			fetcher: &mockCAIFetcher{
				searchResultsPerCall: [][]*assetpb.ResourceSearchResult{
					{
						{Name: defaultNamespaceAssetName, AssetType: "k8s.io/Namespace"},
						{Name: kubeSystemNamespaceAssetName, AssetType: "k8s.io/Namespace"},
					},
					{{Name: podAssetName, AssetType: "k8s.io/Pod"}},
				},
				batchAssets: []*assetpb.TemporalAsset{newPodTemporalAsset(t, startTime, endTime)},
			},
			namespaceFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}},
			wantSearchQueries:    []string{clusterParentQuery, defaultNamespaceParentQuery},
			wantSearchAssetTypes: [][]string{{"k8s.io/Pod", "k8s.io/Namespace"}, {"k8s.io/Pod"}},
			wantCount:            1,
			wantBatchCallCount:   1,
			wantBatchAssets:      []string{podAssetName},
		},
		{
			name:       "skips the namespace parented phase when only cluster scoped resources are requested",
			assetTypes: []string{"k8s.io/Node"},
			fetcher: &mockCAIFetcher{
				searchResultsPerCall: [][]*assetpb.ResourceSearchResult{
					{
						{Name: nodeAssetName, AssetType: "k8s.io/Node"},
						{Name: defaultNamespaceAssetName, AssetType: "k8s.io/Namespace"},
					},
				},
			},
			namespaceFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped"}},
			wantSearchQueries:    []string{clusterParentQuery},
			wantSearchAssetTypes: [][]string{{"k8s.io/Node", "k8s.io/Namespace"}},
			wantCount:            0,
			wantBatchCallCount:   1,
			wantBatchAssets:      []string{nodeAssetName},
		},
		{
			name:       "skips the history lookup when the namespace filter excludes every asset",
			assetTypes: []string{"k8s.io/Pod"},
			fetcher: &mockCAIFetcher{
				searchResultsPerCall: [][]*assetpb.ResourceSearchResult{
					{{Name: defaultNamespaceAssetName, AssetType: "k8s.io/Namespace"}},
				},
			},
			namespaceFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"nonexistent-ns"}},
			wantSearchQueries:    []string{clusterParentQuery},
			wantSearchAssetTypes: [][]string{{"k8s.io/Pod", "k8s.io/Namespace"}},
			wantCount:            0,
			wantBatchCallCount:   0,
		},
		{
			name:       "keeps the excluded Namespace object because Namespace assets are cluster scoped",
			assetTypes: []string{"k8s.io/Namespace", "k8s.io/Pod"},
			fetcher: &mockCAIFetcher{
				searchResultsPerCall: [][]*assetpb.ResourceSearchResult{
					{
						{Name: defaultNamespaceAssetName, AssetType: "k8s.io/Namespace"},
						{Name: kubeSystemNamespaceAssetName, AssetType: "k8s.io/Namespace"},
					},
					{{Name: podAssetName, AssetType: "k8s.io/Pod"}},
				},
				batchAssets: []*assetpb.TemporalAsset{newPodTemporalAsset(t, startTime, endTime)},
			},
			namespaceFilter:      &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{"kube-system"}},
			wantSearchQueries:    []string{clusterParentQuery, defaultNamespaceParentQuery},
			wantSearchAssetTypes: [][]string{{"k8s.io/Namespace", "k8s.io/Pod"}, {"k8s.io/Namespace", "k8s.io/Pod"}},
			wantCount:            1,
			wantBatchCallCount:   1,
			wantBatchAssets:      []string{defaultNamespaceAssetName, kubeSystemNamespaceAssetName, podAssetName},
		},
		{
			name:       "splits the namespace parented phase into several searches",
			assetTypes: []string{"k8s.io/Pod"},
			fetcher: &mockCAIFetcher{
				searchResultsPerCall: [][]*assetpb.ResourceSearchResult{
					splitNamespaceResults,
					{{Name: podAssetName, AssetType: "k8s.io/Pod"}},
					nil,
				},
				batchAssets: []*assetpb.TemporalAsset{newPodTemporalAsset(t, startTime, endTime)},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#namespaced"}},
			wantSearchQueries: []string{
				clusterParentQuery,
				parentSearchQuery(splitNamespaceParents[:gcpcommon.MaxCAISearchQueryComparisons]...),
				parentSearchQuery(splitNamespaceParents[gcpcommon.MaxCAISearchQueryComparisons:]...),
			},
			wantSearchAssetTypes: [][]string{{"k8s.io/Pod", "k8s.io/Namespace"}, {"k8s.io/Pod"}, {"k8s.io/Pod"}},
			wantCount:            1,
			wantBatchCallCount:   1,
			wantBatchAssets:      []string{podAssetName},
		},
		{
			name:       "drops cluster scoped assets when the namespace filter selects namespaces only",
			assetTypes: []string{"k8s.io/Node"},
			fetcher: &mockCAIFetcher{
				searchResultsPerCall: [][]*assetpb.ResourceSearchResult{
					{
						{Name: nodeAssetName, AssetType: "k8s.io/Node"},
						{Name: defaultNamespaceAssetName, AssetType: "k8s.io/Namespace"},
					},
					nil,
				},
			},
			namespaceFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}},
			wantSearchQueries:    []string{clusterParentQuery, defaultNamespaceParentQuery},
			wantSearchAssetTypes: [][]string{{"k8s.io/Node", "k8s.io/Namespace"}, {"k8s.io/Node"}},
			wantCount:            0,
			wantBatchCallCount:   0,
		},
		{
			name:            "propagates search failures",
			assetTypes:      []string{"k8s.io/Pod"},
			fetcher:         &mockCAIFetcher{searchErr: errors.New("search error")},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#namespaced"}},
			wantErr:         true,
		},
		{
			name:       "propagates history lookup failures",
			assetTypes: []string{"k8s.io/Node"},
			fetcher: &mockCAIFetcher{
				searchResultsPerCall: [][]*assetpb.ResourceSearchResult{
					{{Name: nodeAssetName, AssetType: "k8s.io/Node"}},
				},
				batchErr: errors.New("batch error"),
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped"}},
			wantErr:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target := clusterResourceDiscoveryTarget{
				scope:                      "projects/test-project",
				clusterAssetNameCandidates: []string{regionalClusterAssetName, zonalClusterAssetName},
				assetTypes:                 tc.assetTypes,
				namespaceFilter:            tc.namespaceFilter,
			}

			progressMeta := inspectionmetadata.NewTaskProgressMetadata("test")
			ctx := progress.WithContext(t.Context(), progressMeta)
			got, err := gcpcommon.FetchCAIAssetSnapshots(
				ctx,
				tc.fetcher,
				target.scope,
				startTime,
				endTime,
				func(ctx context.Context, f gcpcommon.CAIFetcher) ([]string, error) {
					return discoverClusterResourceAssetNames(ctx, f, target)
				},
			)
			if (err != nil) != tc.wantErr {
				t.Fatalf("FetchCAIAssetSnapshots() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			snap := progressMeta.Snapshot()
			if tc.wantBatchCallCount > 0 && snap.Ratio != 1.0 {
				t.Errorf("progress.Ratio = %f, want 1.0", snap.Ratio)
			}
			if snap.Message == "" {
				t.Errorf("progress.Message is empty")
			}

			if len(got) != tc.wantCount {
				t.Errorf("len(got) = %d, want %d", len(got), tc.wantCount)
			}
			if diff := cmp.Diff(tc.wantSearchQueries, tc.fetcher.recordedSearchQueries()); diff != "" {
				t.Errorf("search queries mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantSearchAssetTypes, tc.fetcher.recordedSearchAssetTypes()); diff != "" {
				t.Errorf("search asset types mismatch (-want +got):\n%s", diff)
			}
			for _, searchCall := range tc.fetcher.searchCalls {
				if searchCall.scope != target.scope {
					t.Errorf("search scope = %s, want %s", searchCall.scope, target.scope)
				}
			}
			if tc.fetcher.batchCallCount != tc.wantBatchCallCount {
				t.Errorf("batchCallCount = %d, want %d", tc.fetcher.batchCallCount, tc.wantBatchCallCount)
			}
			if diff := cmp.Diff(tc.wantBatchAssets, tc.fetcher.gotBatchAssetNames); diff != "" {
				t.Errorf("batch asset names mismatch (-want +got):\n%s", diff)
			}
			if tc.wantBatchCallCount > 0 && tc.fetcher.gotBatchParent != target.scope {
				t.Errorf("batch parent = %s, want %s", tc.fetcher.gotBatchParent, target.scope)
			}
		})
	}
}

func TestClusterResourceSuite_FetcherTask(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	completeCluster := k8scommon.GoogleCloudClusterIdentity{
		ProjectID:   "test-project",
		ClusterName: "test-cluster",
		Location:    "us-central1-a",
	}

	incompleteCluster := k8scommon.GoogleCloudClusterIdentity{
		ProjectID:   "test-project",
		ClusterName: "",
		Location:    "us-central1-a",
	}

	testCases := []struct {
		name                 string
		taskMode             inspectioncore.InspectionTaskModeType
		cluster              k8scommon.GoogleCloudClusterIdentity
		kindFilter           *gcpqueryutil.SetFilterParseResult
		namespaceFilter      *gcpqueryutil.SetFilterParseResult
		searchErr            error
		wantCount            int
		wantSearchQueries    []string
		wantSearchAssetTypes [][]string
	}{
		{
			name:            "returns empty on DryRun mode",
			taskMode:        inspectioncore.TaskModeDryRun,
			cluster:         completeCluster,
			kindFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"pod"}},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#namespaced"}},
			wantCount:       0,
		},
		{
			name:            "returns empty when cluster identity is incomplete",
			taskMode:        inspectioncore.TaskModeRun,
			cluster:         incompleteCluster,
			kindFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"pod"}},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#namespaced"}},
			wantCount:       0,
		},
		{
			name:            "returns empty without querying CAI when the kind filter matches no asset type",
			taskMode:        inspectioncore.TaskModeRun,
			cluster:         completeCluster,
			kindFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{}},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#namespaced"}},
			wantCount:       0,
		},
		{
			name:                 "searches the cluster and its namespaces by exact parent resource name",
			taskMode:             inspectioncore.TaskModeRun,
			cluster:              completeCluster,
			kindFilter:           &gcpqueryutil.SetFilterParseResult{Additives: []string{"pod"}},
			namespaceFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}},
			wantCount:            1,
			wantSearchQueries:    []string{clusterParentQuery, defaultNamespaceParentQuery},
			wantSearchAssetTypes: [][]string{{"k8s.io/Pod", "k8s.io/Namespace"}, {"k8s.io/Pod"}},
		},
		{
			name:                 "returns empty without an error when the CAI search fails",
			taskMode:             inspectioncore.TaskModeRun,
			cluster:              completeCluster,
			kindFilter:           &gcpqueryutil.SetFilterParseResult{Additives: []string{"pod"}},
			namespaceFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}},
			searchErr:            errors.New("permission denied"),
			wantCount:            0,
			wantSearchQueries:    []string{clusterParentQuery},
			wantSearchAssetTypes: [][]string{{"k8s.io/Pod", "k8s.io/Namespace"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := &mockAssetServer{
				searchResultsByQuery: map[string][]*assetpb.ResourceSearchResult{
					clusterParentQuery: {
						{Name: defaultNamespaceAssetName, AssetType: "k8s.io/Namespace"},
						{Name: kubeSystemNamespaceAssetName, AssetType: "k8s.io/Namespace"},
					},
					defaultNamespaceParentQuery: {
						{Name: podAssetName, AssetType: "k8s.io/Pod"},
					},
				},
				searchErr:   tc.searchErr,
				batchAssets: []*assetpb.TemporalAsset{newPodTemporalAsset(t, startTime, endTime)},
			}
			factory := setupMockServer(t, mockServer)

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.RunInspectionTask(ctx, ClusterResourceSuite.FetcherTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(gcpcommon.InputStartTimeTaskID.Ref(), startTime),
				tasktest.NewTaskDependencyValuePair(gcpcommon.InputEndTimeTaskID.Ref(), endTime),
				tasktest.NewTaskDependencyValuePair(gcpcommon.APIClientFactoryTaskID.Ref(), factory),
				tasktest.NewTaskDependencyValuePair(gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
				tasktest.NewTaskDependencyValuePair(k8scommon.ClusterIdentityTaskID.Ref(), tc.cluster),
				tasktest.NewTaskDependencyValuePair(k8scommon.InputKindFilterTaskID.Ref(), tc.kindFilter),
				tasktest.NewTaskDependencyValuePair(k8scommon.InputNamespaceFilterTaskID.Ref(), tc.namespaceFilter),
			)
			if err != nil {
				t.Fatalf("FetcherTask unexpected error: %v", err)
			}

			if len(got) != tc.wantCount {
				t.Errorf("len(got) = %d, want %d", len(got), tc.wantCount)
			}

			var gotQueries []string
			var gotAssetTypes [][]string
			for _, searchRequest := range mockServer.recordedSearchRequests() {
				gotQueries = append(gotQueries, searchRequest.Query)
				gotAssetTypes = append(gotAssetTypes, searchRequest.AssetTypes)
				if searchRequest.Scope != "projects/test-project" {
					t.Errorf("search scope = %s, want projects/test-project", searchRequest.Scope)
				}
			}
			if diff := cmp.Diff(tc.wantSearchQueries, gotQueries); diff != "" {
				t.Errorf("search queries mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantSearchAssetTypes, gotAssetTypes); diff != "" {
				t.Errorf("search asset types mismatch (-want +got):\n%s", diff)
			}
			if tc.wantCount == 0 {
				return
			}

			gotBatchRequests := mockServer.recordedBatchRequests()
			if len(gotBatchRequests) != 1 {
				t.Fatalf("batch request count = %d, want 1", len(gotBatchRequests))
			}
			if diff := cmp.Diff([]string{podAssetName}, gotBatchRequests[0].AssetNames); diff != "" {
				t.Errorf("batch asset names mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveAssetTypes(t *testing.T) {
	allDefaults := defaultAssetTypes()

	var defaultWithoutPod []string
	for _, at := range allDefaults {
		if at != "k8s.io/Pod" {
			defaultWithoutPod = append(defaultWithoutPod, at)
		}
	}

	testCases := []struct {
		name   string
		filter *gcpqueryutil.SetFilterParseResult
		want   []string
	}{
		{
			name: "subtract mode with unmapped kind returns all default types",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"custom.io/CustomResource"},
			},
			want: allDefaults,
		},
		{
			name: "validation error returns nil",
			filter: &gcpqueryutil.SetFilterParseResult{
				ValidationError: "invalid filter",
			},
			want: nil,
		},
		{
			name: "subtract mode with empty subtractives returns all default types",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{},
			},
			want: allDefaults,
		},
		{
			name: "subtract mode with pod returns default types without pod",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"pod"},
			},
			want: defaultWithoutPod,
		},
		{
			name: "additive mode with pod and deployment returns mapped asset types",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: false,
				Additives:    []string{"pod", "deployment"},
			},
			want: []string{"apps.k8s.io/Deployment", "k8s.io/Pod"},
		},
		{
			name: "additive mode with empty additives returns nil",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: false,
				Additives:    []string{},
			},
			want: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveAssetTypes(tc.filter)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("resolveAssetTypes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMatchesNamespaceFilter(t *testing.T) {
	testCases := []struct {
		name      string
		filter    *gcpqueryutil.SetFilterParseResult
		namespace string
		want      bool
	}{
		{
			name: "subtract mode matches non-excluded namespace",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"kube-system"},
			},
			namespace: "default",
			want:      true,
		},
		{
			name: "subtract mode excludes specified namespace",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"kube-system"},
			},
			namespace: "kube-system",
			want:      false,
		},
		{
			name: "subtract mode excludes cluster-scoped when #cluster-scoped is subtracted",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"#cluster-scoped"},
			},
			namespace: "",
			want:      false,
		},
		{
			name: "subtract mode includes namespaced when #cluster-scoped is subtracted",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"#cluster-scoped"},
			},
			namespace: "default",
			want:      true,
		},
		{
			name: "subtract mode excludes namespaced when #namespaced is subtracted",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"#namespaced"},
			},
			namespace: "default",
			want:      false,
		},
		{
			name: "subtract mode includes cluster-scoped when #namespaced is subtracted",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"#namespaced"},
			},
			namespace: "",
			want:      true,
		},
		{
			name: "additive mode matches cluster-scoped when #cluster-scoped is specified",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: false,
				Additives:    []string{"#cluster-scoped"},
			},
			namespace: "",
			want:      true,
		},
		{
			name: "additive mode does not match namespaced when #cluster-scoped is specified",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: false,
				Additives:    []string{"#cluster-scoped"},
			},
			namespace: "default",
			want:      false,
		},
		{
			name: "additive mode matches namespaced when #namespaced is specified",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: false,
				Additives:    []string{"#namespaced"},
			},
			namespace: "default",
			want:      true,
		},
		{
			name: "additive mode does not match cluster-scoped when #namespaced is specified",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: false,
				Additives:    []string{"#namespaced"},
			},
			namespace: "",
			want:      false,
		},
		{
			name: "additive mode matches specified namespace",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: false,
				Additives:    []string{"default"},
			},
			namespace: "default",
			want:      true,
		},
		{
			name: "additive mode does not match unspecified namespace",
			filter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: false,
				Additives:    []string{"default"},
			},
			namespace: "kube-system",
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := matchesNamespaceFilter(tc.filter, tc.namespace)
			if got != tc.want {
				t.Errorf("matchesNamespaceFilter(%v, %q) = %v, want %v", tc.filter, tc.namespace, got, tc.want)
			}
		})
	}
}
