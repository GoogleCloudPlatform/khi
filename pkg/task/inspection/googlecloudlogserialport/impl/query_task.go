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

package googlecloudlogserialport_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8saudit/contract"
	googlecloudlogserialport_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogserialport/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

const MaxNodesPerQuery = 30

func GenerateSerialPortQuery(taskMode inspectioncore_contract.InspectionTaskModeType, foundNodeNames []string, nodeNameSubstrings []string) []string {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return []string{
			generateSerialPortQueryWithInstanceNameFilter("-- instance name filters to be determined after audit log query", generateNodeNameSubstringLogFilter(nodeNameSubstrings)),
		}
	} else {
		result := []string{}
		instanceNameGroups := gcpqueryutil.SplitToChildGroups(foundNodeNames, MaxNodesPerQuery)
		for _, group := range instanceNameGroups {
			instanceNameFilter := fmt.Sprintf(`labels."compute.googleapis.com/resource_name"=(%s)`, strings.Join(gcpqueryutil.WrapDoubleQuoteForStringArray(group), " OR "))
			result = append(result, generateSerialPortQueryWithInstanceNameFilter(instanceNameFilter, generateNodeNameSubstringLogFilter(nodeNameSubstrings)))
		}
		return result
	}
}

func generateNodeNameSubstringLogFilter(nodeNameSubstrings []string) string {
	if len(nodeNameSubstrings) == 0 {
		return "-- No node name substring filters are specified."
	} else {
		return fmt.Sprintf(`labels."compute.googleapis.com/resource_name":(%s)`, strings.Join(gcpqueryutil.WrapDoubleQuoteForStringArray(nodeNameSubstrings), " OR "))
	}
}

func generateSerialPortQueryWithInstanceNameFilter(instanceNameFilter string, nodeNameSubstringFilter string) string {
	return fmt.Sprintf(`LOG_ID("serialconsole.googleapis.com%%2Fserial_port_1_output") OR
LOG_ID("serialconsole.googleapis.com%%2Fserial_port_2_output") OR
LOG_ID("serialconsole.googleapis.com%%2Fserial_port_3_output") OR
LOG_ID("serialconsole.googleapis.com%%2Fserial_port_debug_output")

%s

%s`, instanceNameFilter, nodeNameSubstringFilter)
}

var GKESerialPortLogQueryTask = googlecloudcommon_contract.NewCloudLoggingListLogTask(googlecloudlogserialport_contract.SerialPortLogQueryTaskID, "Serial port log", enum.LogTypeSerialPort, []taskid.UntypedTaskReference{
	googlecloudlogk8saudit_contract.K8sAuditParseTaskID.Ref(),
	googlecloudk8scommon_contract.InputNodeNameFilterTaskID.Ref(),
}, &googlecloudcommon_contract.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)
	nodeNameSubstrings := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputNodeNameFilterTaskID.Ref())

	return GenerateSerialPortQuery(taskMode, builder.ClusterResource.GetNodes(), nodeNameSubstrings), nil
}, GenerateSerialPortQuery(inspectioncore_contract.TaskModeRun, []string{
	"gke-test-cluster-node-1",
	"gke-test-cluster-node-2",
}, []string{})[0])
