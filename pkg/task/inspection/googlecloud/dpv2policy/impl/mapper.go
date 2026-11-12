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

package dpv2policy_impl

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/dpv2policy"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// policyResourceType is the API version and the kind that Kubernetes audit log timelines use for a kind of policy resources.
type policyResourceType struct {
	apiVersion string
	kind       string
	namespaced bool
}

// knownPolicyResourceTypes maps the policy kinds recorded in policy-action logs to their resource types.
// It is used when the UID of a policy is not found in the resources recorded by audit logs.
var knownPolicyResourceTypes = map[string]policyResourceType{
	"NetworkPolicy":        {apiVersion: "networking.k8s.io/v1", kind: "networkpolicy", namespaced: true},
	"FQDNNetworkPolicy":    {apiVersion: "networking.gke.io/v1alpha1", kind: "fqdnnetworkpolicy", namespaced: true},
	"ClusterNetworkPolicy": {apiVersion: "policy.networking.k8s.io/v1alpha2", kind: "clusternetworkpolicy", namespaced: false},
}

// processPolicyActionLog sets the log type, timestamp, severity and summary of a network policy log.
func processPolicyActionLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(dpv2policy.LogTypeNetworkPolicy)
	cs.SetTimestamp(l.Timestamp)

	policyAction := dpv2policy.ExtractPolicyAction(l.NodeReader)

	// Policy-action logs have no severity. Denials are surfaced as warnings so that dropped connections stand out on timelines.
	if policyAction.Disposition == dpv2policy.DispositionDeny {
		cs.SetSeverity(inspectioncore.SeverityWarning)
	} else {
		cs.SetSeverity(inspectioncore.SeverityInfo)
	}
	cs.SetSummary(formatPolicyActionSummary(policyAction))

	return cs, nil
}

// formatPolicyActionSummary formats a network policy log like "[DENY x6] ingress TCP default/sleep:53538 -> default/httpbin:8080 (implicit deny)".
func formatPolicyActionSummary(policyAction dpv2policy.PolicyActionFieldSet) string {
	disposition := strings.ToUpper(policyAction.Disposition)
	if disposition == "" {
		disposition = "UNKNOWN"
	}
	if policyAction.Count > 1 {
		disposition = fmt.Sprintf("%s x%d", disposition, policyAction.Count)
	}

	parts := []string{fmt.Sprintf("[%s]", disposition)}
	if policyAction.Direction != "" {
		parts = append(parts, policyAction.Direction)
	}
	if policyAction.Protocol != "" {
		parts = append(parts, strings.ToUpper(policyAction.Protocol))
	}
	parts = append(parts, formatPolicyActionEndpoint(policyAction.Source), "->", formatPolicyActionEndpoint(policyAction.Destination))
	if policies := formatPolicyActionPolicies(policyAction); policies != "" {
		parts = append(parts, fmt.Sprintf("(%s)", policies))
	}
	return strings.Join(parts, " ")
}

// formatPolicyActionEndpoint formats an endpoint as "<namespace>/<pod>:<port>" for a Pod or "<ip>:<port>" for the other endpoints.
func formatPolicyActionEndpoint(endpoint dpv2policy.PolicyActionEndpoint) string {
	host := endpoint.IP
	switch {
	case endpoint.IsPod():
		host = endpoint.PodNamespace + "/" + endpoint.PodName
	case endpoint.Instance != "":
		host = endpoint.Instance
	}
	if host == "" {
		host = "unknown"
	}
	if endpoint.Port == 0 {
		return host
	}
	return net.JoinHostPort(host, strconv.Itoa(endpoint.Port))
}

// formatPolicyActionPolicies formats the policy rules that decided the connection.
func formatPolicyActionPolicies(policyAction dpv2policy.PolicyActionFieldSet) string {
	if len(policyAction.Policies) == 0 {
		// Cilium omits the policies when a connection to an isolated endpoint matches no allow rule.
		if policyAction.Disposition == dpv2policy.DispositionDeny {
			return "implicit deny"
		}
		return ""
	}
	names := make([]string, 0, len(policyAction.Policies))
	for _, policy := range policyAction.Policies {
		// Cilium can list the same rule more than once in a log, and repeating it adds nothing to the summary.
		if name := formatPolicyActionPolicy(policy); name != "" && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "by " + strings.Join(names, ", ")
}

// formatPolicyActionPolicy formats a policy rule as "<kind> <namespace>/<name>", "<kind> <name>" for a cluster-scoped resource, or "reserved:<rule>".
func formatPolicyActionPolicy(policy dpv2policy.PolicyActionPolicy) string {
	switch {
	case policy.Name != "":
		name := policy.Name
		if policy.Namespace != "" {
			name = policy.Namespace + "/" + policy.Name
		}
		if policy.Kind == "" {
			return name
		}
		return policy.Kind + " " + name
	case policy.ReservedRule != "":
		return "reserved:" + policy.ReservedRule
	default:
		return ""
	}
}

// logIngesterTask ingests the metadata of network policy logs.
var logIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	dpv2policy.LogIngesterTaskID,
	dpv2policy.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		return processPolicyActionLog
	},
)

// groupPolicyActionLog returns the name of the node that reported the log so that the logs of each node are mapped in parallel.
func groupPolicyActionLog(ctx context.Context, l *log.Log) string {
	policyAction := dpv2policy.ExtractPolicyAction(l.NodeReader)
	if policyAction.NodeName == "" {
		return "unknown"
	}
	return policyAction.NodeName
}

