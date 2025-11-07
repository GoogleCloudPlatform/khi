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

package googlecloudloggkeapiaudit_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// GKEAuditLogResourceFieldSet represents the resource-related fields extracted from a GKE audit log entry.
type GKEAuditLogResourceFieldSet struct {
	ClusterName  string
	NodepoolName string
}

// IsCluster returns true if the log entry is related to a GKE cluster operation (i.e., no nodepool name is present).
func (g *GKEAuditLogResourceFieldSet) IsCluster() bool {
	return g.NodepoolName == ""
}

// IsNodepool returns true if the log entry is related to a GKE nodepool operation (i.e., a nodepool name is present).
func (g *GKEAuditLogResourceFieldSet) IsNodepool() bool {
	return g.NodepoolName != ""
}

func (g *GKEAuditLogResourceFieldSet) ResourcePath() resourcepath.ResourcePath {
	if g.IsCluster() {
		return resourcepath.Cluster(g.ClusterName)
	}
	return resourcepath.Nodepool(g.ClusterName, g.NodepoolName)
}

// Kind implements log.FieldSet.
// It returns the kind of the field set, which is "gke_audit".
func (g *GKEAuditLogResourceFieldSet) Kind() string {
	return "gke_audit"
}

var _ log.FieldSet = (*GKEAuditLogResourceFieldSet)(nil)

type GKEAuditLogResourceFieldSetReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (g *GKEAuditLogResourceFieldSetReader) FieldSetKind() string {
	return (&GKEAuditLogResourceFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
// It reads the "resource.labels.cluster_name" and "resource.labels.nodepool_name" fields
// from the provided NodeReader and populates a GKEAuditLogResourceFieldSet.
func (g *GKEAuditLogResourceFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result GKEAuditLogResourceFieldSet
	result.ClusterName = reader.ReadStringOrDefault("resource.labels.cluster_name", "unknown")
	result.NodepoolName = reader.ReadStringOrDefault("resource.labels.nodepool_name", "")
	if result.NodepoolName == "" {
		// UpdateCluster operation for Nodepool may associates with cluster resource type, but actually for nodepool.
		result.NodepoolName = reader.ReadStringOrDefault("protoPayload.request.update.desiredNodePoolId", "")
	}
	return &result, nil
}

var _ log.FieldSetReader = (*GKEAuditLogResourceFieldSetReader)(nil)
