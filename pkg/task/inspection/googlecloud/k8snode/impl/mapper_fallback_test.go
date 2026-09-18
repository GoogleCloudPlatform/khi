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

package k8snode_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8snode"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestOtherLogLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	mapper := &otherNodeLogLogToTimelineMapperSetting{
		StartingMessagesByComponent: map[string]string{
			"component-A": "component-A start",
		},
		TerminatingMessagesByComponent: map[string]string{
			"component-A": "component-A terminate",
		},
	}
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// 1. Initialize Builder
	builder := khifilev6.NewTestBuilder(id.NewGenerator())

	testCases := []struct {
		desc                 string
		inputMessage         string
		component            string
		nodeName             string
		inputClusterIdentity *k8scommon.GoogleCloudClusterIdentity
		inputPodIDInfo       map[string]*k8snode.PodSandboxIDInfo
		inputContainerIDInfo map[string]*k8saudit.ContainerIdentity
		assert               func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc:         "starting log for component-A adds Create revision and event",
			inputMessage: "component-A start",
			component:    "component-A",
			nodeName:     "node-1",
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "test-cluster", "node-1")
				wantComponentPath := k8snode.MustNodeComponentTimeline(ctx, wantNodePath, "component-A")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath).
					HasRevision(wantComponentPath, &khifilev6.StagingRevision{
						VerbType:    k8saudit.VerbCreate,
						StateType:   k8snode.RevisionStateComponentRunning,
						Principal:   "component-A",
						ChangedTime: testTime,
					})
			},
		},
		{
			desc:         "terminating log for component-A adds Delete revision and event",
			inputMessage: "component-A terminate",
			component:    "component-A",
			nodeName:     "node-1",
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "test-cluster", "node-1")
				wantComponentPath := k8snode.MustNodeComponentTimeline(ctx, wantNodePath, "component-A")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath).
					HasRevision(wantComponentPath, &khifilev6.StagingRevision{
						VerbType:    k8saudit.VerbDelete,
						StateType:   k8snode.RevisionStateComponentTerminated,
						Principal:   "component-A",
						ChangedTime: testTime,
					})
			},
		},
		{
			desc:         "applies cluster prefix policy for GKE on AWS/Azure",
			inputMessage: "component-A start",
			component:    "component-A",
			nodeName:     "node-1",
			inputClusterIdentity: &k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
				PrefixPolicy: k8scommon.ClusterPrefixPolicy{
					Prefix: "awsClusters/",
					RequiredUsages: []k8scommon.ClusterNameUsage{
						k8scommon.ClusterNameUsageK8sCluster,
					},
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "awsClusters/test-cluster", "node-1")
				wantComponentPath := k8snode.MustNodeComponentTimeline(ctx, wantNodePath, "component-A")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath)
			},
		},
		{
			desc:         "gcfsd log associates pod and container timelines",
			inputMessage: `time="2026-10-15T09:15:30.123456789Z" level=info msg="Received TaskCreate event: {ContainerID:fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256 Bundle:/run/containerd/io.containerd.runtime.v2.task/k8s.io/fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256 Pid:54321} from namespace: k8s.io" module=containerdgateway`,
			component:    "gcfsd",
			nodeName:     "node-1",
			inputPodIDInfo: map[string]*k8snode.PodSandboxIDInfo{
				"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1": {
					PodName:      "podname",
					PodNamespace: "kube-system",
					PodSandboxID: "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
				},
			},
			inputContainerIDInfo: map[string]*k8saudit.ContainerIdentity{
				"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256": {
					PodSandboxID:  "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
					ContainerName: "fluentbit-gke-init",
					ContainerID:   "fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256",
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "test-cluster", "node-1")
				wantComponentPath := k8snode.MustNodeComponentTimeline(ctx, wantNodePath, "gcfsd")
				wantPodPath := MustK8sPodTimeline(ctx, "test-cluster", "kube-system", "podname")
				wantContainerPath := k8saudit.MustK8sContainerTimeline(ctx, wantPodPath, "fluentbit-gke-init")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath).
					HasEvent(wantContainerPath)
			},
		},
		{
			desc:         "gcfsd log with pod sandbox id associates pod timeline",
			inputMessage: `time="2026-10-15T09:15:30.123456789Z" level=info msg="failed to mount volume for sandbox 6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1" module=containerdgateway`,
			component:    "gcfsd",
			nodeName:     "node-1",
			inputPodIDInfo: map[string]*k8snode.PodSandboxIDInfo{
				"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1": {
					PodName:      "podname",
					PodNamespace: "kube-system",
					PodSandboxID: "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "test-cluster", "node-1")
				wantComponentPath := k8snode.MustNodeComponentTimeline(ctx, wantNodePath, "gcfsd")
				wantPodPath := MustK8sPodTimeline(ctx, "test-cluster", "kube-system", "podname")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath).
					HasEvent(wantPodPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			podIDFinder := patternfinder.NewNaivePatternFinder[*k8snode.PodSandboxIDInfo]()
			for k, v := range tc.inputPodIDInfo {
				podIDFinder.AddPattern(k, v)
			}
			containerIDFinder := patternfinder.NewNaivePatternFinder[*k8saudit.ContainerIdentity]()
			for k, v := range tc.inputContainerIDInfo {
				containerIDFinder.AddPattern(k, v)
			}

			parser := logutil.FallbackRawTextLogParser{}
			message := parser.TryParse(tc.inputMessage)

			l := testlog.NewMockLog(
				testTime,
				k8snode.K8sNodeLogCommonFieldSet{
					Component: tc.component,
					NodeName:  tc.nodeName,
					Message:   message,
				},
			)

			clusterIdent := k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			}
			if tc.inputClusterIdentity != nil {
				clusterIdent = *tc.inputClusterIdentity
			}

			// 2. Setup context with SAME builder and mock tasks
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, k8snode.ClusterIdentityTaskID.Ref(), clusterIdent)
			ctx = tasktest.WithTaskResult(ctx, k8snode.PodSandboxIDDiscoveryTaskID.Ref(), podIDFinder)
			ctx = tasktest.WithTaskResult(ctx, k8saudit.ContainerIDPatternFinderTaskID.Ref(), containerIDFinder)

			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}

			tc.assert(t, ctx, cs)
		})
	}
}
