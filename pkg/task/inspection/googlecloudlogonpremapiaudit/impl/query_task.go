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

package googlecloudlogonpremapiaudit_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogonpremapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogonpremapiaudit/contract"
)

// GenerateOnPremAPIStructuredQuery generates a structured query for OnPrem API audit logs.
func GenerateOnPremAPIStructuredQuery(clusterIdentity googlecloudk8scommon_contract.GoogleCloudClusterIdentity) *logestimator.StructuredLogQuery {
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

// GenerateOnPremAPIQuery generates a query for OnPrem API audit logs.
func GenerateOnPremAPIQuery(clusterIdentity googlecloudk8scommon_contract.GoogleCloudClusterIdentity) string {
	return GenerateOnPremAPIStructuredQuery(clusterIdentity).GenerateCloudLoggingQuery()
}

func generateQuery(clusterIdentity googlecloudk8scommon_contract.GoogleCloudClusterIdentity) string {
	return GenerateOnPremAPIQuery(clusterIdentity)
}

type onpremAPIListLogEntriesTaskSetting struct {
}

// DefaultResourceNames implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (o *onpremAPIListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudlogonpremapiaudit_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (o *onpremAPIListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogonpremapiaudit_contract.ClusterIdentityTaskID.Ref(),
	}
}

// QueryName implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (o *onpremAPIListLogEntriesTaskSetting) QueryName() string {
	return "OnPrem API Logs"
}

// Queries implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (o *onpremAPIListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogonpremapiaudit_contract.ClusterIdentityTaskID.Ref())
	return []*logestimator.StructuredLogQuery{GenerateOnPremAPIStructuredQuery(clusterIdentity)}, nil
}

// TaskID implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (o *onpremAPIListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogonpremapiaudit_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (o *onpremAPIListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.StructuredListLogEntriesTaskSetting = (*onpremAPIListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewStructuredListLogEntriesTask(&onpremAPIListLogEntriesTaskSetting{})
