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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

func TestWithLabelValue(t *testing.T) {
	testStringKey := NewTaskLabelKey[string]("test-string-key")
	testIntKey := NewTaskLabelKey[int]("test-int-key")

	t.Run("string label", func(t *testing.T) {
		testCases := []struct {
			name  string
			value string
			want  string
		}{
			{
				name:  "sets non-empty string value",
				value: "hello",
				want:  "hello",
			},
			{
				name:  "sets empty string value",
				value: "",
				want:  "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				labels := NewLabelSet(WithLabelValue(testStringKey, tc.value))
				got, found := typedmap.Get(labels, testStringKey)
				if !found {
					t.Fatalf("key %q not found in label set", testStringKey)
				}
				if got != tc.want {
					t.Errorf("got %q, want %q", got, tc.want)
				}
			})
		}
	})

	t.Run("int label", func(t *testing.T) {
		testCases := []struct {
			name  string
			value int
			want  int
		}{
			{
				name:  "sets positive integer",
				value: 42,
				want:  42,
			},
			{
				name:  "sets zero",
				value: 0,
				want:  0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				labels := NewLabelSet(WithLabelValue(testIntKey, tc.value))
				got, found := typedmap.Get(labels, testIntKey)
				if !found {
					t.Fatalf("key %q not found in label set", testIntKey)
				}
				if got != tc.want {
					t.Errorf("got %d, want %d", got, tc.want)
				}
			})
		}
	})
}

func TestWithFeatureGate(t *testing.T) {
	testCases := []struct {
		name      string
		ref       taskid.UntypedTaskReference
		wantPanic bool
		wantRefID string
	}{
		{
			name:      "sets valid feature gate reference",
			ref:       taskid.NewTaskReference[any]("my-feature"),
			wantPanic: false,
			wantRefID: "my-feature",
		},
		{
			name:      "panics on nil reference",
			ref:       nil,
			wantPanic: true,
		},
		{
			name:      "panics on empty reference id",
			ref:       taskid.NewTaskReference[any](""),
			wantPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic, got nil")
					}
				}()
			}
			labels := NewLabelSet(WithFeatureGate(tc.ref))
			if tc.wantPanic {
				return
			}
			got, found := typedmap.Get(labels, LabelKeyFeatureGateTaskRef)
			if !found {
				t.Fatalf("key %q not found in label set", LabelKeyFeatureGateTaskRef)
			}
			if got.ReferenceIDString() != tc.wantRefID {
				t.Errorf("got %q, want %q", got.ReferenceIDString(), tc.wantRefID)
			}
		})
	}
}
