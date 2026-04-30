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

package googlecloudk8scommon_contract

import (
	"regexp"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

type NEGNameToResourceIdentityMap = map[string]commonlogk8sauditv2_contract.ResourceIdentity

var NEGNamesInventoryTaskID = taskid.NewDefaultImplementationID[NEGNameToResourceIdentityMap](GoogleCloudCommonK8STaskIDPrefix + "neg-names-inventory")

var NEGNamesInventoryTaskBuilder = inspectiontaskbase.NewInventoryTaskBuilder(NEGNamesInventoryTaskID)

// NEGToBackendServiceMap is a map from NEG name to BackendService name.
type NEGToBackendServiceMap = map[string]string

// NEGToBackendServiceInventoryTaskID is the task ID for the inventory task that provides NEG to BackendService mappings.
var NEGToBackendServiceInventoryTaskID = taskid.NewDefaultImplementationID[NEGToBackendServiceMap](GoogleCloudCommonK8STaskIDPrefix + "neg-to-backend-service-inventory")

// NEGToBackendServiceInventoryBuilder is the inventory task builder for NEG to BackendService mappings.
var NEGToBackendServiceInventoryBuilder = inspectiontaskbase.NewInventoryTaskBuilder(NEGToBackendServiceInventoryTaskID)

var negToBackendServiceRegex = regexp.MustCompile(`NEG "Key\{\\"([^"]+)\\"[^}]*\}" attached to BackendService "Key\{\\"([^"]+)\\"\}"`)

// ExtractNEGToBackendService extracts the NEG name and BackendService name from a given message.
func ExtractNEGToBackendService(message string) (negName string, backendServiceName string) {
	matches := negToBackendServiceRegex.FindStringSubmatch(message)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}
	return "", ""
}
