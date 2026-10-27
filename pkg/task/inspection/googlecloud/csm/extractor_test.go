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

package csm

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	commoncsmcp "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/csmcp"
	"github.com/google/go-cmp/cmp"
)

func TestExtractIstioAccessLog(t *testing.T) {
	testCases := []struct {
		desc    string
		input   string
		want    IstioAccessLogFieldSet
		wantErr bool
	}{
		{
			desc: "server access log",
			input: `
logName: "projects/test-project/logs/server-accesslog-stackdriver"
labels:
  response_flag: "UH"
  source_namespace: "default"
  source_name: "istio-ingressgateway"
  destination_namespace: "default"
  destination_service_name: "productpage"
  destination_service_host: productpage.default.svc.cluster.local
resource:
  labels:
    pod_name: "productpage-v1"
    namespace_name: "default"
    container_name: "istio-proxy"
`,
			want: IstioAccessLogFieldSet{
				Type:                        AccessLogTypeServer,
				ResponseFlags:               logutil.EnvoyResponseFlags{logutil.EnvoyResponseFlagNoHealthyUpstream},
				SourceNamespace:             "default",
				SourceName:                  "istio-ingressgateway",
				DestinationNamespace:        "default",
				DestinationServiceName:      "productpage",
				DestinationServiceNamespace: "default",
				ReporterPodName:             "productpage-v1",
				ReporterPodNamespace:        "default",
				ReporterContainerName:       "istio-proxy",
			},
		},
		{
			desc: "client access log",
			input: `
logName: "projects/test-project/logs/client-accesslog-stackdriver"
labels:
  response_flag: "-"
  source_namespace: "default"
  source_name: "productpage-v1"
  destination_namespace: "default"
  destination_name: "details-v1"
  destination_service_name: "details"
  destination_service_host: details.detailer.svc.cluster.local
resource:
  labels:
    pod_name: "productpage-v1"
    namespace_name: "default"
`,
			want: IstioAccessLogFieldSet{
				Type:                        AccessLogTypeClient,
				ResponseFlags:               logutil.EnvoyResponseFlags{logutil.EnvoyResponseFlagNoError},
				SourceNamespace:             "default",
				SourceName:                  "productpage-v1",
				DestinationNamespace:        "default",
				DestinationServiceNamespace: "detailer",
				DestinationName:             "details-v1",
				DestinationServiceName:      "details",
				ReporterPodName:             "productpage-v1",
				ReporterPodNamespace:        "default",
				ReporterContainerName:       "",
			},
		},
		{
			desc: "server access log with multiple response flags",
			input: `
logName: "projects/test-project/logs/server-accesslog-stackdriver"
labels:
  response_flag: "UH,URX"
  source_namespace: "default"
  source_name: "istio-ingressgateway"
  destination_namespace: "default"
  destination_service_name: "productpage"
  destination_service_host: productpage.default.svc.cluster.local
resource:
  labels:
    pod_name: "productpage-v1"
    namespace_name: "default"
    container_name: "istio-proxy"
`,
			want: IstioAccessLogFieldSet{
				Type:                        AccessLogTypeServer,
				ResponseFlags:               logutil.EnvoyResponseFlags{logutil.EnvoyResponseFlagNoHealthyUpstream, logutil.EnvoyResponseFlagUpstreamRetryLimitExceeded},
				SourceNamespace:             "default",
				SourceName:                  "istio-ingressgateway",
				DestinationNamespace:        "default",
				DestinationServiceName:      "productpage",
				DestinationServiceNamespace: "default",
				ReporterPodName:             "productpage-v1",
				ReporterPodNamespace:        "default",
				ReporterContainerName:       "istio-proxy",
			},
		},
		{
			desc: "with missing labels",
			input: `
logName: "projects/test-project/logs/client-accesslog-stackdriver"
resource:
  labels:
    pod_name: "productpage-v1"
    namespace_name: "default"
`,
			want: IstioAccessLogFieldSet{
				Type:                   AccessLogTypeClient,
				ResponseFlags:          logutil.EnvoyResponseFlags{},
				SourceNamespace:        "",
				SourceName:             "",
				DestinationNamespace:   "",
				DestinationName:        "",
				DestinationServiceName: "",
				ReporterPodName:        "productpage-v1",
				ReporterPodNamespace:   "default",
				ReporterContainerName:  "",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to parse YAML test input: %v", err)
			}
			reader := structured.NewNodeReader(node)
			got, err := ExtractIstioAccessLog(reader)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ExtractIstioAccessLog() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractIstioAccessLog mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("mock node returns mock values", func(t *testing.T) {
		mockFS := IstioAccessLogFieldSet{
			Type:            AccessLogTypeServer,
			SourceNamespace: "mock-ns",
			SourceName:      "mock-pod",
		}
		reader := structured.NewNodeReader(structured.NewMockNode(mockFS))
		got, err := ExtractIstioAccessLog(reader)
		if err != nil {
			t.Fatalf("ExtractIstioAccessLog() error = %v", err)
		}
		if diff := cmp.Diff(mockFS, got); diff != "" {
			t.Errorf("ExtractIstioAccessLog mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestExtractCSMControlPlaneLog(t *testing.T) {
	parsedTS := time.Date(2026, time.July, 3, 15, 46, 12, 0, time.UTC)
	testCases := []struct {
		desc    string
		input   string
		want    CSMControlPlaneLogFieldSet
		wantErr bool
	}{
		{
			desc: "jsonPayload.message with full resource labels",
			input: `
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
    build_id: "build-1234"
    mesh_uid: "proj-1234"
    owner: "helm"
jsonPayload:
  message: "ads: connections 42"
`,
			want: CSMControlPlaneLogFieldSet{
				ProjectID: "fleet-project",
				Location:  "us-central1",
				Revision:  "asm-managed",
				Message:   "ads: connections 42",
			},
		},
		{
			desc: "jsonPayload with pods and timestamp",
			input: `
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
jsonPayload:
  message: "ADS: new delta connection for node:my-pod.default-12 and other-pod.kube-system"
  timestamp: "2026-07-03T15:46:12Z"
`,
			want: CSMControlPlaneLogFieldSet{
				ProjectID: "fleet-project",
				Location:  "us-central1",
				Revision:  "asm-managed",
				Message:   "ADS: new delta connection for node:my-pod.default-12 and other-pod.kube-system",
				Timestamp: &parsedTS,
				Pods: []commoncsmcp.PodIdentifier{
					{Name: "my-pod", Namespace: "default", ConnectionID: "12"},
					{Name: "other-pod", Namespace: "kube-system"},
				},
			},
		},
		{
			desc: "textPayload fallback when jsonPayload.message is absent",
			input: `
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
textPayload: "warning starting istiod"
`,
			want: CSMControlPlaneLogFieldSet{
				ProjectID: "fleet-project",
				Location:  "us-central1",
				Revision:  "asm-managed",
				Message:   "warning starting istiod",
			},
		},
		{
			desc: "missing labels and payload returning empty fields",
			input: `
resource:
  type: istio_control_plane
`,
			want: CSMControlPlaneLogFieldSet{
				ProjectID: "",
				Location:  "",
				Revision:  "",
				Message:   "",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to parse YAML test input: %v", err)
			}
			reader := structured.NewNodeReader(node)
			got, err := ExtractCSMControlPlaneLog(reader)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ExtractCSMControlPlaneLog() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractCSMControlPlaneLog() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("mock node returns mock values", func(t *testing.T) {
		mockFS := CSMControlPlaneLogFieldSet{
			ProjectID: "mock-project",
			Location:  "mock-location",
			Revision:  "mock-rev",
			Message:   "mock-message",
		}
		reader := structured.NewNodeReader(structured.NewMockNode(mockFS))
		got, err := ExtractCSMControlPlaneLog(reader)
		if err != nil {
			t.Fatalf("ExtractCSMControlPlaneLog() error = %v", err)
		}
		if diff := cmp.Diff(mockFS, got); diff != "" {
			t.Errorf("ExtractCSMControlPlaneLog() mismatch (-want +got):\n%s", diff)
		}
	})
}
