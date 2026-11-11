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

package dpv2policy

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

var (
	pathDisposition      = structured.CompileFieldPath("jsonPayload.disposition")
	pathCount            = structured.CompileFieldPath("jsonPayload.count")
	pathNodeName         = structured.CompileFieldPath("jsonPayload.node_name")
	pathResourceNodeName = structured.CompileFieldPath("resource.labels.node_name")
	pathDirection        = structured.CompileFieldPath("jsonPayload.connection.direction")
	pathProtocol         = structured.CompileFieldPath("jsonPayload.connection.protocol")
	pathSourceIP         = structured.CompileFieldPath("jsonPayload.connection.src_ip")
	pathSourcePort       = structured.CompileFieldPath("jsonPayload.connection.src_port")
	pathDestinationIP    = structured.CompileFieldPath("jsonPayload.connection.dest_ip")
	pathDestinationPort  = structured.CompileFieldPath("jsonPayload.connection.dest_port")
	pathSource           = structured.CompileFieldPath("jsonPayload.src")
	pathDestination      = structured.CompileFieldPath("jsonPayload.dest")
	pathPolicies         = structured.CompileFieldPath("jsonPayload.policies")

	// The following paths are relative to an element of jsonPayload.src, jsonPayload.dest or jsonPayload.policies.
	pathPodName      = structured.CompileFieldPath("pod_name")
	pathPodNamespace = structured.CompileFieldPath("pod_namespace")
	pathNamespace    = structured.CompileFieldPath("namespace")
	pathWorkloadKind = structured.CompileFieldPath("workload_kind")
	pathWorkloadName = structured.CompileFieldPath("workload_name")
	pathInstance     = structured.CompileFieldPath("instance")
	pathKind         = structured.CompileFieldPath("kind")
	pathName         = structured.CompileFieldPath("name")
	pathRevision     = structured.CompileFieldPath("revision")
	pathLabels       = structured.CompileFieldPath("labels")
)

const (
	// DispositionAllow is the disposition of a connection allowed by network policies.
	DispositionAllow = "allow"
	// DispositionDeny is the disposition of a connection denied by network policies.
	DispositionDeny = "deny"
)

const (
	// policyUIDLabelPrefix is the prefix of the Cilium policy label holding the UID of the Kubernetes resource that the policy rule derives from.
	policyUIDLabelPrefix = "k8s:io.cilium.k8s.policy.uid="
	// reservedRuleLabelPrefix is the prefix of the Cilium policy label naming a rule that Cilium adds implicitly instead of deriving it from a Kubernetes resource.
	reservedRuleLabelPrefix = "reserved:io.cilium.policy.derived-from="
)

// policyActionCacheKey is the cache key of PolicyActionFieldSet because the ingester, the grouper and the mapper read the same log.
var policyActionCacheKey = structured.NewCacheKey[PolicyActionFieldSet]()

// PolicyActionFieldSet holds the fields of a GKE Dataplane V2 network policy log (logs/policy-action).
type PolicyActionFieldSet struct {
	// Disposition is DispositionAllow or DispositionDeny.
	Disposition string
	// Count is the number of connections the log represents. Denied connections are aggregated in a 5-second window, so it can be larger than 1.
	Count int
	// NodeName is the name of the node where the policy was evaluated.
	NodeName string
	// Direction is "ingress" or "egress", the direction of the connection relative to the endpoint whose policy was evaluated.
	Direction string
	// Protocol is the protocol of the connection in lowercase (e.g., "tcp", "udp", "icmp").
	Protocol string
	// Source is the endpoint initiating the connection.
	Source PolicyActionEndpoint
	// Destination is the endpoint receiving the connection.
	Destination PolicyActionEndpoint
	// Policies are the policy rules that decided the connection.
	// It is empty when a NetworkPolicy or an FQDNNetworkPolicy denies the connection implicitly.
	Policies []PolicyActionPolicy
}

// PolicyActionEndpoint is the source or the destination endpoint of a connection recorded in a policy-action log.
type PolicyActionEndpoint struct {
	// IP is the IP address of the endpoint. Connections to a Service VIP record the IP address of the backend Pod.
	IP string
	// Port is the port of the endpoint. It is 0 for protocols without ports such as ICMP.
	Port int
	// PodNamespace is the namespace of the Pod. It is empty when the endpoint is not a Pod.
	PodNamespace string
	// PodName is the name of the Pod. It is empty when the endpoint is not a Pod.
	PodName string
	// WorkloadKind is the kind of the workload owning the Pod. Cilium only records it for Pods running on the reporting node.
	WorkloadKind string
	// WorkloadName is the name of the workload owning the Pod. Cilium only records it for Pods running on the reporting node.
	WorkloadName string
	// Instance is the IP address recorded for an endpoint that is not a Pod (e.g., an external IP address or a node-local DNS cache).
	Instance string
}

// IsPod reports whether the endpoint is a Pod.
// It only checks the namespace and the name because Cilium omits the workload fields for Pods on remote nodes.
func (e PolicyActionEndpoint) IsPod() bool {
	return e.PodNamespace != "" && e.PodName != ""
}

