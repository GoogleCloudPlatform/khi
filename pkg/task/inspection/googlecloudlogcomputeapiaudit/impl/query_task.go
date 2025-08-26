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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudlogcomputeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomputeapiaudit/contract"
	googlecloudlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8saudit/contract"
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

// ComputeAPIQueryTask defines a task that queries compute API logs from Cloud Logging.
var ComputeAPIQueryTask = gcpqueryutil.NewCloudLoggingListLogTask(googlecloudlogcomputeapiaudit_contract.ComputeAPIQueryTaskID, "Compute API Logs", enum.LogTypeComputeApi, []taskid.UntypedTaskReference{
	googlecloudlogk8saudit_contract.K8sAuditParseTaskID.Ref(),
}, &gcpqueryutil.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)

	return GenerateComputeAPIQuery(i, builder.ClusterResource.GetNodes()), nil
}, GenerateComputeAPIQuery(inspectioncore_contract.TaskModeRun, []string{
	"gke-test-cluster-node-1",
	"gke-test-cluster-node-2",
})[0])
