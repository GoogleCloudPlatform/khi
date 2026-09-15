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

package csmcp

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	commoncsmcp "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/csmcp"
	"github.com/google/go-cmp/cmp"
)

func TestExtract(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    FieldSet
		wantErr bool
	}{
		{
			name: "textPayload delta connection log with resource labels",
			input: `
resource:
  labels:
    cluster_name: my-cluster
    namespace_name: istio-system
    pod_name: istiod-asm-1234
    container_name: discovery
textPayload: "ADS: new delta connection for node:auth-service-647d798687-abcde.backend-ns-61"
`,
			want: FieldSet{
				Timestamp:   nil,
				Message:     "ADS: new delta connection for node:auth-service-647d798687-abcde.backend-ns-61",
				ClusterName: "my-cluster",
				Pods: []commoncsmcp.PodIdentifier{
					{
						Name:         "auth-service-647d798687-abcde",
						Namespace:    "backend-ns",
						ConnectionID: "61",
					},
				},
			},
		},
		{
			name: "textPayload terminated log with quoted ip address",
			input: `
resource:
  labels:
    cluster_name: my-cluster
    namespace_name: istio-system
    pod_name: istiod-asm-1234
    container_name: discovery
textPayload: 'ADS: "192.0.2.1:41780" payment-worker-7f5bcf84bb-fghij.payment-ns-84 terminated'
`,
			want: FieldSet{
				Timestamp:   nil,
				Message:     `ADS: "192.0.2.1:41780" payment-worker-7f5bcf84bb-fghij.payment-ns-84 terminated`,
				ClusterName: "my-cluster",
				Pods: []commoncsmcp.PodIdentifier{
					{
						Name:         "payment-worker-7f5bcf84bb-fghij",
						Namespace:    "payment-ns",
						ConnectionID: "84",
					},
				},
			},
		},
		{
			name: "textPayload push request log without connection id",
			input: `
resource:
  labels:
    cluster_name: my-cluster
    namespace_name: istio-system
    pod_name: istiod-asm-1234
    container_name: discovery
textPayload: "CDS: PUSH request for node:payment-worker-7f5bcf84bb-fghij.payment-ns resources:87 ..."
`,
			want: FieldSet{
				Timestamp:   nil,
				Message:     "CDS: PUSH request for node:payment-worker-7f5bcf84bb-fghij.payment-ns resources:87 ...",
				ClusterName: "my-cluster",
				Pods: []commoncsmcp.PodIdentifier{
					{
						Name:         "payment-worker-7f5bcf84bb-fghij",
						Namespace:    "payment-ns",
						ConnectionID: "",
					},
				},
			},
		},
		{
			name: "jsonPayload message and timestamp",
			input: `
resource:
  labels:
    cluster_name: prod-cluster
    namespace_name: asm-system
    pod_name: istiod-688
    container_name: discovery
jsonPayload:
  message: "ADS: new connection for node:ratings-v1-7d99676f7f-abcde.default-1"
  timestamp: "2026-07-03T15:46:12Z"
`,
			want: FieldSet{
				Timestamp: func() *time.Time {
					t, _ := time.Parse(time.RFC3339, "2026-07-03T15:46:12Z")
					return &t
				}(),
				Message:     "ADS: new connection for node:ratings-v1-7d99676f7f-abcde.default-1",
				ClusterName: "prod-cluster",
				Pods: []commoncsmcp.PodIdentifier{
					{
						Name:         "ratings-v1-7d99676f7f-abcde",
						Namespace:    "default",
						ConnectionID: "1",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			reader := structured.NewNodeReader(node)
			got, err := Extract(reader)
			if (err != nil) != tc.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Extract() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("mock node returns mock values", func(t *testing.T) {
		mockFS := FieldSet{
			Message:     "mock msg",
			ClusterName: "mock-cluster",
		}
		reader := structured.NewNodeReader(structured.NewMockNode(mockFS))
		got, err := Extract(reader)
		if err != nil {
			t.Fatalf("Extract() error = %v", err)
		}
		if diff := cmp.Diff(mockFS, got); diff != "" {
			t.Errorf("Extract() mismatch (-want +got):\n%s", diff)
		}
	})
}
