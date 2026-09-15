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
	"strings"
	"testing"
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
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

// mockCAIFetcher returns canned CAI responses and records the arguments it was called with,
// so that assertions run after the call instead of inside a callback that may never fire.
type mockCAIFetcher struct {
	// searchResultsPerCall holds the results of successive SearchResources calls, one entry per call.
	// A cluster resource lookup searches once per phase, so the entries are indexed by phase.
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

var _ googlecloudcaik8s_contract.CAIFetcher = (*mockCAIFetcher)(nil)

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

func (m *mockCAIFetcher) BatchGetAssetsHistory(ctx context.Context, parent string, assetNames []string, contentType assetpb.ContentType, timeWindow *assetpb.TimeWindow, onProgress func(completedChunks, totalChunks int)) ([]*assetpb.TemporalAsset, error) {
	m.batchCallCount++
	m.gotBatchParent = parent
	m.gotBatchAssetNames = assetNames
	if onProgress != nil && len(assetNames) > 0 {
		totalChunks := (len(assetNames) + maxBatchHistorySize - 1) / maxBatchHistorySize
		for i := 1; i <= totalChunks; i++ {
			onProgress(i, totalChunks)
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

func TestClusterParentCandidates(t *testing.T) {
	testCases := []struct {
		name    string
		cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		want    []string
	}{
		{
			name: "emits the regional and the zonal naming form of the cluster",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				ClusterName: "test-cluster",
				Location:    "us-central1-a",
			},
			want: []string{regionalClusterAssetName, zonalClusterAssetName},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := clusterParentCandidates(tc.cluster)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("clusterParentCandidates() mismatch (-want +got):\n%s", diff)
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

// parentSearchQuery renders the CAI query that matches exactly the given parents. It spells out the
// expected wire format so that the expectation does not depend on the production query builder.
func parentSearchQuery(parents ...string) string {
	var comparisons []string
	for _, parent := range parents {
		comparisons = append(comparisons, fmt.Sprintf("parentFullResourceName=%q", parent))
	}
	return strings.Join(comparisons, " OR ")
}

// assertParentSearchQueryLimits checks that the queries carry the expected number of comparisons,
// that none of them exceeds the CAI query length limit, and that concatenating the comparisons of
// every query reproduces each input parent exactly once in the input order.
// A query holding a single comparison is exempt from the length check because a parent longer than
// the limit cannot be split any further.
func assertParentSearchQueryLimits(t *testing.T, parents, queries []string, wantComparisonsPerQuery []int) {
	t.Helper()
	var gotComparisonsPerQuery []int
	var gotComparisons []string
	for _, query := range queries {
		comparisons := strings.Split(query, " OR ")
		gotComparisonsPerQuery = append(gotComparisonsPerQuery, len(comparisons))
		gotComparisons = append(gotComparisons, comparisons...)
		if len(comparisons) > 1 && len(query) > maxSearchQueryCharacters {
			t.Errorf("query length = %d, want at most %d", len(query), maxSearchQueryCharacters)
		}
	}
	if diff := cmp.Diff(wantComparisonsPerQuery, gotComparisonsPerQuery); diff != "" {
		t.Errorf("comparisons per query mismatch (-want +got):\n%s", diff)
	}

	var wantComparisons []string
	for _, parent := range parents {
		wantComparisons = append(wantComparisons, fmt.Sprintf("parentFullResourceName=%q", parent))
	}
	if diff := cmp.Diff(wantComparisons, gotComparisons); diff != "" {
		t.Errorf("comparisons mismatch (-want +got):\n%s", diff)
	}
}

func TestBuildParentSearchQueries(t *testing.T) {
	testCases := []struct {
		name    string
		parents []string
		want    []string
	}{
		{
			name:    "no parent produces no query",
			parents: nil,
			want:    nil,
		},
		{
			name:    "single parent produces a single exact match",
			parents: []string{regionalClusterAssetName},
			want:    []string{`parentFullResourceName="` + regionalClusterAssetName + `"`},
		},
		{
			name:    "multiple parents are joined with OR",
			parents: []string{regionalClusterAssetName, zonalClusterAssetName},
			want:    []string{clusterParentQuery},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildParentSearchQueries(tc.parents)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("buildParentSearchQueries() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	// A 30 character project ID and a 40 character cluster name leave room for a 63 character
	// namespace name in a 242 character comparison, so only eight of them fit within the limit.
	longClusterParent := fmt.Sprintf("//container.googleapis.com/projects/%s/locations/us-central1/clusters/%s", strings.Repeat("p", 30), strings.Repeat("c", 40))

	splitTestCases := []struct {
		name                    string
		parents                 []string
		wantComparisonsPerQuery []int
	}{
		{
			name:                    "splits once the comparison limit is reached",
			parents:                 makeNamespaceParents(regionalClusterAssetName, maxSearchQueryComparisons+1, 10),
			wantComparisonsPerQuery: []int{maxSearchQueryComparisons, 1},
		},
		{
			name:                    "splits before the character limit is exceeded",
			parents:                 makeNamespaceParents(longClusterParent, 10, 63),
			wantComparisonsPerQuery: []int{8, 2},
		},
		{
			name:                    "keeps a parent longer than the character limit in a query of its own",
			parents:                 makeNamespaceParents(longClusterParent, 2, maxSearchQueryCharacters),
			wantComparisonsPerQuery: []int{1, 1},
		},
	}

	for _, tc := range splitTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildParentSearchQueries(tc.parents)
			assertParentSearchQueryLimits(t, tc.parents, got, tc.wantComparisonsPerQuery)
		})
	}
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

func TestFetchClusterResourceSnapshots(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	// One namespace more than the comparison limit forces the namespace parented phase to split
	// into two searches.
	splitNamespaceParents := makeNamespaceParents(regionalClusterAssetName, maxSearchQueryComparisons+1, 10)
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
			// The history stub returns no asset, so this case asserts the search phases and the
			// client side filtering rather than the resulting snapshots.
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
			// A Namespace object carries no namespace of its own, so the namespace filter treats it
			// as cluster scoped. Excluding kube-system therefore drops its members but keeps the
			// kube-system Namespace object itself; only #cluster-scoped removes it.
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
				parentSearchQuery(splitNamespaceParents[:maxSearchQueryComparisons]...),
				parentSearchQuery(splitNamespaceParents[maxSearchQueryComparisons:]...),
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
			lookup := clusterResourceLookup{
				scope:                   "projects/test-project",
				clusterParentCandidates: []string{regionalClusterAssetName, zonalClusterAssetName},
				assetTypes:              tc.assetTypes,
				namespaceFilter:         tc.namespaceFilter,
				timeWindow: &assetpb.TimeWindow{
					StartTime: timestamppb.New(startTime),
					EndTime:   timestamppb.New(endTime),
				},
			}

			progress := &inspectionmetadata.TaskProgressMetadata{}
			got, err := fetchClusterResourceSnapshots(t.Context(), tc.fetcher, lookup, progress)
			if (err != nil) != tc.wantErr {
				t.Fatalf("fetchClusterResourceSnapshots() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if tc.wantBatchCallCount > 0 && progress.Percentage != 1.0 {
				t.Errorf("progress.Percentage = %f, want 1.0", progress.Percentage)
			}
			if progress.Message == "" {
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
				if searchCall.scope != lookup.scope {
					t.Errorf("search scope = %s, want %s", searchCall.scope, lookup.scope)
				}
			}
			if tc.fetcher.batchCallCount != tc.wantBatchCallCount {
				t.Errorf("batchCallCount = %d, want %d", tc.fetcher.batchCallCount, tc.wantBatchCallCount)
			}
			if diff := cmp.Diff(tc.wantBatchAssets, tc.fetcher.gotBatchAssetNames); diff != "" {
				t.Errorf("batch asset names mismatch (-want +got):\n%s", diff)
			}
			if tc.wantBatchCallCount > 0 && tc.fetcher.gotBatchParent != lookup.scope {
				t.Errorf("batch parent = %s, want %s", tc.fetcher.gotBatchParent, lookup.scope)
			}
		})
	}
}

func TestClusterResourceFetcherTask(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	completeCluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ProjectID:   "test-project",
		ClusterName: "test-cluster",
		Location:    "us-central1-a",
	}

	incompleteCluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ProjectID:   "test-project",
		ClusterName: "",
		Location:    "us-central1-a",
	}

	testCases := []struct {
		name                 string
		taskMode             inspectioncore.InspectionTaskModeType
		cluster              googlecloudk8scommon_contract.GoogleCloudClusterIdentity
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
			got, _, err := inspectiontest.RunInspectionTask(ctx, ClusterResourceFetcherTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputStartTimeTaskID.Ref(), startTime),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputEndTimeTaskID.Ref(), endTime),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientFactoryTaskID.Ref(), factory),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
				tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), tc.cluster),
				tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.InputKindFilterTaskID.Ref(), tc.kindFilter),
				tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref(), tc.namespaceFilter),
			)
			if err != nil {
				t.Fatalf("ClusterResourceFetcherTask unexpected error: %v", err)
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
