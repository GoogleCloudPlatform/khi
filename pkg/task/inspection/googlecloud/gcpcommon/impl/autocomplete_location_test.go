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

package gcpcommon_impl

import (
	"context"
	"errors"
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

// fakeLocationFetcher returns fixed regions or an error and records the project IDs it was called with.
type fakeLocationFetcher struct {
	regions    []string
	err        error
	projectIDs []string
}

// FetchRegions implements gcpcommon.LocationFetcher.
func (f *fakeLocationFetcher) FetchRegions(ctx context.Context, projectID string) ([]string, error) {
	f.projectIDs = append(f.projectIDs, projectID)
	return f.regions, f.err
}

var _ gcpcommon.LocationFetcher = (*fakeLocationFetcher)(nil)

func TestAutocompleteLocationTask(t *testing.T) {
	testCases := []struct {
		desc                  string
		projectID             string
		fetcher               *fakeLocationFetcher
		want                  []string
		wantFetchedProjectIDs []string
	}{
		{
			desc:      "returns the regions of the project",
			projectID: "test-project",
			fetcher: &fakeLocationFetcher{
				regions: []string{"us-central1", "asia-northeast1"},
			},
			want:                  []string{"us-central1", "asia-northeast1"},
			wantFetchedProjectIDs: []string{"test-project"},
		},
		{
			desc:                  "returns no regions without calling the fetcher when the project ID is empty",
			projectID:             "",
			fetcher:               &fakeLocationFetcher{},
			want:                  []string{},
			wantFetchedProjectIDs: nil,
		},
		{
			desc:      "returns no regions when fetching regions fails",
			projectID: "test-project",
			fetcher: &fakeLocationFetcher{
				err: errors.New("fetch failed"),
			},
			want:                  []string{},
			wantFetchedProjectIDs: []string{"test-project"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.Run(t, ctx, autocompleteLocationTask, inspectioncore.TaskModeDryRun, map[string]any{},
				tasktest.Given(gcpcommon.InputProjectIdTaskID.Ref(), tc.projectID),
				tasktest.Given[gcpcommon.LocationFetcher](gcpcommon.LocationFetcherTaskID.Ref(), tc.fetcher),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got.Values); diff != "" {
				t.Errorf("autocompleteLocationTask values mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantFetchedProjectIDs, tc.fetcher.projectIDs); diff != "" {
				t.Errorf("fetched project IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
