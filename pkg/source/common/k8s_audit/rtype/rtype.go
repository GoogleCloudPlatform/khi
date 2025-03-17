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

package rtype

import "github.com/GoogleCloudPlatform/khi/pkg/log/structure"

// rtype.Type indicates the schema of log bodies under request or response.
type Type = int

const (
	RTypeUnknown       Type = 0
	RTypePatch         Type = 1
	RTypeDeleteOptions Type = 2
	RTypeSatus         Type = 3
	RTypeUnusedEnd
)

var AtTypesOnGCPAuditLog map[string]Type = map[string]Type{
	"k8s.io/Patch":                         RTypePatch,
	"meta.k8s.io/__internal.DeleteOptions": RTypeDeleteOptions,
	"core.k8s.io/v1.Status":                RTypeSatus,
}

// RtypeFromOSSK8sObject returns the type of resource from given requestObject or responseObject in OSS k8s audit log.
func RtypeFromOSSK8sObject(bodyReader *structure.Reader) Type {
	apiVersion := bodyReader.ReadStringOrDefault("apiVersion", "unknown")
	kind := bodyReader.ReadStringOrDefault("kind", "unknown")
	if apiVersion == "meta.k8s.io/v1" && kind == "DeleteOptions" {
		return RTypeDeleteOptions
	}
	if apiVersion == "v1" && kind == "Status" {
		return RTypeSatus
	}
	return RTypeUnknown
}
