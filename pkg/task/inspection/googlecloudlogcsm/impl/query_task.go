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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func csmAccessLogsFilter(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, responseFlagsSetFilter *gcpqueryutil.SetFilterParseResult, namespaceSetFilter *gcpqueryutil.SetFilterParseResult) string {
	responseFlagsFilterStr := responseFlagsFilter(responseFlagsSetFilter)
	namespaceFilterStr := namespaceFilter(namespaceSetFilter)
	return fmt.Sprintf(`LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver") 
%s
%s
resource.labels.project_id="%s"
resource.labels.location="%s"
resource.labels.cluster_name="%s"`, responseFlagsFilterStr, namespaceFilterStr, cluster.ProjectID, cluster.Location, cluster.NameWithClusterTypePrefix())
}

func responseFlagsFilter(responseFlagsFilter *gcpqueryutil.SetFilterParseResult) string {
	if responseFlagsFilter.ValidationError != "" {
		return fmt.Sprintf(`-- Failed to generate response flags filter due to the validation error "%s"`, responseFlagsFilter.ValidationError)
	}
	if responseFlagsFilter.SubtractMode {
		if len(responseFlagsFilter.Subtractives) == 0 {
			return "-- No response flags filter"
		}
		return fmt.Sprintf(`-labels.response_flag:(%s)`, strings.Join(responseFlagsFilter.SubtractivesWithQuotes(), " OR "))
	}

	if len(responseFlagsFilter.Additives) == 0 {
		return `-- Invalid: none of the resources will be selected. Ignoring response flag filter.`
	}
	return fmt.Sprintf(`labels.response_flag:(%s)`, strings.Join(responseFlagsFilter.AdditivesWithQuotes(), " OR "))
}

func namespaceFilter(filter *gcpqueryutil.SetFilterParseResult) string {
	if filter.ValidationError != "" {
		return fmt.Sprintf(`-- Failed to generate namespace filter due to the validation error "%s"`, filter.ValidationError)
	}
	if filter.SubtractMode {
		return "-- Unsupported operation"
	} else {
		selectedNamespaces := []string{}
		for _, additive := range filter.Additives {
			if strings.HasPrefix(additive, "#") {
				if additive == "#namespaced" {
					return "-- No namespace filter"
				}
				continue
			}
			selectedNamespaces = append(selectedNamespaces, fmt.Sprintf(`"%s"`, additive))
		}
		if len(selectedNamespaces) == 0 {
			return `resource.labels.namespace_name="" -- Invalid: No namespaces are remained to filter for CSM access log.`
		}
		return fmt.Sprintf(`resource.labels.namespace_name:(%s)`, strings.Join(selectedNamespaces, " OR "))
	}
}

type CSMAccessLogListLogEntryTaskSetting struct{}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *CSMAccessLogListLogEntryTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *CSMAccessLogListLogEntryTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref(),
		googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref(),
		googlecloudlogcsm_contract.InputCSMResponseFlagsTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *CSMAccessLogListLogEntryTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeCSMAccessLog,
		QueryName:      "CSM access logs",
		ExampleQuery: csmAccessLogsFilter(googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
			ProjectID:   "test-project",
			Location:    "test-location",
			ClusterName: "test-cluster",
		}, &gcpqueryutil.SetFilterParseResult{Subtractives: []string{"-"}, SubtractMode: true}, &gcpqueryutil.SetFilterParseResult{SubtractMode: true}),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *CSMAccessLogListLogEntryTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	namespaceFilter := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref())
	responseFlagsFilter := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.InputCSMResponseFlagsTaskID.Ref())
	return []string{csmAccessLogsFilter(cluster, responseFlagsFilter, namespaceFilter)}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *CSMAccessLogListLogEntryTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogcsm_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *CSMAccessLogListLogEntryTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*CSMAccessLogListLogEntryTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&CSMAccessLogListLogEntryTaskSetting{})
