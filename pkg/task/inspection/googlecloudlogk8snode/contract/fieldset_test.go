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

package googlecloudlogk8snode_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
)

func TestK8sNodeLogCommonFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *K8sNodeLogCommonFieldSet
	}{
		{
			desc: "with all parameters",
			input: `jsonPayload:
  MESSAGE: "test message"
  SYSLOG_IDENTIFIER: "test-identifier"
resource:
  labels:
    node_name: node-foo`,
			want: &K8sNodeLogCommonFieldSet{
				Message:   "test message",
				Component: "test-identifier",
				NodeName:  "node-foo",
			},
		},
		{
			desc: "with component name surrounded by ()",
			input: `jsonPayload:
  MESSAGE: "test message"
  SYSLOG_IDENTIFIER: "(dockerd)"
resource:
  labels:
    node_name: node-foo`,
			want: &K8sNodeLogCommonFieldSet{
				Message:   "test message",
				Component: "dockerd",
				NodeName:  "node-foo",
			},
		},
		{
			desc: "kube-proxy logs",
			input: `jsonPayload:
  MESSAGE: "test message"
logName: projects/test-project/logs/kube-proxy
resource:
  labels:
    node_name: node-foo`,
			want: &K8sNodeLogCommonFieldSet{
				Message:   "test message",
				Component: "kube-proxy",
				NodeName:  "node-foo",
			},
		},
		{
			desc:  "without jsonPayload",
			input: `{}`,
			want: &K8sNodeLogCommonFieldSet{
				Message:   "",
				Component: "",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(testCase.input)
			if err != nil {
				t.Errorf("failed to parse test YAML data: %v", err)
			}

			err = l.SetFieldSetReader(&K8sNodeLogCommonFieldSetReader{})
			if err != nil {
				t.Fatalf("K8sNodeLogCommonFieldSetReader returned an unexpected error:%v", err)
			}
			fieldSet := log.MustGetFieldSet(l, &K8sNodeLogCommonFieldSet{})
			if diff := cmp.Diff(testCase.want, fieldSet); diff != "" {
				t.Errorf("K8sNodeLogCommonFieldSetReader mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sNodeLogCommonFieldSet_ParserType(t *testing.T) {
	testCases := []struct {
		desc     string
		fieldSet *K8sNodeLogCommonFieldSet
		want     K8sNodeParserType
	}{
		{
			desc: "containerd parser type",
			fieldSet: &K8sNodeLogCommonFieldSet{
				Component: "containerd",
			},
			want: Containerd,
		},
		{
			desc: "kubelet parser type",
			fieldSet: &K8sNodeLogCommonFieldSet{
				Component: "kubelet",
			},
			want: Kubelet,
		},
		{
			desc: "other parser type",
			fieldSet: &K8sNodeLogCommonFieldSet{
				Component: "other-component",
			},
			want: Other,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.fieldSet.ParserType()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParserType() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sNodeLogCommonFieldSet_ResourcePath(t *testing.T) {
	testCases := []struct {
		desc     string
		fieldSet *K8sNodeLogCommonFieldSet
		want     resourcepath.ResourcePath
	}{
		{
			desc: "kube-proxy resource path",
			fieldSet: &K8sNodeLogCommonFieldSet{
				Component: "kube-proxy",
				NodeName:  "node-foo",
			},
			want: resourcepath.Pod("kube-system", "kube-proxy-node-foo"),
		},
		{
			desc: "other component resource path",
			fieldSet: &K8sNodeLogCommonFieldSet{
				Component: "other-component",
				NodeName:  "node-bar",
			},
			want: resourcepath.NodeComponent("node-bar", "other-component"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.fieldSet.ResourcePath()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ResourcePath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
