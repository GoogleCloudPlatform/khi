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

	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestKubeletLogHistoryModifier(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		desc                 string
		inputNodeLogFieldSet *googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet
		inputPodIDInfo       map[string]*googlecloudlogk8snode_contract.PodSandboxIDInfo
		inputContainerIDInfo map[string]*googlecloudlogk8snode_contract.ContainerIDInfo
		asserter             []testchangeset.ChangeSetAsserter
	}{
		{
			desc: "log with pod sandbox id",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message:   `time="2025-09-29T06:34:07.973711745Z" level=info msg="SyncPod received pod \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\""`,
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
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#kubelet",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: `SyncPod received pod "【podname (Namespace: kube-system)】"`,
				},
			},
		},
		{
			desc: "log with container id",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message:   `time="2025-09-29T06:34:07.973711745Z" level=info msg="ContainerStart: Start container \"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256\""`,
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
			inputContainerIDInfo: map[string]*googlecloudlogk8snode_contract.ContainerIDInfo{
				"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256": {
					PodSandboxID:  "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
					ContainerName: "fluentbit-gke-init",
					ContainerID:   "fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256",
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#kubelet",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname#fluentbit-gke-init",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: `ContainerStart: Start container "【fluentbit-gke-init (Pod:podname, Namespace:kube-system)】"`,
				},
			},
		},
		{
			desc: "log with container id but not in the main message",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message:   `time="2025-09-29T06:34:07.973711745Z" level=info msg="container shim detached" id="fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256"`,
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
			inputContainerIDInfo: map[string]*googlecloudlogk8snode_contract.ContainerIDInfo{
				"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256": {
					PodSandboxID:  "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
					ContainerName: "fluentbit-gke-init",
					ContainerID:   "fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256",
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#kubelet",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname#fluentbit-gke-init",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: `container shim detached 【fluentbit-gke-init (Pod:podname, Namespace:kube-system)】`,
				},
			},
		},
		{
			desc: "log with pod from klog fields",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message:   `time="2025-09-29T06:34:07.973711745Z" level=info msg="Syncing pod" pod="kube-system/podname"`,
				Component: "kubelet",
				NodeName:  "node-1",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#kubelet",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: `Syncing pod 【podname (Namespace: kube-system)】`,
				},
			},
		},
		{
			desc: "log with pod and container name from klog fields",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message:   `time="2025-09-29T06:34:07.973711745Z" level=info msg="Killing container" pod="kube-system/podname" containerName="containername"`,
				Component: "kubelet",
				NodeName:  "node-1",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#kubelet",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname#containername",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: `Killing container 【containername (Pod:podname, Namespace:kube-system)】`,
				},
			},
		},
		{
			desc: "log with pod and container name from klog fields and exitCode",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message:   `time="2025-09-29T06:34:07.973711745Z" level=info msg="Killing container" pod="kube-system/podname" containerName="containername" exitCode="137"`,
				Component: "kubelet",
				NodeName:  "node-1",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#kubelet",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname#containername",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: `Killing container(exitCode=137) 【containername (Pod:podname, Namespace:kube-system)】`,
				},
			},
		},
		{
			desc: "log with pods klog field",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message:   `time="2025-09-29T06:34:07.973711745Z" level=info msg="log with multiple pods" pods="kube-system/podname1,kube-system/podname2"`,
				Component: "kubelet",
				NodeName:  "node-1",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#kubelet",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname1",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname2",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: `log with multiple pods 【podname1 (Namespace: kube-system)】 【podname2 (Namespace: kube-system)】`,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Mock the task results for dependencies
			mockContainerdRelationshipRegistry := googlecloudlogk8snode_contract.NewContainerdRelationshipRegistry()
			if tc.inputPodIDInfo != nil {
				for k, v := range tc.inputPodIDInfo {
					mockContainerdRelationshipRegistry.PodSandboxIDInfoFinder.AddPattern(k, v)
				}
			}
			if tc.inputContainerIDInfo != nil {
				for k, v := range tc.inputContainerIDInfo {
					mockContainerdRelationshipRegistry.ContainerIDInfoFinder.AddPattern(k, v)
				}
			}

			ctx := context.Background()
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogk8snode_contract.ContainerdIDDiscoveryTaskID.Ref(), mockContainerdRelationshipRegistry)

			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				tc.inputNodeLogFieldSet,
			)
			cs := history.NewChangeSet(l)
			modifier := &kubeletNodeLogHistoryModifierSetting{}
			_, err := modifier.ModifyChangeSetFromLog(ctx, l, cs, nil, struct{}{})
			if err != nil {
				t.Fatalf("ModifyChangeSetFromLog() error = %v", err)
			}
			for _, asserter := range tc.asserter {
				asserter.Assert(t, cs)
			}
		})

	}

}
