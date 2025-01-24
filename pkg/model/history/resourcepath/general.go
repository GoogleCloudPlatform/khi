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
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

var PlaceholderForEmptyField = "unknown"

func APIVersionLayerGeneralItem(apiVersion string) ResourcePath {
	if apiVersion == "" {
		apiVersion = PlaceholderForEmptyField
	}
	if apiVersion == "v1" {
		apiVersion = "core/v1"
	}
	return ResourcePath{
		Path:               apiVersion,
		ParentRelationship: enum.RelationshipChild,
	}
}

func KindLayerGeneralItem(apiVersion, kind string) ResourcePath {
	if kind == "" {
		kind = PlaceholderForEmptyField
	}
	return ResourcePath{
		Path:               fmt.Sprintf("%s#%s", APIVersionLayerGeneralItem(apiVersion).Path, kind),
		ParentRelationship: enum.RelationshipChild,
	}
}

func NamespaceLayerGeneralItem(apiVersion, kind, namespace string) ResourcePath {
	if namespace == "" {
		namespace = PlaceholderForEmptyField
	}
	return ResourcePath{
		Path:               fmt.Sprintf("%s#%s", KindLayerGeneralItem(apiVersion, kind).Path, namespace),
		ParentRelationship: enum.RelationshipChild,
	}
}

func NameLayerGeneralItem(apiVersion, kind, namespace, name string) ResourcePath {
	if name == "" {
		name = PlaceholderForEmptyField
	}
	return ResourcePath{
		Path:               fmt.Sprintf("%s#%s", NamespaceLayerGeneralItem(apiVersion, kind, namespace).Path, name),
		ParentRelationship: enum.RelationshipChild,
	}
}

func SubresourceLayerGeneralItem(apiVersion, kind, namespace, name, subresource string) ResourcePath {
	if subresource == "" {
		subresource = PlaceholderForEmptyField
	}
	return ResourcePath{
		Path:               fmt.Sprintf("%s#%s", NameLayerGeneralItem(apiVersion, kind, namespace, name).Path, subresource),
		ParentRelationship: enum.RelationshipChild,
	}
}

func FromK8sOperation(op model.KubernetesObjectOperation) ResourcePath {
	var path string
	if op.SubResourceName != "" {
		path = strings.ToLower(strings.Join([]string{
			op.APIVersion,
			op.GetSingularKindName(),
			op.Namespace,
			op.Name,
			op.SubResourceName,
		}, "#"))
	} else {
		path = strings.ToLower(strings.Join([]string{
			op.APIVersion,
			op.GetSingularKindName(),
			op.Namespace,
			op.Name,
		}, "#"))
	}
	return ResourcePath{
		Path:               path,
		ParentRelationship: enum.RelationshipChild,
	}
}
