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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func TestExtractPolicyAction(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  PolicyActionFieldSet
	}{
		{
			desc: "egress allowed by a NetworkPolicy to a Pod on a remote node",
			input: `
logName: projects/test-project/logs/policy-action
resource:
  type: k8s_node
  labels:
    cluster_name: test-cluster
    location: us-central1
    node_name: gke-test-cluster-default-pool-12345678-abcd
    project_id: test-project
jsonPayload:
  connection:
    dest_ip: 10.0.1.20
    dest_port: 9080
    direction: egress
    protocol: tcp
    src_ip: 10.0.0.10
    src_port: 41952
  count: 1
  dest:
    namespace: bookinfo
    pod_name: reviews-v1-123456789b-fghij
    pod_namespace: bookinfo
  disposition: allow
  node_name: gke-test-cluster-default-pool-12345678-abcd
  policies:
  - kind: NetworkPolicy
    labels:
    - "k8s:io.cilium.k8s.policy.derived-from=NetworkPolicy"
    - "k8s:io.cilium.k8s.policy.name=test-egress-policy"
    - "k8s:io.cilium.k8s.policy.namespace=bookinfo"
    - "k8s:io.cilium.k8s.policy.uid=00000000-0000-0000-0000-000000000001"
    name: test-egress-policy
    namespace: bookinfo
    revision: "9"
  src:
    namespace: bookinfo
    pod_name: productpage-v1-123456789a-abcde
    pod_namespace: bookinfo
    workload_kind: Deployment
    workload_name: productpage-v1
`,
			want: PolicyActionFieldSet{
				Disposition: DispositionAllow,
				Count:       1,
				NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
				Direction:   "egress",
				Protocol:    "tcp",
				Source: PolicyActionEndpoint{
					IP:           "10.0.0.10",
					Port:         41952,
					PodNamespace: "bookinfo",
					PodName:      "productpage-v1-123456789a-abcde",
					WorkloadKind: "Deployment",
					WorkloadName: "productpage-v1",
				},
				Destination: PolicyActionEndpoint{
					IP:           "10.0.1.20",
					Port:         9080,
					PodNamespace: "bookinfo",
					PodName:      "reviews-v1-123456789b-fghij",
				},
				Policies: []PolicyActionPolicy{
					{
						Kind:      "NetworkPolicy",
						Name:      "test-egress-policy",
						Namespace: "bookinfo",
						Revision:  "9",
						UID:       "00000000-0000-0000-0000-000000000001",
					},
				},
			},
		},
		{
			desc: "aggregated implicit denial without policies",
			input: `
jsonPayload:
  connection:
    dest_ip: 10.0.0.20
    dest_port: 8080
    direction: ingress
    protocol: tcp
    src_ip: 10.0.0.23
    src_port: 53538
  count: 6
  dest:
    namespace: default
    pod_name: httpbin-0
    pod_namespace: default
    workload_kind: StatefulSet
    workload_name: httpbin
  disposition: deny
  node_name: gke-test-cluster-default-pool-12345678-abcd
  src:
    namespace: default
    pod_name: sleep-abcde
    pod_namespace: default
    workload_kind: DaemonSet
    workload_name: sleep
`,
			want: PolicyActionFieldSet{
				Disposition: DispositionDeny,
				Count:       6,
				NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
				Direction:   "ingress",
				Protocol:    "tcp",
				Source: PolicyActionEndpoint{
					IP:           "10.0.0.23",
					Port:         53538,
					PodNamespace: "default",
					PodName:      "sleep-abcde",
					WorkloadKind: "DaemonSet",
					WorkloadName: "sleep",
				},
				Destination: PolicyActionEndpoint{
					IP:           "10.0.0.20",
					Port:         8080,
					PodNamespace: "default",
					PodName:      "httpbin-0",
					WorkloadKind: "StatefulSet",
					WorkloadName: "httpbin",
				},
			},
		},
		{
			desc: "denial by a ClusterNetworkPolicy with a reserved rule",
			input: `
jsonPayload:
  connection:
    dest_ip: 10.0.0.24
    dest_port: 9999
    direction: ingress
    protocol: tcp
    src_ip: 10.0.0.26
    src_port: 56434
  count: 2
  dest:
    namespace: default
    pod_name: httpbin-123456789c-klmno
    pod_namespace: default
    workload_kind: Deployment
    workload_name: httpbin
  disposition: deny
  node_name: gke-test-cluster-default-pool-12345678-abcd
  policies:
  - kind: ClusterNetworkPolicy
    labels:
    - "k8s:io.cilium.k8s.policy.derived-from=ClusterNetworkPolicy"
    - "k8s:io.cilium.k8s.policy.name=test-cluster-policy"
    - "k8s:io.cilium.k8s.policy.uid=00000000-0000-0000-0000-000000000003"
    name: test-cluster-policy
    revision: "9"
  - labels:
    - "reserved:io.cilium.policy.derived-from=deny-any-ingress"
    revision: "9"
  src:
    namespace: default
    pod_name: sleep-123456789d-pqrst
    pod_namespace: default
    workload_kind: Deployment
    workload_name: sleep
`,
			want: PolicyActionFieldSet{
				Disposition: DispositionDeny,
				Count:       2,
				NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
				Direction:   "ingress",
				Protocol:    "tcp",
				Source: PolicyActionEndpoint{
					IP:           "10.0.0.26",
					Port:         56434,
					PodNamespace: "default",
					PodName:      "sleep-123456789d-pqrst",
					WorkloadKind: "Deployment",
					WorkloadName: "sleep",
				},
				Destination: PolicyActionEndpoint{
					IP:           "10.0.0.24",
					Port:         9999,
					PodNamespace: "default",
					PodName:      "httpbin-123456789c-klmno",
					WorkloadKind: "Deployment",
					WorkloadName: "httpbin",
				},
				Policies: []PolicyActionPolicy{
					{
						Kind:     "ClusterNetworkPolicy",
						Name:     "test-cluster-policy",
						Revision: "9",
						UID:      "00000000-0000-0000-0000-000000000003",
					},
					{
						Revision:     "9",
						ReservedRule: "deny-any-ingress",
					},
				},
			},
		},
		{
			desc: "egress to an external IP address allowed by an FQDNNetworkPolicy",
			input: `
jsonPayload:
  connection:
    dest_ip: 203.0.113.10
    dest_port: 443
    direction: egress
    protocol: tcp
    src_ip: 10.0.0.31
    src_port: 47030
  count: 1
  dest:
    instance: 203.0.113.10
  disposition: allow
  node_name: gke-test-cluster-default-pool-12345678-abcd
  policies:
  - kind: FQDNNetworkPolicy
    labels:
    - "k8s:io.cilium.k8s.policy.derived-from=FQDNNetworkPolicy"
    - "k8s:io.cilium.k8s.policy.name=test-fqdn-policy"
    - "k8s:io.cilium.k8s.policy.namespace=default"
    - "k8s:io.cilium.k8s.policy.uid=00000000-0000-0000-0000-000000000004"
    name: test-fqdn-policy
    namespace: default
    revision: "8"
  src:
    namespace: default
    pod_name: sleep-job-abcde
    pod_namespace: default
    workload_kind: Job
    workload_name: sleep-job
`,
			want: PolicyActionFieldSet{
				Disposition: DispositionAllow,
				Count:       1,
				NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
				Direction:   "egress",
				Protocol:    "tcp",
				Source: PolicyActionEndpoint{
					IP:           "10.0.0.31",
					Port:         47030,
					PodNamespace: "default",
					PodName:      "sleep-job-abcde",
					WorkloadKind: "Job",
					WorkloadName: "sleep-job",
				},
				Destination: PolicyActionEndpoint{
					IP:       "203.0.113.10",
					Port:     443,
					Instance: "203.0.113.10",
				},
				Policies: []PolicyActionPolicy{
					{
						Kind:      "FQDNNetworkPolicy",
						Name:      "test-fqdn-policy",
						Namespace: "default",
						Revision:  "8",
						UID:       "00000000-0000-0000-0000-000000000004",
					},
				},
			},
		},
		{
			desc: "kubelet health probe allowed by reserved rules",
			input: `
jsonPayload:
  connection:
    dest_ip: 10.0.1.11
    dest_port: 8080
    direction: ingress
    protocol: tcp
    src_ip: 169.254.4.6
    src_port: 43102
  count: 1
  dest:
    namespace: default
    pod_name: httpbin-123456789c-klmno
    pod_namespace: default
    workload_kind: Deployment
    workload_name: httpbin
  disposition: allow
  node_name: gke-test-cluster-default-pool-12345678-efgh
  policies:
  - labels:
    - "reserved:io.cilium.policy.derived-from=allow-localhost-ingress"
    revision: "9"
  - labels:
    - "reserved:io.cilium.policy.derived-from=deny-any-ingress"
    revision: "9"
  src:
    instance: 169.254.4.6
`,
			want: PolicyActionFieldSet{
				Disposition: DispositionAllow,
				Count:       1,
				NodeName:    "gke-test-cluster-default-pool-12345678-efgh",
				Direction:   "ingress",
				Protocol:    "tcp",
				Source: PolicyActionEndpoint{
					IP:       "169.254.4.6",
					Port:     43102,
					Instance: "169.254.4.6",
				},
				Destination: PolicyActionEndpoint{
					IP:           "10.0.1.11",
					Port:         8080,
					PodNamespace: "default",
					PodName:      "httpbin-123456789c-klmno",
					WorkloadKind: "Deployment",
					WorkloadName: "httpbin",
				},
				Policies: []PolicyActionPolicy{
					{
						Revision:     "9",
						ReservedRule: "allow-localhost-ingress",
					},
					{
						Revision:     "9",
						ReservedRule: "deny-any-ingress",
					},
				},
			},
		},
		{
			desc: "ICMP without ports and the node name only in the resource labels",
			input: `
resource:
  type: k8s_node
  labels:
    node_name: gke-test-cluster-default-pool-12345678-abcd
jsonPayload:
  connection:
    dest_ip: 169.254.20.10
    direction: egress
    protocol: icmp
    src_ip: 10.0.0.12
  dest:
    instance: 169.254.20.10
  disposition: deny
  src:
    namespace: default
    pod_name: sleep-abcde
`,
			want: PolicyActionFieldSet{
				Disposition: DispositionDeny,
				NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
				Direction:   "egress",
				Protocol:    "icmp",
				Source: PolicyActionEndpoint{
					IP:           "10.0.0.12",
					PodNamespace: "default",
					PodName:      "sleep-abcde",
				},
				Destination: PolicyActionEndpoint{
					IP:       "169.254.20.10",
					Instance: "169.254.20.10",
				},
			},
		},
		{
			desc:  "log without the policy action payload",
			input: `logName: projects/test-project/logs/policy-action`,
			want:  PolicyActionFieldSet{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to parse YAML test input: %v", err)
			}
			reader := structured.NewNodeReader(node)

			got, err := ExtractPolicyAction(reader)
			if err != nil {
				t.Fatalf("ExtractPolicyAction() returned an unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractPolicyAction() mismatch (-want +got):\n%s", diff)
			}

			cached, err := ExtractPolicyAction(reader)
			if err != nil {
				t.Fatalf("ExtractPolicyAction() returned an unexpected error on the cached reader: %v", err)
			}
			if diff := cmp.Diff(tc.want, cached); diff != "" {
				t.Errorf("ExtractPolicyAction() on the cached reader mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractPolicyAction_Mock(t *testing.T) {
	want := PolicyActionFieldSet{
		Disposition: DispositionDeny,
		Count:       3,
		NodeName:    "gke-test-cluster-default-pool-12345678-abcd",
	}
	reader := structured.NewNodeReader(structured.NewMockNode(want))

	got, err := ExtractPolicyAction(reader)
	if err != nil {
		t.Fatalf("ExtractPolicyAction() returned an unexpected error: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ExtractPolicyAction() mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractPolicyAction_NilReader(t *testing.T) {
	if _, err := ExtractPolicyAction(nil); err == nil {
		t.Errorf("ExtractPolicyAction(nil) returned no error, want an error")
	}
}

func TestPolicyActionEndpoint_IsPod(t *testing.T) {
	testCases := []struct {
		desc     string
		endpoint PolicyActionEndpoint
		want     bool
	}{
		{
			desc:     "Pod on a remote node without workload fields",
			endpoint: PolicyActionEndpoint{PodNamespace: "default", PodName: "httpbin-0"},
			want:     true,
		},
		{
			desc:     "endpoint that is not a Pod",
			endpoint: PolicyActionEndpoint{IP: "169.254.4.6", Instance: "169.254.4.6"},
			want:     false,
		},
		{
			desc:     "Pod name without namespace",
			endpoint: PolicyActionEndpoint{PodName: "httpbin-0"},
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if got := tc.endpoint.IsPod(); got != tc.want {
				t.Errorf("IsPod() = %v, want %v", got, tc.want)
			}
		})
	}
}
