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

package googlecloudlogonpremapiaudit_contract

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

var (
	pathProjectID    = structured.CompileFieldPath("resource.labels.project_id")
	pathResourceName = structured.CompileFieldPath("protoPayload.resourceName")
)

// OnPremClusterType represents the type of on-premises cluster.
type OnPremClusterType = string

const (
	ClusterTypeBaremetalAdmin      OnPremClusterType = "baremetalAdmin"
	ClusterTypeBaremetalStandalone OnPremClusterType = "baremetalStandalone"
	ClusterTypeBaremetalUser       OnPremClusterType = "baremetal"
	ClusterTypeVMWareAdmin         OnPremClusterType = "vmwareAdmin"
	ClusterTypeVMWareUser          OnPremClusterType = "vmware"
	ClusterTypeUnknown             OnPremClusterType = "unknown"
)

// OnPremAPIAuditResourceFieldSet holds structured log data extracted from an OnPrem API log.
type OnPremAPIAuditResourceFieldSet struct {
	Project      string
	ClusterType  OnPremClusterType
	ClusterName  string
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
