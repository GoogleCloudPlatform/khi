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
				if got := tag.ID(); got != tc.wantID {
					t.Errorf("tag.ID() = %q, want %q", got, tc.wantID)
				}
				if got := tag.String(); got != tc.wantID {
					t.Errorf("tag.String() = %q, want %q", got, tc.wantID)
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
			opts:            []taskid.FanInOption{taskid.OrderOnly, taskid.ScopeActiveFeatures},
			wantKind:        taskid.EdgeKindOrderOnly,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveFeatures,
		},
		{
			name:            "tag reference from all",
			opts:            []taskid.FanInOption{taskid.ScopeAll},
			wantKind:        taskid.EdgeKindData,
			wantCondition:   taskid.ConditionRequired,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeAll,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := tag.Ref(tc.opts...)
			if got := ref.Tag(); got != tag.ID() {
				t.Errorf("Tag() = %q, want %q", got, tag.ID())
			}
			if got := ref.DescriptorKind(); got != tc.wantKind {
				t.Errorf("DescriptorKind() = %v, want %v", got, tc.wantKind)
			}
			if got := ref.DescriptorCondition(); got != tc.wantCondition {
				t.Errorf("DescriptorCondition() = %v, want %v", got, tc.wantCondition)
			}
			if got := ref.DescriptorCardinality(); got != tc.wantCardinality {
				t.Errorf("DescriptorCardinality() = %v, want %v", got, tc.wantCardinality)
			}
			if got := ref.DescriptorScope(); got != tc.wantScope {
				t.Errorf("DescriptorScope() = %v, want %v", got, tc.wantScope)
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
			if !found {
				t.Errorf("expected label to be found, but was not")
			}
			if val != tc.wantVal {
				t.Errorf("label value = %v, want %v", val, tc.wantVal)
			}
			key := LabelKeyProvidedTag(tc.tag.ID()).Key()
			if !strings.HasPrefix(key, tc.wantPrefix) {
				t.Errorf("expected prefix %s, got %s", tc.wantPrefix, key)
			}
		})
	}
}
