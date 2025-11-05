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
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

type OnPremClusterType = string

const (
	ClusterTypeBaremetalAdmin      OnPremClusterType = "baremetalAdmin"
	ClusterTypeBaremetalStandalone OnPremClusterType = "baremetalStandalone"
	ClusterTypeBaremetalUser       OnPremClusterType = "baremetal"
	ClusterTypeVMWareAdmin         OnPremClusterType = "vmwareAdmin"
	ClusterTypeVMWareUser          OnPremClusterType = "vmware"
	ClusterTypeUnknown             OnPremClusterType = "unknown"
)

type OnPremAPIAuditResourceFieldSet struct {
	ClusterType  OnPremClusterType
	ClusterName  string
	NodepoolName string
}

// Kind implements log.FieldSet.
func (m *OnPremAPIAuditResourceFieldSet) Kind() string {
	return "onprem_audit"
}

// IsCluster returns true if the log entry is related to a cluster operation (i.e., no nodepool name is present).
func (g *OnPremAPIAuditResourceFieldSet) IsCluster() bool {
	return g.NodepoolName == ""
}

// IsNodepool returns true if the log entry is related to a nodepool operation (i.e., a nodepool name is present).
func (g *OnPremAPIAuditResourceFieldSet) IsNodepool() bool {
	return g.NodepoolName != ""
}

func (g *OnPremAPIAuditResourceFieldSet) ResourcePath() resourcepath.ResourcePath {
	if g.IsCluster() {
		return resourcepath.Cluster(g.ClusterName)
	} else {
		return resourcepath.Nodepool(g.ClusterName, g.NodepoolName)
	}
}

var _ log.FieldSet = (*OnPremAPIAuditResourceFieldSet)(nil)

type OnPremAPIAuditResourceFieldSetReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (m *OnPremAPIAuditResourceFieldSetReader) FieldSetKind() string {
	return (&OnPremAPIAuditResourceFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (m *OnPremAPIAuditResourceFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	result := &OnPremAPIAuditResourceFieldSet{
		ClusterType:  ClusterTypeUnknown,
		NodepoolName: "",
		ClusterName:  "unknown",
	}

	resourceName, err := reader.ReadString("protoPayload.resourceName")
	if err != nil {
		return nil, err
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

var _ log.FieldSetReader = (*OnPremAPIAuditResourceFieldSetReader)(nil)