// PolicyActionPolicy is a policy rule that decided a connection recorded in a policy-action log.
type PolicyActionPolicy struct {
	// Kind is the kind of the Kubernetes resource defining the rule (e.g., "NetworkPolicy"). It is empty for reserved rules.
	Kind string
	// Name is the name of the Kubernetes resource defining the rule. It is empty for reserved rules.
	Name string
	// Namespace is the namespace of the Kubernetes resource defining the rule. It is empty for cluster-scoped resources and reserved rules.
	Namespace string
	// Revision is the revision of the policy repository in the Cilium agent when the rule was evaluated.
	Revision string
	// UID is the UID of the Kubernetes resource defining the rule.
	UID string
	// ReservedRule is the name of the rule that Cilium adds implicitly (e.g., "allow-localhost-ingress"). It is empty for rules derived from Kubernetes resources.
	ReservedRule string
}

// ExtractPolicyAction extracts PolicyActionFieldSet from a policy-action log.
// Missing fields are left empty because the fields present in a log depend on the kind of the endpoints and the disposition.
func ExtractPolicyAction(reader *structured.NodeReader) (PolicyActionFieldSet, error) {
	if mock, ok := structured.GetMock[PolicyActionFieldSet](reader); ok {
		return mock, nil
	}
	if reader == nil {
		return PolicyActionFieldSet{}, fmt.Errorf("failed to extract the policy action: reader is nil")
	}
	if cached, ok := structured.GetCache(reader, policyActionCacheKey); ok {
		return cached, nil
	}
	result := PolicyActionFieldSet{
		Disposition: reader.ReadStringOrDefault(pathDisposition, ""),
		Count:       reader.ReadIntOrDefault(pathCount, 0),
		NodeName:    reader.ReadStringOrDefault(pathNodeName, ""),
		Direction:   reader.ReadStringOrDefault(pathDirection, ""),
		Protocol:    reader.ReadStringOrDefault(pathProtocol, ""),
		Source:      extractPolicyActionEndpoint(reader, pathSource, pathSourceIP, pathSourcePort),
		Destination: extractPolicyActionEndpoint(reader, pathDestination, pathDestinationIP, pathDestinationPort),
		Policies:    extractPolicyActionPolicies(reader),
	}
	if result.NodeName == "" {
		result.NodeName = reader.ReadStringOrDefault(pathResourceNodeName, "")
	}
	structured.SetCache(reader, policyActionCacheKey, result)
	return result, nil
}

// extractPolicyActionEndpoint reads an endpoint from the object at endpointPath and the IP address and the port in jsonPayload.connection.
func extractPolicyActionEndpoint(reader *structured.NodeReader, endpointPath, ipPath, portPath structured.FieldPath) PolicyActionEndpoint {
	endpoint := PolicyActionEndpoint{
		IP:   reader.ReadStringOrDefault(ipPath, ""),
		Port: reader.ReadIntOrDefault(portPath, 0),
	}
	endpointReader, err := reader.GetReader(endpointPath)
	if err != nil {
		return endpoint
	}
	endpoint.PodName = endpointReader.ReadStringOrDefault(pathPodName, "")
	endpoint.PodNamespace = endpointReader.ReadStringOrDefault(pathPodNamespace, "")
	if endpoint.PodNamespace == "" {
		endpoint.PodNamespace = endpointReader.ReadStringOrDefault(pathNamespace, "")
	}
	endpoint.WorkloadKind = endpointReader.ReadStringOrDefault(pathWorkloadKind, "")
	endpoint.WorkloadName = endpointReader.ReadStringOrDefault(pathWorkloadName, "")
	endpoint.Instance = endpointReader.ReadStringOrDefault(pathInstance, "")
	return endpoint
}

// extractPolicyActionPolicies reads the policy rules in jsonPayload.policies.
func extractPolicyActionPolicies(reader *structured.NodeReader) []PolicyActionPolicy {
	policiesReader, err := reader.GetReader(pathPolicies)
	if err != nil {
		return nil
	}
	var policies []PolicyActionPolicy
	for _, policyReader := range policiesReader.Children() {
		policy := PolicyActionPolicy{
			Kind:      policyReader.ReadStringOrDefault(pathKind, ""),
			Name:      policyReader.ReadStringOrDefault(pathName, ""),
			Namespace: policyReader.ReadStringOrDefault(pathNamespace, ""),
			Revision:  policyReader.ReadStringOrDefault(pathRevision, ""),
		}
		if labelsReader, err := policyReader.GetReader(pathLabels); err == nil {
			for _, labelReader := range labelsReader.Children() {
				label, err := labelReader.ReadString(structured.EmptyFieldPath)
				if err != nil {
					continue
				}
				if uid, found := strings.CutPrefix(label, policyUIDLabelPrefix); found {
					policy.UID = uid
				} else if rule, found := strings.CutPrefix(label, reservedRuleLabelPrefix); found {
					policy.ReservedRule = rule
				}
			}
		}
		policies = append(policies, policy)
	}
	return policies
}
