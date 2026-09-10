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
		wantCardinality taskid.EdgeCardinality
		wantScope       taskid.DependencyScope
	}{
		{
			name:            "default tag reference",
			opts:            nil,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveFeatures,
		},
		{
			name:            "tag reference from active graph",
			opts:            []taskid.FanInOption{taskid.ScopeActiveGraph},
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveGraph,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := tag.Ref(tc.opts...)
			if got := ref.Tag(); got != tag.ID() {
				t.Errorf("Tag() = %q, want %q", got, tag.ID())
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
		name         string
		tag          Tag[[]string]
		opts         []ProvidesTagOption
		wantVal      bool
		wantPriority int
		wantPrefix   string
	}{
		{
			name:         "provides tag label with default priority",
			tag:          tag,
			opts:         nil,
			wantVal:      true,
			wantPriority: DefaultTagPriority,
			wantPrefix:   KHISystemPrefix + "provided-tag/",
		},
		{
			name:         "provides tag label with explicit custom priority",
			tag:          tag,
			opts:         []ProvidesTagOption{WithTagPriority(10)},
			wantVal:      true,
			wantPriority: 10,
			wantPrefix:   KHISystemPrefix + "provided-tag/",
		},
		{
			name:         "provides tag label with multiple priorities selects last",
			tag:          tag,
			opts:         []ProvidesTagOption{WithTagPriority(20), WithTagPriority(5)},
			wantVal:      true,
			wantPriority: 5,
			wantPrefix:   KHISystemPrefix + "provided-tag/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opt := ProvidesTag(tc.tag, tc.opts...)
			labels := typedmap.NewTypedMap()
			opt.Write(labels)

			val, found := typedmap.Get(labels, LabelKeyProvidedTag(tc.tag.ID()))
			if !found {
				t.Errorf("expected provided-tag label to be found, but was not")
			}
			if val != tc.wantVal {
				t.Errorf("label value = %v, want %v", val, tc.wantVal)
			}
			key := LabelKeyProvidedTag(tc.tag.ID()).Key()
			if !strings.HasPrefix(key, tc.wantPrefix) {
				t.Errorf("expected prefix %s, got %s", tc.wantPrefix, key)
			}

			priorityVal, priorityFound := typedmap.Get(labels, LabelKeyProvidedTagPriority(tc.tag.ID()))
			if !priorityFound {
				t.Errorf("expected provided-tag-priority label to be found, but was not")
			}
			if priorityVal != tc.wantPriority {
				t.Errorf("priority label value = %d, want %d", priorityVal, tc.wantPriority)
			}
			priorityKey := LabelKeyProvidedTagPriority(tc.tag.ID()).Key()
			if !strings.HasPrefix(priorityKey, LabelKeyProvidedTagPriorityPrefix) {
				t.Errorf("expected prefix %s, got %s", LabelKeyProvidedTagPriorityPrefix, priorityKey)
			}
		})
	}
}
