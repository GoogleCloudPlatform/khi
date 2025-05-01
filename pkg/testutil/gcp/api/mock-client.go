// Copyright 2024 Google LLC
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

package api_test

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
)

type MockApiClient struct {
	GetClusterNamesFunc func(ctx context.Context, projectId string) ([]string, error)
	ListLogEntriesFunc  func(ctx context.Context, resourceNames []string, filter string, logSink chan any) error
}

// GetClusters implements api.GCPClient.
func (m *MockApiClient) GetClusters(ctx context.Context, projectId string) ([]api.Cluster, error) {
	return []api.Cluster{
		{
			Name: "gke-cluster-foo",
		},
		{
			Name: "composer-environment-foo",
			ResourceLabels: map[string]string{
				"goog-composer-environment": "dev",
			},
		},
	}, nil
}

// ListRegions implements api.GCPClient.
func (m *MockApiClient) ListRegions(ctx context.Context, projectId string) ([]string, error) {
	return []string{"us-central1", "us-east1"}, nil
}

// GetAnthosAWSClusterNames implements api.GCPClient.
func (m *MockApiClient) GetAnthosAWSClusterNames(ctx context.Context, projectId string) ([]string, error) {
	if m.GetClusterNamesFunc == nil {
		return []string{"aws-cluster-foo", "aws-cluster-bar"}, nil
	}
	return m.GetClusterNamesFunc(ctx, projectId)
}

// GetAnthosAzureClusterNames implements api.GCPClient.
func (m *MockApiClient) GetAnthosAzureClusterNames(ctx context.Context, projectId string) ([]string, error) {
	if m.GetClusterNamesFunc == nil {
		return []string{"azure-cluster-foo", "azure-cluster-bar"}, nil
	}
	return m.GetClusterNamesFunc(ctx, projectId)
}

// GetAnthosOnBaremetalClusterNames implements api.GCPClient.
func (m *MockApiClient) GetAnthosOnBaremetalClusterNames(ctx context.Context, projectId string) ([]string, error) {
	if m.GetClusterNamesFunc == nil {
		return []string{"baremetal-cluster-foo", "baremetal-cluster-bar"}, nil
	}
	return m.GetClusterNamesFunc(ctx, projectId)
}

// GetAnthosOnVMWareClusterNames implements api.GCPClient.
func (m *MockApiClient) GetAnthosOnVMWareClusterNames(ctx context.Context, projectId string) ([]string, error) {
	if m.GetClusterNamesFunc == nil {
		return []string{"vmware-cluster-foo", "vmware-cluster-bar"}, nil
	}
	return m.GetClusterNamesFunc(ctx, projectId)
}

// GetClusterNames implements api.GCPClient.
func (m *MockApiClient) GetClusterNames(ctx context.Context, projectId string) ([]string, error) {
	if m.GetClusterNamesFunc == nil {
		return []string{"gke-cluster-foo", "gke-cluster-bar"}, nil
	}
	return m.GetClusterNamesFunc(ctx, projectId)
}

func (m *MockApiClient) GetComposerEnvironmentNames(ctx context.Context, projectId string, location string) ([]string, error) {
	// GetClusterNamesFunc is not for Composer environment? Yes, but it's fine since it is a mock! :D
	if m.GetClusterNamesFunc == nil {
		return []string{"composer-environment-foo", "composer-environment-bar"}, nil
	}
	return m.GetClusterNamesFunc(ctx, projectId)
}

// ListLogEntries implements api.GCPClient.
func (m *MockApiClient) ListLogEntries(ctx context.Context, resourceNames []string, filter string, logSink chan any) error {
	if m.ListLogEntriesFunc == nil {
		close(logSink)
		return nil
	}
	return m.ListLogEntriesFunc(ctx, resourceNames, filter, logSink)
}

var _ api.GCPClient = (*MockApiClient)(nil)
