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

package googlecloudclustercomposer_impl

import (
	"context"
	"fmt"
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

type mockComposerClusterFinder struct {
	clusterMapping map[string]string // {projectID}/{environment} -> clusterName
	wantError      bool
}

// GetGKEClusterName implements googlecloudclustercomposer_contract.ComposerEnvironmentClusterFinder.
func (m *mockComposerClusterFinder) GetGKEClusterName(ctx context.Context, projectID string, environment string) (string, error) {
	if m.wantError {
		return "", fmt.Errorf("test error")
	}
	if clusterName, ok := m.clusterMapping[projectID+"/"+environment]; ok {
		return clusterName, nil
	}
	return "", googlecloudclustercomposer_contract.ErrEnvironmentClusterNotFound
}

var _ googlecloudclustercomposer_contract.ComposerEnvironmentClusterFinder = (*mockComposerClusterFinder)(nil)

func TestAutocompleteComposerClusterNamesTask(t *testing.T) {
	testCases := []struct {
		desc           string
		clusterMapping map[string]string
		finderError    bool
		projectIDs     []string
		environments   []string
		want           []*googlecloudk8scommon_contract.AutocompleteResult
	}{
		{
			desc:           "project id is empty",
			clusterMapping: map[string]string{},
			finderError:    false,
			projectIDs:     []string{""},
			environments:   []string{"env1"},
			want: []*googlecloudk8scommon_contract.AutocompleteResult{{
				Values: []string{},
				Error:  "Project ID or Composer environment name is empty",
			}},
		},
		{
			desc:           "environment name is empty",
			clusterMapping: map[string]string{},
			finderError:    false,
			projectIDs:     []string{"foo-project"},
			environments:   []string{""},
			want: []*googlecloudk8scommon_contract.AutocompleteResult{{
				Values: []string{},
				Error:  "Project ID or Composer environment name is empty",
			}},
		},
		{
			desc:           "using cache",
			clusterMapping: map[string]string{"foo-project/env1": "cluster1"},
			finderError:    false,
			projectIDs:     []string{"foo-project", "foo-project"},
			environments:   []string{"env1", "env1"},
			want: []*googlecloudk8scommon_contract.AutocompleteResult{
				{
					Values: []string{"cluster1"},
				},
				{
					Values: []string{"cluster1"},
				},
			},
		},
		{
			desc:           "with error",
			clusterMapping: map[string]string{},
			finderError:    true,
			projectIDs:     []string{"foo-project"},
			environments:   []string{"env1"},
			want: []*googlecloudk8scommon_contract.AutocompleteResult{{
				Values: []string{},
				Error:  "Failed to fetch the list GKE cluster. Please confirm if the Project ID is correct, or retry later",
			}},
		},
		{
			desc:           "environment not found",
			clusterMapping: map[string]string{},
			finderError:    false,
			projectIDs:     []string{"foo-project"},
			environments:   []string{"non-existent-env"},
			want: []*googlecloudk8scommon_contract.AutocompleteResult{{
				Values: []string{},
				Error: `Not found. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.
Note: Composer 3 does not run on your GKE. Please remove all Kubernetes/GKE questies from the previous section.`,
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

			mockComposerClusterFinderInput := tasktest.NewTaskDependencyValuePair[googlecloudclustercomposer_contract.ComposerEnvironmentClusterFinder](
				googlecloudclustercomposer_contract.ComposerEnvironmentClusterFinderTaskID.Ref(),
				&mockComposerClusterFinder{
					clusterMapping: tc.clusterMapping,
					wantError:      tc.finderError,
				},
			)

			for i := 0; i < len(tc.projectIDs); i++ {
				projectIDInput := tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputProjectIdTaskID.Ref(), tc.projectIDs[i])
				environmentNameInput := tasktest.NewTaskDependencyValuePair(googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(), tc.environments[i])
				result, _, err := inspectiontest.RunInspectionTask(ctx, AutocompleteComposerClusterNamesTask, inspectioncore_contract.TaskModeDryRun, map[string]any{}, projectIDInput, environmentNameInput, mockComposerClusterFinderInput)
				if err != nil {
					t.Fatalf("failed to run inspection task in loop %d: %v", i, err)
				}

				if diff := cmp.Diff(tc.want[i], result); diff != "" {
					t.Errorf("result of AutocompleteComposerClusterNamesTask mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
