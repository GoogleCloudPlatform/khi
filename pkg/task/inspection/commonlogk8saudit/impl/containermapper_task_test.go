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

package commonlogk8saudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestContainerStateWalkerV2(t *testing.T) {
	podNamespace := "default"
	podName := "nginx"
	containerName := "nginx-container"
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	nodeComparer := cmp.Comparer(func(a, b structured.Node) bool {
		if a == nil || b == nil {
			return a == b
		}
		aYAML, errA := structured.NewNodeReader(a).Serialize("", &structured.YAMLNodeSerializer{})
		bYAML, errB := structured.NewNodeReader(b).Serialize("", &structured.YAMLNodeSerializer{})
		if errA != nil || errB != nil {
			return false
		}
		return string(aYAML) == string(bYAML)
	})

	parseYAML := func(yamlStr string) structured.Node {
		if yamlStr == "" {
			return nil
		}
		node, err := structured.FromYAML(yamlStr)
		if err != nil {
			t.Fatalf("failed to parse YAML: %v", err)
		}
		return node
	}

	tests := []struct {
		name  string
		steps []struct {
			yaml      string
			timestamp time.Time
			verb      *pb.Verb
		}
		wantRevisions []*khifilev6.StagingRevision
	}{
		{
			name: "Container Waiting",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      *pb.Verb
			}{
				{
					yaml: `
name: nginx-container
state:
  waiting:
    reason: ContainerCreating
`,
					timestamp: baseTime,
					verb:      commonlogk8saudit_contract.VerbUpdate,
				},
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerWaiting,
					Principal:    "user-1",
					ChangedTime:  baseTime,
					ResourceBody: parseYAML("name: nginx-container\nstate:\n  waiting:\n    reason: ContainerCreating\n"),
				},
			},
		},
		{
			name: "Container Running (Ready)",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      *pb.Verb
			}{
				{
					yaml: `
name: nginx-container
ready: true
state:
  running:
    startedAt: "2024-01-01T00:00:00Z"
`,
					timestamp: baseTime.Add(time.Minute),
					verb:      commonlogk8saudit_contract.VerbUpdate,
				},
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerStarted,
					Principal:    "user-1",
					ChangedTime:  baseTime,
					ResourceBody: parseYAML("name: nginx-container\nready: true\nstate:\n  running:\n    startedAt: \"2024-01-01T00:00:00Z\"\n"),
				},
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerRunningReady,
					Principal:    "user-1",
					ChangedTime:  baseTime.Add(time.Minute),
					ResourceBody: parseYAML("name: nginx-container\nready: true\nstate:\n  running:\n    startedAt: \"2024-01-01T00:00:00Z\"\n"),
				},
			},
		},
		{
			name: "Container Running (Not Ready)",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      *pb.Verb
			}{
				{
					yaml: `
name: nginx-container
ready: false
state:
  running:
    startedAt: "2024-01-01T00:00:00Z"
`,
					timestamp: baseTime.Add(time.Minute),
					verb:      commonlogk8saudit_contract.VerbUpdate,
				},
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerStarted,
					Principal:    "user-1",
					ChangedTime:  baseTime,
					ResourceBody: parseYAML("name: nginx-container\nready: false\nstate:\n  running:\n    startedAt: \"2024-01-01T00:00:00Z\"\n"),
				},
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerRunningNonReady,
					Principal:    "user-1",
					ChangedTime:  baseTime.Add(time.Minute),
					ResourceBody: parseYAML("name: nginx-container\nready: false\nstate:\n  running:\n    startedAt: \"2024-01-01T00:00:00Z\"\n"),
				},
			},
		},
		{
			name: "Container Terminated (Success)",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      *pb.Verb
			}{
				{
					yaml: `
name: nginx-container
state:
  terminated:
    exitCode: 0
    startedAt: "2024-01-01T00:00:00Z"
    finishedAt: "2024-01-01T01:00:00Z"
`,
					timestamp: baseTime.Add(2 * time.Hour),
					verb:      commonlogk8saudit_contract.VerbUpdate,
				},
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerStarted,
					Principal:    "user-1",
					ChangedTime:  baseTime,
					ResourceBody: parseYAML("name: nginx-container\nstate:\n  terminated:\n    exitCode: 0\n    startedAt: \"2024-01-01T00:00:00Z\"\n    finishedAt: \"2024-01-01T01:00:00Z\"\n"),
				},
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerTerminatedWithSuccess,
					Principal:    "user-1",
					ChangedTime:  baseTime.Add(time.Hour),
					ResourceBody: parseYAML("name: nginx-container\nstate:\n  terminated:\n    exitCode: 0\n    startedAt: \"2024-01-01T00:00:00Z\"\n    finishedAt: \"2024-01-01T01:00:00Z\"\n"),
				},
			},
		},
		{
			name: "Container Terminated (Error)",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      *pb.Verb
			}{
				{
					yaml: `
name: nginx-container
state:
  terminated:
    exitCode: 1
    startedAt: "2024-01-01T00:00:00Z"
    finishedAt: "2024-01-01T01:00:00Z"
`,
					timestamp: baseTime.Add(2 * time.Hour),
					verb:      commonlogk8saudit_contract.VerbUpdate,
				},
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerStarted,
					Principal:    "user-1",
					ChangedTime:  baseTime,
					ResourceBody: parseYAML("name: nginx-container\nstate:\n  terminated:\n    exitCode: 1\n    startedAt: \"2024-01-01T00:00:00Z\"\n    finishedAt: \"2024-01-01T01:00:00Z\"\n"),
				},
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerTerminatedWithError,
					Principal:    "user-1",
					ChangedTime:  baseTime.Add(time.Hour),
					ResourceBody: parseYAML("name: nginx-container\nstate:\n  terminated:\n    exitCode: 1\n    startedAt: \"2024-01-01T00:00:00Z\"\n    finishedAt: \"2024-01-01T01:00:00Z\"\n"),
				},
			},
		},
		{
			name: "Transition: Waiting -> Running -> Terminated",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      *pb.Verb
			}{
				{
					yaml: `
name: nginx-container
state:
  waiting:
    reason: ContainerCreating
`,
					timestamp: baseTime,
					verb:      commonlogk8saudit_contract.VerbUpdate,
				},
				{
					yaml: `
name: nginx-container
ready: true
state:
  running:
    startedAt: "2024-01-01T00:01:00Z"
`,
					timestamp: baseTime.Add(2 * time.Minute),
					verb:      commonlogk8saudit_contract.VerbUpdate,
				},
				{
					yaml: `
name: nginx-container
state:
  terminated:
    exitCode: 0
    startedAt: "2024-01-01T00:01:00Z"
    finishedAt: "2024-01-01T00:10:00Z"
`,
					timestamp: baseTime.Add(15 * time.Minute),
					verb:      commonlogk8saudit_contract.VerbUpdate,
				},
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerWaiting,
					Principal:    "user-1",
					ChangedTime:  baseTime,
					ResourceBody: parseYAML("name: nginx-container\nstate:\n  waiting:\n    reason: ContainerCreating\n"),
				},
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerStarted,
					Principal:    "user-1",
					ChangedTime:  baseTime.Add(time.Minute),
					ResourceBody: parseYAML("name: nginx-container\nready: true\nstate:\n  running:\n    startedAt: \"2024-01-01T00:01:00Z\"\n"),
				},
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerRunningReady,
					Principal:    "user-1",
					ChangedTime:  baseTime.Add(2 * time.Minute),
					ResourceBody: parseYAML("name: nginx-container\nready: true\nstate:\n  running:\n    startedAt: \"2024-01-01T00:01:00Z\"\n"),
				},
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerTerminatedWithSuccess,
					Principal:    "user-1",
					ChangedTime:  baseTime.Add(10 * time.Minute),
					ResourceBody: parseYAML("name: nginx-container\nstate:\n  terminated:\n    exitCode: 0\n    startedAt: \"2024-01-01T00:01:00Z\"\n    finishedAt: \"2024-01-01T00:10:00Z\"\n"),
				},
			},
		},
		{
			name: "No State (Initial)",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      *pb.Verb
			}{
				{
					yaml:      "", // No state reader
					timestamp: baseTime,
					verb:      commonlogk8saudit_contract.VerbUpdate,
				},
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerStatusNotAvailable,
					Principal:    "user-1",
					ChangedTime:  baseTime,
					ResourceBody: nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			containerPath := MustResolveContainerTimelinePath(ctx, "k8s", podNamespace, podName, containerName)

			walker := &containerStateWalkerV2{
				containerIdentity: &containerStatusIdentity{
					containerName: containerName,
					containerType: ContainerTypeContainer,
				},
				podNamespace: podNamespace,
				podName:      podName,
			}

			l := log.NewLogWithFieldSetsForTest()
			cs := khifilev6.NewTimelineChangeSet(l)

			for _, step := range tt.steps {
				commonFieldSet := &log.CommonFieldSet{
					Timestamp: step.timestamp,
				}
				k8sFieldSet := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
					Verb:        step.verb,
					Principal:   "user-1",
					ClusterName: "k8s",
				}

				var stateReader *structured.NodeReader
				if step.yaml != "" {
					node, err := structured.FromYAML(step.yaml)
					if err != nil {
						t.Fatalf("failed to parse YAML: %v", err)
					}
					stateReader = structured.NewNodeReader(node)
				}

				walker.CheckAndRecord(ctx, stateReader, cs, commonFieldSet, k8sFieldSet)
			}

			if len(tt.wantRevisions) == 0 {
				testchangeset.AssertTimeline(t, cs).HasNoRevision(containerPath)
			} else {
				for _, want := range tt.wantRevisions {
					testchangeset.AssertTimeline(t, cs).HasRevision(containerPath, want, nodeComparer)
				}
			}
		})
	}
}

