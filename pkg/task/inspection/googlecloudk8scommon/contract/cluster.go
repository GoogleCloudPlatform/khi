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

// ClusterNameUsage represents a usage context layer of a cluster name.
type ClusterNameUsage string

const (
	// ClusterNameUsageK8sCluster is the usage context for Kubernetes internal common logs.
	ClusterNameUsageK8sCluster ClusterNameUsage = "k8s_cluster"
	// ClusterNameUsageK8sPlatformAudit is the usage context for Kubernetes platform management audit logs.
	ClusterNameUsageK8sPlatformAudit ClusterNameUsage = "k8s_platform_audit"
	// ClusterNameUsageCSM is the usage context for Cloud Service Mesh logs.
	ClusterNameUsageCSM ClusterNameUsage = "csm"
)

// ClusterPrefixPolicy defines a policy to apply a prefix to a cluster name based on the usage context.
type ClusterPrefixPolicy struct {
	// Prefix is the single cluster prefix string.
	Prefix string
	// RequiredUsages represents the list of usages that require the prefix.
	RequiredUsages []ClusterNameUsage
}

// PrefixFor returns the prefix for the given usage context if the usage is listed in RequiredUsages.
func (c *ClusterPrefixPolicy) PrefixFor(usage ClusterNameUsage) string {
	for _, u := range c.RequiredUsages {
		if u == usage {
			return c.Prefix
		}
	}
	return ""
}

// Apply applies the prefix to the given cluster name if the usage is listed in RequiredUsages.
func (c *ClusterPrefixPolicy) Apply(usage ClusterNameUsage, clusterName string) string {
	return fmt.Sprintf("%s%s", c.PrefixFor(usage), clusterName)
}

// GoogleCloudClusterIdentity is the tuple identify a cluster in Google Cloud.
type GoogleCloudClusterIdentity struct {
	// ProjectID is the project ID of the cluster.
	ProjectID string
	// PrefixPolicy is the prefix policy applied to the cluster name.
	PrefixPolicy ClusterPrefixPolicy
	// ClusterName is the name of the cluster.
	ClusterName string
	// Location is the location of the cluster.
	Location string
}

// NameFor returns the cluster name representation for the given usage context.
func (g *GoogleCloudClusterIdentity) NameFor(usage ClusterNameUsage) string {
	return g.PrefixPolicy.Apply(usage, g.ClusterName)
}

// PrefixFor returns the cluster prefix for the given usage context.
func (g *GoogleCloudClusterIdentity) PrefixFor(usage ClusterNameUsage) string {
	return g.PrefixPolicy.PrefixFor(usage)
}

// UniqueDigest returns an unique string for the cluster identity. This can be used as the cache key depending on a cluster.
func (g *GoogleCloudClusterIdentity) UniqueDigest() string {
	return fmt.Sprintf("%s|%s|%s|%s", g.ProjectID, g.PrefixPolicy.Prefix, g.ClusterName, g.Location)
}
