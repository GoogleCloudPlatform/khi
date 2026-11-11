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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/dpv2policy"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

var (
	testLogTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// crossNodeEgressAllowed is a connection allowed by an egress NetworkPolicy towards a Pod on a remote node.
	crossNodeEgressAllowed = dpv2policy.PolicyActionFieldSet{
		Disposition: dpv2policy.DispositionAllow,
		Count:       1,
		NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
		Direction:   "egress",
		Protocol:    "tcp",
		Source: dpv2policy.PolicyActionEndpoint{
			IP:           "10.0.0.10",
			Port:         41952,
			PodNamespace: "bookinfo",
			PodName:      "productpage-v1-123456789a-abcde",
			WorkloadKind: "Deployment",
			WorkloadName: "productpage-v1",
		},
		Destination: dpv2policy.PolicyActionEndpoint{
			IP:           "10.0.1.20",
			Port:         9080,
			PodNamespace: "bookinfo",
			PodName:      "reviews-v1-123456789b-fghij",
		},
		Policies: []dpv2policy.PolicyActionPolicy{
			{Kind: "NetworkPolicy", Name: "test-egress-policy", Namespace: "bookinfo", Revision: "9", UID: "00000000-0000-0000-0000-000000000001"},
		},
	}

	// implicitDenial is a connection denied without any matching allow rule, aggregated in a 5-second window.
	implicitDenial = dpv2policy.PolicyActionFieldSet{
		Disposition: dpv2policy.DispositionDeny,
		Count:       6,
		NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
		Direction:   "ingress",
		Protocol:    "tcp",
		Source:      dpv2policy.PolicyActionEndpoint{IP: "10.0.0.23", Port: 53538, PodNamespace: "default", PodName: "sleep-abcde"},
		Destination: dpv2policy.PolicyActionEndpoint{IP: "10.0.0.20", Port: 8080, PodNamespace: "default", PodName: "httpbin-0"},
	}

	// clusterNetworkPolicyDenial is a connection denied by an Admin tier ClusterNetworkPolicy.
	clusterNetworkPolicyDenial = dpv2policy.PolicyActionFieldSet{
		Disposition: dpv2policy.DispositionDeny,
		Count:       2,
		NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
		Direction:   "ingress",
		Protocol:    "tcp",
		Source:      dpv2policy.PolicyActionEndpoint{IP: "10.0.0.26", Port: 56434, PodNamespace: "default", PodName: "sleep-123456789d-pqrst"},
		Destination: dpv2policy.PolicyActionEndpoint{IP: "10.0.0.24", Port: 9999, PodNamespace: "default", PodName: "httpbin-123456789c-klmno"},
		Policies: []dpv2policy.PolicyActionPolicy{
			{Kind: "ClusterNetworkPolicy", Name: "test-cluster-policy", Revision: "9", UID: "00000000-0000-0000-0000-000000000003"},
			{Revision: "9", ReservedRule: "deny-any-ingress"},
		},
	}

	// fqdnEgressAllowed is a connection to an external IP address allowed by an FQDNNetworkPolicy.
	fqdnEgressAllowed = dpv2policy.PolicyActionFieldSet{
		Disposition: dpv2policy.DispositionAllow,
		Count:       1,
		NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
		Direction:   "egress",
		Protocol:    "tcp",
		Source:      dpv2policy.PolicyActionEndpoint{IP: "10.0.0.31", Port: 47030, PodNamespace: "default", PodName: "sleep-job-abcde"},
		Destination: dpv2policy.PolicyActionEndpoint{IP: "203.0.113.10", Port: 443, Instance: "203.0.113.10"},
		Policies: []dpv2policy.PolicyActionPolicy{
			{Kind: "FQDNNetworkPolicy", Name: "test-fqdn-policy", Namespace: "default", Revision: "8", UID: "00000000-0000-0000-0000-000000000004"},
		},
	}

	// kubeletProbeAllowed is a health probe from the kubelet allowed by reserved rules.
	kubeletProbeAllowed = dpv2policy.PolicyActionFieldSet{
		Disposition: dpv2policy.DispositionAllow,
		Count:       1,
		NodeName:    "gke-test-cluster-default-pool-12345678-efgh",
		Direction:   "ingress",
		Protocol:    "tcp",
		Source:      dpv2policy.PolicyActionEndpoint{IP: "169.254.4.6", Port: 43102, Instance: "169.254.4.6"},
		Destination: dpv2policy.PolicyActionEndpoint{IP: "10.0.1.11", Port: 8080, PodNamespace: "default", PodName: "httpbin-123456789c-klmno"},
		Policies: []dpv2policy.PolicyActionPolicy{
			{Revision: "9", ReservedRule: "allow-localhost-ingress"},
			{Revision: "9", ReservedRule: "deny-any-ingress"},
		},
	}
)

