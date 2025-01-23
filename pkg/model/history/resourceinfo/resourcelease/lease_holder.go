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

package resourcelease

import "strings"

type K8sResourceLeaseHolder struct {
	Kind      string
	Namespace string
	Name      string
}

func NewK8sResourceLeaseHolder(kind string, namespace string, name string) *K8sResourceLeaseHolder {
	return &K8sResourceLeaseHolder{
		Kind:      strings.ToLower(kind),
		Namespace: strings.ToLower(namespace),
		Name:      strings.ToLower(name),
	}
}

// Equals implements LeaseHolder.
func (k *K8sResourceLeaseHolder) Equals(holder LeaseHolder) bool {
	castedHolder, ok := holder.(*K8sResourceLeaseHolder)
	if !ok {
		return false
	}
	return k.Kind == castedHolder.Kind && k.Namespace == castedHolder.Namespace && k.Name == castedHolder.Name
}

var _ LeaseHolder = (*K8sResourceLeaseHolder)(nil)

func NewContainerLeaseHolder(podSandboxId string, containerName string) *ContainerLeaseHolder {
	return &ContainerLeaseHolder{
		PodSandboxId:  podSandboxId,
		ContainerName: containerName,
	}
}

type ContainerLeaseHolder struct {
	// Container Lease Holder object to hold container IDs
	PodSandboxId  string
	ContainerName string
}

// Equals implements LeaseHolder.
func (c *ContainerLeaseHolder) Equals(holder LeaseHolder) bool {
	castedHolder, ok := holder.(*ContainerLeaseHolder)
	if !ok {
		return false
	}
	return c.ContainerName == castedHolder.ContainerName && c.PodSandboxId == castedHolder.PodSandboxId
}

var _ LeaseHolder = (*ContainerLeaseHolder)(nil)
