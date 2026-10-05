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
		name      string
		kind      InputKind
		want      string
		wantPanic string
	}{
		{name: "required", kind: InputKindRequired, want: "required"},
		{name: "optional", kind: InputKindOptional, want: "optional"},
		{name: "tag", kind: InputKindTag, want: "tag"},
		{name: "ordering", kind: InputKindOrdering, want: "ordering"},
		{name: "unknown kind panics", kind: InputKind(42), wantPanic: "unknown InputKind 42"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			gotPanic := panicMessage(func() { got = tc.kind.String() })
			if gotPanic != tc.wantPanic {
				t.Fatalf("InputKind(%d).String() panic = %q, want %q", int(tc.kind), gotPanic, tc.wantPanic)
			}
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
			name: "merges an input declared twice into one spec",
			bind: func(b *Binder) {
				Use(b, requiredRef)
				Use(b, requiredRef)
			},
			wantSpecs: []string{"required ref:required"},
		},
		{
			name: "merges an ordering dependency into the value input it duplicates",
			bind: func(b *Binder) {
				Use(b, requiredRef)
				After(b, orderingRef)
				After(b, requiredRef)
			},
			wantSpecs: []string{"required ref:required", "ordering ref:ordering"},
		},
		{
			name: "panics when declarations of the same input have conflicting result types",
			bind: func(b *Binder) {
				Use(b, requiredRef)
				After(b, taskid.NewTaskReference[int]("required"))
			},
			wantPanic: "declares input ref:required with conflicting result types string and int",
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
		{
			name: "panics when an input is nil",
			bind: func(b *Binder) {
				After(b, nil)
			},
			wantPanic: "unsupported dependency <nil>",
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

func TestBinder_MergesDuplicateInputs(t *testing.T) {
	ref := taskid.NewTaskReference[string]("merged")
	activeGraphRef := taskid.NewTaskReference[string]("merged", taskid.ScopeActiveGraph)
	activeFeaturesRef := taskid.NewTaskReference[string]("merged", taskid.ScopeActiveFeatures)
	tag := NewTag[string]("binder-merge-tag")

	testCases := []struct {
		name      string
		bind      func(b *Binder)
		wantKind  InputKind
		wantScope taskid.DependencyScope
	}{
		{
			name: "required input merged with an ordering dependency declared before it is required",
			bind: func(b *Binder) {
				After(b, ref)
				Use(b, ref)
			},
			wantKind:  InputKindRequired,
			wantScope: taskid.ScopeAll,
		},
		{
			name: "required input merged with an ordering dependency declared after it is required",
			bind: func(b *Binder) {
				Use(b, ref)
				After(b, activeGraphRef)
			},
			wantKind:  InputKindRequired,
			wantScope: taskid.ScopeAll,
		},
		{
			name: "required input merged with an optional input declared before it is required",
			bind: func(b *Binder) {
				UseOptional(b, activeGraphRef)
				Use(b, ref)
			},
			wantKind:  InputKindRequired,
			wantScope: taskid.ScopeAll,
		},
		{
			name: "required input merged with an optional input declared after it is required",
			bind: func(b *Binder) {
				Use(b, ref)
				UseOptional(b, activeGraphRef)
			},
			wantKind:  InputKindRequired,
			wantScope: taskid.ScopeAll,
		},
		{
			name: "optional input merged with an optional input with a broader scope is optional with the broader scope",
			bind: func(b *Binder) {
				UseOptional(b, activeGraphRef)
				UseOptional(b, activeFeaturesRef)
			},
			wantKind:  InputKindOptional,
			wantScope: taskid.ScopeActiveFeatures,
		},
		{
			name: "optional input merged with a ScopeActiveFeatures ordering dependency declared after it is optional with the broader scope",
			bind: func(b *Binder) {
				UseOptional(b, activeGraphRef)
				After(b, activeFeaturesRef)
			},
			wantKind:  InputKindOptional,
			wantScope: taskid.ScopeActiveFeatures,
		},
		{
			name: "optional input merged with a ScopeActiveFeatures ordering dependency declared before it is optional with the broader scope",
			bind: func(b *Binder) {
				After(b, activeFeaturesRef)
				UseOptional(b, activeGraphRef)
			},
			wantKind:  InputKindOptional,
			wantScope: taskid.ScopeActiveFeatures,
		},
		{
			name: "optional input merged with a ScopeAll ordering dependency declared after it is required",
			bind: func(b *Binder) {
				UseOptional(b, activeGraphRef)
				After(b, ref)
			},
			wantKind:  InputKindRequired,
			wantScope: taskid.ScopeAll,
		},
		{
			name: "optional input merged with a ScopeAll ordering dependency declared before it is required",
			bind: func(b *Binder) {
				After(b, ref)
				UseOptional(b, activeGraphRef)
			},
			wantKind:  InputKindRequired,
			wantScope: taskid.ScopeAll,
		},
		{
			name: "optional input merged with ordering dependencies declared before and after it is required when one uses ScopeAll",
			bind: func(b *Binder) {
				After(b, activeGraphRef)
				UseOptional(b, activeGraphRef)
				After(b, ref)
			},
			wantKind:  InputKindRequired,
			wantScope: taskid.ScopeAll,
		},
		{
			name: "tag input merged with an ordering dependency declared after it is a tag input",
			bind: func(b *Binder) {
				UseTag(b, tag.Ref())
				After(b, tag.Ref())
			},
			wantKind:  InputKindTag,
			wantScope: taskid.ScopeActiveFeatures,
		},
		{
			name: "tag input merged with an ordering dependency declared before it is a tag input",
			bind: func(b *Binder) {
				After(b, tag.Ref(taskid.ScopeActiveGraph))
				UseTag(b, tag.Ref())
			},
			wantKind:  InputKindTag,
			wantScope: taskid.ScopeActiveFeatures,
		},
		{
			name: "tag input merged with a tag input with a broader scope is a tag input with the broader scope",
			bind: func(b *Binder) {
				UseTag(b, tag.Ref(taskid.ScopeActiveGraph))
				UseTag(b, tag.Ref())
			},
			wantKind:  InputKindTag,
			wantScope: taskid.ScopeActiveFeatures,
		},
		{
			name: "ordering dependency merged with an ordering dependency with a broader scope is ordering with the broader scope",
			bind: func(b *Binder) {
				After(b, activeGraphRef)
				After(b, activeFeaturesRef)
			},
			wantKind:  InputKindOrdering,
			wantScope: taskid.ScopeActiveFeatures,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := newBinder(taskid.NewDefaultImplementationID[string]("owner"))
			tc.bind(b)
			if len(b.specs) != 1 {
				t.Fatalf("len(b.specs) = %d, want 1: %v", len(b.specs), describeInputSpecs(b.specs))
			}
			if got := b.specs[0].Kind; got != tc.wantKind {
				t.Errorf("Kind = %v, want %v", got, tc.wantKind)
			}
			if got := b.specs[0].Dependency.DescriptorScope(); got != tc.wantScope {
				t.Errorf("DescriptorScope() = %v, want %v", got, tc.wantScope)
			}
		})
	}
}