func TestProcessPolicyActionLog(t *testing.T) {
	testCases := []struct {
		desc         string
		input        dpv2policy.PolicyActionFieldSet
		wantSummary  string
		wantSeverity *pb.Severity
	}{
		{
			desc:         "egress allowed by a NetworkPolicy to a Pod on a remote node",
			input:        crossNodeEgressAllowed,
			wantSummary:  "[ALLOW] egress TCP bookinfo/productpage-v1-123456789a-abcde:41952 -> bookinfo/reviews-v1-123456789b-fghij:9080 (by NetworkPolicy bookinfo/test-egress-policy)",
			wantSeverity: inspectioncore.SeverityInfo,
		},
		{
			desc:         "aggregated implicit denial",
			input:        implicitDenial,
			wantSummary:  "[DENY x6] ingress TCP default/sleep-abcde:53538 -> default/httpbin-0:8080 (implicit deny)",
			wantSeverity: inspectioncore.SeverityWarning,
		},
		{
			desc:         "denial by a ClusterNetworkPolicy with a reserved rule",
			input:        clusterNetworkPolicyDenial,
			wantSummary:  "[DENY x2] ingress TCP default/sleep-123456789d-pqrst:56434 -> default/httpbin-123456789c-klmno:9999 (by ClusterNetworkPolicy test-cluster-policy, reserved:deny-any-ingress)",
			wantSeverity: inspectioncore.SeverityWarning,
		},
		{
			desc:         "egress to an external IP address allowed by an FQDNNetworkPolicy",
			input:        fqdnEgressAllowed,
			wantSummary:  "[ALLOW] egress TCP default/sleep-job-abcde:47030 -> 203.0.113.10:443 (by FQDNNetworkPolicy default/test-fqdn-policy)",
			wantSeverity: inspectioncore.SeverityInfo,
		},
		{
			desc:         "kubelet health probe allowed by reserved rules",
			input:        kubeletProbeAllowed,
			wantSummary:  "[ALLOW] ingress TCP 169.254.4.6:43102 -> default/httpbin-123456789c-klmno:8080 (by reserved:allow-localhost-ingress, reserved:deny-any-ingress)",
			wantSeverity: inspectioncore.SeverityInfo,
		},
		{
			desc: "denial listing the same reserved rule twice",
			input: dpv2policy.PolicyActionFieldSet{
				Disposition: dpv2policy.DispositionDeny,
				Count:       4,
				Direction:   "ingress",
				Protocol:    "tcp",
				Source:      dpv2policy.PolicyActionEndpoint{IP: "10.0.0.27", Port: 41912, PodNamespace: "default", PodName: "sleep-abcde"},
				Destination: dpv2policy.PolicyActionEndpoint{IP: "10.0.1.11", Port: 8080, PodNamespace: "default", PodName: "httpbin-123456789c-klmno"},
				Policies: []dpv2policy.PolicyActionPolicy{
					{Revision: "9", ReservedRule: "deny-any-ingress"},
					{Revision: "9", ReservedRule: "deny-any-ingress"},
				},
			},
			wantSummary:  "[DENY x4] ingress TCP default/sleep-abcde:41912 -> default/httpbin-123456789c-klmno:8080 (by reserved:deny-any-ingress)",
			wantSeverity: inspectioncore.SeverityWarning,
		},
		{
			desc: "IPv6 destination with a port",
			input: dpv2policy.PolicyActionFieldSet{
				Disposition: dpv2policy.DispositionAllow,
				Count:       1,
				Direction:   "egress",
				Protocol:    "tcp",
				Source:      dpv2policy.PolicyActionEndpoint{IP: "2001:db8::10", Port: 41000, PodNamespace: "default", PodName: "sleep-abcde"},
				Destination: dpv2policy.PolicyActionEndpoint{IP: "2001:db8::1", Port: 443, Instance: "2001:db8::1"},
			},
			wantSummary:  "[ALLOW] egress TCP default/sleep-abcde:41000 -> [2001:db8::1]:443",
			wantSeverity: inspectioncore.SeverityInfo,
		},
		{
			desc: "ICMP without ports",
			input: dpv2policy.PolicyActionFieldSet{
				Disposition: dpv2policy.DispositionDeny,
				Count:       1,
				Direction:   "egress",
				Protocol:    "icmp",
				Source:      dpv2policy.PolicyActionEndpoint{IP: "10.0.0.12", PodNamespace: "default", PodName: "sleep-abcde"},
				Destination: dpv2policy.PolicyActionEndpoint{IP: "169.254.20.10", Instance: "169.254.20.10"},
			},
			wantSummary:  "[DENY] egress ICMP default/sleep-abcde -> 169.254.20.10 (implicit deny)",
			wantSeverity: inspectioncore.SeverityWarning,
		},
		{
			desc:         "log without the policy action payload",
			input:        dpv2policy.PolicyActionFieldSet{},
			wantSummary:  "[UNKNOWN] unknown -> unknown",
			wantSeverity: inspectioncore.SeverityInfo,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := testlog.NewMockLog(testLogTime, tc.input)

			cs, err := processPolicyActionLog(t.Context(), l)
			if err != nil {
				t.Fatalf("processPolicyActionLog() returned an unexpected error: %v", err)
			}

			testchangeset.AssertLog(t, cs).
				HasSummary(tc.wantSummary).
				HasSeverity(tc.wantSeverity).
				HasLogType(dpv2policy.LogTypeNetworkPolicy).
				HasTimestamp(testLogTime)
		})
	}
}

