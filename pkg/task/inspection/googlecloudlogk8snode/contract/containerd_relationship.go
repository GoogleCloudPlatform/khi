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

package googlecloudlogk8snode_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
)

type ContainerIDInfo struct {
	ContainerID   string
	ContainerName string
	PodSandboxID  string
}

func (c *ContainerIDInfo) ResourcePath(podNamespace string, podName string) resourcepath.ResourcePath {
	return resourcepath.Container(podNamespace, podName, c.ContainerName)
}

type PodSandboxIDInfo struct {
	PodName      string
	PodNamespace string
	PodSandboxID string
}

func (p *PodSandboxIDInfo) ResourcePath() resourcepath.ResourcePath {
	return resourcepath.Pod(p.PodNamespace, p.PodName)
}

type ContainerdRelationshipRegistry struct {
	PodSandboxIDInfoFinder patternfinder.PatternFinder[*PodSandboxIDInfo]
	ContainerIDInfoFinder  patternfinder.PatternFinder[*ContainerIDInfo]
}

func NewContainerdRelationshipRegistry() *ContainerdRelationshipRegistry {
	return &ContainerdRelationshipRegistry{
		PodSandboxIDInfoFinder: patternfinder.NewTriePatternFinder[*PodSandboxIDInfo](),
		ContainerIDInfoFinder:  patternfinder.NewTriePatternFinder[*ContainerIDInfo](),
	}
}
