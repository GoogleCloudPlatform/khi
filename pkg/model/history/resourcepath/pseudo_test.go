// Copyright 2024 Google LLC
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

package resourcepath

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

func TestCluster(t *testing.T) {
	expectedParentRelationship := enum.RelationshipChild
	testCases := []struct {
		name        string
		clusterName string
		expected    string
	}{
		{"Cluster name specified", "my-cluster", "@Cluster#controlplane#cluster-scope#my-cluster"},
		{"Empty cluster name", "", "@Cluster#controlplane#cluster-scope#unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Cluster(tc.clusterName)
			if result.Path != tc.expected {
				t.Errorf("Cluster(%v).Path = %v, want %v", tc.clusterName, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("Cluster(%v).ParentRelationship = %v, want %v", tc.clusterName, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestAutoscaler(t *testing.T) {
	expectedParentRelationship := enum.RelationshipControlPlaneComponent
	testCases := []struct {
		name        string
		clusterName string
		expected    string
	}{
		{"Cluster name specified", "my-cluster", "@Cluster#controlplane#cluster-scope#my-cluster#autoscaler"},
		{"Empty cluster name", "", "@Cluster#controlplane#cluster-scope#unknown#autoscaler"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Autoscaler(tc.clusterName)
			if result.Path != tc.expected {
				t.Errorf("Autoscaler(%v).Path = %v, want %v", tc.clusterName, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("Autoscaler(%v).ParentRelationship = %v, want %v", tc.clusterName, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestNodepool(t *testing.T) {
	expectedParentRelationship := enum.RelationshipChild
	testCases := []struct {
		name         string
		clusterName  string
		nodepoolName string
		expected     string
	}{
		{"All specified", "my-cluster", "my-nodepool", "@Cluster#nodepool#my-cluster#my-nodepool"},
		{"Empty cluster name", "", "my-nodepool", "@Cluster#nodepool#unknown#my-nodepool"},
		{"Empty nodepool name", "my-cluster", "", "@Cluster#nodepool#my-cluster#unknown"},
		{"Both empty", "", "", "@Cluster#nodepool#unknown#unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Nodepool(tc.clusterName, tc.nodepoolName)
			if result.Path != tc.expected {
				t.Errorf("Nodepool(%v,%v).Path = %v, want %v", tc.clusterName, tc.nodepoolName, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("Nodepool(%v,%v).ParentRelationship = %v, want %v", tc.clusterName, tc.nodepoolName, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestMig(t *testing.T) {
	expectedParentRelationship := enum.RelationshipManagedInstanceGroup
	testCases := []struct {
		name         string
		clusterName  string
		nodepoolName string
		migName      string
		expected     string
	}{
		{"All specified", "cluster", "nodepool", "mig", "@Cluster#nodepool#cluster#nodepool#mig"},
		{"Empty cluster name", "", "nodepool", "mig", "@Cluster#nodepool#unknown#nodepool#mig"},
		{"Empty nodepool name", "cluster", "", "mig", "@Cluster#nodepool#cluster#unknown#mig"},
		{"Empty mig name", "cluster", "nodepool", "", "@Cluster#nodepool#cluster#nodepool#unknown"},
		{"Two empty", "", "nodepool", "", "@Cluster#nodepool#unknown#nodepool#unknown"},
		{"Two empty #2", "cluster", "", "", "@Cluster#nodepool#cluster#unknown#unknown"},
		{"All empty", "", "", "", "@Cluster#nodepool#unknown#unknown#unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Mig(tc.clusterName, tc.nodepoolName, tc.migName)
			if result.Path != tc.expected {
				t.Errorf("Mig(%v,%v,%v).Path = %v, want %v", tc.clusterName, tc.nodepoolName, tc.migName, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("Mig(%v,%v,%v).ParentRelationship = %v, want %v", tc.clusterName, tc.nodepoolName, tc.migName, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestNodeComponent(t *testing.T) {
	expectedParentRelationship := enum.RelationshipNodeComponent
	testCases := []struct {
		name             string
		nodeName         string
		syslogIdentifier string
		expected         string
	}{
		{"All specified", "my-node", "kubelet", "core/v1#node#cluster-scope#my-node#kubelet"},
		{"Empty node name", "", "kubelet", "core/v1#node#cluster-scope#unknown#kubelet"},
		{"Empty syslog identifier", "my-node", "", "core/v1#node#cluster-scope#my-node#unknown"},
		{"Both empty", "", "", "core/v1#node#cluster-scope#unknown#unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NodeComponent(tc.nodeName, tc.syslogIdentifier)
			if result.Path != tc.expected {
				t.Errorf("NodeComponent(%v,%v).Path = %v, want %v", tc.nodeName, tc.syslogIdentifier, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("NodeComponent(%v,%v).ParentRelationship = %v, want %v", tc.nodeName, tc.syslogIdentifier, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestNodeSerialport(t *testing.T) {
	expectedParentRelationship := enum.RelationshipSerialPort
	testCases := []struct {
		name     string
		nodeName string
		expected string
	}{
		{"Node name specified", "my-node", "core/v1#node#cluster-scope#my-node#serialport"},
		{"Empty node name", "", "core/v1#node#cluster-scope#unknown#serialport"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NodeSerialport(tc.nodeName)
			if result.Path != tc.expected {
				t.Errorf("NodeSerialport(%v).Path = %v, want %v", tc.nodeName, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("NodeSerialport(%v).ParentRelationship = %v, want %v", tc.nodeName, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestNodeBinding(t *testing.T) {
	expectedParentRelationship := enum.RelationshipPodBinding
	testCases := []struct {
		name         string
		nodeName     string
		podNamespace string
		podName      string
		expected     string
	}{
		{"All specified", "my-node", "my-namespace", "my-pod", "core/v1#node#cluster-scope#my-node#my-pod(my-namespace)"},
		{"Empty node name", "", "my-namespace", "my-pod", "core/v1#node#cluster-scope#unknown#my-pod(my-namespace)"},
		{"Empty pod namespace", "my-node", "", "my-pod", "core/v1#node#cluster-scope#my-node#my-pod(unknown)"},
		{"Empty pod name", "my-node", "my-namespace", "", "core/v1#node#cluster-scope#my-node#unknown(my-namespace)"},
		{"Two empty", "", "my-namespace", "", "core/v1#node#cluster-scope#unknown#unknown(my-namespace)"},
		{"Two empty #2", "my-node", "", "", "core/v1#node#cluster-scope#my-node#unknown(unknown)"},
		{"All empty", "", "", "", "core/v1#node#cluster-scope#unknown#unknown(unknown)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NodeBinding(tc.nodeName, tc.podNamespace, tc.podName)
			if result.Path != tc.expected {
				t.Errorf("NodeBinding(%v,%v,%v).Path = %v, want %v", tc.nodeName, tc.podNamespace, tc.podName, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("NodeBinding(%v,%v,%v).ParentRelationship = %v, want %v", tc.nodeName, tc.podNamespace, tc.podName, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestPodEndpointSlice(t *testing.T) {
	expectedParentRelationship := enum.RelationshipEndpointSlice
	testCases := []struct {
		name                   string
		endpointSliceName      string
		endpointSliceNamespace string
		podNamespace           string
		podName                string
		expected               string
	}{
		{"All specified", "my-endpointslice", "my-namespace", "my-namespace", "my-pod", "core/v1#pod#my-namespace#my-pod#my-endpointslice(my-namespace)[endpointslice]"},
		{"Empty endpointSliceName", "", "my-namespace", "my-namespace", "my-pod", "core/v1#pod#my-namespace#my-pod#unknown(my-namespace)[endpointslice]"},
		{"Empty endpointSliceNamespace", "my-endpointslice", "", "my-namespace", "my-pod", "core/v1#pod#my-namespace#my-pod#my-endpointslice(unknown)[endpointslice]"},
		{"Empty pod namespace", "my-endpointslice", "my-namespace", "", "my-pod", "core/v1#pod#unknown#my-pod#my-endpointslice(my-namespace)[endpointslice]"},
		{"Empty pod name", "my-endpointslice", "my-namespace", "my-namespace", "", "core/v1#pod#my-namespace#unknown#my-endpointslice(my-namespace)[endpointslice]"},
		{"Two empty", "", "my-namespace", "", "", "core/v1#pod#unknown#unknown#unknown(my-namespace)[endpointslice]"},
		{"Two empty #2", "my-endpointslice", "", "my-namespace", "", "core/v1#pod#my-namespace#unknown#my-endpointslice(unknown)[endpointslice]"},
		{"All empty", "", "", "", "", "core/v1#pod#unknown#unknown#unknown(unknown)[endpointslice]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PodEndpointSlice(tc.endpointSliceNamespace, tc.endpointSliceName, tc.podNamespace, tc.podName)
			if result.Path != tc.expected {
				t.Errorf("PodEndpointSlice(%v,%v,%v,%v).Path = %v, want %v", tc.endpointSliceNamespace, tc.endpointSliceName, tc.podNamespace, tc.podName, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("PodEndpointSlice(%v,%v,%v,%v).ParentRelationship = %v, want %v", tc.endpointSliceNamespace, tc.endpointSliceName, tc.podNamespace, tc.podName, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestServiceEndpointSlice(t *testing.T) {
	expectedParentRelationship := enum.RelationshipEndpointSlice
	testCases := []struct {
		name              string
		namespace         string
		endpointSliceName string
		serviceName       string
		expected          string
	}{
		{"All specified", "my-namespace", "my-endpointslice", "my-service", "core/v1#service#my-namespace#my-service#my-endpointslice(my-namespace)[endpointslice]"},
		{"Empty endpointSliceName", "my-namespace", "", "my-service", "core/v1#service#my-namespace#my-service#unknown(my-namespace)[endpointslice]"},
		{"Empty namespace", "", "my-endpointslice", "my-service", "core/v1#service#unknown#my-service#my-endpointslice(unknown)[endpointslice]"},
		{"Empty serviceName", "my-namespace", "my-endpointslice", "", "core/v1#service#my-namespace#unknown#my-endpointslice(my-namespace)[endpointslice]"},
		{"Two empty", "", "", "my-service", "core/v1#service#unknown#my-service#unknown(unknown)[endpointslice]"},
		{"Two empty #2", "my-namespace", "", "", "core/v1#service#my-namespace#unknown#unknown(my-namespace)[endpointslice]"},
		{"All empty", "", "", "", "core/v1#service#unknown#unknown#unknown(unknown)[endpointslice]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ServiceEndpointSlice(tc.namespace, tc.endpointSliceName, tc.serviceName)
			if result.Path != tc.expected {
				t.Errorf("ServiceEndpointSlice(%v,%v,%v).Path = %v, want %v", tc.namespace, tc.endpointSliceName, tc.serviceName, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("ServiceEndpointSlice(%v,%v,%v).ParentRelationship = %v, want %v", tc.namespace, tc.endpointSliceName, tc.serviceName, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestOperation(t *testing.T) {
	expectedParentRelationship := enum.RelationshipOperation
	testCases := []struct {
		name            string
		operationOwner  ResourcePath
		operationMethod string
		operationId     string
		expected        string
	}{
		{"All specified", ResourcePath{Path: "foo"}, "GET", "1234567890", "foo#GET-1234567890"},
		{"Empty operation method", ResourcePath{Path: "foo"}, "", "1234567890", "foo#unknown-1234567890"},
		{"Empty operation id", ResourcePath{Path: "foo"}, "GET", "", "foo#GET-unknown"},
		{"Both empty", ResourcePath{Path: "foo"}, "", "", "foo#unknown-unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Operation(tc.operationOwner, tc.operationMethod, tc.operationId)
			if result.Path != tc.expected {
				t.Errorf("Operation(%v,%v,%v).Path = %v, want %v", tc.operationOwner, tc.operationMethod, tc.operationId, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("Operation(%v,%v,%v).ParentRelationship = %v, want %v", tc.operationOwner, tc.operationMethod, tc.operationId, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestStatus(t *testing.T) {
	expectedParentRelationship := enum.RelationshipResourceCondition
	testCases := []struct {
		name        string
		statusOwner ResourcePath
		statusName  string
		expected    string
	}{
		{"All specified", ResourcePath{Path: "foo"}, "status", "foo#status"},
		{"Empty status name", ResourcePath{Path: "foo"}, "", "foo#unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Status(tc.statusOwner, tc.statusName)
			if result.Path != tc.expected {
				t.Errorf("Status(%v,%v).Path = %v, want %v", tc.statusOwner, tc.statusName, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("Status(%v,%v).ParentRelationship = %v, want %v", tc.statusOwner, tc.statusName, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestNetworkEndpointGroupUnderResource(t *testing.T) {
	expectedParentRelationship := enum.RelationshipNetworkEndpointGroup
	testCases := []struct {
		name         string
		parent       ResourcePath
		negNamespace string
		negName      string
		expected     string
	}{
		{"All specified", ResourcePath{Path: "foo"}, "my-namespace", "my-neg", "foo#my-namespace(my-neg)"},
		{"Empty neg namespace", ResourcePath{Path: "foo"}, "", "my-neg", "foo#unknown(my-neg)"},
		{"Empty neg name", ResourcePath{Path: "foo"}, "my-namespace", "", "foo#my-namespace(unknown)"},
		{"Both empty", ResourcePath{Path: "foo"}, "", "", "foo#unknown(unknown)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NetworkEndpointGroupUnderResource(tc.parent, tc.negNamespace, tc.negName)
			if result.Path != tc.expected {
				t.Errorf("NetworkEndpointGroupUnderResource(%v,%v,%v).Path = %v, want %v", tc.parent, tc.negNamespace, tc.negName, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("NetworkEndpointGroupUnderResource(%v,%v,%v).ParentRelationship = %v, want %v", tc.parent, tc.negNamespace, tc.negName, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}

func TestOwnerSubresource(t *testing.T) {
	expectedParentRelationship := enum.RelationshipOwnerReference
	testCases := []struct {
		name              string
		ownerPath         ResourcePath
		ownedResourceName string
		ownedResourceKind string
		expected          string
	}{
		{"All specified", ResourcePath{Path: "foo"}, "bar", "Deployment", "foo#bar[kind:Deployment]"},
		{"Empty ownedResourceName", ResourcePath{Path: "foo"}, "", "Deployment", "foo#unknown[kind:Deployment]"},
		{"Empty ownedResourceKind", ResourcePath{Path: "foo"}, "bar", "", "foo#bar[kind:unknown]"},
		{"Both empty", ResourcePath{Path: "foo"}, "", "", "foo#unknown[kind:unknown]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := OwnerSubresource(tc.ownerPath, tc.ownedResourceName, tc.ownedResourceKind)
			if result.Path != tc.expected {
				t.Errorf("OwnerSubresource(%v,%v,%v).Path = %v, want %v", tc.ownerPath, tc.ownedResourceName, tc.ownedResourceKind, result.Path, tc.expected)
			}
			if result.ParentRelationship != expectedParentRelationship {
				t.Errorf("OwnerSubresource(%v,%v,%v).ParentRelationship = %v, want %v", tc.ownerPath, tc.ownedResourceName, tc.ownedResourceKind, result.ParentRelationship, expectedParentRelationship)
			}
		})
	}
}
