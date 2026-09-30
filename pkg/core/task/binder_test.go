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

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/google/go-cmp/cmp"
)

// panicMessage runs f and returns the message of the recovered panic, or an empty string when f does not panic.
func panicMessage(f func()) (msg string) {
	defer func() {
		r := recover()
		switch v := r.(type) {
		case nil:
			msg = ""
		case error:
			msg = v.Error()
		default:
			msg = fmt.Sprint(v)
		}
	}()
	f()
	return ""
}

// describeInputSpecs converts input specs to comparable strings, because dependency descriptors hold unexported fields.
func describeInputSpecs(specs []InputSpec) []string {
	result := make([]string, 0, len(specs))
	for _, spec := range specs {
		result = append(result, fmt.Sprintf("%s %s", spec.Kind, dependencyKey(spec.Dependency)))
	}
	return result
}

func TestInputKind_String(t *testing.T) {
	testCases := []struct {
		kind InputKind
		want string
	}{
		{kind: InputKindRequired, want: "required"},
		{kind: InputKindOptional, want: "optional"},
		{kind: InputKindTag, want: "tag"},
		{kind: InputKindOrdering, want: "ordering"},
		{kind: InputKind(42), want: "InputKind(42)"},
	}
	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			got := tc.kind.String()
			if got != tc.want {
				t.Errorf("InputKind(%d).String() = %q, want %q", int(tc.kind), got, tc.want)
			}
		})
	}
}

func TestBinder(t *testing.T) {
	requiredRef := taskid.NewTaskReference[string]("required")
	optionalRef := taskid.NewTaskReference[int]("optional", taskid.ScopeActiveGraph)
	orderingRef := taskid.NewTaskReference[struct{}]("ordering")
	tag := NewTag[string]("binder-test-tag")

	testCases := []struct {
		name      string
		bind      func(b *Binder)
		wantSpecs []string
		wantPanic string
	}{
		{
			name: "records inputs in declaration order",
			bind: func(b *Binder) {
				UseTag(b, tag.Ref())
				Use(b, requiredRef)
				After(b, orderingRef)
				UseOptional(b, optionalRef)
			},
			wantSpecs: []string{
				"tag tag:binder-test-tag",
				"required ref:required",
				"ordering ref:ordering",
				"optional ref:optional",
			},
		},
		{
			name:      "records no inputs when nothing is declared",
			bind:      func(b *Binder) {},
			wantSpecs: []string{},
		},
		{
			name: "panics when the same input is declared twice",
			bind: func(b *Binder) {
				Use(b, requiredRef)
				Use(b, requiredRef)
			},
			wantPanic: "declares input ref:required twice",
		},
		{
			name: "panics when an ordering dependency duplicates a value input",
			bind: func(b *Binder) {
				Use(b, requiredRef)
				After(b, requiredRef)
			},
			wantPanic: "declares input ref:required twice",
		},
		{
			name: "panics when a required input uses a scope other than ScopeAll",
			bind: func(b *Binder) {
				Use(b, taskid.NewTaskReference[string]("required", taskid.ScopeActiveGraph))
			},
			wantPanic: "declares required input required with a scope other than ScopeAll",
		},
		{
			name: "panics when an optional input uses ScopeAll",
			bind: func(b *Binder) {
				UseOptional(b, requiredRef)
			},
			wantPanic: "declares optional input required with ScopeAll",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := newBinder(taskid.NewDefaultImplementationID[string]("owner"))
			gotPanic := panicMessage(func() { tc.bind(b) })
			if tc.wantPanic != "" {
				if !strings.Contains(gotPanic, tc.wantPanic) {
					t.Fatalf("bind panic = %q, want substring %q", gotPanic, tc.wantPanic)
				}
				return
			}
			if gotPanic != "" {
				t.Fatalf("bind panicked unexpectedly: %s", gotPanic)
			}
			if diff := cmp.Diff(tc.wantSpecs, describeInputSpecs(b.specs)); diff != "" {
				t.Errorf("Binder specs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