func TestGroupPolicyActionLog(t *testing.T) {
	testCases := []struct {
		desc  string
		input dpv2policy.PolicyActionFieldSet
		want  string
	}{
		{
			desc:  "log with a node name",
			input: kubeletProbeAllowed,
			want:  "gke-test-cluster-default-pool-12345678-efgh",
		},
		{
			desc:  "log without a node name",
			input: dpv2policy.PolicyActionFieldSet{Disposition: dpv2policy.DispositionDeny},
			want:  "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := groupPolicyActionLog(t.Context(), testlog.NewMockLog(testLogTime, tc.input))
			if got != tc.want {
				t.Errorf("groupPolicyActionLog() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMapPolicyActionLog(t *testing.T) {
	testCases := []struct {
		desc         string
		input        dpv2policy.PolicyActionFieldSet
		resourceUIDs k8saudit.UIDToResourceIdentity
		assert       func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc:  "NetworkPolicy links the source Pod, the remote destination Pod and the policy",
			input: crossNodeEgressAllowed,
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(podNetworkPolicyPath(builder, "bookinfo", "productpage-v1-123456789a-abcde", "outbound")).
					HasEvent(podNetworkPolicyPath(builder, "bookinfo", "reviews-v1-123456789b-fghij", "inbound")).
					HasEvent(resourcePath(builder, "networking.k8s.io/v1", "networkpolicy", "bookinfo", "test-egress-policy")).
					HasEventCount(3)
			},
		},
		{
			desc:  "implicit denial links only the Pods",
			input: implicitDenial,
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(podNetworkPolicyPath(builder, "default", "sleep-abcde", "outbound")).
					HasEvent(podNetworkPolicyPath(builder, "default", "httpbin-0", "inbound")).
					HasEventCount(2)
			},
		},
		{
			desc:  "ClusterNetworkPolicy links the cluster-scoped policy and skips the reserved rule",
			input: clusterNetworkPolicyDenial,
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(podNetworkPolicyPath(builder, "default", "sleep-123456789d-pqrst", "outbound")).
					HasEvent(podNetworkPolicyPath(builder, "default", "httpbin-123456789c-klmno", "inbound")).
					HasEvent(resourcePath(builder, "policy.networking.k8s.io/v1alpha2", "clusternetworkpolicy", "cluster-scope", "test-cluster-policy")).
					HasEventCount(3)
			},
		},
		{
			desc:  "FQDNNetworkPolicy links the source Pod and the policy but not the external destination",
			input: fqdnEgressAllowed,
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(podNetworkPolicyPath(builder, "default", "sleep-job-abcde", "outbound")).
					HasEvent(resourcePath(builder, "networking.gke.io/v1alpha1", "fqdnnetworkpolicy", "default", "test-fqdn-policy")).
					HasEventCount(2)
			},
		},
		{
			desc:  "reserved rules link only the destination Pod",
			input: kubeletProbeAllowed,
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(podNetworkPolicyPath(builder, "default", "httpbin-123456789c-klmno", "inbound")).
					HasEventCount(1)
			},
		},
		{
			desc:  "policy found by UID in audit logs uses the API version recorded in audit logs",
			input: fqdnEgressAllowed,
			resourceUIDs: k8saudit.UIDToResourceIdentity{
				"00000000-0000-0000-0000-000000000004": {
					APIVersion: "networking.gke.io/v1beta1",
					Kind:       "fqdnnetworkpolicy",
					Namespace:  "default",
					Name:       "test-fqdn-policy",
				},
			},
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(podNetworkPolicyPath(builder, "default", "sleep-job-abcde", "outbound")).
					HasEvent(resourcePath(builder, "networking.gke.io/v1beta1", "fqdnnetworkpolicy", "default", "test-fqdn-policy")).
					HasNoEvent(resourcePath(builder, "networking.gke.io/v1alpha1", "fqdnnetworkpolicy", "default", "test-fqdn-policy")).
					HasEventCount(2)
			},
		},
		{
			desc: "unknown policy kind and namespaced policy without namespace are not linked",
			input: dpv2policy.PolicyActionFieldSet{
				Disposition: dpv2policy.DispositionAllow,
				NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
				Source:      dpv2policy.PolicyActionEndpoint{PodNamespace: "default", PodName: "sleep-abcde"},
				Destination: dpv2policy.PolicyActionEndpoint{PodNamespace: "default", PodName: "httpbin-0"},
				Policies: []dpv2policy.PolicyActionPolicy{
					{Kind: "UnknownPolicy", Name: "test-unknown-policy", Namespace: "default"},
					{Kind: "NetworkPolicy", Name: "test-policy-without-namespace"},
				},
			},
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(podNetworkPolicyPath(builder, "default", "sleep-abcde", "outbound")).
					HasEvent(podNetworkPolicyPath(builder, "default", "httpbin-0", "inbound")).
					HasEventCount(2)
			},
		},
		{
			desc: "connection between endpoints that are not Pods falls back to the reporting node",
			input: dpv2policy.PolicyActionFieldSet{
				Disposition: dpv2policy.DispositionDeny,
				NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
				Source:      dpv2policy.PolicyActionEndpoint{IP: "10.0.0.2", Instance: "10.0.0.2"},
				Destination: dpv2policy.PolicyActionEndpoint{IP: "203.0.113.20", Port: 443, Instance: "203.0.113.20"},
			},
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(nodeNetworkPolicyPath(builder, "gke-test-cluster-default-pool-12345678-abcd")).
					HasEventCount(1)
			},
		},
		{
			desc: "connection between endpoints that are not Pods without the node name is not linked",
			input: dpv2policy.PolicyActionFieldSet{
				Disposition: dpv2policy.DispositionDeny,
				Source:      dpv2policy.PolicyActionEndpoint{IP: "10.0.0.2", Instance: "10.0.0.2"},
				Destination: dpv2policy.PolicyActionEndpoint{IP: "203.0.113.20", Port: 443, Instance: "203.0.113.20"},
			},
			assert: func(t *testing.T, builder *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEventCount(0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			builder := khifilev6.NewTestBuilder(id.NewGenerator())
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

			cs, err := mapPolicyActionLog(ctx, testlog.NewMockLog(testLogTime, tc.input), "test-cluster", tc.resourceUIDs)
			if err != nil {
				t.Fatalf("mapPolicyActionLog() returned an unexpected error: %v", err)
			}
			tc.assert(t, builder, cs)
		})
	}
}

