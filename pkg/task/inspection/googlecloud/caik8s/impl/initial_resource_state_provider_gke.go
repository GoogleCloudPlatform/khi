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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gkeapiaudit"
)

// caiGKEInitialResourceStateProvider serves initial states of GKE clusters and node pools from CAI.
type caiGKEInitialResourceStateProvider struct {
	clusters  map[string]*gkeapiaudit.InitialResourceState
	nodePools map[string]*gkeapiaudit.InitialResourceState
}

var _ gkeapiaudit.InitialResourceStateProvider = (*caiGKEInitialResourceStateProvider)(nil)

// ClusterInitialState implements gkeapiaudit.InitialResourceStateProvider.
func (p *caiGKEInitialResourceStateProvider) ClusterInitialState(clusterName string) (*gkeapiaudit.InitialResourceState, bool) {
	state, found := p.clusters[clusterName]
	return state, found
}

// NodePoolInitialState implements gkeapiaudit.InitialResourceStateProvider.
func (p *caiGKEInitialResourceStateProvider) NodePoolInitialState(clusterName, nodePoolName string) (*gkeapiaudit.InitialResourceState, bool) {
	state, found := p.nodePools[nodePoolKey(clusterName, nodePoolName)]
	return state, found
}

func nodePoolKey(clusterName, nodePoolName string) string {
	return fmt.Sprintf("%s/%s", clusterName, nodePoolName)
}

func buildCAIGKEInitialResourceStateProvider(states []gcpcommon.CAIActiveAssetState[gkeResourceIdentity]) *caiGKEInitialResourceStateProvider {
	clusters := map[string]*gkeapiaudit.InitialResourceState{}
	nodePools := map[string]*gkeapiaudit.InitialResourceState{}

	for _, state := range states {
		initialState := &gkeapiaudit.InitialResourceState{
			ResourceBody: state.ResourceBody,
		}
		if state.Identity.IsCluster() {
			clusters[state.Identity.ClusterName] = initialState
		} else if state.Identity.IsNodePool() {
			nodePools[nodePoolKey(state.Identity.ClusterName, state.Identity.NodePoolName)] = initialState
		}
	}

	return &caiGKEInitialResourceStateProvider{
		clusters:  clusters,
		nodePools: nodePools,
	}
}

// GKEResourceInitialStateProviderTask supplies initial GKE resource manifests to the audit log parser.
var GKEResourceInitialStateProviderTask = gcpcommon.NewCAIInitialResourceStateProviderTask(
	taskid.NewImplementationID(gkeapiaudit.InitialResourceStateProviderRef, "cai"),
	caik8s.GKEResourceTaskIDs,
	extractGKEIdentity,
	gkeIdentityGroupKey,
	extractGKEResourceBody,
	func(states []gcpcommon.CAIActiveAssetState[gkeResourceIdentity]) gkeapiaudit.InitialResourceStateProvider {
		return buildCAIGKEInitialResourceStateProvider(states)
	},
)
