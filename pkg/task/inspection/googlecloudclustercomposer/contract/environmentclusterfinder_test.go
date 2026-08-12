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

package googlecloudclustercomposer_contract

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractEnvironmentPrefixCandidate(t *testing.T) {
	testCases := []struct {
		name        string
		clusterName string
		location    string
		wantCand    string
		wantOK      bool
	}{
		{
			name:        "valid environment candidate",
			clusterName: "asia-northeast1-my-env-12345678-gke",
			location:    "asia-northeast1",
			wantCand:    "my-env",
			wantOK:      true,
		},
		{
			name:        "valid truncated environment candidate",
			clusterName: "asia-northeast1-very-long-comp-12345678-gke",
			location:    "asia-northeast1",
			wantCand:    "very-long-comp",
			wantOK:      true,
		},
		{
			name:        "missing gke suffix",
			clusterName: "asia-northeast1-my-env-12345678-aks",
			location:    "asia-northeast1",
			wantCand:    "",
			wantOK:      false,
		},
		{
			name:        "location mismatch",
			clusterName: "us-central1-my-env-12345678-gke",
			location:    "asia-northeast1",
			wantCand:    "",
			wantOK:      false,
		},
		{
			name:        "no hyphen after location prefix in rest",
			clusterName: "asia-northeast1-12345678-gke",
			location:    "asia-northeast1",
			wantCand:    "",
			wantOK:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotCand, gotOK := extractEnvironmentPrefixCandidate(tc.clusterName, tc.location)
			if gotOK != tc.wantOK {
				t.Fatalf("extractEnvironmentPrefixCandidate() ok mismatch: want %v, got %v", tc.wantOK, gotOK)
			}
			if diff := cmp.Diff(tc.wantCand, gotCand); diff != "" {
				t.Errorf("extractEnvironmentPrefixCandidate() candidate mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFilterAndMatchComposerGKEClusterNames(t *testing.T) {
	testCases := []struct {
		name          string
		metricsLabels []map[string]string
		location      string
		environment   string
		want          []string
	}{
		{
			name: "matches single cluster",
			metricsLabels: []map[string]string{
				{
					"cluster_name": "asia-northeast1-my-env-12345678-gke",
					"location":     "asia-northeast1",
				},
			},
			location:    "asia-northeast1",
			environment: "my-env",
			want:        []string{"asia-northeast1-my-env-12345678-gke"},
		},
		{
			name: "matches multiple past clusters and sorts them",
			metricsLabels: []map[string]string{
				{
					"cluster_name": "asia-northeast1-my-env-87654321-gke",
					"location":     "asia-northeast1",
				},
				{
					"cluster_name": "asia-northeast1-my-env-12345678-gke",
					"location":     "asia-northeast1",
				},
			},
			location:    "asia-northeast1",
			environment: "my-env",
			want: []string{
				"asia-northeast1-my-env-12345678-gke",
				"asia-northeast1-my-env-87654321-gke",
			},
		},
		{
			name: "ignores different location",
			metricsLabels: []map[string]string{
				{
					"cluster_name": "us-central1-my-env-12345678-gke",
					"location":     "us-central1",
				},
			},
			location:    "asia-northeast1",
			environment: "my-env",
			want:        nil,
		},
		{
			name: "ignores missing gke suffix",
			metricsLabels: []map[string]string{
				{
					"cluster_name": "asia-northeast1-my-env-12345678-aks",
					"location":     "asia-northeast1",
				},
			},
			location:    "asia-northeast1",
			environment: "my-env",
			want:        nil,
		},
		{
			name: "matches truncated environment name",
			metricsLabels: []map[string]string{
				{
					"cluster_name": "asia-northeast1-very-long-comp-12345678-gke",
					"location":     "asia-northeast1",
				},
			},
			location:    "asia-northeast1",
			environment: "very-long-composer-environment-name",
			want:        []string{"asia-northeast1-very-long-comp-12345678-gke"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterAndMatchComposerGKEClusterNames(tc.metricsLabels, tc.location, tc.environment)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("filterAndMatchComposerGKEClusterNames() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
