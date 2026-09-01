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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func TestExtractK8sNodeLogCommon(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		mock  *K8sNodeLogCommonFieldSet
		want  K8sNodeLogCommonFieldSet
	}{
		{
			desc: "with all parameters",
			input: `jsonPayload:
  MESSAGE: "test message"
  SYSLOG_IDENTIFIER: "test-identifier"
resource:
  labels:
    node_name: node-foo`,
			want: K8sNodeLogCommonFieldSet{
				Message: &logutil.ParseStructuredLogResult{Fields: map[string]any{
					logutil.OriginalMessageFieldKey:       "test message",
					logutil.MainMessageStructuredFieldKey: "test message",
				}},
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
			want: K8sNodeLogCommonFieldSet{
				Message: &logutil.ParseStructuredLogResult{Fields: map[string]any{
					logutil.OriginalMessageFieldKey:       "test message",
					logutil.MainMessageStructuredFieldKey: "test message",
				}}, Component: "dockerd",
				NodeName: "node-foo",
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
			want: K8sNodeLogCommonFieldSet{
				Message: &logutil.ParseStructuredLogResult{Fields: map[string]any{
					logutil.OriginalMessageFieldKey:       "test message",
					logutil.MainMessageStructuredFieldKey: "test message",
				}}, Component: "kube-proxy",
				NodeName: "node-foo",
			},
		},
		{
			desc:  "without jsonPayload",
			input: `{}`,
			want: K8sNodeLogCommonFieldSet{
				Message: &logutil.ParseStructuredLogResult{Fields: map[string]any{
					logutil.OriginalMessageFieldKey:       "",
					logutil.MainMessageStructuredFieldKey: "",
				}}, Component: "",
			},
		},
		{
			desc: "from mock",
			mock: &K8sNodeLogCommonFieldSet{
				Message: &logutil.ParseStructuredLogResult{Fields: map[string]any{
					logutil.OriginalMessageFieldKey: "mock-message",
				}},
				Component: "mock-comp",
				NodeName:  "mock-node",
			},
			want: K8sNodeLogCommonFieldSet{
				Message: &logutil.ParseStructuredLogResult{Fields: map[string]any{
					logutil.OriginalMessageFieldKey: "mock-message",
				}},
				Component: "mock-comp",
				NodeName:  "mock-node",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var l *log.Log
			var err error
			if tc.mock != nil {
				l = testlog.NewMockLog(*tc.mock)
			} else {
				l, err = log.NewLogFromYAMLString(tc.input)
				if err != nil {
					t.Fatalf("failed to parse test YAML data: %v", err)
				}
			}

			got, err := ExtractK8sNodeLogCommon(l.NodeReader, nil)
			if err != nil {
				t.Fatalf("ExtractK8sNodeLogCommon() returned unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractK8sNodeLogCommon() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestK8sNodeLogCommonFieldSet_ParserType(t *testing.T) {
	testCases := []struct {
		desc     string
		fieldSet K8sNodeLogCommonFieldSet
		want     K8sNodeParserType
	}{
		{
			desc: "containerd parser type",
			fieldSet: K8sNodeLogCommonFieldSet{
				Component: "containerd",
			},
			want: Containerd,
		},
		{
			desc: "kubelet parser type",
			fieldSet: K8sNodeLogCommonFieldSet{
				Component: "kubelet",
			},
			want: Kubelet,
		},
		{
			desc: "other parser type",
			fieldSet: K8sNodeLogCommonFieldSet{
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
