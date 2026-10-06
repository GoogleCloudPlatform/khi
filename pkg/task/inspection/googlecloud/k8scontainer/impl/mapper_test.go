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
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontainer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

// TestProcessContainerLog tests processContainerLog.
func TestProcessContainerLog(t *testing.T) {
	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "successful container log ingestion",
			input: testlog.NewMockLog(
				time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				inspectioncore.DefaultSeverityFieldSet{
					Severity: inspectioncore.SeverityInfo,
				},
				k8scontainer.K8sContainerLogFieldSet{
					Namespace:     "test-namespace",
					PodName:       "test-pod",
					ContainerName: "test-container",
					Message:       "test message",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("test message").
					HasSeverity(inspectioncore.SeverityInfo).
					HasLogType(k8scontainer.LogTypeContainer).
					HasTimestamp(time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC))
			},
		},
		{
			name: "container log with structured klog error",
			input: testlog.NewMockLog(
				time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				inspectioncore.DefaultSeverityFieldSet{
					Severity: inspectioncore.SeverityInfo,
				},
				k8scontainer.K8sContainerLogFieldSet{
					Namespace:     "kube-system",
					PodName:       "kube-apiserver-pod",
					ContainerName: "kube-apiserver",
					Message:       `E0929 08:20:24.205299    1949 server.go:100] "Failed to reconcile" error="timeout"`,
					ParsedMessage: logutil.NewKLogTextParser(true).TryParse(`E0929 08:20:24.205299    1949 server.go:100] "Failed to reconcile" error="timeout"`),
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("Failed to reconcile").
					HasSeverity(inspectioncore.SeverityError).
					HasLogType(k8scontainer.LogTypeContainer)
			},
		},
		{
			name: "container log with structured jsonl warning",
			input: testlog.NewMockLog(
				time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				inspectioncore.DefaultSeverityFieldSet{
					Severity: inspectioncore.SeverityInfo,
				},
				k8scontainer.K8sContainerLogFieldSet{
					Namespace:     "app-namespace",
					PodName:       "app-pod",
					ContainerName: "app-container",
					Message:       `{"level":"warn","msg":"cache degraded"}`,
					ParsedMessage: logutil.NewJsonlTextParser().TryParse(`{"level":"warn","msg":"cache degraded"}`),
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("cache degraded").
					HasSeverity(inspectioncore.SeverityWarning).
					HasLogType(k8scontainer.LogTypeContainer)
			},
		},
		{
			name: "container log with istio envoy access log with error response flag",
			input: testlog.NewMockLog(
				time.Date(2026, 8, 10, 8, 50, 55, 0, time.UTC),
				inspectioncore.DefaultSeverityFieldSet{
					Severity: inspectioncore.SeverityInfo,
				},
				k8scontainer.K8sContainerLogFieldSet{
					Namespace:     "default",
					PodName:       "frontend-pod",
					ContainerName: "istio-proxy",
					Message:       `[2026-08-10T08:50:55.958Z] "GET / HTTP/1.1" 503 UF - - "-" 0 0 6 - "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "10.4.0.5" "10.4.0.5:80" outbound|80||foo.default.svc.cluster.local - - 10.4.1.8:59778 - -`,
					ParsedMessage: logutil.NewEnvoyAccessLogTextParser().TryParse(`[2026-08-10T08:50:55.958Z] "GET / HTTP/1.1" 503 UF - - "-" 0 0 6 - "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "10.4.0.5" "10.4.0.5:80" outbound|80||foo.default.svc.cluster.local - - 10.4.1.8:59778 - -`),
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("【Upstream connection failure(UF)】503 GET http://10.4.0.5/").
					HasSeverity(inspectioncore.SeverityError).
					HasLogType(k8scontainer.LogTypeContainer)
			},
		},
		{
			name: "istio-proxy non-access-log control plane log",
			input: testlog.NewMockLog(
				time.Date(2026, 8, 20, 4, 46, 31, 353884563, time.UTC),
				inspectioncore.DefaultSeverityFieldSet{
					Severity: inspectioncore.SeverityInfo,
				},
				k8scontainer.K8sContainerLogFieldSet{
					Namespace:     "default",
					PodName:       "nginx-server-8646bbcd65-6d969",
					ContainerName: "istio-proxy",
					Message:       "2026-08-20T04:46:31.353537Z\tinfo\txdsproxy\tconnected to upstream XDS server: meshconfig.googleapis.com:443",
					ParsedMessage: (&logutil.FallbackRawTextLogParser{}).TryParse("2026-08-20T04:46:31.353537Z\tinfo\txdsproxy\tconnected to upstream XDS server: meshconfig.googleapis.com:443"),
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("2026-08-20T04:46:31.353537Z\tinfo\txdsproxy\tconnected to upstream XDS server: meshconfig.googleapis.com:443").
					HasSeverity(inspectioncore.SeverityInfo).
					HasLogType(k8scontainer.LogTypeContainer).
					HasTimestamp(time.Date(2026, 8, 20, 4, 46, 31, 353884563, time.UTC))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := processContainerLog(t.Context(), tc.input)
			if err != nil {
				t.Fatalf("processContainerLog() returned unexpected error: %v", err)
			}
			tc.assert(t, cs)
		})
	}
}

// TestLogToTimelineMapper_ProcessLogByGroup tests mapContainerLog, which containerLogTimelineMapper.ProcessLogByGroup delegates to.
func TestLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

	clusterTimeline := k8saudit.MustK8sClusterTimeline(ctx, "test-cluster")
	apiVersionTimeline := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := k8saudit.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
	namespaceTimeline := k8saudit.MustK8sNamespaceTimeline(ctx, kindTimeline, "test-namespace")
	podTimeline := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod")
	expectedPath := k8saudit.MustK8sContainerTimeline(ctx, podTimeline, "test-container")

	testCases := []struct {
		name     string
		inputLog *log.Log
		cluster  k8scommon.GoogleCloudClusterIdentity
		assert   func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "simple container log mapping",
			inputLog: testlog.NewMockLog(
				k8scontainer.K8sContainerLogFieldSet{
					Namespace:     "test-namespace",
					PodName:       "test-pod",
					ContainerName: "test-container",
					Message:       "test message",
				},
			),
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath)
			},
		},
		{
			name: "container log with empty message",
			inputLog: testlog.NewMockLog(
				k8scontainer.K8sContainerLogFieldSet{
					Namespace:     "test-namespace",
					PodName:       "test-pod",
					ContainerName: "test-container",
					Message:       "",
				},
			),
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

			cs, err := mapContainerLog(ctx, tc.inputLog, tc.cluster.ClusterName)
			if err != nil {
				t.Fatalf("mapContainerLog() returned unexpected error: %v", err)
			}
			tc.assert(t, ctx, cs)
		})
	}
}

