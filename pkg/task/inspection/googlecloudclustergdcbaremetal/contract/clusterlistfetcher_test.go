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

package googlecloudclustergdcbaremetal_contract

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetAdminAndUserClusters(t *testing.T) {
	tests := []struct {
		name              string
		project           string
		fetchAdminCluster fetchClusterFunc
		fetchUserCluster  fetchClusterFunc
		want              []string
		wantErr           bool
		expectedErr       error
	}{
		{
			name:    "successful fetch",
			project: "test-project",
			fetchAdminCluster: func(ctx context.Context, parent string) ([]string, error) {
				return []string{"admin-cluster-1", "admin-cluster-2"}, nil
			},
			fetchUserCluster: func(ctx context.Context, parent string) ([]string, error) {
				return []string{"user-cluster-1", "user-cluster-2"}, nil
			},
			want:    []string{"admin-cluster-1", "admin-cluster-2", "user-cluster-1", "user-cluster-2"},
			wantErr: false,
		},
		{
			name:    "error in fetchAdminCluster",
			project: "test-project",
			fetchAdminCluster: func(ctx context.Context, parent string) ([]string, error) {
				return nil, fmt.Errorf("admin cluster fetch error")
			},
			fetchUserCluster: func(ctx context.Context, parent string) ([]string, error) {
				return []string{"user-cluster-1"}, nil
			},
			wantErr:     true,
			expectedErr: fmt.Errorf("admin cluster fetch error"),
		},
		{
			name:    "error in fetchUserCluster",
			project: "test-project",
			fetchAdminCluster: func(ctx context.Context, parent string) ([]string, error) {
				return []string{"admin-cluster-1"}, nil
			},
			fetchUserCluster: func(ctx context.Context, parent string) ([]string, error) {
				return nil, fmt.Errorf("user cluster fetch error")
			},
			wantErr:     true,
			expectedErr: fmt.Errorf("user cluster fetch error"),
		},
		{
			name:    "empty results",
			project: "test-project",
			fetchAdminCluster: func(ctx context.Context, parent string) ([]string, error) {
				return []string{}, nil
			},
			fetchUserCluster: func(ctx context.Context, parent string) ([]string, error) {
				return []string{}, nil
			},
			want:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAdminAndUserClusters(context.Background(), tt.project, tt.fetchAdminCluster, tt.fetchUserCluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAdminAndUserClusters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.expectedErr.Error() {
					t.Errorf("getAdminAndUserClusters() error = %v, expectedErr %v", err, tt.expectedErr)
				}
				return
			}

			// Use a map to ignore order
			gotMap := make(map[string]bool)
			for _, s := range got {
				gotMap[s] = true
			}
			wantMap := make(map[string]bool)
			for _, s := range tt.want {
				wantMap[s] = true
			}

			if diff := cmp.Diff(gotMap, wantMap); diff != "" {
				t.Errorf("getAdminAndUserClusters() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestToShortClusterName(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid user cluster name",
			input: "projects/my-project/locations/us-central1/baremetalClusters/user-cluster-1",
			want:  "user-cluster-1",
		},
		{
			name:  "valid admin cluster name",
			input: "projects/my-project/locations/us-central1/baremetalAdminClusters/admin-cluster-1",
			want:  "admin-cluster-1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := toShortClusterName(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("toShortClusterName() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
