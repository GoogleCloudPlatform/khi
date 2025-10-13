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
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

type mockComposerEnvironmentFetcher struct {
	responsePairs map[string][]string // {projectID}/{location}
	responseError bool
}

// GetEnvironmentNames implements googlecloudclustercomposer_contract.ComposerEnvironmentListFetcher.
func (m *mockComposerEnvironmentFetcher) GetEnvironmentNames(ctx context.Context, projectID string, location string) ([]string, error) {
	if m.responseError {
		return nil, fmt.Errorf("test error")
	}
	return m.responsePairs[projectID+"/"+location], nil
}

var _ googlecloudclustercomposer_contract.ComposerEnvironmentListFetcher = (*mockComposerEnvironmentFetcher)(nil)

func TestAutocompleteComposerEnvironmentNamesTask(t *testing.T) {
	testCases := []struct {
		desc                            string
		projectIDs                      []string
		locations                       []string
		projectIDLocationToClusterNames map[string][]string // {projectID}/{location}
		listError                       bool
		want                            [][]string
	}{
		{
			desc:                            "project id is empty",
			projectIDs:                      []string{""},
			locations:                       []string{"us-central1"},
			projectIDLocationToClusterNames: map[string][]string{},
			listError:                       false,
			want:                            [][]string{{}},
		},
		{
			desc:                            "location is empty",
			projectIDs:                      []string{"foo-project"},
			locations:                       []string{""},
			projectIDLocationToClusterNames: map[string][]string{},
			listError:                       false,
			want:                            [][]string{{}},
		},
		{
			desc:                            "using cache",
			projectIDs:                      []string{"foo-project", "foo-project"},
			locations:                       []string{"us-central1", "us-central1"},
			projectIDLocationToClusterNames: map[string][]string{"foo-project/us-central1": {"env1", "env2"}},
			listError:                       false,
			want:                            [][]string{{"env1", "env2"}, {"env1", "env2"}},
		},
		{
			desc:                            "with error",
			projectIDs:                      []string{"foo-project"},
			locations:                       []string{"us-central1"},
			projectIDLocationToClusterNames: map[string][]string{},
			listError:                       true,
			want:                            [][]string{{}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

			mockComposerEnvironmentFetcherInput := tasktest.NewTaskDependencyValuePair[googlecloudclustercomposer_contract.ComposerEnvironmentListFetcher](
				googlecloudclustercomposer_contract.ComposerEnvironmentListFetcherTaskID.Ref(),
				&mockComposerEnvironmentFetcher{
					responsePairs: tc.projectIDLocationToClusterNames,
					responseError: tc.listError,
				},
			)

			for i := 0; i < len(tc.projectIDs); i++ {
				projectIDInput := tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputProjectIdTaskID.Ref(), tc.projectIDs[i])
				locationInput := tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputLocationsTaskID.Ref(), tc.locations[i])
				result, _, err := inspectiontest.RunInspectionTask(ctx, AutocompleteComposerEnvironmentNamesTask, inspectioncore_contract.TaskModeDryRun, map[string]any{}, projectIDInput, locationInput, mockComposerEnvironmentFetcherInput)
				if err != nil {
					t.Fatalf("failed to run inspection task in loop %d: %v", i, err)
				}

				if diff := cmp.Diff(tc.want[i], result); diff != "" {
					t.Errorf("result of AutocompleteComposerEnvironmentNamesTask mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}

}