// resourcePath returns the expected timeline path of a Kubernetes resource in the test cluster.
func resourcePath(builder *khifilev6.Builder, apiVersion, kind, namespace, name string) *khifilev6.TimelinePath {
	return builder.TimelineAccumulator.GetPath(nil,
		khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
		khifilev6.PathSegment{Name: apiVersion, Type: inspectioncore.TimelineTypeAPIVersion},
		khifilev6.PathSegment{Name: kind, Type: inspectioncore.TimelineTypeKind},
		khifilev6.PathSegment{Name: namespace, Type: inspectioncore.TimelineTypeNamespace},
		khifilev6.PathSegment{Name: name, Type: inspectioncore.TimelineTypeResource},
	)
}

// podNetworkPolicyPath returns the expected network policy timeline path under a Pod.
func podNetworkPolicyPath(builder *khifilev6.Builder, namespace, podName, timelineName string) *khifilev6.TimelinePath {
	return builder.TimelineAccumulator.GetPath(resourcePath(builder, "core/v1", "pod", namespace, podName),
		khifilev6.PathSegment{Name: timelineName, Type: dpv2policy.TimelineTypeNetworkPolicy},
	)
}

// nodeNetworkPolicyPath returns the expected network policy timeline path under a Node.
func nodeNetworkPolicyPath(builder *khifilev6.Builder, nodeName string) *khifilev6.TimelinePath {
	return builder.TimelineAccumulator.GetPath(resourcePath(builder, "core/v1", "node", "cluster-scope", nodeName),
		khifilev6.PathSegment{Name: "policy-action", Type: dpv2policy.TimelineTypeNetworkPolicy},
	)
}
