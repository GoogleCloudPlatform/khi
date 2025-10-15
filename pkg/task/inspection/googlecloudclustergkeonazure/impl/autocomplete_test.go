// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package googlecloudclustergkeonazure_impl

import (
	"context"
	"fmt"
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudclustergkeonazure_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonazure/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

type mockGKEOnAzureClusterListFetcher struct {
	responsePairs     map[string][]string
	responseWithError bool
}

func (m *mockGKEOnAzureClusterListFetcher) GetClusterNames(ctx context.Context, projectID string) ([]string, error) {
	if m.responseWithError {
		return nil, fmt.Errorf("test error")
	}
	return m.responsePairs[projectID], nil
}

var _ googlecloudclustergkeonazure_contract.ClusterListFetcher = (*mockGKEOnAzureClusterListFetcher)(nil)

func TestAutocompleteGKEOnAzureClusterNames(t *testing.T) {
	testCase := []struct {
		desc        string
		clusterList map[string][]string
		listError   bool
		projectIDs  []string
		want        []*googlecloudk8scommon_contract.AutocompleteClusterNameList
	}{
		{
			desc:        "project id is empty",
			clusterList: map[string][]string{},
			projectIDs:  []string{""},
			want: []*googlecloudk8scommon_contract.AutocompleteClusterNameList{
				{
					ClusterNames: []string{},
					Error:        "Project ID is empty",
				},
			},
		},
		{
			desc: "multiple call for single project",
			clusterList: map[string][]string{
				"foo": {"azure-cluster-1", "azure-cluster-2"},
			},
			projectIDs: []string{"foo", "foo"},
			want: []*googlecloudk8scommon_contract.AutocompleteClusterNameList{
				{
					ClusterNames: []string{"azure-cluster-1", "azure-cluster-2"},
					Error:        "",
				},
				{
					ClusterNames: []string{"azure-cluster-1", "azure-cluster-2"},
					Error:        "",
				},
			},
		},
		{
			desc: "multiple projects",
			clusterList: map[string][]string{
				"foo": {"azure-cluster-1", "azure-cluster-2"},
				"bar": {"test-cluster-a", "test-cluster-b"},
			},
			projectIDs: []string{"foo", "bar"},
			want: []*googlecloudk8scommon_contract.AutocompleteClusterNameList{
				{
					ClusterNames: []string{"azure-cluster-1", "azure-cluster-2"},
					Error:        "",
				},
				{
					ClusterNames: []string{"test-cluster-a", "test-cluster-b"},
					Error:        "",
				},
			},
		},
		{
			desc:        "with error",
			clusterList: map[string][]string{},
			listError:   true,
			projectIDs:  []string{"foo"},
			want: []*googlecloudk8scommon_contract.AutocompleteClusterNameList{
				{
					ClusterNames: []string{},
					Error:        "Failed to get the list from API:test error",
				},
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

			mockClusterListFetcherInput := tasktest.NewTaskDependencyValuePair[googlecloudclustergkeonazure_contract.ClusterListFetcher](googlecloudclustergkeonazure_contract.ClusterListFetcherTaskID.Ref(), &mockGKEOnAzureClusterListFetcher{
				responsePairs:     tc.clusterList,
				responseWithError: tc.listError,
			})
			for i := 0; i < len(tc.projectIDs); i++ {
				projectIDInput := tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputProjectIdTaskID.Ref(), tc.projectIDs[i])
				result, _, err := inspectiontest.RunInspectionTask(ctx, AutocompleteGKEOnAzureClusterNamesTask, inspectioncore_contract.TaskModeDryRun, map[string]any{}, projectIDInput, mockClusterListFetcherInput)
				if err != nil {
					t.Fatalf("failed to run inspection task in loop %d: %v", i, err)
				}

				if diff := cmp.Diff(tc.want[i], result); diff != "" {
					t.Errorf("result of AutocompleteGKEOnAzureClusterNames mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
