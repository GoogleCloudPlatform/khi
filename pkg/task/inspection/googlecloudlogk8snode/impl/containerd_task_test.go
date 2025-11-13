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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestProcessPodSandboxIDDiscoveryForLog(t *testing.T) {
	podSandboxID := "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1"
	testCases := []struct {
		desc                   string
		inputComponentFieldSet *googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet
		want                   *googlecloudlogk8snode_contract.PodSandboxIDInfo
	}{
		{
			desc: "valid log message",
			inputComponentFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message: &logutil.ParseStructuredLogResult{
					Fields: map[string]any{
						logutil.MainMessageStructuredFieldKey: "RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"",
					},
				},
			},
			want: &googlecloudlogk8snode_contract.PodSandboxIDInfo{
				PodName:      "podname",
				PodNamespace: "kube-system",
				PodSandboxID: podSandboxID,
			},
		},
		{
			desc: "empty message",
			inputComponentFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message: &logutil.ParseStructuredLogResult{
					Fields: map[string]any{
						logutil.MainMessageStructuredFieldKey: "",
					},
				},
			},
			want: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(tc.inputComponentFieldSet)
			repository := googlecloudlogk8snode_contract.NewContainerdRelationshipRegistry()
			processPodSandboxIDDiscoveryForLog(t.Context(), l, repository)

			got, _ := repository.PodSandboxIDInfoFinder.GetPattern(podSandboxID)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("PodSandboxIDInfoFinder mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindPodSandboxIDInfo(t *testing.T) {
	testCases := []struct {
		desc    string
		log     string
		want    *googlecloudlogk8snode_contract.PodSandboxIDInfo
		wantErr bool
	}{
		{
			desc: "valid log message",
			log:  "RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\"",
			want: &googlecloudlogk8snode_contract.PodSandboxIDInfo{
				PodName:      "podname",
				PodNamespace: "kube-system",
				PodSandboxID: "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
			},
			wantErr: false,
		},
		{
			desc:    "log message without RunPodSandbox prefix",
			log:     "Some other log message",
			want:    nil,
			wantErr: true,
		},
		{
			desc:    "log message without sandbox id",
			log:     "RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,}",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			input := logutil.ParseStructuredLogResult{
				Fields: map[string]any{
					logutil.MainMessageStructuredFieldKey: tc.log,
				},
			}
			got, err := findPodSandboxIDInfo(&input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("findPodSandboxIDInfo() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("findPodSandboxIDInfo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessContainerIDDiscoveryForLog(t *testing.T) {
	containerID := "container123"
	testCases := []struct {
		desc                   string
		inputComponentFieldSet *googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet
		want                   *googlecloudlogk8snode_contract.ContainerIDInfo
	}{
		{
			desc: "valid log message",
			inputComponentFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message: &logutil.ParseStructuredLogResult{
					Fields: map[string]any{
						logutil.MainMessageStructuredFieldKey: "CreateContainer within sandbox \"sandbox123\" for &ContainerMetadata{Name:container-name,Attempt:0,} returns container id \"container123\"",
					},
				},
			},
			want: &googlecloudlogk8snode_contract.ContainerIDInfo{
				PodSandboxID:  "sandbox123",
				ContainerID:   "container123",
				ContainerName: "container-name",
			},
		},
		{
			desc: "empty log message",
			inputComponentFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Message: &logutil.ParseStructuredLogResult{
					Fields: map[string]any{
						logutil.MainMessageStructuredFieldKey: "",
					},
				},
			},
			want: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(tc.inputComponentFieldSet)
			repository := googlecloudlogk8snode_contract.NewContainerdRelationshipRegistry()
			processContainerIDDiscoveryForLog(t.Context(), l, repository)

			got, _ := repository.ContainerIDInfoFinder.GetPattern(containerID)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ContainerIDInfoFinder mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindContainerIDInfo(t *testing.T) {
	testCases := []struct {
		desc    string
		log     string
		want    *googlecloudlogk8snode_contract.ContainerIDInfo
		wantErr bool
	}{
		{
			desc: "valid log message",
			log:  "CreateContainer within sandbox \"sandbox123\" for &ContainerMetadata{Name:container-name,Attempt:0,} returns container id \"container123\"",
			want: &googlecloudlogk8snode_contract.ContainerIDInfo{
				PodSandboxID:  "sandbox123",
				ContainerName: "container-name",
				ContainerID:   "container123",
			},
			wantErr: false,
		},
		{
			desc:    "log message without CreateContainer prefix",
			log:     "Some other log message",
			want:    nil,
			wantErr: true,
		},
		{
			desc:    "log message without sandbox id",
			log:     "CreateContainer for &ContainerMetadata{Name:container-name,Attempt:0,} returns container id \"container123\"",
			want:    nil,
			wantErr: true,
		},
		{
			desc:    "log message without container id",
			log:     "CreateContainer within sandbox \"sandbox123\" for &ContainerMetadata{Name:container-name,Attempt:0,}",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			input := logutil.ParseStructuredLogResult{
				Fields: map[string]any{
					logutil.MainMessageStructuredFieldKey: tc.log,
				},
			}
			got, err := findContainerIDInfo(&input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("findContainerIDInfo() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("findContainerIDInfo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestContainerdIDDiscoveryTask(t *testing.T) {
	testCases := []struct {
		desc              string
		messages          []string
		wantPodInfo       map[string]googlecloudlogk8snode_contract.PodSandboxIDInfo
		wantContainerInfo map[string]googlecloudlogk8snode_contract.ContainerIDInfo
	}{
		{
			desc: "single pod sandbox and container discovery",
			messages: []string{
				`time="2025-09-29T06:34:07.973711745Z" level=info msg="RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\""`,
				`time="2025-09-29T06:34:07.973711745Z" level=info msg="CreateContainer within sandbox \"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1\" for &ContainerMetadata{Name:fluentbit-gke-init,Attempt:0,} returns container id \"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256\""`,
			},
			wantPodInfo: map[string]googlecloudlogk8snode_contract.PodSandboxIDInfo{
				"6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1": {
					PodName:      "podname",
					PodNamespace: "kube-system",
					PodSandboxID: "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
				},
			},
			wantContainerInfo: map[string]googlecloudlogk8snode_contract.ContainerIDInfo{
				"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256": {
					PodSandboxID:  "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
					ContainerName: "fluentbit-gke-init",
					ContainerID:   "fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			logs := []*log.Log{}
			logfmtParser := logutil.NewLogfmtTextParser()
			for _, msg := range tc.messages {
				input := logfmtParser.TryParse(msg)
				logs = append(logs, log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
						Message: input,
					},
				))
			}

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.RunInspectionTask(ctx, ContainerdIDDiscoveryTask, inspectioncore_contract.TaskModeRun, map[string]any{}, tasktest.NewTaskDependencyValuePair(
				googlecloudlogk8snode_contract.ContainerdLogFilterTaskID.Ref(), logs,
			))
			if err != nil {
				t.Fatalf("ContainerdIDDiscoveryTask error = %v", err)
			}

			for k, wantPod := range tc.wantPodInfo {
				gotPod, err := got.PodSandboxIDInfoFinder.GetPattern(k)
				if err != nil {
					t.Errorf("PodSandboxIDInfoFinder.GetPattern(%s) error = %v", k, err)
				}
				if diff := cmp.Diff(wantPod, *gotPod); diff != "" {
					t.Errorf("PodSandboxIDInfoFinder mismatch for key %s (-want +got):\n%s", k, diff)
				}
			}
			for k, wantContainer := range tc.wantContainerInfo {
				gotContainer, err := got.ContainerIDInfoFinder.GetPattern(k)
				if err != nil {
					t.Errorf("ContainerIDInfoFinder.GetPattern(%s) error = %v", k, err)
				}
				if diff := cmp.Diff(wantContainer, *gotContainer); diff != "" {
					t.Errorf("ContainerIDInfoFinder mismatch for key %s (-want +got):\n%s", k, diff)
				}
			}
		})
	}

}

func TestContainerdHistoryModifierTask(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		desc                 string
		inputMessage         string
		inputNodeLogFieldSet *googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet
		inputPodIDInfo       map[string]*googlecloudlogk8snode_contract.PodSandboxIDInfo
		inputContainerIDInfo map[string]*googlecloudlogk8snode_contract.ContainerIDInfo
		asserter             []testchangeset.ChangeSetAsserter
	}{
		{
			desc:         "starting log for containerd",
			inputMessage: `time="2025-09-29T06:34:07.973711745Z" level=info msg="starting containerd"`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "containerd",
				NodeName:  "node-1",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#containerd",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateExisting,
						Requestor:  "containerd",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#containerd",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "starting containerd",
				},
			},
		},
		{
			desc:         "terminating log for containerd",
			inputMessage: `time="2025-09-29T06:34:07.973711745Z" level=info msg="Stop CRI service"`,
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "containerd",
				NodeName:  "node-1",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#containerd",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						Requestor:  "containerd",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#containerd",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "Stop CRI service",
				},
			},
		},
		{
			desc:         "log with pod sandbox id",
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
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#containerd",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: `RunPodSandbox for &PodSandboxMetadata{Name:podname,Uid:b86b49f2431d244c613996c6472eb864,Namespace:kube-system,Attempt:0,} returns sandbox id "【podname (Namespace: kube-system)】"`,
				},
			},
		},
		{
			desc:         "log with container id",
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
			inputContainerIDInfo: map[string]*googlecloudlogk8snode_contract.ContainerIDInfo{
				"fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256": {
					PodSandboxID:  "6123c6aacf0c78dc38ec4f0ff72edd3cf04eb82ca0e3e7dddd3950ea9753bdf1",
					ContainerName: "fluentbit-gke-init",
					ContainerID:   "fc3e6702e38e918ec02567358c4c889b38fc628838645222d9a08b0b68c90256",
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#containerd",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname",
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#kube-system#podname#fluentbit-gke-init",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: `CreateContainer within sandbox "【podname (Namespace: kube-system)】" for &ContainerMetadata{Name:fluentbit-gke-init,Attempt:0,} returns container id "【fluentbit-gke-init (Pod:podname, Namespace:kube-system)】"`,
				},
			},
		},
	}
	for _, tc := range testCases {
		logfmtParser := logutil.NewLogfmtTextParser()
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
			message := logfmtParser.TryParse(tc.inputMessage)
			tc.inputNodeLogFieldSet.Message = message
			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				tc.inputNodeLogFieldSet,
			)
			cs := history.NewChangeSet(l)
			modifier := &containerdNodeLogHistoryModifierSetting{}
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
