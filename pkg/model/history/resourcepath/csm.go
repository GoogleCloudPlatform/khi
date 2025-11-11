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

package resourcepath

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

func csmAccessPath(base ResourcePath, direction string, containerName string) ResourcePath {
	path := fmt.Sprintf("%s#%s", base.Path, direction)
	if containerName != "" {
		path += ":" + containerName
	}
	return ResourcePath{
		Path:               path,
		ParentRelationship: enum.RelationshipCSMAccessLog,
	}
}

func CSMServerAccess(podNamespace string, podName string, containerName string) ResourcePath {
	return csmAccessPath(Pod(podNamespace, podName), "server", containerName)
}

func CSMServiceServerAccess(serviceNamespace string, serviceName string) ResourcePath {
	return csmAccessPath(Service(serviceNamespace, serviceName), "server", "")
}

func CSMClientAccess(podNamespace string, podName string) ResourcePath {
	return csmAccessPath(Pod(podNamespace, podName), "client", "")
}

func CSMServiceClientAccess(serviceNamespace string, serviceName string) ResourcePath {
	return csmAccessPath(Service(serviceNamespace, serviceName), "client", "")
}
