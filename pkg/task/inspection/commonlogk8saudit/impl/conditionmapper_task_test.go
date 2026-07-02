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
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestConditionWalker(t *testing.T) {
	builder := khifilev6.NewBuilder()
	cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
	api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
	kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
	ns := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
	parentPath := builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "nginx", Type: inspectioncore_contract.TimelineTypeResource})
	conditionPath := builder.TimelineAccumulator.GetPath(parentPath, khifilev6.PathSegment{Name: "Ready", Type: commonlogk8saudit_contract.TimelineTypeResourceCondition})

	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	commonFieldSet := &log.CommonFieldSet{
		Timestamp: baseTime,
	}
	k8sFieldSet := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
		Verb:      commonlogk8saudit_contract.VerbUpdate,
		Principal: "user-1",
	}

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
		name      string
		condition *model.K8sResourceStatusCondition
		want      *khifilev6.StagingRevision
	}

	scenarios := []struct {
		name  string
		steps []step
	}{
		{
			name: "Standard Lifecycle",
			steps: []step{
				{
					name: "Initial Condition (TransitionTime)",
					condition: &model.K8sResourceStatusCondition{
						Type:               "Ready",
						Status:             "True",
						LastTransitionTime: baseTime.Format(time.RFC3339),
					},
					want: &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						ResourceBody: parseYAML("lastTransitionTime: \"2024-01-01T00:00:00Z\"\nstatus: \"True\"\ntype: Ready\n"),
						Principal:    "user-1",
						ChangedTime:  baseTime,
						StateType:    commonlogk8saudit_contract.RevisionStateConditionTrue,
					},
				},
				{
					name: "No Change",
					condition: &model.K8sResourceStatusCondition{
						Type:               "Ready",
						Status:             "True",
						LastTransitionTime: baseTime.Format(time.RFC3339),
					},
					want: nil,
				},
				{
					name: "Status Change (TransitionTime)",
					condition: &model.K8sResourceStatusCondition{
						Type:               "Ready",
						Status:             "False",
						LastTransitionTime: baseTime.Add(1 * time.Hour).Format(time.RFC3339),
					},
					want: &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						ResourceBody: parseYAML("lastTransitionTime: \"2024-01-01T01:00:00Z\"\nstatus: \"False\"\ntype: Ready\n"),
						Principal:    "user-1",
						ChangedTime:  baseTime.Add(1 * time.Hour),
						StateType:    commonlogk8saudit_contract.RevisionStateConditionFalse,
					},
				},
				{
					name: "Probe Time Change (ProbeLikeTime)",
					condition: &model.K8sResourceStatusCondition{
						Type:               "Ready",
						Status:             "False",
						LastTransitionTime: baseTime.Add(1 * time.Hour).Format(time.RFC3339),
						LastHeartbeatTime:  baseTime.Add(2 * time.Hour).Format(time.RFC3339),
					},
					want: &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						ResourceBody: parseYAML("lastHeartbeatTime: \"2024-01-01T02:00:00Z\"\nlastTransitionTime: \"2024-01-01T01:00:00Z\"\nstatus: \"False\"\ntype: Ready\n"),
						Principal:    "user-1",
						ChangedTime:  baseTime.Add(2 * time.Hour),
						StateType:    commonlogk8saudit_contract.RevisionStateConditionFalse,
					},
				},
				{
					name: "No change on LastTransitionTime but changes on LastHeartbeatTime",
					condition: &model.K8sResourceStatusCondition{
						Type:               "Ready",
						Status:             "False",
						LastTransitionTime: baseTime.Add(1 * time.Hour).Format(time.RFC3339),
						LastHeartbeatTime:  baseTime.Add(3 * time.Hour).Format(time.RFC3339),
					},
					want: &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						ResourceBody: parseYAML("lastHeartbeatTime: \"2024-01-01T03:00:00Z\"\nlastTransitionTime: \"2024-01-01T01:00:00Z\"\nstatus: \"False\"\ntype: Ready\n"),
						Principal:    "user-1",
						ChangedTime:  baseTime.Add(3 * time.Hour),
						StateType:    commonlogk8saudit_contract.RevisionStateConditionFalse,
					},
				},
				{
					name:      "Condition Removal",
					condition: nil,
					want: &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						ResourceBody: nil,
						Principal:    "user-1",
						ChangedTime:  baseTime,
						StateType:    commonlogk8saudit_contract.RevisionStateConditionNotGiven,
					},
				},
				{
					name:      "Condition Removal (Already Removed)",
					condition: nil,
					want:      nil,
				},
			},
		},
		{
			name: "patch conditions without the full status information",
			steps: []step{
				{
					name: "initial patch without status",
					condition: &model.K8sResourceStatusCondition{
						Type:               "Ready",
						LastTransitionTime: baseTime.Add(1 * time.Hour).Format(time.RFC3339),
					},
					want: &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						ResourceBody: parseYAML("lastTransitionTime: \"2024-01-01T01:00:00Z\"\ntype: Ready\n"),
						Principal:    "user-1",
						ChangedTime:  baseTime.Add(1 * time.Hour),
						StateType:    commonlogk8saudit_contract.RevisionStateConditionNoAvailableInfo,
					},
				},
				{
					name: "patch without status, with heartbeat",
					condition: &model.K8sResourceStatusCondition{
						Type:               "Ready",
						LastTransitionTime: baseTime.Add(1 * time.Hour).Format(time.RFC3339),
						LastHeartbeatTime:  baseTime.Add(2 * time.Hour).Format(time.RFC3339),
					},
					want: &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						ResourceBody: parseYAML("lastHeartbeatTime: \"2024-01-01T02:00:00Z\"\nlastTransitionTime: \"2024-01-01T01:00:00Z\"\ntype: Ready\n"),
						Principal:    "user-1",
						ChangedTime:  baseTime.Add(2 * time.Hour),
						StateType:    commonlogk8saudit_contract.RevisionStateConditionNoAvailableInfo,
					},
				},
				{
					name: "patch with status added",
					condition: &model.K8sResourceStatusCondition{
						Type:               "Ready",
						LastTransitionTime: baseTime.Add(3 * time.Hour).Format(time.RFC3339),
						LastHeartbeatTime:  baseTime.Add(2 * time.Hour).Format(time.RFC3339),
						Status:             "True",
					},
					want: &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						ResourceBody: parseYAML("lastHeartbeatTime: \"2024-01-01T02:00:00Z\"\nlastTransitionTime: \"2024-01-01T03:00:00Z\"\nstatus: \"True\"\ntype: Ready\n"),
						Principal:    "user-1",
						ChangedTime:  baseTime.Add(3 * time.Hour),
						StateType:    commonlogk8saudit_contract.RevisionStateConditionTrue,
					},
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			walker := newConditionWalker(conditionPath, "Ready")
			for _, tt := range scenario.steps {
				t.Run(tt.name, func(t *testing.T) {
					l := log.NewLogWithFieldSetsForTest()
					cs := khifilev6.NewTimelineChangeSet(l)
					walker.CheckAndRecord(ctx, commonFieldSet, k8sFieldSet, tt.condition, cs)

					if tt.want == nil {
						testchangeset.AssertTimeline(t, cs).HasNoRevision(conditionPath)
					} else {
						testchangeset.AssertTimeline(t, cs).HasRevision(conditionPath, tt.want, nodeComparer)
					}
				})
			}
		})
	}
}

