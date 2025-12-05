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

package googlecloudlognetworkapiaudit_impl

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
	googlecloudlognetworkapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlognetworkapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// generateGCPNetworkAPIQuery generates a query for network API logs.
func generateGCPNetworkAPIQuery(taskMode inspectioncore_contract.InspectionTaskModeType, negNames []string) []string {
	nodeNamesWithNetworkEndpointGroups := []string{}
	for _, negName := range negNames {
		nodeNamesWithNetworkEndpointGroups = append(nodeNamesWithNetworkEndpointGroups, fmt.Sprintf("networkEndpointGroups/%s", negName))
	}
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return []string{queryFromNegNameFilter("-- neg name filters to be determined after audit log query")}
	} else {
		result := []string{}
		groups := gcpqueryutil.SplitToChildGroups(nodeNamesWithNetworkEndpointGroups, 10)
		for _, group := range groups {
			negNameFilter := fmt.Sprintf("protoPayload.resourceName:(%s)", strings.Join(group, " OR "))
			result = append(result, queryFromNegNameFilter(negNameFilter))
		}
		return result
	}
}

func queryFromNegNameFilter(negNameFilter string) string {
	return fmt.Sprintf(`resource.type="gce_network"
-protoPayload.methodName:("list" OR "get" OR "watch")
%s
`, negNameFilter)
}

type networkAPIListLogEntiesTaskSetting struct{}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (n *networkAPIListLogEntiesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", projectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (n *networkAPIListLogEntiesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
		googlecloudk8scommon_contract.NEGNamesInventoryTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (n *networkAPIListLogEntiesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeNetworkAPI,
		QueryName:      "GCP network log",
		ExampleQuery:   generateGCPNetworkAPIQuery(inspectioncore_contract.TaskModeRun, []string{"neg-id-1", "neg-id-2"})[0],
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (n *networkAPIListLogEntiesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	negs := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.NEGNamesInventoryTaskID.Ref())
	negNames := []string{}
	for negName := range negs {
		negNames = append(negNames, negName)
	}
	return generateGCPNetworkAPIQuery(taskMode, negNames), nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (n *networkAPIListLogEntiesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlognetworkapiaudit_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (n *networkAPIListLogEntiesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*networkAPIListLogEntiesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&networkAPIListLogEntiesTaskSetting{})
