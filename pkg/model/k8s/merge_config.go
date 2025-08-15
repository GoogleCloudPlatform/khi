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
	"fmt"
	"log/slog"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

// K8sManifestMergeConfigRegistry holds merge configurations for Kubernetes manifests.
type K8sManifestMergeConfigRegistry struct {
	defaultResolver      *structured.MergeConfigResolver
	mergeConfigResolvers map[string]*structured.MergeConfigResolver
}

// Register adds a new merge configuration for a specific apiVersion and kind.
// If a configuration for the same apiVersion and kind already exists, it logs an error.
func (r *K8sManifestMergeConfigRegistry) Register(apiVersion string, kind string, childResolver *structured.MergeConfigResolver) {
	mapKey := fmt.Sprintf("%s-%s", apiVersion, kind)
	if _, found := r.mergeConfigResolvers[mapKey]; found {
		slog.Error(fmt.Sprintf("Merge config for apiVersion: %s, kind:%s is already registered", apiVersion, kind))
	}
	r.mergeConfigResolvers[mapKey] = childResolver
}

// Get retrieves the merge configuration for a specific apiVersion and kind.
// If a specific configuration is not found, it returns the default resolver.
func (r *K8sManifestMergeConfigRegistry) Get(apiVersion string, kind string) *structured.MergeConfigResolver {
	mapKey := fmt.Sprintf("%s-%s", apiVersion, kind)
	if resolver, found := r.mergeConfigResolvers[mapKey]; found {
		return resolver
	} else {
		slog.Warn(fmt.Sprintf("Merge config for apiVersion: %s, kind:%s was not found", apiVersion, kind))
		return r.defaultResolver
	}
}
