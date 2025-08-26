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

package googlecloudk8scommon_impl

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

var nodeNameSubstringValidator = regexp.MustCompile("^[-a-z0-9]*$")

// getNodeNameSubstringsFromRawInput splits input by spaces and returns result in array.
// This removes surround spaces and removes empty string.
func getNodeNameSubstringsFromRawInput(value string) []string {
	result := []string{}
	nodeNameSubstrings := strings.Split(value, " ")
	for _, v := range nodeNameSubstrings {
		nodeNameSubstring := strings.TrimSpace(v)
		if nodeNameSubstring != "" {
			result = append(result, nodeNameSubstring)
		}
	}
	return result
}

// InputNodeNameFilterTask is a task to collect list of substrings of node names. This input value is used in querying k8s_node or serialport logs.
var InputNodeNameFilterTask = formtask.NewTextFormTaskBuilder(googlecloudk8scommon_contract.InputNodeNameFilterTaskID, googlecloudcommon_contract.PriorityForK8sResourceFilterGroup+3000, "Node names").
	WithDefaultValueConstant("", true).
	WithDescription("A space-separated list of node name substrings used to collect node-related logs. If left blank, KHI gathers logs from all nodes in the cluster.").
	WithValidator(func(ctx context.Context, value string) (string, error) {
		nodeNameSubstrings := getNodeNameSubstringsFromRawInput(value)
		for _, name := range nodeNameSubstrings {
			if !nodeNameSubstringValidator.Match([]byte(name)) {
				return fmt.Sprintf("substring `%s` is not valid as a substring of node name", name), nil
			}
		}
		return "", nil
	}).WithConverter(func(ctx context.Context, value string) ([]string, error) {
	return getNodeNameSubstringsFromRawInput(value), nil
}).Build()
