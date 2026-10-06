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

package csm_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csm"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// generateCSMTrafficLogsStructuredQuery generates a structured query for CSM Traffic logs.
func generateCSMTrafficLogsStructuredQuery(cluster k8scommon.GoogleCloudClusterIdentity, responseFlagsSetFilter *gcpqueryutil.SetFilterParseResult, namespaceSetFilter *gcpqueryutil.SetFilterParseResult) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(cluster.ProjectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(cluster.Location)),
		logestimator.ResourceLabel("cluster_name", logestimator.Exact(cluster.NameFor(k8scommon.ClusterNameUsageCSM))),
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

// csmTrafficLogQuerySource builds the Cloud Logging query for the CSM traffic logs of the cluster.
type csmTrafficLogQuerySource struct {
	clusterIdentity     coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	namespaceFilter     coretask.Input[*gcpqueryutil.SetFilterParseResult]
	responseFlagsFilter coretask.Input[*gcpqueryutil.SetFilterParseResult]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *csmTrafficLogQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := s.clusterIdentity.Get(ctx)
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *csmTrafficLogQuerySource) Queries(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*logestimator.StructuredLogQuery, error) {
	cluster := s.clusterIdentity.Get(ctx)
	namespaceFilter := s.namespaceFilter.Get(ctx)
	responseFlagsFilter := s.responseFlagsFilter.Get(ctx)
	return []*logestimator.StructuredLogQuery{generateCSMTrafficLogsStructuredQuery(cluster, responseFlagsFilter, namespaceFilter)}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *csmTrafficLogQuerySource) TimePartitionCount() int {
	return 10
}

var _ gcpcommon.StructuredLogQuerySource = (*csmTrafficLogQuerySource)(nil)

// listLogEntriesTask queries the CSM traffic logs of the cluster from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(
	csm.ListLogEntriesTaskID,
	"CSM Traffic logs",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &csmTrafficLogQuerySource{
			clusterIdentity:     coretask.Use(b, csm.ClusterIdentityTaskID.Ref()),
			namespaceFilter:     coretask.Use(b, k8scommon.InputNamespaceFilterTaskID.Ref()),
			responseFlagsFilter: coretask.Use(b, csm.InputCSMResponseFlagsTaskID.Ref()),
		}
	},
)
