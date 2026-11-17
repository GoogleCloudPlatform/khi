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

package k8scontainer_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontainer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestEnrichCiliumAgentLog(t *testing.T) {
	testCases := []struct {
		name            string
		input           *log.Log
		initialSeverity *pb.Severity
		assert          func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name:            "cilium-agent JSON log with subsystem, error, level and pod reference",
			initialSeverity: inspectioncore.SeverityInfo,
			input: testlog.NewMockLog(
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				k8scontainer.K8sContainerLogFieldSet{
					Namespace:     "kube-system",
					PodName:       "anetd-abcde",
					ContainerName: "cilium-agent",
					Message:       "Regeneration of endpoint failed",
					ParsedMessage: (&logutil.FallbackRawTextLogParser{}).TryParse("Regeneration of endpoint failed"),
				},
				k8scontainer.CiliumAgentLogFieldSet{
					Subsystem:    "endpoint",
					Error:        "context deadline exceeded",
					Severity:     inspectioncore.SeverityError,
					PodNamespace: "default",
					PodName:      "httpbin-0",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("[endpoint] Regeneration of endpoint failed(error=context deadline exceeded) 【httpbin-0 (Namespace: default)】").
					HasSeverity(inspectioncore.SeverityError)
			},
		},
		{
			name:            "cilium-agent log without JSON level keeps existing severity",
			initialSeverity: inspectioncore.SeverityWarning,
			input: testlog.NewMockLog(
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				k8scontainer.K8sContainerLogFieldSet{
					Namespace:     "kube-system",
					PodName:       "anetd-abcde",
					ContainerName: "cilium-agent",
					Message:       `time=2026-01-01T00:00:00.000000000Z level=warn msg=Reconciling subsys=sfc reasons=[pod:default/httpbin-0]`,
					ParsedMessage: logutil.NewLogfmtTextParser().TryParse(`time=2026-01-01T00:00:00.000000000Z level=warn msg=Reconciling subsys=sfc reasons=[pod:default/httpbin-0]`),
				},
				k8scontainer.CiliumAgentLogFieldSet{
					Subsystem: "sfc",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("[sfc] Reconciling").
					HasSeverity(inspectioncore.SeverityWarning)
			},
		},
		{
			name:            "cilium-agent logrus JSON log entry",
			initialSeverity: inspectioncore.SeverityWarning,
			input: testlog.MustLogFromYAML(`
timestamp: "2026-01-01T00:00:00Z"
severity: WARNING
resource:
  type: k8s_container
  labels:
    cluster_name: test-cluster
    container_name: cilium-agent
    namespace_name: kube-system
    pod_name: anetd-abcde
jsonPayload:
  ciliumEndpointName: kube-system/kube-dns-123456789a-abcde
  endpointID: 688
  k8sPodName: kube-system/kube-dns-123456789a-abcde
  level: info
  msg: Identity of endpoint changed
  subsys: endpoint
`),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("[endpoint] Identity of endpoint changed 【kube-dns-123456789a-abcde (Namespace: kube-system)】").
					HasSeverity(inspectioncore.SeverityInfo)
			},
		},
		{
			name:            "cilium-agent JSON error log entry written to stdout",
			initialSeverity: inspectioncore.SeverityInfo,
			input: testlog.MustLogFromYAML(`
timestamp: "2026-01-01T00:00:00Z"
severity: INFO
resource:
  type: k8s_container
  labels:
    cluster_name: test-cluster
    container_name: cilium-agent
    namespace_name: kube-system
    pod_name: anetd-abcde
jsonPayload:
  error: "failed to list <unspecified>: the server could not find the requested resource (get ciliumpodippools.cilium.io)"
  level: error
  msg: k8sError
  subsys: k8s
`),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("[k8s] k8sError(error=failed to list <unspecified>: the server could not find the requested resource (get ciliumpodippools.cilium.io))").
					HasSeverity(inspectioncore.SeverityError)
			},
		},
		{
			name:            "cilium-agent log without subsystem",
			initialSeverity: inspectioncore.SeverityInfo,
			input: testlog.NewMockLog(
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				k8scontainer.K8sContainerLogFieldSet{
					Namespace:     "kube-system",
					PodName:       "anetd-abcde",
					ContainerName: "cilium-agent",
					Message:       "Failed to initialize daemon",
					ParsedMessage: (&logutil.FallbackRawTextLogParser{}).TryParse("Failed to initialize daemon"),
				},
				k8scontainer.CiliumAgentLogFieldSet{
					Error:    "permission denied",
					Severity: inspectioncore.SeverityError,
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("Failed to initialize daemon(error=permission denied)").
					HasSeverity(inspectioncore.SeverityError)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := khifilev6.NewLogChangeSet(tc.input)
			if err != nil {
				t.Fatalf("NewLogChangeSet() returned unexpected error: %v", err)
			}
			cs.SetSeverity(tc.initialSeverity)

			containerFields, err := k8scontainer.ExtractK8sContainerLog(tc.input.NodeReader, nil)
			if err != nil {
				t.Fatalf("ExtractK8sContainerLog() returned unexpected error: %v", err)
			}
			summary := containerFields.Message
			if containerFields.ParsedMessage != nil {
				if msg, err := containerFields.ParsedMessage.MainMessage(); err == nil && msg != "" {
					summary = msg
				}
			}

			summary = enrichCiliumAgentLog(cs, tc.input, containerFields, summary)
			cs.SetSummary(summary)
			tc.assert(t, cs)
		})
	}
}
