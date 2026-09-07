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
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/google/go-cmp/cmp"
)

func TestNewTag(t *testing.T) {
	testCases := []struct {
		name        string
		inputID     string
		wantID      string
		shouldPanic bool
		panicMatch  string
	}{
		{
			name:        "valid tag id",
			inputID:     "khi.google.com/tag/test",
			wantID:      "khi.google.com/tag/test",
			shouldPanic: false,
		},
		{
			name:        "empty tag id panics",
			inputID:     "",
			shouldPanic: true,
			panicMatch:  "tag id must not be empty",
		},
		{
			name:        "whitespace-only tag id panics",
			inputID:     "   ",
			shouldPanic: true,
			panicMatch:  "tag id must not be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldPanic {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected panic containing %q, but none occurred", tc.panicMatch)
						return
					}
					msg := fmt.Sprint(r)
					if !strings.Contains(msg, tc.panicMatch) {
						t.Errorf("expected panic message to contain %q, got: %v", tc.panicMatch, msg)
					}
				}()
			}

			tag := NewTag[string](tc.inputID)
			if !tc.shouldPanic {
				if diff := cmp.Diff(tc.wantID, tag.ID()); diff != "" {
					t.Errorf("tag.ID() mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tc.wantID, tag.String()); diff != "" {
					t.Errorf("tag.String() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestTagRef(t *testing.T) {
	tag := NewTag[int]("khi.google.com/tag/numbers")

	testCases := []struct {
		name            string
		opts            []taskid.FanInOption
		wantKind        taskid.EdgeKind
		wantCondition   taskid.EdgeCondition
		wantCardinality taskid.EdgeCardinality
		wantScope       taskid.DependencyScope
	}{
		{
			name:            "default tag reference",
			opts:            nil,
			wantKind:        taskid.EdgeKindData,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveGraph,
		},
		{
			name:            "order-only tag reference from active features",
			opts:            []taskid.FanInOption{taskid.OrderOnly, taskid.FromActiveFeatures},
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveFeatures,
		},
		{
			name:            "tag reference from all",
			opts:            []taskid.FanInOption{taskid.FromAll},
			wantKind:        taskid.EdgeKindData,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeAll,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := tag.Ref(tc.opts...)
			if diff := cmp.Diff(tag.ID(), ref.Tag()); diff != "" {
				t.Errorf("Tag() mismatch (-want +got):\n%s", diff)
			}
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
		})
	}
}

func TestProvidesTag(t *testing.T) {
	tag := NewTag[[]string]("khi.google.com/tag/names")

	testCases := []struct {
		name       string
		tag        Tag[[]string]
		wantLabel  string
		wantVal    bool
		wantPrefix string
	}{
		{
			name:       "provides tag label is set correctly",
			tag:        tag,
			wantLabel:  LabelKeyProvidedTag(tag.ID()).Key(),
			wantVal:    true,
			wantPrefix: KHISystemPrefix + "provided-tag/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opt := ProvidesTag(tc.tag)
			labels := typedmap.NewTypedMap()
			opt.Write(labels)

			val, found := typedmap.Get(labels, LabelKeyProvidedTag(tc.tag.ID()))
			if diff := cmp.Diff(true, found); diff != "" {
				t.Errorf("label found mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantVal, val); diff != "" {
				t.Errorf("label value mismatch (-want +got):\n%s", diff)
			}
			key := LabelKeyProvidedTag(tc.tag.ID()).Key()
			if !strings.HasPrefix(key, tc.wantPrefix) {
				t.Errorf("expected prefix %s, got %s", tc.wantPrefix, key)
			}
		})
	}
}
