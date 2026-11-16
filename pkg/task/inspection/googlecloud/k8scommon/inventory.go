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

package k8scommon

import (
	"regexp"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
)

var (
	// NEGNamesInventoryTaskID is the task ID for the inventory task that aggregates discovered NEG names.
	NEGNamesInventoryTaskID = taskid.NewDefaultImplementationID[NEGNameToResourceIdentityMap](GoogleCloudCommonK8STaskIDPrefix + "neg-names-inventory")
	// TagNEGNamesDiscovery is the task tag for tasks that discover NEG name to resource identity mappings.
	TagNEGNamesDiscovery = coretask.NewTag[NEGNameToResourceIdentityMap]("googlecloud-neg-names-discovery")
	// NEGToBackendServiceInventoryTaskID is the task ID for the inventory task that provides NEG to BackendService mappings.
	NEGToBackendServiceInventoryTaskID = taskid.NewDefaultImplementationID[NEGToBackendServiceMap](GoogleCloudCommonK8STaskIDPrefix + "neg-to-backend-service-inventory")
	// TagNEGToBackendServiceDiscovery is the task tag for tasks that discover NEG to BackendService mappings.
	TagNEGToBackendServiceDiscovery = coretask.NewTag[NEGToBackendServiceMap]("googlecloud-neg-to-backend-service-discovery")
)

var negToBackendServiceRegex = regexp.MustCompile(`NEG "Key\{\\"([^"]+)\\"[^}]*\}" attached to BackendService "Key\{\\"([^"]+)\\"\}"`)

// NEGNameToResourceIdentityMap maps NEG names to their Kubernetes ServiceNetworkEndpointGroup resource identities.
type NEGNameToResourceIdentityMap = map[string]k8saudit.ResourceIdentity

// NEGToBackendServiceMap is a map from NEG name to BackendService name.
type NEGToBackendServiceMap = map[string]string

// ExtractNEGToBackendService extracts the NEG name and BackendService name from a given message.
func ExtractNEGToBackendService(message string) (negName string, backendServiceName string) {
	matches := negToBackendServiceRegex.FindStringSubmatch(message)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}
	return "", ""
}
