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
				Namespace:     "test-namespace",
				PodName:       "test-pod",
				ContainerName: "test-container",
			},
			input: `resource:
  labels:
    namespace_name: test-namespace
    pod_name: test-pod
    container_name: test-container`,
		},
		{
			desc: "missing resource labels",
			want: &K8sContainerLogFieldSet{
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
			if diff := cmp.Diff(tc.want, containerFieldSet, cmpopts.IgnoreFields(K8sContainerLogFieldSet{}, "Message")); diff != "" {
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
