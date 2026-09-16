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

package caik8s_impl

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

// caiInitialResourceStateProvider serves the manifests Cloud Asset Inventory reported for the moment
// the inspection window opened.
type caiInitialResourceStateProvider struct {
	bodiesByIdentity map[string]*structured.NodeReader
}

var _ k8saudit.InitialResourceStateProvider = (*caiInitialResourceStateProvider)(nil)

// InitialResourceState implements k8saudit.InitialResourceStateProvider.
func (p *caiInitialResourceStateProvider) InitialResourceState(identity *k8saudit.ResourceIdentity) (*structured.NodeReader, bool) {
	body, found := p.bodiesByIdentity[initialResourceStateKey(identity)]
	return body, found
}

func buildCAIInitialResourceStateProvider(states []gcpcommon.CAIActiveAssetState[*k8saudit.ResourceIdentity]) *caiInitialResourceStateProvider {
	bodiesByIdentity := make(map[string]*structured.NodeReader, len(states))
	for _, state := range states {
		bodiesByIdentity[initialResourceStateKey(state.Identity)] = structured.NewNodeReader(state.ResourceBody)
	}
	return &caiInitialResourceStateProvider{bodiesByIdentity: bodiesByIdentity}
}

// initialResourceStateKey renders the lookup key of a CAI resource identity. CAI leaves the namespace
// empty for cluster scoped resources while the audit log pipeline names it ClusterScopeNamespace, so
// the key adopts the audit log convention.
func initialResourceStateKey(identity *k8saudit.ResourceIdentity) string {
	if identity.Namespace != "" {
		return identity.String()
	}
	clusterScoped := *identity
	clusterScoped.Namespace = k8saudit.ClusterScopeNamespace
	return clusterScoped.String()
}

// ClusterResourceInitialStateProviderTask supplies the audit log parser with the resource manifests CAI
// observed before the inspection window.
var ClusterResourceInitialStateProviderTask = gcpcommon.NewCAIInitialResourceStateProviderTask(
	taskid.NewImplementationID(k8saudit.InitialResourceStateProviderRef, "cai"),
	caik8s.ClusterResourceTaskIDs,
	extractK8sIdentity,
	initialResourceStateKey,
	extractK8sResourceBody,
	func(states []gcpcommon.CAIActiveAssetState[*k8saudit.ResourceIdentity]) k8saudit.InitialResourceStateProvider {
		return buildCAIInitialResourceStateProvider(states)
	},
)
