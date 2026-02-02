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

package commonlogk8sauditv2_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestContainerStateWalker(t *testing.T) {
	podNamespace := "default"
	podName := "nginx"
	containerName := "nginx-container"
	containerPath := resourcepath.Container(podNamespace, podName, containerName)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		steps []struct {
			yaml      string
			timestamp time.Time
			verb      enum.RevisionVerb
		}
		asserters []testchangeset.ChangeSetAsserter
	}{
		{
			name: "Container Waiting",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      enum.RevisionVerb
			}{
				{
					yaml: `
name: nginx-container
state:
  waiting:
    reason: ContainerCreating
`,
					timestamp: baseTime,
					verb:      enum.RevisionVerbUpdate,
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerWaiting,
						Requestor:  "user-1",
						ChangeTime: baseTime,
						Body:       "name: nginx-container\nstate:\n  waiting:\n    reason: ContainerCreating\n",
					},
				},
			},
		},
		{
			name: "Container Running (Ready)",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      enum.RevisionVerb
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
					verb:      enum.RevisionVerbUpdate,
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerStarted,
						Requestor:  "user-1",
						ChangeTime: baseTime,
						Body:       "name: nginx-container\nready: true\nstate:\n  running:\n    startedAt: \"2024-01-01T00:00:00Z\"\n",
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerRunningReady,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(time.Minute),
						Body:       "name: nginx-container\nready: true\nstate:\n  running:\n    startedAt: \"2024-01-01T00:00:00Z\"\n",
					},
				},
			},
		},
		{
			name: "Container Running (Not Ready)",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      enum.RevisionVerb
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
					verb:      enum.RevisionVerbUpdate,
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerStarted,
						Requestor:  "user-1",
						ChangeTime: baseTime,
						Body:       "name: nginx-container\nready: false\nstate:\n  running:\n    startedAt: \"2024-01-01T00:00:00Z\"\n",
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerRunningNonReady,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(time.Minute),
						Body:       "name: nginx-container\nready: false\nstate:\n  running:\n    startedAt: \"2024-01-01T00:00:00Z\"\n",
					},
				},
			},
		},
		{
			name: "Container Terminated (Success)",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      enum.RevisionVerb
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
					verb:      enum.RevisionVerbUpdate,
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerStarted,
						Requestor:  "user-1",
						ChangeTime: baseTime,
						Body:       "name: nginx-container\nstate:\n  terminated:\n    exitCode: 0\n    startedAt: \"2024-01-01T00:00:00Z\"\n    finishedAt: \"2024-01-01T01:00:00Z\"\n",
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerTerminatedWithSuccess,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(time.Hour),
						Body:       "name: nginx-container\nstate:\n  terminated:\n    exitCode: 0\n    startedAt: \"2024-01-01T00:00:00Z\"\n    finishedAt: \"2024-01-01T01:00:00Z\"\n",
					},
				},
			},
		},
		{
			name: "Container Terminated (Error)",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      enum.RevisionVerb
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
					verb:      enum.RevisionVerbUpdate,
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerTerminatedWithError,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(time.Hour),
						Body:       "name: nginx-container\nstate:\n  terminated:\n    exitCode: 1\n    startedAt: \"2024-01-01T00:00:00Z\"\n    finishedAt: \"2024-01-01T01:00:00Z\"\n",
					},
				},
			},
		},
		{
			name: "Transition: Waiting -> Running -> Terminated",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      enum.RevisionVerb
			}{
				{
					yaml: `
name: nginx-container
state:
  waiting:
    reason: ContainerCreating
`,
					timestamp: baseTime,
					verb:      enum.RevisionVerbUpdate,
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
					verb:      enum.RevisionVerbUpdate,
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
					verb:      enum.RevisionVerbUpdate,
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerWaiting,
						Requestor:  "user-1",
						ChangeTime: baseTime,
						Body:       "name: nginx-container\nstate:\n  waiting:\n    reason: ContainerCreating\n",
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerStarted,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(time.Minute),
						Body:       "name: nginx-container\nready: true\nstate:\n  running:\n    startedAt: \"2024-01-01T00:01:00Z\"\n",
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerRunningReady,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(2 * time.Minute),
						Body:       "name: nginx-container\nready: true\nstate:\n  running:\n    startedAt: \"2024-01-01T00:01:00Z\"\n",
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerTerminatedWithSuccess,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(10 * time.Minute),
						Body:       "name: nginx-container\nstate:\n  terminated:\n    exitCode: 0\n    startedAt: \"2024-01-01T00:01:00Z\"\n    finishedAt: \"2024-01-01T00:10:00Z\"\n",
					},
				},
			},
		},
		{
			name: "No State (Initial)",
			steps: []struct {
				yaml      string
				timestamp time.Time
				verb      enum.RevisionVerb
			}{
				{
					yaml:      "", // No state reader
					timestamp: baseTime,
					verb:      enum.RevisionVerbUpdate,
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: containerPath.Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerStatusNotAvailable,
						Requestor:  "user-1",
						ChangeTime: baseTime,
						Body:       "# No state for this container is recorded yet",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walker := &containerStateWalker{
				containerIdentity: &containerStatusIdentity{
					containerName: containerName,
					containerType: ContainerTypeContainer,
				},
				podNamespace: podNamespace,
				podName:      podName,
			}
			l := log.NewLogWithFieldSetsForTest()
			cs := history.NewChangeSet(l)

			for _, step := range tt.steps {
				commonFieldSet := &log.CommonFieldSet{
					Timestamp: step.timestamp,
				}
				k8sFieldSet := &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
					K8sOperation: &model.KubernetesObjectOperation{
						Verb: step.verb,
					},
					Principal: "user-1",
				}

				var stateReader *structured.NodeReader
				if step.yaml != "" {
					stateReader = mustParseYAML(t, step.yaml)
				}

				walker.CheckAndRecord(stateReader, cs, commonFieldSet, k8sFieldSet)
			}

			for _, asserter := range tt.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}

func TestContainerLogToTimelineMapperTask_Process(t *testing.T) {
	task := &containerLogToTimelineMapperTaskSetting{}
	ctx := context.Background()
	podNamespace := "default"
	podName := "nginx"

	tests := []struct {
		name         string
		pass         int
		yaml         string
		nilBody      bool
		initialState *containerLogToTimelineMapperTaskState
		wantState    *containerLogToTimelineMapperTaskState
		asserters    []testchangeset.ChangeSetAsserter
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
			wantState: &containerLogToTimelineMapperTaskState{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
					"init-container(init)": {
						containerName: "init-container",
						containerType: ContainerTypeInitContainer,
					},
					"debug-container(ephemeral)": {
						containerName: "debug-container",
						containerType: ContainerTypeEphemeral,
					},
				},
				containerStateWalkers: map[string]*containerStateWalker{},
			},
			asserters: []testchangeset.ChangeSetAsserter{},
		},
		{
			name:    "Pass 0: Nil Body",
			pass:    0,
			nilBody: true,
			wantState: &containerLogToTimelineMapperTaskState{
				containerIdentities:   map[string]*containerStatusIdentity{},
				containerStateWalkers: map[string]*containerStateWalker{},
			},
			asserters: []testchangeset.ChangeSetAsserter{},
		},
		{
			name: "Pass 1: Process Containers",
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
			initialState: &containerLogToTimelineMapperTaskState{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
				},
				containerStateWalkers: map[string]*containerStateWalker{},
			},
			wantState: &containerLogToTimelineMapperTaskState{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
				},
				containerStateWalkers: map[string]*containerStateWalker{
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
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.Container(podNamespace, podName, "main-container").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerStarted,
						Requestor:  "user-1",
						ChangeTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						Body:       "name: main-container\nstate:\n  running:\n    startedAt: \"2024-01-01T00:00:00Z\"\nready: true\n",
					},
				},
			},
		},
		{
			name: "Pass 1: Missing Status",
			pass: 1,
			yaml: `
status:
  containerStatuses: []
`,
			initialState: &containerLogToTimelineMapperTaskState{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
				},
				containerStateWalkers: map[string]*containerStateWalker{},
			},
			wantState: &containerLogToTimelineMapperTaskState{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
				},
				containerStateWalkers: map[string]*containerStateWalker{
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
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.Container(podNamespace, podName, "main-container").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateContainerStatusNotAvailable,
						Requestor:  "user-1",
						ChangeTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						Body:       "# No state for this container is recorded yet",
					},
				},
			},
		},
		{
			name:    "Pass 1: Nil Body",
			pass:    1,
			nilBody: true,
			initialState: &containerLogToTimelineMapperTaskState{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
				},
				containerStateWalkers: map[string]*containerStateWalker{},
			},
			wantState: &containerLogToTimelineMapperTaskState{
				containerIdentities: map[string]*containerStatusIdentity{
					"main-container": {
						containerName: "main-container",
						containerType: ContainerTypeContainer,
					},
				},
				containerStateWalkers: map[string]*containerStateWalker{},
			},
			asserters: []testchangeset.ChangeSetAsserter{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var reader *structured.NodeReader
			if !tc.nilBody {
				reader = mustParseYAML(t, tc.yaml)
			}
			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{},
				&commonlogk8sauditv2_contract.K8sAuditLogFieldSet{},
			)
			commonFieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
			commonFieldSet.Timestamp = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			k8sFieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
			k8sFieldSet.K8sOperation = &model.KubernetesObjectOperation{Verb: enum.RevisionVerbUpdate}
			k8sFieldSet.Principal = "user-1"

			event := commonlogk8sauditv2_contract.ResourceChangeEvent{
				Log:                   l,
				EventType:             commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
				EventTargetBodyReader: reader,
				EventTargetResource: &commonlogk8sauditv2_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  podNamespace,
					Name:       podName,
				},
			}

			cs := history.NewChangeSet(l)
			nextState, err := task.Process(ctx, tc.pass, event, cs, nil, tc.initialState)
			if err != nil {
				t.Fatalf("Process(%d) failed: %v", tc.pass, err)
			}

			if diff := cmp.Diff(tc.wantState, nextState, cmp.AllowUnexported(containerLogToTimelineMapperTaskState{}, containerStatusIdentity{}, containerStateWalker{})); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
