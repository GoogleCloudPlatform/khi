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

package k8scommon_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

func TestFilterAndTrimPrefixFromClusterNames(t *testing.T) {
	tests := []struct {
		name         string
		clusterNames []string
		prefix       string
		expected     []string
	}{
		{
			name:         "basic",
			clusterNames: []string{"awsClusters/cluster1", "cluster2", "awsClusters/cluster3"},
			prefix:       "awsClusters/",
			expected:     []string{"cluster1", "cluster3"},
		},
		{
			name:         "no match",
			clusterNames: []string{"cluster1", "cluster2", "cluster3"},
			prefix:       "awsClusters/",
			expected:     []string{},
		},
		{
			name:         "empty prefix(GKE)",
			clusterNames: []string{"cluster1", "awsClusters/cluster2", "cluster3"},
			prefix:       "",
			expected:     []string{"cluster1", "cluster3"},
		},
		{
			name:         "empty cluster names",
			clusterNames: []string{},
			prefix:       "awsClusters/",
			expected:     []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricsLabels := []map[string]string{}
			for _, clusterName := range tt.clusterNames {
				metricsLabels = append(metricsLabels, map[string]string{
					"cluster_name": clusterName,
				})
			}
			got := filterAndTrimPrefixFromClusterNames(metricsLabels, tt.prefix)
			gotClusterNames := []string{}
			for _, clusterName := range got {
				gotClusterNames = append(gotClusterNames, clusterName["cluster_name"])
			}
			if diff := cmp.Diff(tt.expected, gotClusterNames); diff != "" {
				t.Errorf("filterAndTrimPrefixFromClusterNames() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClusterScopedAutocompleteTasks_IncompleteClusterIdentity(t *testing.T) {
	testCases := []struct {
		name       string
		task       coretask.Task[*inspectioncore.AutocompleteResult[string]]
		metricsRef taskid.TaskReference[string]
		cluster    k8scommon.GoogleCloudClusterIdentity
		wantHint   string
	}{
		{
			name:       "autocompleteNamespacesTask returns hint when project ID is missing",
			task:       autocompleteNamespacesTask,
			metricsRef: k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref(),
			cluster:    k8scommon.GoogleCloudClusterIdentity{},
			wantHint:   "Namespace names are suggested after the project ID, cluster name, and location are provided.",
		},
		{
			name:       "autocompletePodNamesTask returns hint when cluster name is missing",
			task:       autocompletePodNamesTask,
			metricsRef: k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref(),
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ProjectID: "test-project",
			},
			wantHint: "Pod names are suggested after the project ID, cluster name, and location are provided.",
		},
		{
			name:       "autocompleteNodeNamesTask returns hint when location is missing",
			task:       autocompleteNodeNamesTask,
			metricsRef: k8scommon.AutocompleteMetricsK8sNodeTaskID.Ref(),
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				ClusterName: "test-cluster",
			},
			wantHint: "Node names are suggested after the project ID, cluster name, and location are provided.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.Run(t, ctx, tc.task, inspectioncore.TaskModeDryRun, map[string]any{},
				tasktest.Given(k8scommon.ClusterIdentityTaskID.Ref(), tc.cluster),
				tasktest.Given(gcpcommon.InputStartTimeTaskID.Ref(), time.Unix(1000, 0)),
				tasktest.Given(gcpcommon.InputEndTimeTaskID.Ref(), time.Unix(2000, 0)),
				tasktest.Given(gcpcommon.APIClientFactoryTaskID.Ref(), (*googlecloud.ClientFactory)(nil)),
				tasktest.Given(gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(), (*googlecloud.CallOptionInjector)(nil)),
				tasktest.Given(tc.metricsRef, "test-metric"),
			)
			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}
			want := &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   tc.wantHint,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Run() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
