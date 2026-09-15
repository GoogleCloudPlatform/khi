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

package googlecloudk8scommon_impl

import (
	"context"
	"testing"
	"time"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
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
		name     string
		task     coretask.Task[*inspectioncore.AutocompleteResult[string]]
		cluster  googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		wantHint string
	}{
		{
			name:     "AutocompleteNamespacesTask returns hint when project ID is missing",
			task:     AutocompleteNamespacesTask,
			cluster:  googlecloudk8scommon_contract.GoogleCloudClusterIdentity{},
			wantHint: "Namespace names are suggested after the project ID, cluster name, and location are provided.",
		},
		{
			name: "AutocompletePodNamesTask returns hint when cluster name is missing",
			task: AutocompletePodNamesTask,
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID: "test-project",
			},
			wantHint: "Pod names are suggested after the project ID, cluster name, and location are provided.",
		},
		{
			name: "AutocompleteNodeNamesTask returns hint when location is missing",
			task: AutocompleteNodeNamesTask,
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				ClusterName: "test-cluster",
			},
			wantHint: "Node names are suggested after the project ID, cluster name, and location are provided.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			got, err := tasktest.RunTask(ctx, tc.task,
				tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), tc.cluster),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputStartTimeTaskID.Ref(), time.Unix(1000, 0)),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputEndTimeTaskID.Ref(), time.Unix(2000, 0)),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientFactoryTaskID.Ref(), nil),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(), nil),
				tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.AutocompleteMetricsK8sContainerTaskID.Ref(), "test-metric"),
				tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.AutocompleteMetricsK8sNodeTaskID.Ref(), "test-metric"),
			)
			if err != nil {
				t.Fatalf("RunTask() unexpected error: %v", err)
			}
			want := &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   tc.wantHint,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("RunTask() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
