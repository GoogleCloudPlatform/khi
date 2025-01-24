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

package serialport

import (
	"context"
	"fmt"
	"strings"

	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/k8saudittask"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const SerialPortLogQueryTaskID = query.GKEQueryPrefix + "serialport"

const MaxNodesPerQuery = 30

func GenerateSerialPortQuery(taskMode int, foundNodeNames []string, nodeNameSubstrings []string) []string {
	if taskMode == inspection_task.TaskModeDryRun {
		return []string{
			generateSerialPortQueryWithInstanceNameFilter("-- instance name filters to be determined after audit log query", generateNodeNameSubstringLogFilter(nodeNameSubstrings)),
		}
	} else {
		result := []string{}
		instanceNameGroups := queryutil.SplitToChildGroups(foundNodeNames, MaxNodesPerQuery)
		for _, group := range instanceNameGroups {
			instanceNameFilter := fmt.Sprintf(`labels."compute.googleapis.com/resource_name"=(%s)`, strings.Join(queryutil.WrapDoubleQuoteForStringArray(group), " OR "))
			result = append(result, generateSerialPortQueryWithInstanceNameFilter(instanceNameFilter, generateNodeNameSubstringLogFilter(nodeNameSubstrings)))
		}
		return result
	}
}

func generateNodeNameSubstringLogFilter(nodeNameSubstrings []string) string {
	if len(nodeNameSubstrings) == 0 {
		return "-- No node name substring filters are specified."
	} else {
		return fmt.Sprintf(`labels."compute.googleapis.com/resource_name":(%s)`, strings.Join(queryutil.WrapDoubleQuoteForStringArray(nodeNameSubstrings), " OR "))
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

var GKESerialPortLogQueryTask = query.NewQueryGeneratorTask(SerialPortLogQueryTaskID, "Serial port log", enum.LogTypeSerialPort, []string{
	k8saudittask.K8sAuditParseTaskID,
	gcp_task.InputNodeNameFilterTaskID,
}, func(ctx context.Context, taskMode int, vs *task.VariableSet) ([]string, error) {
	builder, err := inspection_task.GetHistoryBuilderFromTaskVariable(vs)
	if err != nil {
		return nil, err
	}
	nodeNameSubstrings, err := gcp_task.GetNodeNameFilterFromTaskVaraible(vs)
	if err != nil {
		return nil, err
	}
	return GenerateSerialPortQuery(taskMode, builder.ClusterResource.GetNodes(), nodeNameSubstrings), nil
})
