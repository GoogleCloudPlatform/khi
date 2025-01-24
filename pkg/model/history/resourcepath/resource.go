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

package resourcepath

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

const nonSpecifiedPlaceholder = "unknown"

func Container(namespace string, name string, containerName string) ResourcePath {
	if namespace == "" {
		namespace = nonSpecifiedPlaceholder
	}
	if name == "" {
		name = nonSpecifiedPlaceholder
	}
	if containerName == "" {
		containerName = nonSpecifiedPlaceholder
	}
	containerResourcePath := SubresourceLayerGeneralItem("core/v1", "pod", namespace, name, containerName)
	containerResourcePath.ParentRelationship = enum.RelationshipContainer
	return containerResourcePath
}

func Pod(namespace string, name string) ResourcePath {
	if namespace == "" {
		namespace = nonSpecifiedPlaceholder
	}
	if name == "" {
		name = nonSpecifiedPlaceholder
	}
	return NameLayerGeneralItem("core/v1", "pod", namespace, name)
}

func Service(namespace string, name string) ResourcePath {
	if namespace == "" {
		namespace = nonSpecifiedPlaceholder
	}
	if name == "" {
		name = nonSpecifiedPlaceholder
	}
	return NameLayerGeneralItem("core/v1", "service", namespace, name)
}

func Node(name string) ResourcePath {
	if name == "" {
		name = nonSpecifiedPlaceholder
	}
	return NameLayerGeneralItem("core/v1", "node", "cluster-scope", name)
}
