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

package composerairflow_impl

import (
	"context"
	"fmt"
	"slices"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

// GenerateComposerLogsStructuredQuery generates a structured query for Composer environment logs.
func GenerateComposerLogsStructuredQuery(projectID, location, environmentName string, selectedComponents []string) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(projectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(location)),
		logestimator.ResourceLabel("environment_name", logestimator.Exact(environmentName)),
	}

	if !slices.Contains(selectedComponents, "@any") && len(selectedComponents) > 0 {
		filters = append(filters, logestimator.LogID(logestimator.OneOf(selectedComponents...)))
	}

	filters = append(filters, logestimator.LogID(logestimator.NoneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")))

	return &logestimator.StructuredLogQuery{
		Incomplete:    projectID == "" || location == "" || environmentName == "",
		ResourceTypes: []string{"cloud_composer_environment"},
		Filters:       filters,
	}
}

// GenerateComposerLogsQuery generates a query for Composer environment logs.
func GenerateComposerLogsQuery(projectID, location, environmentName string, selectedComponents []string) string {
	return GenerateComposerLogsStructuredQuery(projectID, location, environmentName, selectedComponents).GenerateCloudLoggingQuery()
}

// composerLogsQuerySource builds the Cloud Composer logs query for the cluster, environment, and components read through the inputs.
type composerLogsQuerySource struct {
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	environmentName coretask.Input[string]
	components      coretask.Input[[]string]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *composerLogsQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	clusterIdentity := s.clusterIdentity.Get(ctx)
	return []string{fmt.Sprintf("projects/%s", clusterIdentity.ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *composerLogsQuerySource) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	clusterIdentity := s.clusterIdentity.Get(ctx)
	environmentName := s.environmentName.Get(ctx)
	selectedComponents := s.components.Get(ctx)

	return []*logestimator.StructuredLogQuery{GenerateComposerLogsStructuredQuery(clusterIdentity.ProjectID, clusterIdentity.Location, environmentName, selectedComponents)}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *composerLogsQuerySource) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ gcpcommon.StructuredLogQuerySource = (*composerLogsQuerySource)(nil)

// composerLogsQueryTask defines a task that gathers logs from Cloud Logging for multiple Composer components.
var composerLogsQueryTask = gcpcommon.DefineStructuredListLogEntriesTask(
	composerairflow.ComposerLogsQueryTaskID,
	"Composer Environment Logs",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &composerLogsQuerySource{
			clusterIdentity: coretask.Use(b, composercluster.ClusterIdentityTaskID.Ref()),
			environmentName: coretask.Use(b, composercluster.InputComposerEnvironmentNameTaskID.Ref()),
			components:      coretask.Use(b, composerairflow.InputComposerComponentsTaskID.Ref()),
		}
	},
)