// logGrouperTask groups network policy logs by their reporting node.
var logGrouperTask = inspectiontaskbase.DefineLogGrouperTask(
	dpv2policy.LogGrouperTaskID,
	dpv2policy.ListLogEntriesTaskID.Ref(),
	func(b *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
		return groupPolicyActionLog
	},
)

// policyActionTimelineMapper maps network policy logs to the timelines of Pods, network policies and Nodes.
type policyActionTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	resourceUIDs    coretask.Input[k8saudit.UIDToResourceIdentity]
}

// ProcessLogByGroup implements inspectiontaskbase.TimelineMapper.
func (m *policyActionTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	clusterIdentity := m.clusterIdentity.Get(ctx)
	cs := mapPolicyActionLog(ctx, l, clusterIdentity.NameFor(k8scommon.ClusterNameUsageK8sCluster), m.resourceUIDs.Get(ctx))
	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*policyActionTimelineMapper)(nil)

// mapPolicyActionLog adds events for a network policy log to the timelines of its source and destination Pods and of the network policies that decided the connection.
// It adds the event to the timeline of the reporting node instead when neither endpoint is a Pod.
func mapPolicyActionLog(ctx context.Context, l *log.Log, clusterName string, resourceUIDs k8saudit.UIDToResourceIdentity) *khifilev6.TimelineChangeSet {
	policyAction := dpv2policy.ExtractPolicyAction(l.NodeReader)

	cs := khifilev6.NewTimelineChangeSet(l)

	// Remote Pods have no workload fields, so Pods are identified only by their namespace and name.
	if policyAction.Source.IsPod() {
		cs.AddEvent(dpv2policy.MustNetworkPolicyTimeline(ctx, mustPodTimeline(ctx, clusterName, policyAction.Source), dpv2policy.PodOutboundTimelineName))
	}
	if policyAction.Destination.IsPod() {
		cs.AddEvent(dpv2policy.MustNetworkPolicyTimeline(ctx, mustPodTimeline(ctx, clusterName, policyAction.Destination), dpv2policy.PodInboundTimelineName))
	}
	for _, policy := range policyAction.Policies {
		if resource, found := resolvePolicyResource(policy, resourceUIDs); found {
			cs.AddEvent(k8saudit.MustResourceTimeline(ctx, clusterName, resource))
		}
	}
	if !policyAction.Source.IsPod() && !policyAction.Destination.IsPod() && policyAction.NodeName != "" {
		cs.AddEvent(dpv2policy.MustNetworkPolicyTimeline(ctx, mustNodeTimeline(ctx, clusterName, policyAction.NodeName), dpv2policy.NodeTimelineName))
	}

	return cs
}

// resolvePolicyResource returns the identity of the Kubernetes resource defining the policy rule.
// It prefers the resource recorded with the same UID in audit logs because its API version is the one that the cluster actually serves.
func resolvePolicyResource(policy dpv2policy.PolicyActionPolicy, resourceUIDs k8saudit.UIDToResourceIdentity) (*k8saudit.ResourceIdentity, bool) {
	// Reserved rules have no name because Cilium adds them implicitly without any Kubernetes resource.
	if policy.Name == "" {
		return nil, false
	}
	if policy.UID != "" {
		if resource, found := resourceUIDs[policy.UID]; found && resource.Type() == k8saudit.Resource {
			return resource, true
		}
	}
	resourceType, found := knownPolicyResourceTypes[policy.Kind]
	if !found {
		return nil, false
	}
	namespace := k8saudit.ClusterScopeNamespace
	if resourceType.namespaced {
		if policy.Namespace == "" {
			return nil, false
		}
		namespace = policy.Namespace
	}
	return &k8saudit.ResourceIdentity{
		APIVersion: resourceType.apiVersion,
		Kind:       resourceType.kind,
		Namespace:  namespace,
		Name:       policy.Name,
	}, true
}

// mustPodTimeline returns the resource timeline of the Pod endpoint.
func mustPodTimeline(ctx context.Context, clusterName string, endpoint dpv2policy.PolicyActionEndpoint) *khifilev6.TimelinePath {
	return k8saudit.MustResourceTimeline(ctx, clusterName, &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Namespace:  endpoint.PodNamespace,
		Name:       endpoint.PodName,
	})
}

// mustNodeTimeline returns the resource timeline of the node.
func mustNodeTimeline(ctx context.Context, clusterName string, nodeName string) *khifilev6.TimelinePath {
	return k8saudit.MustResourceTimeline(ctx, clusterName, &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "node",
		Namespace:  k8saudit.ClusterScopeNamespace,
		Name:       nodeName,
	})
}

// logToTimelineMapperTask maps network policy logs to timelines.
var logToTimelineMapperTask = inspectiontaskbase.DefineLogToTimelineMapperTask(
	dpv2policy.LogToTimelineMapperTaskID,
	inspectiontaskbase.TimelineMapperInputs{
		LogIngester: dpv2policy.LogIngesterTaskID.Ref(),
		GroupedLogs: dpv2policy.LogGrouperTaskID.Ref(),
	},
	func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
		return &policyActionTimelineMapper{
			clusterIdentity: coretask.Use(b, dpv2policy.ClusterIdentityTaskID.Ref()),
			resourceUIDs:    coretask.Use(b, k8saudit.ResourceUIDInventoryTaskID.Ref()),
		}
	},
	inspectioncore.FeatureTaskLabel(
		"Dataplane V2 Network Policy Logs",
		"Gather GKE Dataplane V2 network policy logs (policy-action) to visualize allowed and denied connections on the timelines of the source and destination Pods and the network policies that decided them. Note: Network policy logging must be enabled on the cluster, and the log volume can be large in clusters with heavy traffic.",
		3500,
		false,
	),
)
