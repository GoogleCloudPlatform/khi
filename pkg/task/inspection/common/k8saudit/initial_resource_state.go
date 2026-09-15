// Copyright 2026 Google LLC
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

package k8saudit

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

// ClusterScopeNamespace is the namespace value the audit log pipeline uses for cluster scoped resources.
// Kubernetes leaves the namespace empty for those resources, but KHI needs a non empty name to place them
// under a namespace timeline.
const ClusterScopeNamespace = "cluster-scope"

// InitialResourceStateProvider supplies the resource manifest that existed before the audit logs begin.
// An environment backed by a resource inventory can fill the gap between the resource creation and the
// first audit log in the inspection window.
//
// Implementations of this interface must be thread-safe, as InitialResourceState may be called
// concurrently from multiple goroutines.
type InitialResourceStateProvider interface {
	// InitialResourceState returns the manifest observed before the inspection window opened, and whether
	// the inventory covered the resource.
	InitialResourceState(identity *ResourceIdentity) (*structured.NodeReader, bool)
}