func TestContainerLogToTimelineMapperTaskV2_ProcessLog(t *testing.T) {
	taskSetting := &containerLogToTimelineMapperTaskSettingV2{}
	podNamespace := "default"
	podName := "nginx"
	testTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	nodeComparer := cmp.Comparer(func(a, b structured.Node) bool {
		if a == nil || b == nil {
			return a == b
		}
		aYAML, errA := structured.NewNodeReader(a).Serialize("", &structured.YAMLNodeSerializer{})
		bYAML, errB := structured.NewNodeReader(b).Serialize("", &structured.YAMLNodeSerializer{})
		if errA != nil || errB != nil {
			return false
		}
		return string(aYAML) == string(bYAML)
	})

	parseYAML := func(yamlStr string) structured.Node {
		if yamlStr == "" {
			return nil
		}
		node, err := structured.FromYAML(yamlStr)
		if err != nil {
			t.Fatalf("failed to parse YAML: %v", err)
		}
		return node
	}

	tests := []struct {
		name          string
		pass          int
		yaml          string
		nilBody       bool
		eventType     commonlogk8saudit_contract.ChangeEventTypeV2
		verb          *pb.Verb
		initialState  *containerLogToTimelineMapperTaskStateV2
		wantState     *containerLogToTimelineMapperTaskStateV2
		wantRevisions []*khifilev6.StagingRevision
	}{
		{
			name: "Pass 0: Collect Identities",
			pass: 0,
			yaml: `
status:
  containerStatuses:
  - name: main-container
  initContainerStatuses:
  - name: init-container
  ephemeralContainerStatuses:
  - name: debug-container
`,
			initialState: nil,
			wantState: &containerLogToTimelineMapperTaskStateV2{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
					"init-container": {
						containerName: "init-container",
						containerType: ContainerTypeInitContainer,
					},
					"debug-container": {
						containerName: "debug-container",
						containerType: ContainerTypeEphemeral,
					},
				},
				containerStateWalkers: map[string]*containerStateWalkerV2{},
			},
			wantRevisions: []*khifilev6.StagingRevision{},
		},
		{
			name:    "Pass 0: Nil Body",
			pass:    0,
			nilBody: true,
			wantState: &containerLogToTimelineMapperTaskStateV2{
				containerIdentities:   map[string]*containerStatusIdentity{},
				containerStateWalkers: map[string]*containerStateWalkerV2{},
			},
			wantRevisions: []*khifilev6.StagingRevision{},
		},
		{
			name: "Process Containers",
			pass: 1,
			yaml: `
status:
  containerStatuses:
  - name: main-container
    state:
      running:
        startedAt: "2024-01-01T00:00:00Z"
    ready: true
`,
			initialState: &containerLogToTimelineMapperTaskStateV2{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
				},
				containerStateWalkers: map[string]*containerStateWalkerV2{},
			},
			wantState: &containerLogToTimelineMapperTaskStateV2{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
				},
				containerStateWalkers: map[string]*containerStateWalkerV2{
					"main-container": {
						containerIdentity: &containerStatusIdentity{
							containerName: "main-container",
							containerType: ContainerTypeContainer,
						},
						podNamespace:   podNamespace,
						podName:        podName,
						lastState:      "ready",
						lastStartTime:  "2024-01-01T00:00:00Z",
						lastFinishTime: "",
					},
				},
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerStarted,
					Principal:    "user-1",
					ChangedTime:  testTime,
					ResourceBody: parseYAML("name: main-container\nstate:\n  running:\n    startedAt: \"2024-01-01T00:00:00Z\"\nready: true\n"),
				},
			},
		},
		{
			name: "Missing Status",
			pass: 1,
			yaml: `
status:
  containerStatuses: []
`,
			initialState: &containerLogToTimelineMapperTaskStateV2{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
				},
				containerStateWalkers: map[string]*containerStateWalkerV2{},
			},
			wantState: &containerLogToTimelineMapperTaskStateV2{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
				},
				containerStateWalkers: map[string]*containerStateWalkerV2{
					"main-container": {
						containerIdentity: &containerStatusIdentity{
							containerName: "main-container",
							containerType: ContainerTypeContainer,
						},
						podNamespace:   podNamespace,
						podName:        podName,
						lastState:      "no state",
						lastStartTime:  "",
						lastFinishTime: "",
					},
				},
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerStatusNotAvailable,
					Principal:    "user-1",
					ChangedTime:  testTime,
					ResourceBody: nil,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			containerPath := MustResolveContainerTimelinePath(ctx, "k8s", podNamespace, podName, "main-container")

			var reader *structured.NodeReader
			if !tc.nilBody {
				node, err := structured.FromYAML(tc.yaml)
				if err != nil {
					t.Fatalf("failed to parse YAML: %v", err)
				}
				reader = structured.NewNodeReader(node)
			}

			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{},
				&commonlogk8saudit_contract.K8sAuditLogFieldSet{},
			)
			commonFieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
			commonFieldSet.Timestamp = testTime
			k8sFieldSet := log.MustGetFieldSet(l, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})

			verb := tc.verb
			if verb == nil {
				verb = commonlogk8saudit_contract.VerbUpdate
			}
			k8sFieldSet.Verb = verb
			k8sFieldSet.Principal = "user-1"
			k8sFieldSet.ClusterName = "k8s"

			resIdentity := &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  podNamespace,
				Name:       podName,
			}

			groupSet := commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"pod": {
						Resource: resIdentity,
						Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
							{Log: l, ResourceBodyReader: reader, ResourceBodyYAML: tc.yaml},
						},
					},
				},
			}

			event := commonlogk8saudit_contract.MultiGroupLogEvent{
				Log:              l,
				GroupRole:        "pod",
				ResourceIdentity: resIdentity,
				EventType:        tc.eventType,
				GroupSet:         groupSet,
			}

			if tc.pass == 0 {
				nextState, err := taskSetting.PreProcessLog(ctx, 0, event, tc.initialState)
				if err != nil {
					t.Fatalf("PreProcessLog failed: %v", err)
				}
				if diff := cmp.Diff(tc.wantState, nextState, cmp.AllowUnexported(containerLogToTimelineMapperTaskStateV2{}, containerStatusIdentity{}, containerStateWalkerV2{})); diff != "" {
					t.Errorf("state mismatch (-want +got):\n%s", diff)
				}
			} else {
				cs, nextState, err := taskSetting.ProcessLog(ctx, event, tc.initialState)
				if err != nil {
					t.Fatalf("ProcessLog failed: %v", err)
				}
				if diff := cmp.Diff(tc.wantState, nextState, cmp.AllowUnexported(containerLogToTimelineMapperTaskStateV2{}, containerStatusIdentity{}, containerStateWalkerV2{})); diff != "" {
					t.Errorf("state mismatch (-want +got):\n%s", diff)
				}

				if len(tc.wantRevisions) == 0 {
					testchangeset.AssertTimeline(t, cs).HasNoRevision(containerPath)
				} else {
					for _, want := range tc.wantRevisions {
						testchangeset.AssertTimeline(t, cs).HasRevision(containerPath, want, nodeComparer)
					}
				}
			}
		})
	}
}
