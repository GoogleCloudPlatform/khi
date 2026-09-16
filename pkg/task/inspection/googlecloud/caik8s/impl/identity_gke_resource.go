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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

// gkeResourceIdentity holds identifying details extracted from a GKE asset name.
type gkeResourceIdentity struct {
	ClusterName  string
	NodePoolName string
}

// IsCluster returns true if the asset identity corresponds to a cluster.
func (i gkeResourceIdentity) IsCluster() bool {
	return i.ClusterName != "" && i.NodePoolName == ""
}

// IsNodePool returns true if the asset identity corresponds to a node pool.
func (i gkeResourceIdentity) IsNodePool() bool {
	return i.ClusterName != "" && i.NodePoolName != ""
}

// parseGKEAssetName parses a GKE full resource name into a gkeResourceIdentity.
func parseGKEAssetName(name string) gkeResourceIdentity {
	parts := strings.Split(name, "/")
	var clusterName, nodePoolName string
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "clusters" {
			clusterName = parts[i+1]
		} else if parts[i] == "nodePools" {
			nodePoolName = parts[i+1]
		}
	}
	return gkeResourceIdentity{
		ClusterName:  clusterName,
		NodePoolName: nodePoolName,
	}
}

func extractGKEIdentity(reader *structured.NodeReader) (gkeResourceIdentity, bool) {
	assetName := gcpcommon.ExtractCAIAssetName(reader)
	identity := parseGKEAssetName(assetName)
	if identity.ClusterName == "" {
		return gkeResourceIdentity{}, false
	}
	return identity, true
}

func gkeIdentityGroupKey(identity gkeResourceIdentity) string {
	if identity.IsNodePool() {
		return fmt.Sprintf("nodepool/%s/%s", identity.ClusterName, identity.NodePoolName)
	}
	return fmt.Sprintf("cluster/%s", identity.ClusterName)
}

func formatGKEResourceLogSummary(identity gkeResourceIdentity) string {
	if identity.IsNodePool() {
		return fmt.Sprintf("CAI resource snapshot: NodePool/%s", identity.NodePoolName)
	}
	return fmt.Sprintf("CAI resource snapshot: Cluster/%s", identity.ClusterName)
}
