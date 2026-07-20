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

package googlecloudlogk8scontainer_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

// TestLogIngester_ProcessLog tests the containerLogIngester.ProcessLog function.
func TestLogIngester_ProcessLog(t *testing.T) {
	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "successful container log ingestion",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
					Namespace:     "test-namespace",
					PodName:       "test-pod",
					ContainerName: "test-container",
					Message:       "test message",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("test message").
					HasSeverity(inspectioncore_contract.SeverityInfo).
					HasLogType(googlecloudlogk8scontainer_contract.LogTypeContainer).
					HasTimestamp(time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC))
			},
		},
		{
			name: "container log with structured klog error",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
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
					HasSeverity(inspectioncore_contract.SeverityError).
					HasLogType(googlecloudlogk8scontainer_contract.LogTypeContainer)
			},
		},
		{
			name: "container log with structured jsonl warning",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
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
					HasSeverity(inspectioncore_contract.SeverityWarning).
					HasLogType(googlecloudlogk8scontainer_contract.LogTypeContainer)
			},
		},
	}

	ingester := &containerLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := ingester.ProcessLog(t.Context(), tc.input)
			if err != nil {
				t.Fatalf("ProcessLog() returned unexpected error: %v", err)
			}
			tc.assert(t, cs)
		})
	}
}

// TestLogToTimelineMapper_ProcessLogByGroup tests the containerLogLogToTimelineMapper.ProcessLogByGroup function.
func TestLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
	namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "test-namespace")
	podTimeline := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod")
	expectedPath := commonlogk8saudit_contract.MustK8sContainerTimeline(ctx, podTimeline, "test-container")

	testCases := []struct {
		name     string
		inputLog *log.Log
		cluster  googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		assert   func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "simple container log mapping",
			inputLog: log.NewLogWithFieldSetsForTest(
				&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
					Namespace:     "test-namespace",
					PodName:       "test-pod",
					ContainerName: "test-container",
					Message:       "test message",
				},
			),
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath)
			},
		},
		{
			name: "container log with empty message",
			inputLog: log.NewLogWithFieldSetsForTest(
				&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
					Namespace:     "test-namespace",
					PodName:       "test-pod",
					ContainerName: "test-container",
					Message:       "",
				},
			),
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath)
			},
		},
	}

	mapper := &containerLogLogToTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref(), tc.cluster)

			cs, _, err := mapper.ProcessLogByGroup(ctx, tc.inputLog, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}
			tc.assert(t, ctx, cs)
		})
	}
}

// TestPodPhaseTimelineMapper_ProcessLogByGroup tests the containerLogPodPhaseTimelineMapper.ProcessLogByGroup function.
func TestPodPhaseTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
	namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "test-namespace")
	podPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod")
	bindingPath := commonlogk8saudit_contract.MustK8sSubresourceTimeline(ctx, podPath, "binding")
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

	makePodNode := func(nodeName string, labels map[string]string) structured.Node {
		labelsMap := map[string]any{}
		for k, v := range labels {
			labelsMap[k] = v
		}
		manifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      "test-pod",
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

	makeBindingNode := func(nodeName string) structured.Node {
		manifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Binding",
			"metadata": map[string]any{
				"name":      "test-pod",
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

	testCases := []struct {
		name                   string
		inputLogs              []*log.Log
		cluster                googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		resourceRevisionResult inspectiontaskbase.TimelineMapperResult
		assert                 func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet)
	}{
		{
			name: "skipped because NodeName is empty",
			inputLogs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
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
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, css[0]).
					HasRevision(expectedPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", nil),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", nil),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer).
					HasRevision(bindingPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeBindingNode("test-node"),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)
			},
		},
		{
			name: "skipped because audit log has Pod resource timeline",
			inputLogs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			resourceRevisionResult: inspectiontaskbase.TimelineMapperResult{
				Revisions: map[*khifilev6.TimelinePath]int{
					podPath: 1,
				},
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
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			resourceRevisionResult: inspectiontaskbase.TimelineMapperResult{
				Revisions: map[*khifilev6.TimelinePath]int{
					bindingPath: 1,
				},
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
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
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
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", nil),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer).
					HasRevision(bindingPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeBindingNode("test-node"),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)
				if css[1] != nil {
					t.Errorf("expected second changeset to be nil, got %v", css[1])
				}
			},
		},
		{
			name: "regenerate Pod when labels change",
			inputLogs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
						PodLabels: map[string]string{
							"a": "1",
						},
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
						PodLabels: map[string]string{
							"a": "2",
						},
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
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
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", map[string]string{"a": "1"}),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer).
					HasRevision(bindingPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeBindingNode("test-node"),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)

				// Second log generates only Pod (labels changed, node remained same)
				testchangeset.AssertTimeline(t, css[1]).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node", map[string]string{"a": "2"}),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)

				if css[1].Revisions[bindingPath] != nil {
					t.Errorf("expected no revision on bindingPath in second changeset, but got one")
				}

				// Verify PodPhase timeline does NOT have a revision in the second changeset
				if css[1].Revisions[expectedPath] != nil {
					t.Errorf("expected no revision on podPhasePath in second changeset, but got one")
				}
			},
		},
		{
			name: "regenerate all when node changes",
			inputLogs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node-1",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node-2",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
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
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node-1", nil),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer).
					HasRevision(bindingPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeBindingNode("test-node-1"),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)

				// Second log generates all 3 on node 2
				testchangeset.AssertTimeline(t, css[1]).
					HasRevision(expectedPath2, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node-2", nil),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makePodNode("test-node-2", nil),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer).
					HasRevision(bindingPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: makeBindingNode("test-node-2"),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUnknown,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
					}, nodeComparer)
			},
		},
	}

	mapper := &containerLogPodPhaseTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref(), tc.cluster)
			ctx = tasktest.WithTaskResult(ctx, commonlogk8saudit_contract.ResourceRevisionLogToTimelineMapperTaskID.Ref(), tc.resourceRevisionResult)

			var css []*khifilev6.TimelineChangeSet
			var state *containerLogPodPhaseMapperState
			for _, l := range tc.inputLogs {
				cs, nextState, err := mapper.ProcessLogByGroup(ctx, l, state)
				if err != nil {
					t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
				}
				css = append(css, cs)
				state = nextState
			}
			tc.assert(t, ctx, css)
		})
	}
}
