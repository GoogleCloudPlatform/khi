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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestK8sNodeLogIngester_ProcessLog(t *testing.T) {
	ingester := &K8sNodeLogIngester{}
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		desc                 string
		inputMessage         string
		inputNodeLogFieldSet *googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet
		inputPodIDInfo       map[string]*googlecloudlogk8snode_contract.PodSandboxIDInfo
		inputContainerIDInfo map[string]*commonlogk8saudit_contract.ContainerIdentity
		inputResourceUIDInfo map[string]*commonlogk8saudit_contract.ResourceIdentity
		assert               func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			desc:         "standard ingestion of info log",
			inputMessage: "component-A start",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "component-A",
				NodeName:  "node-1",
			},
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("component-A start").
					HasSeverity(inspectioncore_contract.SeverityInfo).
					HasLogType(googlecloudlogk8snode_contract.LogTypeNode)
			},
		},
		{
			desc:         "kubelet log with pod sandbox id",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "Generic (PLEG): container finished" podID="4cba26fb-f074-44fe-9afa-5195e903c337" podID="6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1"`,
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
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary(`Generic (PLEG): container finished 【podname (Namespace: kube-system)】`).
					HasSeverity(inspectioncore_contract.SeverityInfo)
			},
		},
		{
			desc:         "kubelet log with container id",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "ContainerStart: Start container \"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256\"" podID="4cba26fb-f074-44fe-9afa-5195e903c337"`,
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
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary(`ContainerStart: Start container "【fluentbit-gke-init (Pod: podname, Namespace: kube-system)】"`).
					HasSeverity(inspectioncore_contract.SeverityInfo)
			},
		},
		{
			desc:         "kubelet log with pod from klog fields",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "Syncing pod" podID="4cba26fb-f074-44fe-9afa-5195e903c337" msg="Syncing pod" pod="kube-system/podname"`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "kubelet",
				NodeName:  "node-1",
			},
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary(`Syncing pod 【podname (Namespace: kube-system)】`).
					HasSeverity(inspectioncore_contract.SeverityInfo)
			},
		},
		{
			desc:         "kubelet log with pod and container name from klog fields",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "Killing container" podID="4cba26fb-f074-44fe-9afa-5195e903c337" msg="Syncing pod" pod="kube-system/podname" containerName="containername"`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "kubelet",
				NodeName:  "node-1",
			},
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary(`Killing container 【containername (Pod: podname, Namespace: kube-system)】`).
					HasSeverity(inspectioncore_contract.SeverityInfo)
			},
		},
		{
			desc:         "kubelet log with pod and container name from klog fields and exitCode",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "Killing container" podID="4cba26fb-f074-44fe-9afa-5195e903c337" msg="Syncing pod" pod="kube-system/podname" containerName="containername" exitCode=137`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "kubelet",
				NodeName:  "node-1",
			},
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary(`Killing container(exitCode=137) 【containername (Pod: podname, Namespace: kube-system)】`).
					HasSeverity(inspectioncore_contract.SeverityError)
			},
		},
		{
			desc:         "kubelet log with pods klog field",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "log with multiple pods" podID="4cba26fb-f074-44fe-9afa-5195e903c337" msg="Syncing pod" pods=["kube-system/podname1","kube-system/podname2"]`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "kubelet",
				NodeName:  "node-1",
			},
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary(`log with multiple pods 【podname1 (Namespace: kube-system)】 【podname2 (Namespace: kube-system)】`).
					HasSeverity(inspectioncore_contract.SeverityInfo)
			},
		},
		{
			desc:         "kubelet log with pods uid field",
			inputMessage: `I0929 08:30:43.794472    1949 generic.go:334] "log with multiple pods" podID="4cba26fb-f074-44fe-9afa-5195e903c337" msg="Syncing pod"`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "kubelet",
				NodeName:  "node-1",
			},
			inputResourceUIDInfo: map[string]*commonlogk8saudit_contract.ResourceIdentity{
				"4cba26fb-f074-44fe-9afa-5195e903c337": {
					Name:       "podname1",
					Namespace:  "kube-system",
					Kind:       "pod",
					APIVersion: "core/v1",
				},
			},
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary(`log with multiple pods 【podname1 (Namespace: kube-system, APIVersion: core/v1, Kind: pod)】`).
					HasSeverity(inspectioncore_contract.SeverityInfo)
			},
		},
		{
			desc:         "containerd run pod sandbox log ingestion",
			inputMessage: `time="2025-09-29T06:34:07.973711745Z" level=info msg="RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\""`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "containerd",
				NodeName:  "node-1",
			},
			inputPodIDInfo: map[string]*googlecloudlogk8snode_contract.PodSandboxIDInfo{
				"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1": {
					PodName:      "podname",
					PodNamespace: "kube-system",
					PodSandboxID: "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
				},
			},
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary(`RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id "【podname (Namespace: kube-system)】"`).
					HasSeverity(inspectioncore_contract.SeverityInfo)
			},
		},
		{
			desc:         "containerd create container log ingestion",
			inputMessage: `time="2025-09-29T06:34:07.973711745Z" level=info msg="CreateContainer within sandbox \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\" for &ContainerMetadata{Name:fluentbit-gke-init,Attempt:0,} returns container id \"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256\""`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "containerd",
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
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary(`CreateContainer within sandbox "【podname (Namespace: kube-system)】" for &ContainerMetadata{Name:fluentbit-gke-init,Attempt:0,} returns container id "【fluentbit-gke-init (Pod: podname, Namespace: kube-system)】"`).
					HasSeverity(inspectioncore_contract.SeverityInfo)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var podIDFinder patternfinder.PatternFinder[*googlecloudlogk8snode_contract.PodSandboxIDInfo] = patternfinder.NewNaivePatternFinder[*googlecloudlogk8snode_contract.PodSandboxIDInfo]()
			if tc.inputPodIDInfo != nil {
				naiveFinder := patternfinder.NewNaivePatternFinder[*googlecloudlogk8snode_contract.PodSandboxIDInfo]()
				for k, v := range tc.inputPodIDInfo {
					naiveFinder.AddPattern(k, v)
				}
				podIDFinder = naiveFinder
			}

			var containerIDFinder patternfinder.PatternFinder[*commonlogk8saudit_contract.ContainerIdentity] = patternfinder.NewNaivePatternFinder[*commonlogk8saudit_contract.ContainerIdentity]()
			if tc.inputContainerIDInfo != nil {
				naiveFinder := patternfinder.NewNaivePatternFinder[*commonlogk8saudit_contract.ContainerIdentity]()
				for k, v := range tc.inputContainerIDInfo {
					naiveFinder.AddPattern(k, v)
				}
				containerIDFinder = naiveFinder
			}

			var resourceUIDFinder patternfinder.PatternFinder[*commonlogk8saudit_contract.ResourceIdentity] = patternfinder.NewNaivePatternFinder[*commonlogk8saudit_contract.ResourceIdentity]()
			if tc.inputResourceUIDInfo != nil {
				naiveFinder := patternfinder.NewNaivePatternFinder[*commonlogk8saudit_contract.ResourceIdentity]()
				for k, v := range tc.inputResourceUIDInfo {
					naiveFinder.AddPattern(k, v)
				}
				resourceUIDFinder = naiveFinder
			}

			ctx := tasktest.WithTaskResult(t.Context(), googlecloudlogk8snode_contract.PodSandboxIDDiscoveryTaskID.Ref(), podIDFinder)
			ctx = tasktest.WithTaskResult(ctx, commonlogk8saudit_contract.ContainerIDPatternFinderTaskID.Ref(), containerIDFinder)
			ctx = tasktest.WithTaskResult(ctx, commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref(), resourceUIDFinder)

			// Detect type of parser needed for test message
			var message *logutil.ParseStructuredLogResult
			switch tc.inputNodeLogFieldSet.Component {
			case "kubelet":
				klogParser := logutil.NewKLogTextParser(true)
				message = klogParser.TryParse(tc.inputMessage)
			case "containerd":
				logfmtParser := logutil.NewLogfmtTextParser()
				message = logfmtParser.TryParse(tc.inputMessage)
			default:
				rawParser := logutil.FallbackRawTextLogParser{}
				message = rawParser.TryParse(tc.inputMessage)
			}
			tc.inputNodeLogFieldSet.Message = message

			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				tc.inputNodeLogFieldSet,
			)

			cs, err := ingester.ProcessLog(ctx, l)
			if err != nil {
				t.Fatalf("ProcessLog() returned unexpected error: %v", err)
			}

			tc.assert(t, cs)
		})
	}
}
