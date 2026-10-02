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

package onpremapiaudit_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/onpremapiaudit"
)

// generateOnPremAPIStructuredQuery generates a structured query for OnPrem API audit logs.
func generateOnPremAPIStructuredQuery(clusterIdentity k8scommon.GoogleCloudClusterIdentity) *logestimator.StructuredLogQuery {
	return &logestimator.StructuredLogQuery{
		Incomplete:    !clusterIdentity.IsComplete(),
		ResourceTypes: []string{"audited_resource"},
		Filters: []logestimator.LoggingMonitoringMatcher{
			logestimator.ResourceLabel("service", logestimator.Exact("gkeonprem.googleapis.com")),
			logestimator.ResourceLabel("method", logestimator.ContainsAny("Update", "Create", "Delete", "Enroll", "Unenroll")),
			logestimator.LogID(logestimator.OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
			logestimator.CustomFilter(fmt.Sprintf(`protoPayload.resourceName:"projects/%s/locations/%s/"`, clusterIdentity.ProjectID, clusterIdentity.Location)),
			logestimator.CustomFilter(fmt.Sprintf(`protoPayload.resourceName:"%s"`, clusterIdentity.ClusterName)),
		},
	}
}

// onPremAuditQuerySource builds the on-prem API audit log query for the cluster read through the cluster identity input.
type onPremAuditQuerySource struct {
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *onPremAuditQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return []string{fmt.Sprintf("projects/%s", s.clusterIdentity.Get(ctx).ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *onPremAuditQuerySource) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	return []*logestimator.StructuredLogQuery{generateOnPremAPIStructuredQuery(s.clusterIdentity.Get(ctx))}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *onPremAuditQuerySource) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ gcpcommon.StructuredLogQuerySource = (*onPremAuditQuerySource)(nil)

// listLogEntriesTask queries on-prem API logs from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(onpremapiaudit.ListLogEntriesTaskID, "OnPrem API Logs", func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
	return &onPremAuditQuerySource{clusterIdentity: coretask.Use(b, onpremapiaudit.ClusterIdentityTaskID.Ref())}
})
