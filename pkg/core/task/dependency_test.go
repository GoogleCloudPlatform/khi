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

package coretask

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/google/go-cmp/cmp"
)

func TestTagReference(t *testing.T) {
	testCases := []struct {
		name            string
		tag             string
		opts            []taskid.FanInOption
		wantKind        taskid.EdgeKind
		wantCondition   taskid.EdgeCondition
		wantCardinality taskid.EdgeCardinality
		wantScope       taskid.DependencyScope
		wantTag         string
	}{
		{
			name:            "default tag reference",
			tag:             "test/tag",
			opts:            nil,
			wantKind:        taskid.EdgeKindData,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveGraph,
			wantTag:         "test/tag",
		},
		{
			name:            "order-only tag reference",
			tag:             "test/order_only_tag",
			opts:            []taskid.FanInOption{taskid.OrderOnly},
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveGraph,
			wantTag:         "test/order_only_tag",
		},
		{
			name:            "tag reference from active features",
			tag:             "test/features_tag",
			opts:            []taskid.FanInOption{taskid.FromActiveFeatures},
			wantKind:        taskid.EdgeKindData,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveFeatures,
			wantTag:         "test/features_tag",
		},
		{
			name:            "tag reference from all",
			tag:             "test/all_tag",
			opts:            []taskid.FanInOption{taskid.FromAll},
			wantKind:        taskid.EdgeKindData,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeAll,
			wantTag:         "test/all_tag",
		},
		{
			name:            "tag reference order-only from all",
			tag:             "test/order_all_tag",
			opts:            []taskid.FanInOption{taskid.OrderOnly, taskid.FromAll},
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeAll,
			wantTag:         "test/order_all_tag",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := NewTagReference[int](tc.tag, tc.opts...)

			if diff := cmp.Diff(tc.wantKind, ref.DescriptorKind()); diff != "" {
				t.Errorf("DescriptorKind() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantCondition, ref.DescriptorCondition()); diff != "" {
				t.Errorf("DescriptorCondition() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantCardinality, ref.DescriptorCardinality()); diff != "" {
				t.Errorf("DescriptorCardinality() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantScope, ref.DescriptorScope()); diff != "" {
				t.Errorf("DescriptorScope() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantTag, ref.Tag()); diff != "" {
				t.Errorf("Tag() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, ref.GetZeroValue()); diff != "" {
				t.Errorf("GetZeroValue() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToOrderOnly(t *testing.T) {
	testCases := []struct {
		name            string
		input           Dependency
		wantKind        taskid.EdgeKind
		wantCondition   taskid.EdgeCondition
		wantCardinality taskid.EdgeCardinality
		wantScope       taskid.DependencyScope
		wantRefID       string
		wantTag         string
	}{
		{
			name:            "point-to-point data to order-only",
			input:           taskid.NewTaskReference[string]("task.a"),
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityPointToPoint,
			wantScope:       taskid.ScopeAll,
			wantRefID:       "task.a",
		},
		{
			name:            "point-to-point optional to order-only preserving optional",
			input:           taskid.NewTaskReference[string]("task.b", taskid.Optional, taskid.FromActiveFeatures),
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionOptional,
			wantCardinality: taskid.CardinalityPointToPoint,
			wantScope:       taskid.ScopeActiveFeatures,
			wantRefID:       "task.b",
		},
		{
			name:            "point-to-point from active graph to order-only",
			input:           taskid.NewTaskReference[string]("task.c", taskid.FromActiveGraph),
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityPointToPoint,
			wantScope:       taskid.ScopeActiveGraph,
			wantRefID:       "task.c",
		},
		{
			name:            "tag reference data to order-only",
			input:           NewTagReference[string]("tag.foo", taskid.FromAll),
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeAll,
			wantTag:         "tag.foo",
		},
		{
			name:            "tag reference default active graph to order-only",
			input:           NewTagReference[string]("tag.bar"),
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveGraph,
			wantTag:         "tag.bar",
		},
		{
			name:            "tag reference from active features to order-only",
			input:           NewTagReference[string]("tag.features", taskid.FromActiveFeatures),
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveFeatures,
			wantTag:         "tag.features",
		},
		{
			name:            "already order-only remains unchanged",
			input:           taskid.NewTaskReference[string]("task.d", taskid.OrderOnly),
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityPointToPoint,
			wantScope:       taskid.ScopeAll,
			wantRefID:       "task.d",
		},
		{
			name:            "already order-only tag reference remains unchanged",
			input:           NewTagReference[string]("tag.already_order_only", taskid.OrderOnly),
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveGraph,
			wantTag:         "tag.already_order_only",
		},
		{
			name:            "custom dependency descriptor returns unchanged",
			input:           customDependency{kind: taskid.EdgeKindData},
			wantKind:        taskid.EdgeKindData,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityPointToPoint,
			wantScope:       taskid.ScopeAll,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ToOrderOnly(tc.input)

			if diff := cmp.Diff(tc.wantKind, got.DescriptorKind()); diff != "" {
				t.Errorf("DescriptorKind() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantCondition, got.DescriptorCondition()); diff != "" {
				t.Errorf("DescriptorCondition() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantCardinality, got.DescriptorCardinality()); diff != "" {
				t.Errorf("DescriptorCardinality() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantScope, got.DescriptorScope()); diff != "" {
				t.Errorf("DescriptorScope() mismatch (-want +got):\n%s", diff)
			}
			if tc.wantRefID != "" {
				p2p, ok := got.(taskid.PointToPointDescriptor)
				if !ok {
					t.Fatalf("expected PointToPointDescriptor, got %T", got)
				}
				if diff := cmp.Diff(tc.wantRefID, p2p.ReferenceID()); diff != "" {
					t.Errorf("ReferenceID() mismatch (-want +got):\n%s", diff)
				}
			}
			if tc.wantTag != "" {
				fanIn, ok := got.(taskid.FanInDescriptor)
				if !ok {
					t.Fatalf("expected FanInDescriptor, got %T", got)
				}
				if diff := cmp.Diff(tc.wantTag, fanIn.Tag()); diff != "" {
					t.Errorf("Tag() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

type customDependency struct {
	kind taskid.EdgeKind
}

func (c customDependency) DescriptorKind() taskid.EdgeKind {
	return c.kind
}

func (c customDependency) DescriptorCondition() taskid.EdgeCondition {
	return taskid.ConditionRequired
}

func (c customDependency) DescriptorCardinality() taskid.EdgeCardinality {
	return taskid.CardinalityPointToPoint
}

func (c customDependency) DescriptorScope() taskid.DependencyScope {
	return taskid.ScopeAll
}

var _ taskid.DependencyDescriptor = (*customDependency)(nil)
