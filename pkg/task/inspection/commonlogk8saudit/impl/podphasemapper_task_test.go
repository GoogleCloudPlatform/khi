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
	"context"
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

func TestPodPhaseLogToTimelineMapperTaskSetting_ProcessLog(t *testing.T) {
	baseTime := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

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

	type step struct {
		role      string
		yaml      string
		eventType commonlogk8saudit_contract.ChangeEventType
		verb      *pb.Verb
		time      time.Time
	}

	type wantRevision struct {
		uid          string
		nodeName     string
		changedTime  time.Time
		stateType    *pb.RevisionState
		verbType     *pb.Verb
		resourceBody structured.Node
		principal    string
	}

	testCases := []struct {
		name          string
		podName       string
		namespace     string
		clusterName   string
		steps         []step
		wantRevisions []wantRevision
	}{
		{
			// Input:
			// 1. A pod creation event at t=5s, verb: Create, phase: Pending. Unscheduled at this point.
			// 2. A binding creation log event at t=10s, mapping the pod to node-1.
			//    The binding log contains a binding-specific UID and creationTimestamp, which should be ignored by the pod phase mapper.
			// 3. A pod status update event at t=20s, specifying uid-1, nodeName as node-1, and phase as Succeeded.
			//
			// Expected Output:
			// - A single pod phase timeline path: k8s-cluster/node-1/default/my-pod[uid-1]
			// - Revision at t=5s: StateType=PodPhasePending, Verb=Create, body=pod creation body.
			// - Revision at t=10s: StateType=PodPhaseScheduled, Verb=Create, body=nil.
			// - Revision at t=20s: StateType=PodPhaseSucceeded, Verb=Update, body=pod Succeeded body.
			name:        "Standard lifecycle starting with pod creation followed by binding with binding metadata and Succeeded update",
			podName:     "my-pod",
			namespace:   "default",
			clusterName: "k8s-cluster",
			steps: []step{
				{
					role: "pod",
					yaml: `
metadata:
  uid: uid-1
  creationTimestamp: "2026-06-26T12:00:05Z"
spec: {}
status:
  phase: Pending
`,
					eventType: commonlogk8saudit_contract.ChangeEventTypeCreation,
					verb:      commonlogk8saudit_contract.VerbCreate,
					time:      baseTime.Add(5 * time.Second),
				},
				{
					role: "binding",
					yaml: `
metadata:
  uid: binding-uid
  creationTimestamp: "2026-06-26T12:00:10Z"
target:
  name: node-1
`,
					eventType: commonlogk8saudit_contract.ChangeEventTypeCreation,
					verb:      commonlogk8saudit_contract.VerbCreate,
					time:      baseTime.Add(10 * time.Second),
				},
				{
					role: "pod",
					yaml: `
metadata:
  uid: uid-1
  creationTimestamp: "2026-06-26T12:00:05Z"
spec:
  nodeName: node-1
status:
  phase: Succeeded
`,
					eventType: commonlogk8saudit_contract.ChangeEventTypeModification,
					verb:      commonlogk8saudit_contract.VerbUpdate,
					time:      baseTime.Add(20 * time.Second),
				},
			},
			wantRevisions: []wantRevision{
				{
					uid:         "uid-1",
					nodeName:    "node-1",
					changedTime: baseTime.Add(5 * time.Second),
					stateType:   commonlogk8saudit_contract.RevisionStatePodPhasePending,
					verbType:    commonlogk8saudit_contract.VerbCreate,
					resourceBody: parseYAML(`
metadata:
  uid: uid-1
  creationTimestamp: "2026-06-26T12:00:05Z"
spec: {}
status:
  phase: Pending
`),
					principal: "admin",
				},
				{
					uid:         "uid-1",
					nodeName:    "node-1",
					changedTime: baseTime.Add(10 * time.Second),
					stateType:   commonlogk8saudit_contract.RevisionStatePodPhaseScheduled,
					verbType:    commonlogk8saudit_contract.VerbCreate,
					resourceBody: parseYAML(`
metadata:
  uid: uid-1
  creationTimestamp: "2026-06-26T12:00:05Z"
spec: {}
status:
  phase: Pending
`),
					principal: "admin",
				},
				{
					uid:         "uid-1",
					nodeName:    "node-1",
					changedTime: baseTime.Add(20 * time.Second),
					stateType:   commonlogk8saudit_contract.RevisionStatePodPhaseSucceeded,
					verbType:    commonlogk8saudit_contract.VerbUpdate,
					resourceBody: parseYAML(`
metadata:
  uid: uid-1
  creationTimestamp: "2026-06-26T12:00:05Z"
spec:
  nodeName: node-1
status:
  phase: Succeeded
`),
					principal: "admin",
				},
			},
		},
		{
			// Input:
			// 1. A binding creation log event at t=10s, mapping the pod to node-1.
			//    The binding log contains a binding-specific UID and creationTimestamp, which should be ignored.
			// 2. A pod update event at t=20s, specifying uid-1, nodeName node-1, and phase Failed.
			//    The pod metadata includes creationTimestamp = 12:00:00Z.
			//
			// Expected Output:
			// - An inferred creation revision at the creationTimestamp (12:00:00Z) with StateType=PodPhaseUnknown, Verb=Create.
			// - A revision at t=20s: StateType=PodPhaseFailed, Verb=Update.
			name:        "Inferred creation revision when the first pod event is an update, preceded by binding with binding metadata",
			podName:     "my-pod",
			namespace:   "default",
			clusterName: "k8s-cluster",
			steps: []step{
				{
					role: "binding",
					yaml: `
metadata:
  uid: binding-uid
  creationTimestamp: "2026-06-26T12:00:10Z"
target:
  name: node-1
`,
					eventType: commonlogk8saudit_contract.ChangeEventTypeCreation,
					verb:      commonlogk8saudit_contract.VerbCreate,
					time:      baseTime.Add(10 * time.Second),
				},
				{
					role: "pod",
					yaml: `
metadata:
  uid: uid-1
  creationTimestamp: "2026-06-26T12:00:00Z"
spec:
  nodeName: node-1
status:
  phase: Failed
`,
					eventType: commonlogk8saudit_contract.ChangeEventTypeCreation,
					verb:      commonlogk8saudit_contract.VerbUpdate,
					time:      baseTime.Add(20 * time.Second),
				},
			},
			wantRevisions: []wantRevision{
				{
					uid:          "uid-1",
					nodeName:     "node-1",
					changedTime:  time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC),
					stateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					verbType:     commonlogk8saudit_contract.VerbCreate,
					resourceBody: nil,
					principal:    "N/A",
				},
				{
					uid:         "uid-1",
					nodeName:    "node-1",
					changedTime: baseTime.Add(20 * time.Second),
					stateType:   commonlogk8saudit_contract.RevisionStatePodPhaseFailed,
					verbType:    commonlogk8saudit_contract.VerbUpdate,
					resourceBody: parseYAML(`
metadata:
  uid: uid-1
  creationTimestamp: "2026-06-26T12:00:00Z"
spec:
  nodeName: node-1
status:
  phase: Failed
`),
					principal: "admin",
				},
			},
		},
		{
			// Input:
			// 1. A binding creation log event at t=10s, mapping the pod to node-1.
			//    The binding log contains a binding-specific UID and creationTimestamp, which should be ignored.
			// 2. A pod patch event at t=20s, specifying phase Running. This event lacks pod metadata (uid and creationTimestamp).
			// 3. A pod status update event at t=30s, specifying uid-1, nodeName as node-1, and phase as Succeeded.
			//    The pod metadata includes creationTimestamp = 12:00:00Z.
			//
			// Expected Output:
			// - A single pod phase timeline path: k8s-cluster/node-1/default/my-pod[uid-1]
			// - Revision at t=0s (12:00:00Z): StateType=PodPhaseUnknown, Verb=Create, body=nil.
			// - Revision at t=10s: StateType=PodPhaseScheduled, Verb=Create, body=nil.
			// - Revision at t=20s: StateType=PodPhaseRunning, Verb=Patch, body=patch body.
			// - Revision at t=30s: StateType=PodPhaseSucceeded, Verb=Update, body=pod update body.
			name:        "Resolve pod lifecycle stages with chronological fallback when metadata is missing in early events",
			podName:     "my-pod",
			namespace:   "default",
			clusterName: "k8s-cluster",
			steps: []step{
				{
					role: "binding",
					yaml: `
metadata:
  uid: binding-uid
  creationTimestamp: "2026-06-26T12:00:10Z"
target:
  name: node-1
`,
					eventType: commonlogk8saudit_contract.ChangeEventTypeCreation,
					verb:      commonlogk8saudit_contract.VerbCreate,
					time:      baseTime.Add(10 * time.Second),
				},
				{
					role: "pod",
					yaml: `
status:
  phase: Running
`,
					eventType: commonlogk8saudit_contract.ChangeEventTypeCreation,
					verb:      commonlogk8saudit_contract.VerbPatch,
					time:      baseTime.Add(20 * time.Second),
				},
				{
					role: "pod",
					yaml: `
metadata:
  uid: uid-1
  creationTimestamp: "2026-06-26T12:00:00Z"
spec:
  nodeName: node-1
status:
  phase: Succeeded
`,
					eventType: commonlogk8saudit_contract.ChangeEventTypeModification,
					verb:      commonlogk8saudit_contract.VerbUpdate,
					time:      baseTime.Add(30 * time.Second),
				},
			},
			wantRevisions: []wantRevision{
				{
					uid:          "uid-1",
					nodeName:     "node-1",
					changedTime:  time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC),
					stateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					verbType:     commonlogk8saudit_contract.VerbCreate,
					resourceBody: nil,
					principal:    "N/A",
				},
				{
					uid:          "uid-1",
					nodeName:     "node-1",
					changedTime:  baseTime.Add(10 * time.Second),
					stateType:    commonlogk8saudit_contract.RevisionStatePodPhaseScheduled,
					verbType:     commonlogk8saudit_contract.VerbCreate,
					resourceBody: nil,
					principal:    "admin",
				},
				{
					uid:         "uid-1",
					nodeName:    "node-1",
					changedTime: baseTime.Add(20 * time.Second),
					stateType:   commonlogk8saudit_contract.RevisionStatePodPhaseRunning,
					verbType:    commonlogk8saudit_contract.VerbPatch,
					resourceBody: parseYAML(`
status:
  phase: Running
`),
					principal: "admin",
				},
				{
					uid:         "uid-1",
					nodeName:    "node-1",
					changedTime: baseTime.Add(30 * time.Second),
					stateType:   commonlogk8saudit_contract.RevisionStatePodPhaseSucceeded,
					verbType:    commonlogk8saudit_contract.VerbUpdate,
					resourceBody: parseYAML(`
metadata:
  uid: uid-1
  creationTimestamp: "2026-06-26T12:00:00Z"
spec:
  nodeName: node-1
status:
  phase: Succeeded
`),
					principal: "admin",
				},
			},
		},
		{
			// Input:
			// 1. A binding creation log event at t=10s, mapping the pod to node-1.
			//    The binding log contains a binding-specific UID and creationTimestamp, which should be ignored.
			// 2. A pod patch event at t=20s, specifying uid-1 and phase Running.
			//    The pod metadata does NOT include creationTimestamp.
			//
			// Expected Output:
			// - An inferred creation revision at Unix Epoch (1970-01-01T00:00:00Z) with StateType=PodPhaseUnknown, Verb=Create.
			// - Revision at t=10s: StateType=PodPhaseScheduled, Verb=Create, body=nil.
			// - Revision at t=20s: StateType=PodPhaseRunning, Verb=Patch, body=patch body.
			name:        "Inferred creation revision fallback to Unix Epoch when creationTimestamp is missing",
			podName:     "my-pod",
			namespace:   "default",
			clusterName: "k8s-cluster",
			steps: []step{
				{
					role: "binding",
					yaml: `
metadata:
  uid: binding-uid
  creationTimestamp: "2026-06-26T12:00:10Z"
target:
  name: node-1
`,
					eventType: commonlogk8saudit_contract.ChangeEventTypeCreation,
					verb:      commonlogk8saudit_contract.VerbCreate,
					time:      baseTime.Add(10 * time.Second),
				},
				{
					role: "pod",
					yaml: `
metadata:
  uid: uid-1
status:
  phase: Running
`,
					eventType: commonlogk8saudit_contract.ChangeEventTypeCreation,
					verb:      commonlogk8saudit_contract.VerbPatch,
					time:      baseTime.Add(20 * time.Second),
				},
			},
			wantRevisions: []wantRevision{
				{
					uid:          "uid-1",
					nodeName:     "node-1",
					changedTime:  time.Unix(0, 0).UTC(),
					stateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					verbType:     commonlogk8saudit_contract.VerbCreate,
					resourceBody: nil,
					principal:    "N/A",
				},
				{
					uid:          "uid-1",
					nodeName:     "node-1",
					changedTime:  baseTime.Add(10 * time.Second),
					stateType:    commonlogk8saudit_contract.RevisionStatePodPhaseScheduled,
					verbType:     commonlogk8saudit_contract.VerbCreate,
					resourceBody: nil,
					principal:    "admin",
				},
				{
					uid:         "uid-1",
					nodeName:    "node-1",
					changedTime: baseTime.Add(20 * time.Second),
					stateType:   commonlogk8saudit_contract.RevisionStatePodPhaseRunning,
					verbType:    commonlogk8saudit_contract.VerbPatch,
					resourceBody: parseYAML(`
metadata:
  uid: uid-1
status:
  phase: Running
`),
					principal: "admin",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			mapperSetting := &podPhaseLogToTimelineMapperTaskSetting{}

			podResource := &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  tc.namespace,
				Name:       tc.podName,
			}
			bindingResource := &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion:      "core/v1",
				Kind:            "pod",
				Namespace:       tc.namespace,
				Name:            tc.podName,
				SubresourceName: "binding",
			}

			podGroup := &commonlogk8saudit_contract.ResourceManifestLogGroup{
				Resource: podResource,
				Logs:     []*commonlogk8saudit_contract.ResourceManifestLog{},
			}
			bindingGroup := &commonlogk8saudit_contract.ResourceManifestLogGroup{
				Resource: bindingResource,
				Logs:     []*commonlogk8saudit_contract.ResourceManifestLog{},
			}

			type stepLogInfo struct {
				manifestLog *commonlogk8saudit_contract.ResourceManifestLog
				role        string
				identity    *commonlogk8saudit_contract.ResourceIdentity
				eventType   commonlogk8saudit_contract.ChangeEventType
				verb        *pb.Verb
			}
			var stepInfos []stepLogInfo

			for _, step := range tc.steps {
				k8sFieldSet := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
					Principal:    "admin",
					APIVersion:   "core/v1",
					PluralKind:   "pods",
					ResourceName: tc.podName,
					Namespace:    tc.namespace,
					ClusterName:  tc.clusterName,
					Verb:         step.verb,
				}
				commonFs := &log.CommonFieldSet{
					Timestamp: step.time,
				}
				logObj := log.NewLogWithFieldSetsForTest(k8sFieldSet, commonFs)
				node, err := structured.FromYAML(step.yaml)
				if err != nil {
					t.Fatalf("failed to parse test YAML: %v", err)
				}
				var nodeReader *structured.NodeReader
				if node != nil {
					nodeReader = structured.NewNodeReader(node)
				}

				mLog := &commonlogk8saudit_contract.ResourceManifestLog{
					Log:                logObj,
					ResourceBodyReader: nodeReader,
					ResourceBodyYAML:   step.yaml,
				}

				var identity *commonlogk8saudit_contract.ResourceIdentity
				if step.role == "pod" {
					identity = podResource
					podGroup.Logs = append(podGroup.Logs, mLog)
				} else {
					identity = bindingResource
					bindingGroup.Logs = append(bindingGroup.Logs, mLog)
				}

				stepInfos = append(stepInfos, stepLogInfo{
					manifestLog: mLog,
					role:        step.role,
					identity:    identity,
					eventType:   step.eventType,
					verb:        step.verb,
				})
			}

			groupSet := commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"pod":     podGroup,
					"binding": bindingGroup,
				},
			}

			var events []commonlogk8saudit_contract.MultiGroupLogEvent
			for _, info := range stepInfos {
				events = append(events, commonlogk8saudit_contract.MultiGroupLogEvent{
					Log:              info.manifestLog.Log,
					GroupRole:        info.role,
					ResourceIdentity: info.identity,
					EventType:        info.eventType,
					GroupSet:         groupSet,
				})
			}

			// 1. Pass 0 of PreProcessLog
			var state *podPhaseTaskState
			for _, ev := range events {
				var err error
				state, err = mapperSetting.PreProcessLog(ctx, 0, ev, state)
				if err != nil {
					t.Fatalf("PreProcessLog(pass=0) failed: %v", err)
				}
			}

			// 2. Pass 1 of PreProcessLog
			for _, ev := range events {
				var err error
				state, err = mapperSetting.PreProcessLog(ctx, 1, ev, state)
				if err != nil {
					t.Fatalf("PreProcessLog(pass=1) failed: %v", err)
				}
			}

			// 3. ProcessLog
			var changeSets []*khifilev6.TimelineChangeSet
			for _, ev := range events {
				cs, nextState, err := mapperSetting.ProcessLog(ctx, ev, state)
				if err != nil {
					t.Fatalf("ProcessLog() failed: %v", err)
				}
				state = nextState
				changeSets = append(changeSets, cs)
			}

			mergedCS := khifilev6.NewTimelineChangeSet(log.NewLogWithFieldSetsForTest())
			for _, cs := range changeSets {
				if cs != nil {
					for path, revs := range cs.Revisions {
						for _, r := range revs {
							mergedCS.AddRevision(path, r)
						}
					}
					for path := range cs.Events {
						mergedCS.AddEvent(path)
					}
					for aliasPath, targetPath := range cs.Aliases {
						mergedCS.AddAlias(aliasPath, targetPath)
					}
				}
			}

			for _, want := range tc.wantRevisions {
				path := MustResolvePodPhaseTimelinePath(ctx, tc.clusterName, want.nodeName, tc.namespace, tc.podName, want.uid)
				wantStagingRev := &khifilev6.StagingRevision{
					ChangedTime:  want.changedTime,
					ResourceBody: want.resourceBody,
					Principal:    want.principal,
					VerbType:     want.verbType,
					StateType:    want.stateType,
				}
				testchangeset.AssertTimeline(t, mergedCS).HasRevision(path, wantStagingRev, nodeComparer)
			}
		})
	}
}

// MustResolvePodPhaseTimelinePath resolves the TimelinePath for the pod phase.
func MustResolvePodPhaseTimelinePath(ctx context.Context, clusterName, nodeName, namespace, podName, uid string) *khifilev6.TimelinePath {
	return MustPodPhaseTimelinePath(ctx, clusterName, nodeName, namespace, podName, uid)
}