type mockInitialResourceStateProvider struct {
	states map[k8saudit.ResourceIdentity]*structured.NodeReader
}

func (m *mockInitialResourceStateProvider) InitialResourceState(identity *k8saudit.ResourceIdentity) (*structured.NodeReader, bool) {
	if m == nil || m.states == nil {
		return nil, false
	}
	state, ok := m.states[*identity]
	return state, ok
}

var _ k8saudit.InitialResourceStateProvider = (*mockInitialResourceStateProvider)(nil)

// TestPodPhaseTimelineMapper_ProcessLogByGroup tests mapContainerLogPodPhase, which containerLogPodPhaseTimelineMapper.ProcessLogByGroup delegates to.
func TestPodPhaseTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

	clusterTimeline := k8saudit.MustK8sClusterTimeline(ctx, "test-cluster")
	apiVersionTimeline := k8saudit.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := k8saudit.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
	namespaceTimeline := k8saudit.MustK8sNamespaceTimeline(ctx, kindTimeline, "test-namespace")
	podPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod")
	bindingPath := k8saudit.MustK8sSubresourceTimeline(ctx, podPath, "binding")
	expectedPath := mustPodPhaseTimelinePath(ctx, "test-cluster", "test-node", "test-namespace", "test-pod", "unknown")

	nodeComparer := cmp.Comparer(func(x, y structured.Node) bool {
		if x == nil && y == nil {
			return true
		}
		if x == nil || y == nil {
			return false
		}
		serializer := &structured.JSONNodeSerializer{}
		xBytes, err := serializer.Serialize(x)
		if err != nil {
			return false
		}
		yBytes, err := serializer.Serialize(y)
		if err != nil {
			return false
		}
		return string(xBytes) == string(yBytes)
	})

	makeNamedPodNode := func(podName, nodeName string, labels map[string]string) structured.Node {
		labelsMap := map[string]any{}
		for k, v := range labels {
			labelsMap[k] = v
		}
		manifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      podName,
				"namespace": "test-namespace",
				"labels":    labelsMap,
			},
			"spec": map[string]any{
				"nodeName": nodeName,
			},
		}
		node, err := structured.FromGoValue(manifest, &structured.AlphabeticalGoMapKeyOrderProvider{})
		if err != nil {
			t.Fatalf("failed to generate expected pod manifest: %v", err)
		}
		return node
	}

	makePodNode := func(nodeName string, labels map[string]string) structured.Node {
		return makeNamedPodNode("test-pod", nodeName, labels)
	}

	makeNamedBindingNode := func(podName, nodeName string) structured.Node {
		manifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Binding",
			"metadata": map[string]any{
				"name":      podName,
				"namespace": "test-namespace",
			},
			"target": map[string]any{
				"kind": "Node",
				"name": nodeName,
			},
		}
		node, err := structured.FromGoValue(manifest, &structured.AlphabeticalGoMapKeyOrderProvider{})
		if err != nil {
			t.Fatalf("failed to generate expected binding manifest: %v", err)
		}
		return node
	}

	makeBindingNode := func(nodeName string) structured.Node {
		return makeNamedBindingNode("test-pod", nodeName)
	}

	makeCAIPodNodeReader := func(uid string) *structured.NodeReader {
		manifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"uid": uid,
			},
		}
		node, err := structured.FromGoValue(manifest, &structured.AlphabeticalGoMapKeyOrderProvider{})
		if err != nil {
			t.Fatalf("failed to generate CAI pod node: %v", err)
		}
		return structured.NewNodeReader(node)
	}

	testCases := []struct {
		name                  string
		inputLogs             []*log.Log
		cluster               k8scommon.GoogleCloudClusterIdentity
		initialStateProvider  *mockInitialResourceStateProvider
		timelineCreationTimes func() k8saudit.TimelineCreationTimes
		assert                func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet)
	}{
		{
			name: "skipped because NodeName is empty",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if css[0] != nil {
					t.Errorf("expected cs to be nil, got %v", css[0])
				}
			},
		},
		{
			name: "mapped successfully (no audit log)",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, css[0]).
					HasRevision(expectedPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", nil),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", nil),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer).
					HasRevision(bindingPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeBindingNode("test-node"),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)
			},
		},
		{
			name: "skipped because audit log has Pod resource timeline",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod-audit",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			timelineCreationTimes: func() k8saudit.TimelineCreationTimes {
				auditPodPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod-audit")
				return k8saudit.TimelineCreationTimes{auditPodPath: []time.Time{time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC)}}
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if css[0] != nil {
					t.Errorf("expected cs to be nil, got %v", css[0])
				}
			},
		},
		{
			name: "skipped because audit log has Pod binding timeline",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod-binding",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			timelineCreationTimes: func() k8saudit.TimelineCreationTimes {
				bindingPodPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod-binding")
				subresourcePath := k8saudit.MustK8sSubresourceTimeline(ctx, bindingPodPath, "binding")
				return k8saudit.TimelineCreationTimes{subresourcePath: []time.Time{time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC)}}
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if css[0] != nil {
					t.Errorf("expected cs to be nil, got %v", css[0])
				}
			},
		},
		{
			name: "mapped once and skipped for subsequent logs with same node",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if len(css) != 2 {
					t.Fatalf("expected 2 changesets, got %d", len(css))
				}
				testchangeset.AssertTimeline(t, css[0]).
					HasRevision(expectedPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", nil),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", nil),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer).
					HasRevision(bindingPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeBindingNode("test-node"),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)
				if css[1] != nil {
					t.Errorf("expected second changeset to be nil, got %v", css[1])
				}
			},
		},
		{
			name: "regenerate Pod when labels change",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
						PodLabels: map[string]string{
							"a": "1",
						},
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
						PodLabels: map[string]string{
							"a": "2",
						},
					},
					time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if len(css) != 2 {
					t.Fatalf("expected 2 changesets, got %d", len(css))
				}
				// First log generates all 3
				testchangeset.AssertTimeline(t, css[0]).
					HasRevision(expectedPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", map[string]string{"a": "1"}),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", map[string]string{"a": "1"}),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer).
					HasRevision(bindingPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeBindingNode("test-node"),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)

				// Second log generates only Pod (labels changed, node remained same)
				testchangeset.AssertTimeline(t, css[1]).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
						ResourceBody: makePodNode("test-node", map[string]string{"a": "2"}),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)

				if len(css[1].GetRevisions(bindingPath)) > 0 {
					t.Errorf("expected no revision on bindingPath in second changeset, but got one")
				}

				// Verify PodPhase timeline does NOT have a revision in the second changeset
				if len(css[1].GetRevisions(expectedPath)) > 0 {
					t.Errorf("expected no revision on podPhasePath in second changeset, but got one")
				}
			},
		},
		{
			name: "regenerate all when node changes",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node-1",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node-2",
					},
					time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if len(css) != 2 {
					t.Fatalf("expected 2 changesets, got %d", len(css))
				}
				expectedPath1 := mustPodPhaseTimelinePath(ctx, "test-cluster", "test-node-1", "test-namespace", "test-pod", "unknown")
				expectedPath2 := mustPodPhaseTimelinePath(ctx, "test-cluster", "test-node-2", "test-namespace", "test-pod", "unknown")

				// First log generates all 3 on node 1
				testchangeset.AssertTimeline(t, css[0]).
					HasRevision(expectedPath1, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node-1", nil),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node-1", nil),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer).
					HasRevision(bindingPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeBindingNode("test-node-1"),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)

				// Second log generates all 3 on node 2
				testchangeset.AssertTimeline(t, css[1]).
					HasRevision(expectedPath2, &khifilev6.StagingRevision{
						ChangedTime:  time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
						ResourceBody: makePodNode("test-node-2", nil),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
						ResourceBody: makePodNode("test-node-2", nil),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer).
					HasRevision(bindingPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
						ResourceBody: makeBindingNode("test-node-2"),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)
			},
		},
		{
			name: "CAI known Pod: maps pod phase under node with resolved CAI UID without supplementing Pod or Binding revisions",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod-cai",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			initialStateProvider: &mockInitialResourceStateProvider{
				states: map[k8saudit.ResourceIdentity]*structured.NodeReader{
					{
						APIVersion: "core/v1",
						Kind:       "pod",
						Namespace:  "test-namespace",
						Name:       "test-pod-cai",
					}: makeCAIPodNodeReader("cai-pod-uid-123"),
				},
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				caiPodPhasePath := mustPodPhaseTimelinePath(ctx, "test-cluster", "test-node", "test-namespace", "test-pod-cai", "cai-pod-uid-123")
				caiPodPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod-cai")
				caiBindingPath := k8saudit.MustK8sSubresourceTimeline(ctx, caiPodPath, "binding")

				testchangeset.AssertTimeline(t, css[0]).
					HasRevision(caiPodPhasePath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeNamedPodNode("test-pod-cai", "test-node", nil),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasNoRevision(caiPodPath).
					HasNoRevision(caiBindingPath)
			},
		},
		{
			name: "CAI known Pod with existing CAI revision on podPath: maps pod phase under node",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod-cai-rev",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			initialStateProvider: &mockInitialResourceStateProvider{
				states: map[k8saudit.ResourceIdentity]*structured.NodeReader{
					{
						APIVersion: "core/v1",
						Kind:       "pod",
						Namespace:  "test-namespace",
						Name:       "test-pod-cai-rev",
					}: {},
				},
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				caiPodPhasePath := mustPodPhaseTimelinePath(ctx, "test-cluster", "test-node", "test-namespace", "test-pod-cai-rev", "unknown")
				caiPodPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod-cai-rev")
				caiBindingPath := k8saudit.MustK8sSubresourceTimeline(ctx, caiPodPath, "binding")

				testchangeset.AssertTimeline(t, css[0]).
					HasRevision(caiPodPhasePath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeNamedPodNode("test-pod-cai-rev", "test-node", nil),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUnknown,
						StateType:    k8saudit.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasNoRevision(caiPodPath).
					HasNoRevision(caiBindingPath)
			},
		},
		{
			name: "CAI known Pod with audit log Pod revision: skipped because audit log takes precedence",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod-cai-audit",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			initialStateProvider: &mockInitialResourceStateProvider{
				states: map[k8saudit.ResourceIdentity]*structured.NodeReader{
					{
						APIVersion: "core/v1",
						Kind:       "pod",
						Namespace:  "test-namespace",
						Name:       "test-pod-cai-audit",
					}: {},
				},
			},
			timelineCreationTimes: func() k8saudit.TimelineCreationTimes {
				caiPodPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod-cai-audit")
				return k8saudit.TimelineCreationTimes{caiPodPath: []time.Time{time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC)}}
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if css[0] != nil {
					t.Errorf("expected cs to be nil, got %v", css[0])
				}
			},
		},
		{
			name: "CAI known Pod: subsequent log with only label changes returns nil changeset",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod-cai-labels",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName:  "test-node",
						PodLabels: map[string]string{"a": "1"},
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod-cai-labels",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName:  "test-node",
						PodLabels: map[string]string{"a": "2"},
					},
					time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			initialStateProvider: &mockInitialResourceStateProvider{
				states: map[k8saudit.ResourceIdentity]*structured.NodeReader{
					{
						APIVersion: "core/v1",
						Kind:       "pod",
						Namespace:  "test-namespace",
						Name:       "test-pod-cai-labels",
					}: {},
				},
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if len(css) != 2 {
					t.Fatalf("expected 2 changesets, got %d", len(css))
				}
				if css[0] == nil {
					t.Errorf("expected first changeset to be non-nil")
				}
				if css[1] != nil {
					t.Errorf("expected second changeset to be nil, got %v", css[1])
				}
			},
		},
		{
			name: "CAI known Pod with audit log binding revision: skipped because audit log takes precedence",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod-cai-binding",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			initialStateProvider: &mockInitialResourceStateProvider{
				states: map[k8saudit.ResourceIdentity]*structured.NodeReader{
					{
						APIVersion: "core/v1",
						Kind:       "pod",
						Namespace:  "test-namespace",
						Name:       "test-pod-cai-binding",
					}: {},
				},
			},
			timelineCreationTimes: func() k8saudit.TimelineCreationTimes {
				caiPodPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod-cai-binding")
				subresourcePath := k8saudit.MustK8sSubresourceTimeline(ctx, caiPodPath, "binding")
				return k8saudit.TimelineCreationTimes{subresourcePath: []time.Time{time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC)}}
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if css[0] != nil {
					t.Errorf("expected cs to be nil, got %v", css[0])
				}
			},
		},
		{
			name: "CAI known Pod with existing PodPhase revision from audit log: skipped because audit log takes precedence",
			inputLogs: []*log.Log{
				testlog.NewMockLog(
					k8scontainer.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod-cai-phase",
						ContainerName: "test-container",
					},
					k8scontainer.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				),
			},
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			initialStateProvider: &mockInitialResourceStateProvider{
				states: map[k8saudit.ResourceIdentity]*structured.NodeReader{
					{
						APIVersion: "core/v1",
						Kind:       "pod",
						Namespace:  "test-namespace",
						Name:       "test-pod-cai-phase",
					}: makeCAIPodNodeReader("cai-pod-uid-456"),
				},
			},
			timelineCreationTimes: func() k8saudit.TimelineCreationTimes {
				existingPhasePath := mustPodPhaseTimelinePath(ctx, "test-cluster", "test-node", "test-namespace", "test-pod-cai-phase", "cai-pod-uid-456")
				return k8saudit.TimelineCreationTimes{existingPhasePath: []time.Time{time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC)}}
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if css[0] != nil {
					t.Errorf("expected cs to be nil, got %v", css[0])
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
			var provider k8saudit.InitialResourceStateProvider = &mockInitialResourceStateProvider{}
			if tc.initialStateProvider != nil {
				provider = tc.initialStateProvider
			}

			creationTimes := k8saudit.TimelineCreationTimes{}
			if tc.timelineCreationTimes != nil {
				creationTimes = tc.timelineCreationTimes()
			}

			var css []*khifilev6.TimelineChangeSet
			var state *containerLogPodPhaseMapperState
			for _, l := range tc.inputLogs {
				cs, nextState, err := mapContainerLogPodPhase(ctx, l, state, tc.cluster.ClusterName, provider, creationTimes)
				if err != nil {
					t.Fatalf("mapContainerLogPodPhase() returned unexpected error: %v", err)
				}
				css = append(css, cs)
				state = nextState
			}
			tc.assert(t, ctx, css)
		})
	}
}
