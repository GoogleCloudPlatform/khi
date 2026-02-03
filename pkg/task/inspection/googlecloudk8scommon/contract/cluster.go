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

package googlecloudk8scommon_contract

import "fmt"

// GoogleCloudClusterIdentity is the tuple identify a cluster in Google Cloud.
type GoogleCloudClusterIdentity struct {
	// ProjectID is the project ID of the cluster.
	ProjectID string
	// ClusterTypePrefix is an empty string for GKE & GDC, "awsClusters/" for GKE on AWS and "azureClusters/" for GKE on Azure.
	ClusterTypePrefix string
	// ClusterName is the name of the cluster.
	ClusterName string
	// Location is the location of the cluster.
	Location string
}

func (g *GoogleCloudClusterIdentity) NameWithClusterTypePrefix() string {
	return fmt.Sprintf("%s%s", g.ClusterTypePrefix, g.ClusterName)
}

// UniqueDigest returns an unique string for the cluster identity. This can be used as the cache key depending on a cluster.
func (g *GoogleCloudClusterIdentity) UniqueDigest() string {
	return fmt.Sprintf("%s|%s|%s|%s", g.ProjectID, g.ClusterTypePrefix, g.ClusterName, g.Location)
}