func TestConditionLogToTimelineMapperTask_ProcessLog(t *testing.T) {
	taskSetting := &conditionLogToTimelineMapperTaskSetting{
		minimumDeltaTimeToCreateInferredCreationRevision: 10 * time.Second,
	}

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

	t.Run("PreProcessLog and ProcessLog standard lifecycle", func(t *testing.T) {
		builder := khifilev6.NewBuilder()
		cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
		api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
		kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
		ns := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
		parentPath := builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "nginx", Type: inspectioncore_contract.TimelineTypeResource})
		conditionPath := builder.TimelineAccumulator.GetPath(parentPath, khifilev6.PathSegment{Name: "Ready", Type: commonlogk8saudit_contract.TimelineTypeResourceCondition})

		ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

		commonFieldSet := &log.CommonFieldSet{
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		k8sFieldSet := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
			Verb:        commonlogk8saudit_contract.VerbUpdate,
			Principal:   "user-1",
			ClusterName: "k8s",
		}
		logObj := log.NewLogWithFieldSetsForTest(commonFieldSet, k8sFieldSet)

		bodyYAML := `
status:
  conditions:
  - type: Ready
    status: "True"
    lastTransitionTime: "2024-01-01T00:00:00Z"
`
		bodyNode := parseYAML(bodyYAML)
		bodyReader := structured.NewNodeReader(bodyNode)

		resIdentity := &commonlogk8saudit_contract.ResourceIdentity{
			APIVersion: "core/v1",
			Kind:       "pod",
			Namespace:  "default",
			Name:       "nginx",
		}

		groupSet := commonlogk8saudit_contract.RelatedGroupSet{
			Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
				"target": {
					Resource: resIdentity,
					Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
						{Log: logObj, ResourceBodyReader: bodyReader, ResourceBodyYAML: bodyYAML},
					},
				},
			},
		}

		event := commonlogk8saudit_contract.MultiGroupLogEvent{
			Log:              logObj,
			GroupRole:        "target",
			ResourceIdentity: resIdentity,
			EventType:        commonlogk8saudit_contract.ChangeEventTypeModification,
			GroupSet:         groupSet,
		}

		// 1. Run PreProcessLog
		state, err := taskSetting.PreProcessLog(ctx, 0, event, nil)
		if err != nil {
			t.Fatalf("PreProcessLog failed: %v", err)
		}

		// Verify state after pre-processing
		if _, exists := state.AvailableTypes["Ready"]; !exists {
			t.Errorf("expected 'Ready' in AvailableTypes")
		}
		walker := state.ConditionWalkers["Ready"]
		if walker == nil {
			t.Fatalf("expected condition walker for Ready")
		}
		if len(walker.lastTransitionStates) != 1 {
			t.Errorf("expected 1 last transition state, got %d", len(walker.lastTransitionStates))
		}

		// 2. Run ProcessLog
		cs, nextState, err := taskSetting.ProcessLog(ctx, event, state)
		if err != nil {
			t.Fatalf("ProcessLog failed: %v", err)
		}

		if nextState == nil {
			t.Fatalf("expected nextState not nil")
		}

		testchangeset.AssertTimeline(t, cs).
			HasRevision(conditionPath, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbUpdate,
				ResourceBody: parseYAML("lastTransitionTime: \"2024-01-01T00:00:00Z\"\nstatus: \"True\"\ntype: Ready\n"),
				Principal:    "user-1",
				ChangedTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				StateType:    commonlogk8saudit_contract.RevisionStateConditionTrue,
			}, nodeComparer)
	})

	t.Run("PreProcessLog and ProcessLog inferred creation", func(t *testing.T) {
		builder := khifilev6.NewBuilder()
		cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
		api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
		kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
		ns := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
		parentPath := builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "nginx", Type: inspectioncore_contract.TimelineTypeResource})
		conditionPath := builder.TimelineAccumulator.GetPath(parentPath, khifilev6.PathSegment{Name: "Ready", Type: commonlogk8saudit_contract.TimelineTypeResourceCondition})

		ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

		commonFieldSet := &log.CommonFieldSet{
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		k8sFieldSet := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
			Verb:        commonlogk8saudit_contract.VerbCreate,
			Principal:   "user-1",
			ClusterName: "k8s",
		}
		logObj := log.NewLogWithFieldSetsForTest(commonFieldSet, k8sFieldSet)

		bodyYAML := `
metadata:
  uid: uid-1
  creationTimestamp: "2023-12-31T23:59:00Z"
status:
  conditions: []
`
		bodyNode := parseYAML(bodyYAML)
		bodyReader := structured.NewNodeReader(bodyNode)

		resIdentity := &commonlogk8saudit_contract.ResourceIdentity{
			APIVersion: "core/v1",
			Kind:       "pod",
			Namespace:  "default",
			Name:       "nginx",
		}

		groupSet := commonlogk8saudit_contract.RelatedGroupSet{
			Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
				"target": {
					Resource: resIdentity,
					Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
						{Log: logObj, ResourceBodyReader: bodyReader, ResourceBodyYAML: bodyYAML},
					},
				},
			},
		}

		event := commonlogk8saudit_contract.MultiGroupLogEvent{
			Log:              logObj,
			GroupRole:        "target",
			ResourceIdentity: resIdentity,
			EventType:        commonlogk8saudit_contract.ChangeEventTypeCreation,
			GroupSet:         groupSet,
		}

		initialState := &conditionLogToTimelineMapperTaskState{
			AvailableTypes: map[string]struct{}{"Ready": {}},
			ConditionWalkers: map[string]*conditionWalker{
				"Ready": newConditionWalker(conditionPath, "Ready"),
			},
			uidToCreationTimestampMap: map[string]time.Time{
				"uid-1": time.Date(2023, 12, 31, 23, 59, 0, 0, time.UTC),
			},
		}

		cs, _, err := taskSetting.ProcessLog(ctx, event, initialState)
		if err != nil {
			t.Fatalf("ProcessLog failed: %v", err)
		}

		testchangeset.AssertTimeline(t, cs).
			HasRevision(conditionPath, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbCreate,
				ResourceBody: nil,
				Principal:    "user-1",
				ChangedTime:  time.Date(2023, 12, 31, 23, 59, 0, 0, time.UTC),
				StateType:    commonlogk8saudit_contract.RevisionStateConditionNoAvailableInfo,
			}, nodeComparer)
	})

	t.Run("PreProcessLog and ProcessLog inferred creation with UID tracking", func(t *testing.T) {
		builder := khifilev6.NewBuilder()
		cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
		api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
		kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
		ns := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
		parentPath := builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "nginx", Type: inspectioncore_contract.TimelineTypeResource})
		conditionPath := builder.TimelineAccumulator.GetPath(parentPath, khifilev6.PathSegment{Name: "Ready", Type: commonlogk8saudit_contract.TimelineTypeResourceCondition})

		ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

		// Log 1: update event at t=10s, contains UID and creationTimestamp
		logObj1 := log.NewLogWithFieldSetsForTest(
			&log.CommonFieldSet{Timestamp: time.Date(2024, 1, 1, 0, 0, 10, 0, time.UTC)},
			&commonlogk8saudit_contract.K8sAuditLogFieldSet{Verb: commonlogk8saudit_contract.VerbUpdate, Principal: "user-1", ClusterName: "k8s"},
		)
		bodyYAML1 := `
metadata:
  uid: "uid-1"
  creationTimestamp: "2023-12-31T23:59:00Z"
status:
  conditions:
  - type: Ready
    status: "True"
    lastTransitionTime: "2024-01-01T00:00:00Z"
`
		bodyReader1 := structured.NewNodeReader(parseYAML(bodyYAML1))

		// Log 2: patch event at t=5s (first event), lacks UID and creationTimestamp in body
		logObj2 := log.NewLogWithFieldSetsForTest(
			&log.CommonFieldSet{Timestamp: time.Date(2024, 1, 1, 0, 0, 5, 0, time.UTC)},
			&commonlogk8saudit_contract.K8sAuditLogFieldSet{Verb: commonlogk8saudit_contract.VerbPatch, Principal: "user-1", ClusterName: "k8s"},
		)
		bodyYAML2 := `
status:
  conditions:
  - type: Ready
    status: "True"
    lastTransitionTime: "2024-01-01T00:00:00Z"
`
		bodyReader2 := structured.NewNodeReader(parseYAML(bodyYAML2))

		resIdentity := &commonlogk8saudit_contract.ResourceIdentity{
			APIVersion: "core/v1",
			Kind:       "pod",
			Namespace:  "default",
			Name:       "nginx",
		}

		groupSet := commonlogk8saudit_contract.RelatedGroupSet{
			Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
				"target": {
					Resource: resIdentity,
					Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
						{Log: logObj2, ResourceBodyReader: bodyReader2, ResourceBodyYAML: bodyYAML2},
						{Log: logObj1, ResourceBodyReader: bodyReader1, ResourceBodyYAML: bodyYAML1},
					},
				},
			},
		}

		event1 := commonlogk8saudit_contract.MultiGroupLogEvent{
			Log:              logObj2,
			GroupRole:        "target",
			ResourceIdentity: resIdentity,
			EventType:        commonlogk8saudit_contract.ChangeEventTypeCreation,
			GroupSet:         groupSet,
		}
		event2 := commonlogk8saudit_contract.MultiGroupLogEvent{
			Log:              logObj1,
			GroupRole:        "target",
			ResourceIdentity: resIdentity,
			EventType:        commonlogk8saudit_contract.ChangeEventTypeModification,
			GroupSet:         groupSet,
		}

		var state *conditionLogToTimelineMapperTaskState
		var err error

		// Run PreProcessLog
		state, err = taskSetting.PreProcessLog(ctx, 0, event2, state)
		if err != nil {
			t.Fatalf("PreProcessLog 1 failed: %v", err)
		}
		state, err = taskSetting.PreProcessLog(ctx, 0, event1, state)
		if err != nil {
			t.Fatalf("PreProcessLog 2 failed: %v", err)
		}

		// Run ProcessLog for the first event (Log 2)
		cs, _, err := taskSetting.ProcessLog(ctx, event1, state)
		if err != nil {
			t.Fatalf("ProcessLog failed: %v", err)
		}

		// The creationTimestamp resolved from uid-1 is "2023-12-31T23:59:00Z".
		// Log 2 timestamp is "2024-01-01T00:00:05Z".
		// Difference is > 10 seconds, so it should generate inferred creation revision.
		testchangeset.AssertTimeline(t, cs).
			HasRevision(conditionPath, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbPatch,
				ResourceBody: nil,
				Principal:    "user-1",
				ChangedTime:  time.Date(2023, 12, 31, 23, 59, 0, 0, time.UTC),
				StateType:    commonlogk8saudit_contract.RevisionStateConditionNoAvailableInfo,
			}, nodeComparer)
	})

	t.Run("PreProcessLog and ProcessLog deletion", func(t *testing.T) {
		builder := khifilev6.NewBuilder()
		cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
		api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
		kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
		ns := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
		parentPath := builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "nginx", Type: inspectioncore_contract.TimelineTypeResource})
		conditionPath := builder.TimelineAccumulator.GetPath(parentPath, khifilev6.PathSegment{Name: "Ready", Type: commonlogk8saudit_contract.TimelineTypeResourceCondition})

		ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

		commonFieldSet := &log.CommonFieldSet{
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		k8sFieldSet := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
			Verb:        commonlogk8saudit_contract.VerbDelete,
			Principal:   "user-1",
			ClusterName: "k8s",
		}
		logObj := log.NewLogWithFieldSetsForTest(commonFieldSet, k8sFieldSet)

		bodyYAML := `
status:
  conditions: []
`
		bodyNode := parseYAML(bodyYAML)
		bodyReader := structured.NewNodeReader(bodyNode)

		resIdentity := &commonlogk8saudit_contract.ResourceIdentity{
			APIVersion: "core/v1",
			Kind:       "pod",
			Namespace:  "default",
			Name:       "nginx",
		}

		groupSet := commonlogk8saudit_contract.RelatedGroupSet{
			Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
				"target": {
					Resource: resIdentity,
					Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
						{Log: logObj, ResourceBodyReader: bodyReader, ResourceBodyYAML: bodyYAML},
					},
				},
			},
		}

		event := commonlogk8saudit_contract.MultiGroupLogEvent{
			Log:              logObj,
			GroupRole:        "target",
			ResourceIdentity: resIdentity,
			EventType:        commonlogk8saudit_contract.ChangeEventTypeDeletion,
			GroupSet:         groupSet,
		}

		initialState := &conditionLogToTimelineMapperTaskState{
			AvailableTypes: map[string]struct{}{"Ready": {}},
			ConditionWalkers: map[string]*conditionWalker{
				"Ready": newConditionWalker(conditionPath, "Ready"),
			},
		}

		cs, _, err := taskSetting.ProcessLog(ctx, event, initialState)
		if err != nil {
			t.Fatalf("ProcessLog failed: %v", err)
		}

		testchangeset.AssertTimeline(t, cs).
			HasRevision(conditionPath, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbDelete,
				ResourceBody: nil,
				Principal:    "user-1",
				ChangedTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleted,
			}, nodeComparer)
	})
}
