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

package googlecloudlogcsm_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
)

// GenerateCSMTrafficLogsStructuredQuery generates a structured query for CSM Traffic logs.
func GenerateCSMTrafficLogsStructuredQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, responseFlagsSetFilter *gcpqueryutil.SetFilterParseResult, namespaceSetFilter *gcpqueryutil.SetFilterParseResult) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(cluster.ProjectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(cluster.Location)),
		logestimator.ResourceLabel("cluster_name", logestimator.Exact(cluster.NameFor(googlecloudk8scommon_contract.ClusterNameUsageCSM))),
	}

	if nsFilter := namespaceStructuredMatcher(namespaceSetFilter); nsFilter != nil {
		filters = append(filters, nsFilter)
	}

	filters = append(filters, logestimator.LogID(logestimator.OneOf("server-accesslog-stackdriver", "client-accesslog-stackdriver")))

	if responseFlagsFilter := responseFlagsStructuredMatcher(responseFlagsSetFilter); responseFlagsFilter != nil {
		filters = append(filters, responseFlagsFilter)
	}

	return &logestimator.StructuredLogQuery{
		Incomplete: !cluster.IsComplete(),
		Filters:    filters,
	}
}

// GenerateCSMTrafficLogsQuery generates a query for CSM Traffic logs.
func GenerateCSMTrafficLogsQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, responseFlagsSetFilter *gcpqueryutil.SetFilterParseResult, namespaceSetFilter *gcpqueryutil.SetFilterParseResult) string {
	return GenerateCSMTrafficLogsStructuredQuery(cluster, responseFlagsSetFilter, namespaceSetFilter).GenerateCloudLoggingQuery()
}

func responseFlagsStructuredMatcher(responseFlagsFilter *gcpqueryutil.SetFilterParseResult) logestimator.LoggingMonitoringMatcher {
	if responseFlagsFilter == nil {
		return nil
	}
	if responseFlagsFilter.ValidationError != "" {
		return logestimator.Comment(fmt.Sprintf(`Failed to generate response flags filter due to the validation error "%s"`, responseFlagsFilter.ValidationError))
	}
	if responseFlagsFilter.SubtractMode {
		if len(responseFlagsFilter.Subtractives) == 0 {
			return nil
		}
		return logestimator.CustomFilter(fmt.Sprintf(`-labels.response_flag:(%s)`, strings.Join(responseFlagsFilter.SubtractivesWithQuotes(), " OR ")))
	}

	if len(responseFlagsFilter.Additives) == 0 {
		return logestimator.Comment(`Invalid: none of the resources will be selected. Ignoring response flag filter.`)
	}
	return logestimator.CustomFilter(fmt.Sprintf(`labels.response_flag:(%s)`, strings.Join(responseFlagsFilter.AdditivesWithQuotes(), " OR ")))
}

func namespaceStructuredMatcher(filter *gcpqueryutil.SetFilterParseResult) logestimator.LoggingMonitoringMatcher {
	if filter == nil {
		return nil
	}
	if filter.ValidationError != "" {
		return logestimator.Comment(fmt.Sprintf(`Failed to generate namespace filter due to the validation error "%s"`, filter.ValidationError))
	}
	if filter.SubtractMode {
		return logestimator.Comment("Unsupported operation")
	}
	selectedNamespaces := []string{}
	for _, additive := range filter.Additives {
		if strings.HasPrefix(additive, "#") {
			if additive == "#namespaced" {
				return nil
			}
			continue
		}
		selectedNamespaces = append(selectedNamespaces, additive)
	}
	if len(selectedNamespaces) == 0 {
		return logestimator.WithComment(logestimator.ResourceLabel("namespace_name", logestimator.Exact("")), "Invalid: No namespaces remain to filter for CSM traffic logs.")
	}
	return logestimator.ResourceLabel("namespace_name", logestimator.ContainsAny(selectedNamespaces...))
}

type CSMTrafficLogListLogEntryTaskSetting struct{}

// DefaultResourceNames implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *CSMTrafficLogListLogEntryTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *CSMTrafficLogListLogEntryTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogcsm_contract.ClusterIdentityTaskID.Ref(),
		googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref(),
		googlecloudlogcsm_contract.InputCSMResponseFlagsTaskID.Ref(),
	}
}

// QueryName implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *CSMTrafficLogListLogEntryTaskSetting) QueryName() string {
	return "CSM Traffic logs"
}

// Queries implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *CSMTrafficLogListLogEntryTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.ClusterIdentityTaskID.Ref())
	namespaceFilter := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref())
	responseFlagsFilter := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.InputCSMResponseFlagsTaskID.Ref())
	return []*logestimator.StructuredLogQuery{GenerateCSMTrafficLogsStructuredQuery(cluster, responseFlagsFilter, namespaceFilter)}, nil
}

// TaskID implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *CSMTrafficLogListLogEntryTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogcsm_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *CSMTrafficLogListLogEntryTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.StructuredListLogEntriesTaskSetting = (*CSMTrafficLogListLogEntryTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewStructuredListLogEntriesTask(&CSMTrafficLogListLogEntryTaskSetting{})
