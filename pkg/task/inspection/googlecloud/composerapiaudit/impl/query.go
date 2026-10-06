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

package composerapiaudit_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// generateComposerAuditStructuredQuery generates a structured query for Cloud Composer audit logs.
func generateComposerAuditStructuredQuery(projectID, location, environmentName string) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(projectID)),
	}
	if location != "" && location != "all" {
		filters = append(filters, logestimator.ResourceLabel("location", logestimator.Exact(location)))
	}
	if environmentName != "" {
		filters = append(filters, logestimator.ResourceLabel("environment_name", logestimator.Exact(environmentName)))
	}
	filters = append(filters,
		logestimator.LogID(logestimator.OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
		logestimator.CustomFilter(`protoPayload.serviceName="composer.googleapis.com"`),
	)

	return &logestimator.StructuredLogQuery{
		Incomplete:    projectID == "" || environmentName == "",
		ResourceTypes: []string{"cloud_composer_environment"},
		Filters:       filters,
	}
}

// composerAuditQuerySource builds the Cloud Composer audit log query for the cluster and environment read through the inputs.
type composerAuditQuerySource struct {
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	environmentName coretask.Input[string]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *composerAuditQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return []string{fmt.Sprintf("projects/%s", s.clusterIdentity.Get(ctx).ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *composerAuditQuerySource) Queries(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*logestimator.StructuredLogQuery, error) {
	clusterIdentity := s.clusterIdentity.Get(ctx)
	environmentName := s.environmentName.Get(ctx)
	return []*logestimator.StructuredLogQuery{generateComposerAuditStructuredQuery(clusterIdentity.ProjectID, clusterIdentity.Location, environmentName)}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *composerAuditQuerySource) TimePartitionCount() int {
	return 1
}

var _ gcpcommon.StructuredLogQuerySource = (*composerAuditQuerySource)(nil)

// listLogEntriesTask queries Cloud Composer audit logs from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(composerapiaudit.ListLogEntriesTaskID, "Composer API Audit logs", func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
	return &composerAuditQuerySource{
		clusterIdentity: coretask.Use(b, composerapiaudit.ClusterIdentityTaskID.Ref()),
		environmentName: coretask.Use(b, composercluster.InputComposerEnvironmentNameTaskID.Ref()),
	}
})
