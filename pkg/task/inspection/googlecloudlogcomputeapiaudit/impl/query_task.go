// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package googlecloudlogcomputeapiaudit_impl defines the implementation of the googlecloudlogcomputeapiaudit task.
package googlecloudlogcomputeapiaudit_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcomputeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomputeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GenerateComputeAPIQuery generates a query for compute API logs.
func GenerateComputeAPIQuery(taskMode inspectioncore_contract.InspectionTaskModeType, nodeNames []string) []string {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return []string{
			generateComputeAPIQueryWithInstanceNameFilter("-- instance name filters to be determined after audit log query"),
		}
	} else {
		result := []string{}
		instanceNameGroups := gcpqueryutil.SplitToChildGroups(nodeNames, 30)
		for _, group := range instanceNameGroups {
			nodeNamesWithInstance := []string{}
			for _, nodeName := range group {
				nodeNamesWithInstance = append(nodeNamesWithInstance, fmt.Sprintf("instances/%s", nodeName))
			}
			instanceNameFilter := fmt.Sprintf("protoPayload.resourceName:(%s)", strings.Join(nodeNamesWithInstance, " OR "))
			result = append(result, generateComputeAPIQueryWithInstanceNameFilter(instanceNameFilter))
		}
		return result
	}
}

func generateComputeAPIQueryWithInstanceNameFilter(instanceNameFilter string) string {
	return fmt.Sprintf(`resource.type="gce_instance"
-protoPayload.methodName:("list" OR "get" OR "watch")
%s
`, instanceNameFilter)
}

type computeAPIListLogEntriesTaskSetting struct {
}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", projectID)}, nil

}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		commonlogk8sauditv2_contract.NodeNameInventoryTaskID.Ref(),
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeComputeApi,
		QueryName:      "Compute API Audit log",
		ExampleQuery: GenerateComputeAPIQuery(inspectioncore_contract.TaskModeRun, []string{
			"gke-test-cluster-node-1",
			"gke-test-cluster-node-2",
		})[0],
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	nodeNames := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.NodeNameInventoryTaskID.Ref())
	return GenerateComputeAPIQuery(taskMode, nodeNames), nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogcomputeapiaudit_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*computeAPIListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&computeAPIListLogEntriesTaskSetting{})
