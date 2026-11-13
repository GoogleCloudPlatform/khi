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

package onpremapiaudit

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

const (
	// ClusterTypeBaremetalAdmin identifies a bare-metal admin cluster.
	ClusterTypeBaremetalAdmin OnPremClusterType = "baremetalAdmin"
	// ClusterTypeBaremetalStandalone identifies a bare-metal standalone cluster.
	ClusterTypeBaremetalStandalone OnPremClusterType = "baremetalStandalone"
	// ClusterTypeBaremetalUser identifies a bare-metal user cluster.
	ClusterTypeBaremetalUser OnPremClusterType = "baremetal"
	// ClusterTypeVMWareAdmin identifies a VMware admin cluster.
	ClusterTypeVMWareAdmin OnPremClusterType = "vmwareAdmin"
	// ClusterTypeVMWareUser identifies a VMware user cluster.
	ClusterTypeVMWareUser OnPremClusterType = "vmware"
	// ClusterTypeUnknown identifies an unknown on-premises cluster type.
	ClusterTypeUnknown OnPremClusterType = "unknown"
)

var (
	pathProjectID    = structured.CompileFieldPath("resource.labels.project_id")
	pathResourceName = structured.CompileFieldPath("protoPayload.resourceName")
)

// OnPremClusterType represents the type of on-premises cluster.
type OnPremClusterType = string

// OnPremAPIAuditResourceFieldSet holds structured log data extracted from an OnPrem API log.
type OnPremAPIAuditResourceFieldSet struct {
	// Project is the GCP project ID or number associated with the cluster.
	Project string
	// ClusterType is the on-premises cluster variant (e.g., baremetal, vmware).
	ClusterType OnPremClusterType
	// ClusterName is the name of the on-premises cluster.
	ClusterName string
	// NodepoolName is the name of the node pool when the log targets a node pool, or empty otherwise.
	NodepoolName string
}

// IsCluster returns true if the log entry is related to a cluster operation (i.e., no nodepool name is present).
func (g *OnPremAPIAuditResourceFieldSet) IsCluster() bool {
	return g.NodepoolName == ""
}

// IsNodepool returns true if the log entry is related to a nodepool operation (i.e., a nodepool name is present).
func (g *OnPremAPIAuditResourceFieldSet) IsNodepool() bool {
	return g.NodepoolName != ""
}

// ExtractOnPremAPIAuditResource extracts OnPrem resource fields from a NodeReader.
func ExtractOnPremAPIAuditResource(reader *structured.NodeReader) (OnPremAPIAuditResourceFieldSet, error) {
	if mock, ok := structured.GetMock[OnPremAPIAuditResourceFieldSet](reader); ok {
		return mock, nil
	}
	result := OnPremAPIAuditResourceFieldSet{
		Project:      "unknown",
		ClusterType:  ClusterTypeUnknown,
		NodepoolName: "",
		ClusterName:  "unknown",
	}

	if projectID, err := reader.ReadString(pathProjectID); err == nil && projectID != "" {
		result.Project = projectID
	}

	resourceName, err := reader.ReadString(pathResourceName)
	if err != nil {
		return result, err
	}

	// resourceName should be in the format of
	// projects/<PROJECT_NUMBER>/locations/<LOCATION>/(baremetal*|vmware*)Clusters/<CLUSTER_NAME>(/(baremetal*|vmware*)NodePools/<NODEPOOL_NAME>)
	splited := strings.Split(resourceName, "/")
	if len(splited) > 5 {
		result.ClusterName = splited[5]
	}
	if len(splited) > 7 {
		result.NodepoolName = splited[7]
	}
	if len(splited) > 4 {
		result.ClusterType = strings.TrimSuffix(splited[4], "Clusters")
	}

	return result, nil
}
