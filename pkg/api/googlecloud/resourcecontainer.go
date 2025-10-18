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

package googlecloud

// ResourceContainerType represents the type of a Google Cloud resource container.
type ResourceContainerType int

// ResourceContainer is an interface that represents a container for Google Cloud resources (e.g., project, organization, folder, or billing account).
// ClientFactory receives a resource container to generate a client. This is needed because KHI can use multiple clients when it needs to gather logs or resource info from multiple projects.
type ResourceContainer interface {
	GetType() ResourceContainerType
}

const (
	// ResourceContainerInvalid represents an ResourceContainerInvalid resource container type. This is used for test mock types.
	ResourceContainerInvalid ResourceContainerType = iota
	// ResourceContainerProject represents a Google Cloud ResourceContainerProject resource container.
	ResourceContainerProject ResourceContainerType = iota
)

// ProjectResourceContainer is an interface that represents a Google Cloud project resource container.
type ProjectResourceContainer interface {
	ResourceContainer
	ProjectID() string
}

// projectResourceContainerImpl is an implementation of ResourceContainer for a Google Cloud project.
type projectResourceContainerImpl struct {
	projectID string
}

// Project creates a new ResourceContainer for a Google Cloud project with the given project ID.
func Project(projectID string) ResourceContainer {
	return &projectResourceContainerImpl{
		projectID: projectID,
	}
}

// GetType returns the ResourceContainerType for a projectResourceContainer, which is 'project'.
func (p *projectResourceContainerImpl) GetType() ResourceContainerType {
	return ResourceContainerProject
}

// ProjectID returns the projectID of this container.
func (p *projectResourceContainerImpl) ProjectID() string {
	return p.projectID
}

var _ ProjectResourceContainer = (*projectResourceContainerImpl)(nil)
