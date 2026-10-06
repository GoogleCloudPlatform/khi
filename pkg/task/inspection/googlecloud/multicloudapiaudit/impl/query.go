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

package multicloudapiaudit_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/multicloudapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// generateMultiCloudAPIStructuredQuery generates a structured query for multicloud API logs.
func generateMultiCloudAPIStructuredQuery(clusterIdentity k8scommon.GoogleCloudClusterIdentity) *logestimator.StructuredLogQuery {
	return &logestimator.StructuredLogQuery{
		Incomplete:    !clusterIdentity.IsComplete(),
		ResourceTypes: []string{"audited_resource"},
		Filters: []logestimator.LoggingMonitoringMatcher{
			logestimator.ResourceLabel("service", logestimator.Exact("gkemulticloud.googleapis.com")),
			logestimator.ResourceLabel("method", logestimator.ContainsAny("Update", "Create", "Delete")),
			logestimator.LogID(logestimator.OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
			logestimator.CustomFilter(fmt.Sprintf(`protoPayload.resourceName:"projects/%s/locations/%s/"`, clusterIdentity.ProjectID, clusterIdentity.Location)),
			logestimator.CustomFilter(fmt.Sprintf(`protoPayload.resourceName:"%s"`, clusterIdentity.NameFor(k8scommon.ClusterNameUsageK8sPlatformAudit))),
		},
	}
}

// multiCloudAuditQuerySource builds the multicloud API audit log query for the cluster read through the cluster identity input.
type multiCloudAuditQuerySource struct {
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *multiCloudAuditQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return []string{fmt.Sprintf("projects/%s", s.clusterIdentity.Get(ctx).ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *multiCloudAuditQuerySource) Queries(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*logestimator.StructuredLogQuery, error) {
	return []*logestimator.StructuredLogQuery{generateMultiCloudAPIStructuredQuery(s.clusterIdentity.Get(ctx))}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *multiCloudAuditQuerySource) TimePartitionCount() int {
	return 1
}

var _ gcpcommon.StructuredLogQuerySource = (*multiCloudAuditQuerySource)(nil)

// listLogEntriesTask queries multicloud API logs from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(multicloudapiaudit.ListLogEntriesTaskID, "Multicloud API Logs", func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
	return &multiCloudAuditQuerySource{clusterIdentity: coretask.Use(b, multicloudapiaudit.ClusterIdentityTaskID.Ref())}
})
