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
	"google.golang.org/protobuf/testing/protocmp"
)

func TestEndpointLogToTimelineMapperTask_ProcessLog(t *testing.T) {
	task := &endpointResourceLogToTimelineMapperTaskSettingV2{}
	timestamp := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

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

	testCases := []struct {
		name         string
		isPreProcess bool
		yaml         string
		eventType    commonlogk8saudit_contract.ChangeEventTypeV2
		verb         *pb.Verb
		initialState *endpointResourceLogToTimelineMapperStateV2
		wantState    *endpointResourceLogToTimelineMapperStateV2
		assert       func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder)
	}{
		{
			name:         "Pass 0: Collect Service Name",
			isPreProcess: true,
			yaml: `
metadata:
  ownerReferences:
  - kind: Service
    name: my-service
`,
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			verb:      commonlogk8saudit_contract.VerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				if len(cs.Events) > 0 || len(cs.Revisions) > 0 || len(cs.Aliases) > 0 {
					t.Errorf("expected empty timeline changeset, but got: %v", cs)
				}
			},
		},
		{
			name:         "Pass 0: Collect Pod Identity",
			isPreProcess: true,
			yaml: `
endpoints:
- targetRef:
    kind: Pod
    name: my-pod
    namespace: default
    uid: pod-uid-1
`,
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			verb:      commonlogk8saudit_contract.VerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {
						uid:       "pod-uid-1",
						name:      "my-pod",
						namespace: "default",
					},
				},
				lastStates: map[string]*pb.RevisionState{},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				if len(cs.Events) > 0 || len(cs.Revisions) > 0 || len(cs.Aliases) > 0 {
					t.Errorf("expected empty timeline changeset, but got: %v", cs)
				}
			},
		},
		{
			name:         "Pass 1: Standard Update (Ready)",
			isPreProcess: false,
			yaml: `
endpoints:
- conditions:
    ready: true
  targetRef:
    kind: Pod
    name: my-pod
    namespace: default
    uid: pod-uid-1
`,
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			verb:      commonlogk8saudit_contract.VerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]*pb.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]*pb.RevisionState{
					"pod-uid-1": commonlogk8saudit_contract.RevisionStateEndpointReady,
				},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				clusterPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})

				podApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				podKind := builder.TimelineAccumulator.GetPath(podApi, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
				podNs := builder.TimelineAccumulator.GetPath(podKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				podPath := builder.TimelineAccumulator.GetPath(podNs, khifilev6.PathSegment{Name: "my-pod", Type: inspectioncore_contract.TimelineTypeResource})
				expectedPodPath := builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{Name: "my-endpoint(default)", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				epsApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "discovery.k8s.io/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				epsKind := builder.TimelineAccumulator.GetPath(epsApi, khifilev6.PathSegment{Name: "endpointslice", Type: inspectioncore_contract.TimelineTypeKind})
				epsNs := builder.TimelineAccumulator.GetPath(epsKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				epsPath := builder.TimelineAccumulator.GetPath(epsNs, khifilev6.PathSegment{Name: "my-endpoint", Type: inspectioncore_contract.TimelineTypeResource})
				expectedEpsPath := builder.TimelineAccumulator.GetPath(epsPath, khifilev6.PathSegment{Name: "my-pod", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				svcApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				svcKind := builder.TimelineAccumulator.GetPath(svcApi, khifilev6.PathSegment{Name: "service", Type: inspectioncore_contract.TimelineTypeKind})
				svcNs := builder.TimelineAccumulator.GetPath(svcKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				svcPath := builder.TimelineAccumulator.GetPath(svcNs, khifilev6.PathSegment{Name: "my-service", Type: inspectioncore_contract.TimelineTypeResource})
				expectedSvcPath := builder.TimelineAccumulator.GetPath(svcPath, khifilev6.PathSegment{Name: "my-endpoint", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				testchangeset.AssertTimeline(t, cs).
					HasRevision(expectedPodPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateEndpointReady,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: parseYAML("conditions:\n  ready: true\ntargetRef:\n  kind: Pod\n  name: my-pod\n  namespace: default\n  uid: pod-uid-1\n"),
					}, nodeComparer).
					HasRevision(expectedEpsPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateEndpointReady,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: parseYAML("conditions:\n  ready: true\ntargetRef:\n  kind: Pod\n  name: my-pod\n  namespace: default\n  uid: pod-uid-1\n"),
					}, nodeComparer).
					HasRevision(expectedSvcPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateEndpointReady,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: parseYAML("endpoints:\n- conditions:\n    ready: true\n  targetRef:\n    kind: Pod\n    name: my-pod\n    namespace: default\n    uid: pod-uid-1\n"),
					}, nodeComparer)
			},
		},
		{
			name:         "Pass 1: Standard Update (Terminating)",
			isPreProcess: false,
			yaml: `
endpoints:
- conditions:
    terminating: true
  targetRef:
    kind: Pod
    name: my-pod
    namespace: default
    uid: pod-uid-1
`,
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			verb:      commonlogk8saudit_contract.VerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]*pb.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]*pb.RevisionState{
					"pod-uid-1": commonlogk8saudit_contract.RevisionStateEndpointTerminating,
				},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				clusterPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})

				podApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				podKind := builder.TimelineAccumulator.GetPath(podApi, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
				podNs := builder.TimelineAccumulator.GetPath(podKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				podPath := builder.TimelineAccumulator.GetPath(podNs, khifilev6.PathSegment{Name: "my-pod", Type: inspectioncore_contract.TimelineTypeResource})
				expectedPodPath := builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{Name: "my-endpoint(default)", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				testchangeset.AssertTimeline(t, cs).
					HasRevision(expectedPodPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateEndpointTerminating,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: parseYAML("conditions:\n  terminating: true\ntargetRef:\n  kind: Pod\n  name: my-pod\n  namespace: default\n  uid: pod-uid-1\n"),
					}, nodeComparer)
			},
		},
		{
			name:         "Pass 1: Standard Update (Unready)",
			isPreProcess: false,
			yaml: `
endpoints:
- conditions:
    ready: false
  targetRef:
    kind: Pod
    name: my-pod
    namespace: default
    uid: pod-uid-1
`,
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			verb:      commonlogk8saudit_contract.VerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]*pb.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]*pb.RevisionState{
					"pod-uid-1": commonlogk8saudit_contract.RevisionStateEndpointUnready,
				},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				clusterPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})

				podApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				podKind := builder.TimelineAccumulator.GetPath(podApi, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
				podNs := builder.TimelineAccumulator.GetPath(podKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				podPath := builder.TimelineAccumulator.GetPath(podNs, khifilev6.PathSegment{Name: "my-pod", Type: inspectioncore_contract.TimelineTypeResource})
				expectedPodPath := builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{Name: "my-endpoint(default)", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				testchangeset.AssertTimeline(t, cs).
					HasRevision(expectedPodPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateEndpointUnready,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: parseYAML("conditions:\n  ready: false\ntargetRef:\n  kind: Pod\n  name: my-pod\n  namespace: default\n  uid: pod-uid-1\n"),
					}, nodeComparer)
			},
		},
		{
			name:         "Pass 1: Service State (Ready)",
			isPreProcess: false,
			yaml: `
endpoints:
- conditions:
    ready: true
`,
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			verb:      commonlogk8saudit_contract.VerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				clusterPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})

				svcApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				svcKind := builder.TimelineAccumulator.GetPath(svcApi, khifilev6.PathSegment{Name: "service", Type: inspectioncore_contract.TimelineTypeKind})
				svcNs := builder.TimelineAccumulator.GetPath(svcKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				svcPath := builder.TimelineAccumulator.GetPath(svcNs, khifilev6.PathSegment{Name: "my-service", Type: inspectioncore_contract.TimelineTypeResource})
				expectedSvcPath := builder.TimelineAccumulator.GetPath(svcPath, khifilev6.PathSegment{Name: "my-endpoint", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				testchangeset.AssertTimeline(t, cs).
					HasRevision(expectedSvcPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateEndpointReady,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: parseYAML("endpoints:\n- conditions:\n    ready: true\n"),
					}, nodeComparer)
			},
		},
		{
			name:         "Pass 1: Service State (Terminating)",
			isPreProcess: false,
			yaml: `
endpoints:
- conditions:
    terminating: true
`,
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			verb:      commonlogk8saudit_contract.VerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				clusterPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})

				svcApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				svcKind := builder.TimelineAccumulator.GetPath(svcApi, khifilev6.PathSegment{Name: "service", Type: inspectioncore_contract.TimelineTypeKind})
				svcNs := builder.TimelineAccumulator.GetPath(svcKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				svcPath := builder.TimelineAccumulator.GetPath(svcNs, khifilev6.PathSegment{Name: "my-service", Type: inspectioncore_contract.TimelineTypeResource})
				expectedSvcPath := builder.TimelineAccumulator.GetPath(svcPath, khifilev6.PathSegment{Name: "my-endpoint", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				testchangeset.AssertTimeline(t, cs).
					HasRevision(expectedSvcPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateEndpointTerminating,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: parseYAML("endpoints:\n- conditions:\n    terminating: true\n"),
					}, nodeComparer)
			},
		},
		{
			name:         "Pass 1: Service State (Unready)",
			isPreProcess: false,
			yaml: `
endpoints:
- conditions:
    ready: false
`,
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			verb:      commonlogk8saudit_contract.VerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				clusterPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})

				svcApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				svcKind := builder.TimelineAccumulator.GetPath(svcApi, khifilev6.PathSegment{Name: "service", Type: inspectioncore_contract.TimelineTypeKind})
				svcNs := builder.TimelineAccumulator.GetPath(svcKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				svcPath := builder.TimelineAccumulator.GetPath(svcNs, khifilev6.PathSegment{Name: "my-service", Type: inspectioncore_contract.TimelineTypeResource})
				expectedSvcPath := builder.TimelineAccumulator.GetPath(svcPath, khifilev6.PathSegment{Name: "my-endpoint", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				testchangeset.AssertTimeline(t, cs).
					HasRevision(expectedSvcPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateEndpointUnready,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: parseYAML("endpoints:\n- conditions:\n    ready: false\n"),
					}, nodeComparer)
			},
		},
		{
			name:         "Pass 1: Endpoint Removal (Implicit)",
			isPreProcess: false,
			yaml: `
endpoints: []
`,
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			verb:      commonlogk8saudit_contract.VerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]*pb.RevisionState{
					"pod-uid-1": commonlogk8saudit_contract.RevisionStateEndpointReady,
				},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]*pb.RevisionState{},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				clusterPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})

				podApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				podKind := builder.TimelineAccumulator.GetPath(podApi, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
				podNs := builder.TimelineAccumulator.GetPath(podKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				podPath := builder.TimelineAccumulator.GetPath(podNs, khifilev6.PathSegment{Name: "my-pod", Type: inspectioncore_contract.TimelineTypeResource})
				expectedPodPath := builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{Name: "my-endpoint(default)", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				testchangeset.AssertTimeline(t, cs).
					HasRevision(expectedPodPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: nil,
					}, nodeComparer)
			},
		},
		{
			name:         "Pass 1: Target Deletion",
			isPreProcess: false,
			yaml: `
metadata:
  name: my-endpoint
`,
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Deletion,
			verb:      commonlogk8saudit_contract.VerbDelete,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]*pb.RevisionState{
					"pod-uid-1": commonlogk8saudit_contract.RevisionStateEndpointReady,
				},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]*pb.RevisionState{},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				clusterPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})

				podApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				podKind := builder.TimelineAccumulator.GetPath(podApi, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
				podNs := builder.TimelineAccumulator.GetPath(podKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				podPath := builder.TimelineAccumulator.GetPath(podNs, khifilev6.PathSegment{Name: "my-pod", Type: inspectioncore_contract.TimelineTypeResource})
				expectedPodPath := builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{Name: "my-endpoint(default)", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				svcApi := builder.TimelineAccumulator.GetPath(clusterPath, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
				svcKind := builder.TimelineAccumulator.GetPath(svcApi, khifilev6.PathSegment{Name: "service", Type: inspectioncore_contract.TimelineTypeKind})
				svcNs := builder.TimelineAccumulator.GetPath(svcKind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
				svcPath := builder.TimelineAccumulator.GetPath(svcNs, khifilev6.PathSegment{Name: "my-service", Type: inspectioncore_contract.TimelineTypeResource})
				expectedSvcPath := builder.TimelineAccumulator.GetPath(svcPath, khifilev6.PathSegment{Name: "my-endpoint", Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice})

				testchangeset.AssertTimeline(t, cs).
					HasRevision(expectedPodPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: nil,
					}, nodeComparer).
					HasRevision(expectedSvcPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
						ChangedTime:  timestamp,
						Principal:    "user-1",
						ResourceBody: nil,
					}, nodeComparer)
			},
		},
		{
			name:         "Pass 0: No EndpointSlice body",
			isPreProcess: true,
			eventType:    commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			verb:         commonlogk8saudit_contract.VerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperStateV2{
				serviceNames: map[string]struct{}{},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]*pb.RevisionState{},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, builder *khifilev6.Builder) {
				if len(cs.Events) > 0 || len(cs.Revisions) > 0 || len(cs.Aliases) > 0 {
					t.Errorf("expected empty timeline changeset, but got: %v", cs)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			var reader *structured.NodeReader
			if tc.yaml != "" {
				node, err := structured.FromYAML(tc.yaml)
				if err != nil {
					t.Fatalf("failed to parse YAML: %v", err)
				}
				reader = structured.NewNodeReader(node)
			}

			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: timestamp,
				},
				&commonlogk8saudit_contract.K8sAuditLogFieldSet{
					Verb:        tc.verb,
					Principal:   "user-1",
					ClusterName: "k8s",
				},
			)

			resIdentity := &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "discovery.k8s.io/v1",
				Kind:       "endpointslice",
				Namespace:  "default",
				Name:       "my-endpoint",
			}

			groupSet := commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"target": {
						Resource: resIdentity,
						Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
							{Log: l, ResourceBodyReader: reader, ResourceBodyYAML: tc.yaml},
						},
					},
				},
			}

			event := commonlogk8saudit_contract.MultiGroupLogEvent{
				Log:              l,
				GroupRole:        "target",
				ResourceIdentity: resIdentity,
				EventType:        tc.eventType,
				GroupSet:         groupSet,
			}

			if tc.isPreProcess {
				nextState, err := task.PreProcessLog(ctx, 0, event, tc.initialState)
				if err != nil {
					t.Fatalf("PreProcessLog failed: %v", err)
				}
				if diff := cmp.Diff(tc.wantState, nextState, cmp.AllowUnexported(endpointResourceLogToTimelineMapperStateV2{}, podIdentity{}), protocmp.Transform()); diff != "" {
					t.Errorf("PreProcessLog state mismatch (-want +got):\n%s", diff)
				}
			} else {
				cs, nextState, err := task.ProcessLog(ctx, event, tc.initialState)
				if err != nil {
					t.Fatalf("ProcessLog failed: %v", err)
				}
				if diff := cmp.Diff(tc.wantState, nextState, cmp.AllowUnexported(endpointResourceLogToTimelineMapperStateV2{}, podIdentity{}), protocmp.Transform()); diff != "" {
					t.Errorf("ProcessLog state mismatch (-want +got):\n%s", diff)
				}
				tc.assert(t, cs, builder)
			}
		})
	}
}
