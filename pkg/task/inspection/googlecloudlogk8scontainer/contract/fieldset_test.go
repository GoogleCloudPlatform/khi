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

package googlecloudlogk8scontainer_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestK8sContainerLogFieldSetReader_ResourceLabels(t *testing.T) {
	testCase := []struct {
		desc  string
		want  *K8sContainerLogFieldSet
		input string
	}{
		{
			desc: "from resource labels",
			want: &K8sContainerLogFieldSet{
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
			want: &K8sContainerLogFieldSet{
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
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse log from yaml: %v", err)
			}
			l.SetFieldSetReader(&K8sContainerLogFieldSetReader{})
			containerFieldSet, err := log.GetFieldSet(l, &K8sContainerLogFieldSet{})
			if err != nil {
				t.Fatalf("failed to extract message field: %v", err)
			}
			if diff := cmp.Diff(tc.want, containerFieldSet, cmpopts.IgnoreFields(K8sContainerLogFieldSet{}, "Message", "ParsedMessage")); diff != "" {
				t.Errorf("K8sContainerLogFieldSetReader mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sContainerLogFieldSetReader_MainMessage(t *testing.T) {
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
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse log from yaml: %v", err)
			}
			l.SetFieldSetReader(&K8sContainerLogFieldSetReader{})
			k8sContainerLogFieldSet, err := log.GetFieldSet(l, &K8sContainerLogFieldSet{})
			if err != nil {
				t.Fatalf("failed to extract message field: %v", err)
			}
			if k8sContainerLogFieldSet.Message != tc.want {
				t.Errorf("expected main message: %v, got: %v", tc.want, k8sContainerLogFieldSet.Message)
			}
		})
	}

}

func TestK8sContainerLogFieldSetReader_IstioProxyParsedMessage(t *testing.T) {
	inputYAML := `resource:
  labels:
    cluster_name: test-cluster
    namespace_name: default
    pod_name: frontend-pod
    container_name: istio-proxy
textPayload: '[2026-08-10T08:50:55.958Z] "HEAD / HTTP/1.1" 502 - via_upstream - "-" 0 0 6 5 "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "123.45.167.189" "123.45.167.189:80" PassthroughCluster 10.4.1.8:33606 123.45.167.189:80 10.4.1.8:59778 - allow_any'`

	l, err := log.NewLogFromYAMLString(inputYAML)
	if err != nil {
		t.Fatalf("failed to parse log from yaml: %v", err)
	}
	l.SetFieldSetReader(&K8sContainerLogFieldSetReader{})
	containerFieldSet, err := log.GetFieldSet(l, &K8sContainerLogFieldSet{})
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
	wantMsg := "502 HEAD http://123.45.167.189/"
	if mainMsg != wantMsg {
		t.Errorf("MainMessage() = %q, want %q", mainMsg, wantMsg)
	}
}

func TestK8sContainerLogFieldSetReader_IstioProxyNonAccessLog(t *testing.T) {
	inputYAML := `insertId: d041l2rc48z6lx5t
logName: projects/tse-kakeru/logs/stdout
labels:
  compute.googleapis.com/resource_name: gke-keigof-0804-clus-keigof-0804-node-dc914732-d5fd
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
    cluster_name: keigof-0804-cluster
    container_name: istio-proxy
    location: asia-northeast1
    namespace_name: default
    pod_name: nginx-server-8646bbcd65-6d969
    project_id: tse-kakeru
severity: INFO
receiveTimestamp: '2026-08-20T04:46:34.341075649Z'
timestamp: '2026-08-20T04:46:31.353884563Z'`

	l, err := log.NewLogFromYAMLString(inputYAML)
	if err != nil {
		t.Fatalf("failed to parse log from yaml: %v", err)
	}
	l.SetFieldSetReader(&K8sContainerLogFieldSetReader{})
	containerFieldSet, err := log.GetFieldSet(l, &K8sContainerLogFieldSet{})
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

func TestGCPContainerLogNodeNameLabelFieldSetReader(t *testing.T) {
	testCase := []struct {
		desc  string
		want  *GCPContainerLogNodeNameLabelFieldSet
		input string
	}{
		{
			desc: "from labels",
			want: &GCPContainerLogNodeNameLabelFieldSet{
				NodeName:  "test-node",
				PodLabels: map[string]string{},
			},
			input: `labels:
  compute.googleapis.com/resource_name: test-node`,
		},
		{
			desc: "missing labels",
			want: &GCPContainerLogNodeNameLabelFieldSet{
				NodeName:  "",
				PodLabels: map[string]string{},
			},
			input: `labels:
  foo: bar`,
		},
		{
			desc: "with k8s-pod labels",
			want: &GCPContainerLogNodeNameLabelFieldSet{
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
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse log from yaml: %v", err)
			}
			l.SetFieldSetReader(&GCPContainerLogNodeNameLabelFieldSetReader{})
			nodeFieldSet, err := log.GetFieldSet(l, &GCPContainerLogNodeNameLabelFieldSet{})
			if err != nil {
				t.Fatalf("failed to extract node field: %v", err)
			}
			if diff := cmp.Diff(tc.want, nodeFieldSet); diff != "" {
				t.Errorf("GCPContainerLogNodeNameLabelFieldSetReader mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
