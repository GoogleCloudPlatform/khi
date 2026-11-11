// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8scontainer

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// severityComparer compares severities by identity because every severity is a registered singleton.
var severityComparer = cmp.Comparer(func(a, b *pb.Severity) bool {
	return a == b
})

func TestExtractK8sContainerLog_ResourceLabels(t *testing.T) {
	testCase := []struct {
		desc  string
		want  K8sContainerLogFieldSet
		input string
	}{
		{
			desc: "from resource labels",
			want: K8sContainerLogFieldSet{
				ClusterName:   "test-cluster",
				Namespace:     "test-namespace",
				PodName:       "test-pod",
				ContainerName: "test-container",
			},
			input: `resource:
  labels:
    cluster_name: test-cluster
    namespace_name: test-namespace
    pod_name: test-pod
    container_name: test-container`,
		},
		{
			desc: "missing resource labels",
			want: K8sContainerLogFieldSet{
				ClusterName:   "unknown",
				Namespace:     "unknown",
				PodName:       "unknown",
				ContainerName: "unknown",
			},
			input: `resource:
  labels:
    foo: bar`,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.desc, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			reader := structured.NewNodeReader(node)
			containerFieldSet, err := ExtractK8sContainerLog(reader, nil)
			if err != nil {
				t.Fatalf("failed to extract message field: %v", err)
			}
			if diff := cmp.Diff(tc.want, containerFieldSet, cmpopts.IgnoreFields(K8sContainerLogFieldSet{}, "Message", "ParsedMessage")); diff != "" {
				t.Errorf("ExtractK8sContainerLog mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractK8sContainerLog_MainMessage(t *testing.T) {
	testCase := []struct {
		desc  string
		want  string
		input string
	}{
		{
			desc:  "from textPayload field",
			want:  "foo",
			input: `textPayload: foo`,
		},
		{
			desc: "from jsonPayload.message field",
			want: "bar",
			input: `jsonPayload:
  message: bar`,
		},
		{
			desc: "from jsonPayload.MESSAGE field",
			want: "bar",
			input: `jsonPayload:
  MESSAGE: bar`,
		},
		{
			desc: "from jsonPayload.msg field",
			want: "bar",
			input: `jsonPayload:
  msg: bar`,
		},
		{
			desc: "from jsonPayload.log field",
			want: "bar",
			input: `jsonPayload:
  log: bar`,
		},
		{
			desc: "from the whole jsonPayload field",
			want: `{"foo":"bar"}`,
			input: `jsonPayload:
  foo: bar`,
		},
		{
			desc: "from the whole labels field",
			want: `{"foo":"bar"}`,
			input: `labels:
  foo: bar`,
		},
		{
			desc: "ignore when the message is protoPayload even labels are provided",
			want: "",
			input: `labels:
  foo: bar
protoPayload:
  qux: quux`,
		},
		{
			desc:  "empty if no proper field is given",
			want:  "",
			input: `foo: bar`,
		},
		{
			desc: "prioritize textPayload rather than jsonPayload.msg or labels",
			want: "bar",
			input: `jsonPayload:
  msg: foo
textPayload: bar
labels:
  qux: quux`,
		},
		{
			desc: "prioritize jsonPayload.msg over labels",
			want: "foo",
			input: `jsonPayload:
  msg: foo
labels:
  qux: quux`,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.desc, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			reader := structured.NewNodeReader(node)
			k8sContainerLogFieldSet, err := ExtractK8sContainerLog(reader, nil)
			if err != nil {
				t.Fatalf("failed to extract message field: %v", err)
			}
			if k8sContainerLogFieldSet.Message != tc.want {
				t.Errorf("expected main message: %v, got: %v", tc.want, k8sContainerLogFieldSet.Message)
			}
		})
	}
}

func TestExtractK8sContainerLog_IstioProxyParsedMessage(t *testing.T) {
	inputYAML := `resource:
  labels:
    cluster_name: test-cluster
    namespace_name: default
    pod_name: frontend-pod
    container_name: istio-proxy
textPayload: '[2026-08-10T08:50:55.958Z] "HEAD / HTTP/1.1" 502 - via_upstream - "-" 0 0 6 5 "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "192.0.2.1" "192.0.2.1:80" PassthroughCluster 10.4.1.8:33606 192.0.2.1:80 10.4.1.8:59778 - allow_any'`

	node, err := structured.FromYAML(inputYAML)
	if err != nil {
		t.Fatalf("failed to parse yaml: %v", err)
	}
	reader := structured.NewNodeReader(node)
	containerFieldSet, err := ExtractK8sContainerLog(reader, nil)
	if err != nil {
		t.Fatalf("failed to extract container field set: %v", err)
	}
	if containerFieldSet.ParsedMessage == nil {
		t.Fatalf("expected ParsedMessage not to be nil for istio-proxy")
	}
	mainMsg, err := containerFieldSet.ParsedMessage.MainMessage()
	if err != nil {
		t.Fatalf("MainMessage() failed: %v", err)
	}
	wantMsg := "502 HEAD http://192.0.2.1/"
	if mainMsg != wantMsg {
		t.Errorf("MainMessage() = %q, want %q", mainMsg, wantMsg)
	}
}

func TestExtractK8sContainerLog_IstioProxyNonAccessLog(t *testing.T) {
	inputYAML := `insertId: d041l2rc48z6lx5t
logName: projects/test-project/logs/stdout
labels:
  compute.googleapis.com/resource_name: gke-test-cluster-default-pool-dc914732-d5fd
  k8s-pod/app: nginx-server
  k8s-pod/pod-template-hash: 8646bbcd65
  k8s-pod/security_istio_io/tlsMode: istio
  k8s-pod/service_istio_io/canonical-name: nginx-server
  k8s-pod/service_istio_io/canonical-revision: latest
  logging.gke.io/top_level_controller_name: nginx-server
  logging.gke.io/top_level_controller_type: Deployment
textPayload: "2026-08-20T04:46:31.353537Z\tinfo\txdsproxy\tconnected to upstream XDS server: meshconfig.googleapis.com:443"
resource:
  type: k8s_container
  labels:
    cluster_name: test-cluster
    container_name: istio-proxy
    location: asia-northeast1
    namespace_name: default
    pod_name: nginx-server-8646bbcd65-6d969
    project_id: test-project
severity: INFO
receiveTimestamp: '2026-08-20T04:46:34.341075649Z'
timestamp: '2026-08-20T04:46:31.353884563Z'`

	node, err := structured.FromYAML(inputYAML)
	if err != nil {
		t.Fatalf("failed to parse yaml: %v", err)
	}
	reader := structured.NewNodeReader(node)
	containerFieldSet, err := ExtractK8sContainerLog(reader, nil)
	if err != nil {
		t.Fatalf("failed to extract container field set: %v", err)
	}
	wantMsg := "2026-08-20T04:46:31.353537Z\tinfo\txdsproxy\tconnected to upstream XDS server: meshconfig.googleapis.com:443"
	if containerFieldSet.Message != wantMsg {
		t.Errorf("Message = %q, want %q", containerFieldSet.Message, wantMsg)
	}
	if containerFieldSet.ParsedMessage == nil {
		t.Fatalf("expected ParsedMessage not to be nil")
	}
	mainMsg, err := containerFieldSet.ParsedMessage.MainMessage()
	if err != nil {
		t.Fatalf("MainMessage() failed: %v", err)
	}
	if mainMsg != wantMsg {
		t.Errorf("MainMessage() = %q, want %q", mainMsg, wantMsg)
	}
}

func TestK8sContainerLogFieldSet_GroupKey(t *testing.T) {
	fs := &K8sContainerLogFieldSet{
		Namespace: "test-namespace",
		PodName:   "test-pod",
	}
	want := "test-namespace/test-pod"
	got := fs.GroupKey()
	if got != want {
		t.Errorf("GroupKey() = %q, want %q", got, want)
	}
}

func TestExtractGCPContainerLogNodeNameLabel(t *testing.T) {
	testCase := []struct {
		desc  string
		want  GCPContainerLogNodeNameLabelFieldSet
		input string
	}{
		{
			desc: "from labels",
			want: GCPContainerLogNodeNameLabelFieldSet{
				NodeName:  "test-node",
				PodLabels: map[string]string{},
			},
			input: `labels:
  compute.googleapis.com/resource_name: test-node`,
		},
		{
			desc: "missing labels",
			want: GCPContainerLogNodeNameLabelFieldSet{
				NodeName:  "",
				PodLabels: map[string]string{},
			},
			input: `labels:
  foo: bar`,
		},
		{
			desc: "with k8s-pod labels",
			want: GCPContainerLogNodeNameLabelFieldSet{
				NodeName: "test-node",
				PodLabels: map[string]string{
					"app":               "my-app",
					"pod-template-hash": "12345",
				},
			},
			input: `labels:
  compute.googleapis.com/resource_name: test-node
  k8s-pod/app: my-app
  k8s-pod/pod-template-hash: "12345"
  other-label: foo`,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.desc, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			reader := structured.NewNodeReader(node)
			nodeFieldSet, err := ExtractGCPContainerLogNodeNameLabel(reader)
			if err != nil {
				t.Fatalf("failed to extract node field: %v", err)
			}
			if diff := cmp.Diff(tc.want, nodeFieldSet); diff != "" {
				t.Errorf("ExtractGCPContainerLogNodeNameLabel mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("mock node returns mock values", func(t *testing.T) {
		mockFS := GCPContainerLogNodeNameLabelFieldSet{
			NodeName:  "mock-node",
			PodLabels: map[string]string{"foo": "bar"},
		}
		reader := structured.NewNodeReader(structured.NewMockNode(mockFS))
		got, err := ExtractGCPContainerLogNodeNameLabel(reader)
		if err != nil {
			t.Fatalf("ExtractGCPContainerLogNodeNameLabel() error = %v", err)
		}
		if diff := cmp.Diff(mockFS, got); diff != "" {
			t.Errorf("ExtractGCPContainerLogNodeNameLabel mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestExtractCiliumAgentLog(t *testing.T) {
	logfmtParser := logutil.NewLogfmtTextParser()

	testCases := []struct {
		desc   string
		input  string
		parsed *logutil.ParseStructuredLogResult
		want   CiliumAgentLogFieldSet
	}{
		{
			desc: "logrus log with a pod reference combining the namespace and the name",
			input: `
jsonPayload:
  ciliumEndpointName: kube-system/kube-dns-123456789a-abcde
  containerID: 2bbd0dd28e
  endpointID: 688
  ipv4: 10.0.0.9
  k8sPodName: kube-system/kube-dns-123456789a-abcde
  level: info
  msg: Identity of endpoint changed
  new-identity: "7301"
  old-identity: no identity
  subsys: endpoint
`,
			want: CiliumAgentLogFieldSet{
				Subsystem:    "endpoint",
				Severity:     inspectioncore.SeverityInfo,
				PodNamespace: "kube-system",
				PodName:      "kube-dns-123456789a-abcde",
			},
		},
		{
			desc: "Hive slog log with the namespace and the name in separate fields",
			input: `
jsonPayload:
  containerID: 6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1
  endpointID: 169
  k8sNamespace: kube-system
  k8sPodName: konnectivity-agent-123456789b-fghij
  level: info
  module: agent.controlplane.endpoint-api
  msg: Delete endpoint by containerID request
`,
			want: CiliumAgentLogFieldSet{
				Subsystem:    "endpoint-api",
				Severity:     inspectioncore.SeverityInfo,
				PodNamespace: "kube-system",
				PodName:      "konnectivity-agent-123456789b-fghij",
			},
		},
		{
			desc: "error log without a pod reference",
			input: `
jsonPayload:
  error: "failed to list <unspecified>: the server could not find the requested resource (get ciliumpodippools.cilium.io)"
  level: error
  msg: k8sError
  subsys: k8s
`,
			want: CiliumAgentLogFieldSet{
				Subsystem: "k8s",
				Error:     "failed to list <unspecified>: the server could not find the requested resource (get ciliumpodippools.cilium.io)",
				Severity:  inspectioncore.SeverityError,
			},
		},
		{
			desc: "pod reference only in the endpoint name with an uppercase level",
			input: `
jsonPayload:
  ciliumEndpointName: default/httpbin-0
  level: WARN
  msg: Regeneration of endpoint failed
  subsys: endpoint
`,
			want: CiliumAgentLogFieldSet{
				Subsystem:    "endpoint",
				Severity:     inspectioncore.SeverityWarning,
				PodNamespace: "default",
				PodName:      "httpbin-0",
			},
		},
		{
			desc: "malformed pod reference and unknown level",
			input: `
jsonPayload:
  k8sPodName: default/httpbin/0
  level: trace
  msg: Unexpected pod reference
`,
			want: CiliumAgentLogFieldSet{},
		},
		{
			desc:   "text payload in logfmt",
			input:  `textPayload: 'time=2026-01-01T00:00:00.000000000Z level=info msg=Reconciling subsys=sfc k8sNamespace=default k8sPodName=httpbin-0'`,
			parsed: logfmtParser.TryParse(`time=2026-01-01T00:00:00.000000000Z level=info msg=Reconciling subsys=sfc k8sNamespace=default k8sPodName=httpbin-0`),
			want: CiliumAgentLogFieldSet{
				Subsystem:    "sfc",
				PodNamespace: "default",
				PodName:      "httpbin-0",
			},
		},
		{
			desc:   "text payload that is not logfmt",
			input:  `textPayload: plain text message`,
			parsed: (&logutil.FallbackRawTextLogParser{}).TryParse("plain text message"),
			want:   CiliumAgentLogFieldSet{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			reader := structured.NewNodeReader(node)

			got, err := ExtractCiliumAgentLog(reader, tc.parsed)
			if err != nil {
				t.Fatalf("ExtractCiliumAgentLog() returned an unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got, severityComparer); diff != "" {
				t.Errorf("ExtractCiliumAgentLog() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractCiliumAgentLog_Mock(t *testing.T) {
	want := CiliumAgentLogFieldSet{
		Subsystem:    "endpoint",
		Severity:     inspectioncore.SeverityWarning,
		PodNamespace: "default",
		PodName:      "httpbin-0",
	}
	reader := structured.NewNodeReader(structured.NewMockNode(want))

	got, err := ExtractCiliumAgentLog(reader, nil)
	if err != nil {
		t.Fatalf("ExtractCiliumAgentLog() returned an unexpected error: %v", err)
	}
	if diff := cmp.Diff(want, got, severityComparer); diff != "" {
		t.Errorf("ExtractCiliumAgentLog() mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractCiliumAgentLog_NilReader(t *testing.T) {
	if _, err := ExtractCiliumAgentLog(nil, nil); err == nil {
		t.Errorf("ExtractCiliumAgentLog(nil, nil) returned no error, want an error")
	}
}

func TestCiliumAgentLogFieldSet_HasPodReference(t *testing.T) {
	testCases := []struct {
		desc   string
		fields CiliumAgentLogFieldSet
		want   bool
	}{
		{
			desc:   "namespace and name",
			fields: CiliumAgentLogFieldSet{PodNamespace: "default", PodName: "httpbin-0"},
			want:   true,
		},
		{
			desc:   "name without namespace",
			fields: CiliumAgentLogFieldSet{PodName: "httpbin-0"},
			want:   false,
		},
		{
			desc:   "no pod reference",
			fields: CiliumAgentLogFieldSet{Subsystem: "daemon"},
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if got := tc.fields.HasPodReference(); got != tc.want {
				t.Errorf("HasPodReference() = %v, want %v", got, tc.want)
			}
		})
	}
}
