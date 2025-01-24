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

package k8s

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

func ParseKubernetesOperation(resourceName string, methodName string) *model.KubernetesObjectOperation {
	resourceNameFragments := strings.Split(resourceName, "/")
	methodNameFragments := strings.Split(methodName, ".")
	pluralKind := ""
	namespace := ""
	name := "unknown"
	subResourceName := ""
	if methodNameFragments[4] == "namespaces" {
		// Branch for namespace resource
		namespace = "Cluster-Scope"
		name = resourceNameFragments[3]
		pluralKind = "namespaces"
		if len(resourceNameFragments) > 4 {
			subResourceName = resourceNameFragments[4]
		}
	} else if resourceNameFragments[2] == "namespaces" && len(resourceNameFragments) >= 5 {
		namespace = resourceNameFragments[3]
		pluralKind = resourceNameFragments[4]
		if len(resourceNameFragments) > 5 {
			name = resourceNameFragments[5]
		}
		if len(resourceNameFragments) > 6 {
			subResourceName = resourceNameFragments[6]
		}
	} else if len(resourceNameFragments) >= 3 {
		namespace = "Cluster-Scope"
		if len(resourceNameFragments) > 3 {
			name = resourceNameFragments[3]
		}
		pluralKind = resourceNameFragments[2]
		if len(resourceNameFragments) > 4 {
			subResourceName = resourceNameFragments[4]
		}
	}
	verb := methodNameFragments[len(methodNameFragments)-1]
	if verb == "deletecollection" {
		name = ""
	}
	return &model.KubernetesObjectOperation{
		APIVersion:      resourceNameFragments[0] + "/" + resourceNameFragments[1],
		PluralKind:      pluralKind,
		Namespace:       namespace,
		Name:            name,
		SubResourceName: subResourceName,
		Verb:            parseVerb(verb),
	}
}

func parseVerb(verbInStr string) enum.RevisionVerb {
	switch verbInStr {
	case "create":
		return enum.RevisionVerbCreate
	case "update":
		return enum.RevisionVerbUpdate
	case "patch":
		return enum.RevisionVerbPatch
	case "delete":
		return enum.RevisionVerbDelete
	case "deletecollection":
		return enum.RevisionVerbDeleteCollection
	default:
		return enum.RevisionVerbUnknown
	}
}
