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

package googlecloudlogmulticloudapiaudit_contract

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

var pathResourceName = structured.CompileFieldPath("protoPayload.resourceName")

// MultiCloudClusterType represents the type of multicloud cluster.
type MultiCloudClusterType = string

const (
	ClusterTypeAWS     MultiCloudClusterType = "aws"
	ClusterTypeAzure   MultiCloudClusterType = "azure"
	ClusterTypeUnknown MultiCloudClusterType = "unknown"
)

// MulticloudAPIAuditResourceFieldSet represents the parsed resource fields of a multicloud audit log.
type MulticloudAPIAuditResourceFieldSet struct {
	ClusterType  MultiCloudClusterType
	ClusterName  string
	NodepoolName string
}

// IsCluster returns true if the log entry is related to a cluster operation (i.e., no nodepool name is present).
func (g *MulticloudAPIAuditResourceFieldSet) IsCluster() bool {
	return g.NodepoolName == ""
}

// IsNodepool returns true if the log entry is related to a nodepool operation (i.e., a nodepool name is present).
func (g *MulticloudAPIAuditResourceFieldSet) IsNodepool() bool {
	return g.NodepoolName != ""
}

// ExtractMulticloudAPIAuditResource extracts Multicloud resource information from a NodeReader.
func ExtractMulticloudAPIAuditResource(reader *structured.NodeReader) (MulticloudAPIAuditResourceFieldSet, error) {
	if mock, ok := structured.GetMock[MulticloudAPIAuditResourceFieldSet](reader); ok {
		return mock, nil
	}
	result := MulticloudAPIAuditResourceFieldSet{
		ClusterType:  ClusterTypeUnknown,
		NodepoolName: "",
		ClusterName:  "unknown",
	}

	resourceName, err := reader.ReadString(pathResourceName)
	if err != nil {
		return result, err
	}

	// resourceName should be in the format of
	// projects/<PROJECT_NUMBER>/locations/<LOCATION>/(aws|azure)Clusters/<CLUSTER_NAME>(/(aws|azure)NodePools/<NODEPOOL_NAME>)
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
