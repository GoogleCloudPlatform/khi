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

package noderesource

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
)

type ResourceBinding interface {
	// GetUniqueIdentifier returns ids/names that needs to be included in the log body when the resource associates with it.
	GetUniqueIdentifier() string
	// GetResourcePath returns the path on where this resource should have the event.
	GetResourcePath() resourcepath.ResourcePath
	// RewriteLogSummary receives summary from original or privious another resource binding and return rewritten summary.
	RewriteLogSummary(summary string) string
}

// PodResourceBinding is a ResourceBinding for a Pod resource on a node.
type PodResourceBinding struct {
	PodSandboxID string
	PodName      string
	PodNamespace string
}

// NewPodResourceBinding returns a new PodResourceBinding instance.
func NewPodResourceBinding(podSandboxID string, podNamespace string, podName string) *PodResourceBinding {
	return &PodResourceBinding{
		PodSandboxID: podSandboxID,
		PodName:      podName,
		PodNamespace: podNamespace,
	}
}

// GetResourcePath implements ResourceBinding.
func (p *PodResourceBinding) GetResourcePath() resourcepath.ResourcePath {
	return resourcepath.Pod(p.PodNamespace, p.PodName)
}

// GetUniqueIdentifier implements ResourceBinding.
func (p *PodResourceBinding) GetUniqueIdentifier() string {
	return p.PodSandboxID
}

// RewriteLogSummary implements ResourceBinding.
func (p *PodResourceBinding) RewriteLogSummary(summary string) string {
	return rewriteIdWithReadableName(p.PodSandboxID, fmt.Sprintf("%s/%s", p.PodNamespace, p.PodName), fmt.Sprintf("%s【%s/%s】", summary, p.PodNamespace, p.PodName))
}

// NewContainerResourceBinding returns an instance of ContainerRersourceBinding that is a child of this Pod.
func (p *PodResourceBinding) NewContainerResourceBinding(containerID string, containerName string) *ContainerResourceBinding {
	return &ContainerResourceBinding{
		ConainerID:    containerID,
		ContainerName: containerName,
		PodNamespace:  p.PodNamespace,
		PodName:       p.PodName,
	}
}

var _ ResourceBinding = (*PodResourceBinding)(nil)

// ContainerResourceBinding is a ResourceBinding for a container on a node.
type ContainerResourceBinding struct {
	ConainerID    string
	ContainerName string
	PodNamespace  string
	PodName       string
}

// GetResourcePath implements ResourceBinding.
func (c *ContainerResourceBinding) GetResourcePath() resourcepath.ResourcePath {
	return resourcepath.Container(c.PodNamespace, c.PodName, c.ContainerName)
}

// GetUniqueIdentifier implements ResourceBinding.
func (c *ContainerResourceBinding) GetUniqueIdentifier() string {
	return c.ConainerID
}

// RewriteLogSummary implements ResourceBinding.
func (c *ContainerResourceBinding) RewriteLogSummary(summary string) string {
	return rewriteIdWithReadableName(c.ConainerID, fmt.Sprintf("%s in %s/%s", c.ContainerName, c.PodNamespace, c.PodName), fmt.Sprintf("%s 【%s in %s/%s】", summary, c.ContainerName, c.PodNamespace, c.PodName))
}

var _ ResourceBinding = (*ContainerResourceBinding)(nil)

func rewriteIdWithReadableName(replaceTarget string, readableName string, originalMessage string) string {
	converted := fmt.Sprintf("%s...(%s)", replaceTarget[:min(len(replaceTarget), 7)], readableName)
	return strings.ReplaceAll(originalMessage, replaceTarget, converted)
}
