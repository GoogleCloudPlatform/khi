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

package googlecloudlogk8snode_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestKubeletLogLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	mapper := &kubeletNodeLogLogToTimelineMapperSetting{}
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	builder := khifilev6.NewBuilder()

	testCases := []struct {
		desc                 string
		inputMessage         string
		inputNodeLogFieldSet *googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet
		inputClusterIdentity *googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		inputPodIDInfo       map[string]*googlecloudlogk8snode_contract.PodSandboxIDInfo
		inputContainerIDInfo map[string]*commonlogk8saudit_contract.ContainerIdentity
		inputResourceUIDInfo map[string]*commonlogk8saudit_contract.ResourceIdentity
		assert               func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc:         "adds pod sandbox timeline event and node component event",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "Generic (PLEG): container finished" podID="6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1"`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "kubelet",
				NodeName:  "node-1",
			},
			inputPodIDInfo: map[string]*googlecloudlogk8snode_contract.PodSandboxIDInfo{
				"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1": {
					PodName:      "podname",
					PodNamespace: "kube-system",
					PodSandboxID: "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "test-cluster", "node-1")
				wantComponentPath := googlecloudlogk8snode_contract.MustNodeComponentTimeline(ctx, wantNodePath, "kubelet")
				wantPodPath := MustK8sPodTimeline(ctx, "test-cluster", "kube-system", "podname")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath).
					HasEvent(wantPodPath)
			},
		},
		{
			desc:         "adds container timeline event",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "ContainerStart: Start container \"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256\""`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "kubelet",
				NodeName:  "node-1",
			},
			inputPodIDInfo: map[string]*googlecloudlogk8snode_contract.PodSandboxIDInfo{
				"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1": {
					PodName:      "podname",
					PodNamespace: "kube-system",
					PodSandboxID: "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
				},
			},
			inputContainerIDInfo: map[string]*commonlogk8saudit_contract.ContainerIdentity{
				"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256": {
					PodSandboxID:  "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
					ContainerName: "fluentbit-gke-init",
					ContainerID:   "fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256",
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "test-cluster", "node-1")
				wantComponentPath := googlecloudlogk8snode_contract.MustNodeComponentTimeline(ctx, wantNodePath, "kubelet")
				wantPodPath := MustK8sPodTimeline(ctx, "test-cluster", "kube-system", "podname")
				wantContainerPath := commonlogk8saudit_contract.MustK8sContainerTimeline(ctx, wantPodPath, "fluentbit-gke-init")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath).
					HasEvent(wantContainerPath)
			},
		},
		{
			desc:         "adds custom resource UID event",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "log with custom resource" podID="4cba26fb-f074-44fe-9afa-5195e903c337"`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "kubelet",
				NodeName:  "node-1",
			},
			inputResourceUIDInfo: map[string]*commonlogk8saudit_contract.ResourceIdentity{
				"4cba26fb-f074-44fe-9afa-5195e903c337": {
					Name:       "my-custom-res",
					Namespace:  "default",
					Kind:       "mykind",
					APIVersion: "custom.api/v1",
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "test-cluster", "node-1")
				wantComponentPath := googlecloudlogk8snode_contract.MustNodeComponentTimeline(ctx, wantNodePath, "kubelet")

				wantResourceIdent := &commonlogk8saudit_contract.ResourceIdentity{
					Name:       "my-custom-res",
					Namespace:  "default",
					Kind:       "mykind",
					APIVersion: "custom.api/v1",
				}
				wantResourcePath := commonlogk8saudit_contract.MustResourceTimeline(ctx, "test-cluster", wantResourceIdent)

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath).
					HasEvent(wantResourcePath)
			},
		},
		{
			desc:         "applies cluster prefix policy for GKE on AWS/Azure",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "Generic (PLEG): container finished" podID="6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1"`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "kubelet",
				NodeName:  "node-1",
			},
			inputClusterIdentity: &googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
				PrefixPolicy: googlecloudk8scommon_contract.ClusterPrefixPolicy{
					Prefix: "awsClusters/",
					RequiredUsages: []googlecloudk8scommon_contract.ClusterNameUsage{
						googlecloudk8scommon_contract.ClusterNameUsageK8sCluster,
					},
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "awsClusters/test-cluster", "node-1")
				wantComponentPath := googlecloudlogk8snode_contract.MustNodeComponentTimeline(ctx, wantNodePath, "kubelet")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			podIDFinder := patternfinder.NewNaivePatternFinder[*googlecloudlogk8snode_contract.PodSandboxIDInfo]()
			if tc.inputPodIDInfo != nil {
				for k, v := range tc.inputPodIDInfo {
					podIDFinder.AddPattern(k, v)
				}
			}
			containerIDFinder := patternfinder.NewNaivePatternFinder[*commonlogk8saudit_contract.ContainerIdentity]()
			if tc.inputContainerIDInfo != nil {
				for k, v := range tc.inputContainerIDInfo {
					containerIDFinder.AddPattern(k, v)
				}
			}
			finder := patternfinder.NewNaivePatternFinder[*commonlogk8saudit_contract.ResourceIdentity]()
			if tc.inputResourceUIDInfo != nil {
				for k, v := range tc.inputResourceUIDInfo {
					finder.AddPattern(k, v)
				}
			}

			clusterIdent := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			}
			if tc.inputClusterIdentity != nil {
				clusterIdent = *tc.inputClusterIdentity
			}

			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogk8snode_contract.ClusterIdentityTaskID.Ref(), clusterIdent)
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogk8snode_contract.PodSandboxIDDiscoveryTaskID.Ref(), podIDFinder)
			ctx = tasktest.WithTaskResult(ctx, commonlogk8saudit_contract.ContainerIDPatternFinderTaskID.Ref(), containerIDFinder)
			ctx = tasktest.WithTaskResult(ctx, commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref(), finder)

			klogParser := logutil.NewKLogTextParser(true)
			message := klogParser.TryParse(tc.inputMessage)
			tc.inputNodeLogFieldSet.Message = message

			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				tc.inputNodeLogFieldSet,
			)

			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}

			tc.assert(t, ctx, cs)
		})
	}
}
