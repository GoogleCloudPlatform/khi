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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

func Cluster(name string) ResourcePath {
	if name == "" {
		name = nonSpecifiedPlaceholder
	}
	return NameLayerGeneralItem("@Cluster", "controlplane", "cluster-scope", name)
}

func Autoscaler(clusterName string) ResourcePath {
	cluster := Cluster(clusterName)
	cluster.Path = fmt.Sprintf("%s#autoscaler", cluster.Path)
	cluster.ParentRelationship = enum.RelationshipControlPlaneComponent
	return cluster
}

func Nodepool(clusterName string, nodepoolName string) ResourcePath {
	if clusterName == "" {
		clusterName = nonSpecifiedPlaceholder
	}
	if nodepoolName == "" {
		nodepoolName = nonSpecifiedPlaceholder
	}
	return NameLayerGeneralItem("@Cluster", "nodepool", clusterName, nodepoolName)
}

func Mig(clusterName string, nodepoolName string, migName string) ResourcePath {
	if migName == "" {
		migName = nonSpecifiedPlaceholder
	}
	nodepool := Nodepool(clusterName, nodepoolName)
	nodepool.Path = fmt.Sprintf("%s#%s", nodepool.Path, migName)
	nodepool.ParentRelationship = enum.RelationshipManagedInstanceGroup
	return nodepool
}

func NodeComponent(nodeName string, syslogIdentifier string) ResourcePath {
	if syslogIdentifier == "" {
		syslogIdentifier = nonSpecifiedPlaceholder
	}
	node := Node(nodeName)
	node.ParentRelationship = enum.RelationshipNodeComponent
	node.Path = fmt.Sprintf("%s#%s", node.Path, syslogIdentifier)
	return node
}

// NodeSerialport returns a ResourcePath for the pseudo serial port timeline under nodes.
func NodeSerialport(nodeName string) ResourcePath {
	node := Node(nodeName)
	node.ParentRelationship = enum.RelationshipSerialPort
	node.Path = fmt.Sprintf("%s#serialport", node.Path)
	return node
}

// NodeBinding returns a ResourcePath for the pseudo binding timeline under nodes.
func NodeBinding(nodeName string, podNamespace string, podName string) ResourcePath {
	if podName == "" {
		podName = nonSpecifiedPlaceholder
	}
	if podNamespace == "" {
		podNamespace = nonSpecifiedPlaceholder
	}
	node := Node(nodeName)
	node.Path = fmt.Sprintf("%s#%s(%s)", node.Path, podName, podNamespace)
	node.ParentRelationship = enum.RelationshipPodBinding
	return node
}

// PodEndpointSlice returns a ResourcePath for the pseudo endpointslice timeline under pods.
func PodEndpointSlice(endpointSliceNamespace string, endpointSliceName string, podNamespace string, podName string) ResourcePath {
	if endpointSliceName == "" {
		endpointSliceName = nonSpecifiedPlaceholder
	}
	if endpointSliceNamespace == "" {
		endpointSliceNamespace = nonSpecifiedPlaceholder
	}
	pod := Pod(podNamespace, podName)
	pod.Path = fmt.Sprintf("%s#%s(%s)[endpointslice]", pod.Path, endpointSliceName, endpointSliceNamespace)
	pod.ParentRelationship = enum.RelationshipEndpointSlice
	return pod
}

// ServiceEndpointSlice returns a ResourcePath for the pseudo endpointslice timeline under services.
func ServiceEndpointSlice(namespace string, endpointSliceName string, serviceName string) ResourcePath {
	if namespace == "" {
		namespace = nonSpecifiedPlaceholder
	}
	if endpointSliceName == "" {
		endpointSliceName = nonSpecifiedPlaceholder
	}
	service := Service(namespace, serviceName)
	service.Path = fmt.Sprintf("%s#%s(%s)[endpointslice]", service.Path, endpointSliceName, namespace)
	service.ParentRelationship = enum.RelationshipEndpointSlice
	return service
}

// Operation returns a ResourcePath for the pseudo operation timeline under the given name layer resource.
func Operation(operationOwner ResourcePath, operationMethod string, operationId string) ResourcePath {
	if operationMethod == "" {
		operationMethod = nonSpecifiedPlaceholder
	}
	if operationId == "" {
		operationId = nonSpecifiedPlaceholder
	}
	return ResourcePath{
		Path:               fmt.Sprintf("%s#%s-%s", operationOwner.Path, operationMethod, operationId),
		ParentRelationship: enum.RelationshipOperation,
	}
}

// Status returns a ResourcePath for the pseudo status timeline under the given name layer resource.
func Status(statusOwner ResourcePath, statusName string) ResourcePath {
	if statusName == "" {
		statusName = nonSpecifiedPlaceholder
	}
	return ResourcePath{
		Path:               fmt.Sprintf("%s#%s", statusOwner.Path, statusName),
		ParentRelationship: enum.RelationshipResourceCondition,
	}
}

// NetworkEndpointGroupUnderResource returns the pseudo neg timeline under the given name layer resource.
func NetworkEndpointGroupUnderResource(parent ResourcePath, negNamespace string, negName string) ResourcePath {
	if negNamespace == "" {
		negNamespace = nonSpecifiedPlaceholder
	}
	if negName == "" {
		negName = nonSpecifiedPlaceholder
	}
	return ResourcePath{
		Path:               fmt.Sprintf("%s#%s(%s)", parent.Path, negNamespace, negName),
		ParentRelationship: enum.RelationshipNetworkEndpointGroup,
	}
}

// OwnerSubresource returns a ResourcePath for the pseudo owner reference timeline under the given owner resource timeline at the name layer.
func OwnerSubresource(ownerPath ResourcePath, ownedResourceName string, ownedResourceKind string) ResourcePath {
	if ownedResourceName == "" {
		ownedResourceName = nonSpecifiedPlaceholder
	}
	if ownedResourceKind == "" {
		ownedResourceKind = nonSpecifiedPlaceholder
	}
	return ResourcePath{
		Path:               fmt.Sprintf("%s#%s[kind:%s]", ownerPath.Path, ownedResourceName, ownedResourceKind),
		ParentRelationship: enum.RelationshipOwnerReference,
	}
}
