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
)

var (
	pathClusterName       = structured.CompileFieldPath("resource.labels.cluster_name")
	pathNodepoolName      = structured.CompileFieldPath("resource.labels.nodepool_name")
	pathDesiredNodePoolID = structured.CompileFieldPath("protoPayload.request.update.desiredNodePoolId")
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

// ExtractGKEAuditLogResource extracts GKE Audit Log resource fields from a NodeReader.
func ExtractGKEAuditLogResource(reader *structured.NodeReader) (GKEAuditLogResourceFieldSet, error) {
	if mock, ok := structured.GetMock[GKEAuditLogResourceFieldSet](reader); ok {
		return mock, nil
	}
	var result GKEAuditLogResourceFieldSet
	result.ClusterName = reader.ReadStringOrDefault(pathClusterName, "unknown")
	result.NodepoolName = reader.ReadStringOrDefault(pathNodepoolName, "")
	if result.NodepoolName == "" {
		// UpdateCluster operation for Nodepool may associate with cluster resource type, but actually for nodepool.
		result.NodepoolName = reader.ReadStringOrDefault(pathDesiredNodePoolID, "")
	}
	return result, nil
}
