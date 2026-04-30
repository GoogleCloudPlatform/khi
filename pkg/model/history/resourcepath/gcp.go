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

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// NetworkEndpointGroup returns the ResourcePath of timeline for NEG.
func NetworkEndpointGroup(negNamespace string, negName string) ResourcePath {
	if negNamespace == "" {
		negNamespace = nonSpecifiedPlaceholder
	}
	if negName == "" {
		negName = nonSpecifiedPlaceholder
	}
	return NameLayerGeneralItem("networking.gke.io/v1beta1", "servicenetworkendpointgroup", negNamespace, negName)
}

// GCPResource returns the ResourcePath for a generic GCP resource.
// It parses resource names like "projects/(project)/locations/(location)/(type)/(name)"
// or "projects/(project)/regions/(region)/(type)/(name)"
// and generates a ResourcePath under "@GCP" pseudo API version.
func GCPResource(resourceName string) ResourcePath {
	if resourceName == "" || resourceName == "unknown" {
		return ResourcePath{
			Path:               fmt.Sprintf("@GCP#CSM#unknown#%s", nonSpecifiedPlaceholder),
			ParentRelationship: enum.RelationshipChild,
		}
	}

	parts := strings.Split(resourceName, "/")
	// projects/(project)/locations/(location)/(type)/(name) -> 6 parts
	// projects/(project)/global/(type)/(name) -> 5 parts
	// We want to extract the type and name, and potentially the location.

	var resourceType, name string
	if len(parts) >= 2 {
		resourceType = parts[len(parts)-2]
		name = parts[len(parts)-1]
	} else {
		resourceType = "unknown"
		name = resourceName
	}

	return ResourcePath{
		Path:               fmt.Sprintf("@GCP#CSM#%s#%s", resourceType, name),
		ParentRelationship: enum.RelationshipChild,
	}
}
