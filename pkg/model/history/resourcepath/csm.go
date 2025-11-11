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

func CSMServerAccess(podNamespace string, podName string, containerName string) ResourcePath {
	base := Pod(podNamespace, podName)
	if containerName == "" {
		return ResourcePath{
			Path:               fmt.Sprintf("%s#server", base.Path),
			ParentRelationship: enum.RelationshipCSMAccessLog,
		}
	}
	return ResourcePath{
		Path:               fmt.Sprintf("%s#server:%s", base.Path, containerName),
		ParentRelationship: enum.RelationshipCSMAccessLog,
	}
}

func CSMClientAccess(podNamespace string, podName string) ResourcePath {
	base := Pod(podNamespace, podName)
	return ResourcePath{
		Path:               fmt.Sprintf("%s#client", base.Path),
		ParentRelationship: enum.RelationshipCSMAccessLog,
	}
}

func CSMServiceClientAccess(serviceNamespace string, serviceName string) ResourcePath {
	base := Service(serviceNamespace, serviceName)
	return ResourcePath{
		Path:               fmt.Sprintf("%s#client", base.Path),
		ParentRelationship: enum.RelationshipCSMAccessLog,
	}
}

func CSMServiceServerAccess(serviceNamespace string, serviceName string) ResourcePath {
	base := Service(serviceNamespace, serviceName)
	return ResourcePath{
		Path:               fmt.Sprintf("%s#server", base.Path),
		ParentRelationship: enum.RelationshipCSMAccessLog,
	}
}
