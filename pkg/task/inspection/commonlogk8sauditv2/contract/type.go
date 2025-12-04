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

package commonlogk8sauditv2_contract

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

type ResourceIdentityType int

const (
	// Namespace indicates the resource is entire specific resource in a namespace
	Namespace ResourceIdentityType = iota
	// Resource indicates the resource is specific resource in a namespace
	Resource
	// Subresource indicates the resource is specific subresource of a resource
	Subresource
)

// ResourceIdentity represents the identity of a kubernetes resource.
type ResourceIdentity struct {
	APIVersion      string
	Kind            string
	Name            string
	Namespace       string
	SubresourceName string
}

// Equals implements resourcelease.LeaseHolder.
func (r *ResourceIdentity) Equals(holder resourcelease.LeaseHolder) bool {
	if castedHolder, ok := holder.(*ResourceIdentity); ok {
		return r.APIVersion == castedHolder.APIVersion && r.Kind == castedHolder.Kind && r.Name == castedHolder.Name && r.Namespace == castedHolder.Namespace && r.SubresourceName == castedHolder.SubresourceName
	}
	return false
}

func ResourceIdentityFromKubernetesOperation(op *model.KubernetesObjectOperation) *ResourceIdentity {
	return &ResourceIdentity{
		APIVersion:      op.APIVersion,
		Kind:            op.GetSingularKindName(),
		Name:            op.Name,
		Namespace:       op.Namespace,
		SubresourceName: op.SubResourceName,
	}
}

// Type returns the type of the resource identity.
func (r *ResourceIdentity) Type() ResourceIdentityType {
	switch {
	case r.Name == "":
		return Namespace
	case r.SubresourceName != "":
		return Subresource
	default:
		return Resource
	}
}

// ResourcePathString returns the resource path string.
func (r *ResourceIdentity) ResourcePathString() string {
	switch r.Type() {
	case Namespace:
		return resourcepath.NameLayerGeneralItem(r.APIVersion, r.Kind, r.Namespace, "@namespace").Path
	case Resource:
		return resourcepath.NameLayerGeneralItem(r.APIVersion, r.Kind, r.Namespace, r.Name).Path
	case Subresource:
		return resourcepath.SubresourceLayerGeneralItem(r.APIVersion, r.Kind, r.Namespace, r.Name, r.SubresourceName).Path
	default:
		panic(fmt.Sprintf("unknown resource identity type: %d", r.Type()))
	}
}

// SubresourceIdentity returns the resource identity of the subresource.
func (r *ResourceIdentity) SubresourceIdentity(subresourceName string) *ResourceIdentity {
	return &ResourceIdentity{
		APIVersion:      r.APIVersion,
		Kind:            r.Kind,
		Name:            r.Name,
		Namespace:       r.Namespace,
		SubresourceName: subresourceName,
	}
}

// ParentIdentity returns the resource identity of the parent resource.
func (r *ResourceIdentity) ParentIdentity() *ResourceIdentity {
	switch r.Type() {
	case Namespace:
		return nil // We don't define kind layer or its parent because these can't have events or revisions.
	case Resource:
		return &ResourceIdentity{
			APIVersion: r.APIVersion,
			Kind:       r.Kind,
			Namespace:  r.Namespace,
		}
	case Subresource:
		return &ResourceIdentity{
			APIVersion: r.APIVersion,
			Kind:       r.Kind,
			Namespace:  r.Namespace,
			Name:       r.Name,
		}
	default:
		panic(fmt.Sprintf("unknown resource identity type: %d", r.Type()))
	}
}

var _ resourcelease.LeaseHolder = (*ResourceIdentity)(nil)

type ContainerIdentity struct {
	ContainerID   string
	ContainerName string
	PodSandboxID  string
}

func (c *ContainerIdentity) Merge(other *ContainerIdentity) *ContainerIdentity {
	result := *c
	if other.ContainerID != "" {
		result.ContainerID = other.ContainerID
	}
	if other.ContainerName != "" {
		result.ContainerName = other.ContainerName
	}
	if other.PodSandboxID != "" {
		result.PodSandboxID = other.PodSandboxID
	}
	return &result
}

func (c *ContainerIdentity) ResourcePath(podNamespace string, podName string) resourcepath.ResourcePath {
	return resourcepath.Container(podNamespace, podName, c.ContainerName)
}

// ResourceLogGroup is the group of the logs associated with k8s resource.
type ResourceLogGroup struct {
	// Resource is the resource identity.
	Resource *ResourceIdentity
	// Logs is the list of the logs associated with the resource.
	Logs []*log.Log
}

// ResourceLogGroupMap is the map of the resource log groups.
type ResourceLogGroupMap = map[string]*ResourceLogGroup

// ResourceManifestLog is the log with the resource manifest information.
type ResourceManifestLog struct {
	// Log is the log.
	Log *log.Log
	// ResourceBodyYAML is the YAML representation of the resource body.
	ResourceBodyYAML string
	// ResourceBodyReader is the reader for the resource body.
	ResourceBodyReader *structured.NodeReader
	// ResourceCreated is true if the resource is created.
	ResourceCreated bool
	// ResourceDeleted is true if the resource is deleted.
	ResourceDeleted bool
}

// ResourceManifestLogGroup is the group of the resource change logs.
type ResourceManifestLogGroup struct {
	// Resource is the resource identity.
	Resource *ResourceIdentity
	// Logs is the list of the resource change logs.
	Logs []*ResourceManifestLog
}

// ResourceManifestLogGroupMap is the map of the resource change log groups.
type ResourceManifestLogGroupMap = map[string]*ResourceManifestLogGroup
