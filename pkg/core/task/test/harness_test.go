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

package tasktest

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// fatalRecorder records the message passed to Fatalf and stops the calling goroutine like testing.T does.
type fatalRecorder struct {
	testing.TB
	fatalMessage string
}

func (r *fatalRecorder) Helper() {}

func (r *fatalRecorder) Fatalf(format string, args ...any) {
	r.fatalMessage = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// runWithFatalRecorder runs Run in a separate goroutine so that Fatalf can stop it without stopping the test.
func runWithFatalRecorder[T any](t *testing.T, task coretask.DefinedTask[T], inputs ...InputValue) (T, string, error) {
	recorder := &fatalRecorder{TB: t}
	var result T
	var err error
	done := make(chan struct{})
	go func() {
		defer close(done)
		result, err = Run(recorder, t.Context(), task, inputs...)
	}()
	<-done
	return result, recorder.fatalMessage, err
}

func TestRun(t *testing.T) {
	requiredRef := taskid.NewTaskReference[string]("harness-test.required")
	optionalRef := taskid.NewTaskReference[int]("harness-test.optional", taskid.ScopeActiveGraph)
	orderingRef := taskid.NewTaskReference[struct{}]("harness-test.ordering")
	tag := coretask.NewTag[string]("harness-test-tag")
	task := coretask.Define(taskid.NewDefaultImplementationID[string]("harness-test.task"), func(b *coretask.Binder) func(ctx context.Context) (string, error) {
		required := coretask.Use(b, requiredRef)
		optional := coretask.UseOptional(b, optionalRef)
		tagged := coretask.UseTag(b, tag.Ref())
		coretask.After(b, orderingRef)
		return func(ctx context.Context) (string, error) {
			optionalValue, found := optional.Get(ctx)
			return fmt.Sprintf("required=%s optional=%d,%t tagged=%v", required.Get(ctx), optionalValue, found, tagged.Get(ctx)), nil
		}
	})

	testCases := []struct {
		name             string
		inputs           []InputValue
		want             string
		wantFatalSubstrs []string
	}{
		{
			name: "all inputs given",
			inputs: []InputValue{
				Given(requiredRef, "foo"),
				Given(optionalRef, 42),
				GivenTag(tag.Ref(), "a", "b"),
			},
			want: "required=foo optional=42,true tagged=[a b]",
		},
		{
			name: "omitted optional and tag inputs read defaults",
			inputs: []InputValue{
				Given(requiredRef, "foo"),
			},
			want: "required=foo optional=0,false tagged=[]",
		},
		{
			name:             "missing required input",
			inputs:           nil,
			wantFatalSubstrs: []string{"missing required input ref:harness-test.required (string)"},
		},
		{
			name: "undeclared input",
			inputs: []InputValue{
				Given(requiredRef, "foo"),
				Given(taskid.NewTaskReference[string]("harness-test.undeclared"), "bar"),
			},
			wantFatalSubstrs: []string{"undeclared input ref:harness-test.undeclared is given"},
		},
		{
			name: "input given more than once",
			inputs: []InputValue{
				Given(requiredRef, "foo"),
				Given(requiredRef, "bar"),
			},
			wantFatalSubstrs: []string{"input ref:harness-test.required is given more than once"},
		},
		{
			name: "value given to ordering-only input",
			inputs: []InputValue{
				Given(requiredRef, "foo"),
				Given(orderingRef, struct{}{}),
			},
			wantFatalSubstrs: []string{"input ref:harness-test.ordering is an ordering-only dependency and does not take a value"},
		},
		{
			name: "value with mismatched type",
			inputs: []InputValue{
				Given(taskid.NewTaskReference[int]("harness-test.required"), 1),
			},
			wantFatalSubstrs: []string{"input ref:harness-test.required expects string, but the given value is int"},
		},
		{
			name: "tag values with mismatched type",
			inputs: []InputValue{
				Given(requiredRef, "foo"),
				GivenTag(coretask.NewTag[int]("harness-test-tag").Ref(), 1),
			},
			wantFatalSubstrs: []string{"input tag:harness-test-tag expects string, but the given value is int"},
		},
		{
			name: "all problems reported together",
			inputs: []InputValue{
				Given(taskid.NewTaskReference[string]("harness-test.undeclared"), "bar"),
				Given(orderingRef, struct{}{}),
			},
			wantFatalSubstrs: []string{
				"task harness-test.task",
				"undeclared input ref:harness-test.undeclared is given",
				"input ref:harness-test.ordering is an ordering-only dependency",
				"missing required input ref:harness-test.required",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, fatalMessage, err := runWithFatalRecorder(t, task, tc.inputs...)
			if len(tc.wantFatalSubstrs) > 0 {
				if fatalMessage == "" {
					t.Fatalf("Run() did not fail the test, want failure containing %q", tc.wantFatalSubstrs)
				}
				for _, want := range tc.wantFatalSubstrs {
					if !strings.Contains(fatalMessage, want) {
						t.Errorf("Run() failure message = %q, want substring %q", fatalMessage, want)
					}
				}
				return
			}
			if fatalMessage != "" {
				t.Fatalf("Run() failed the test unexpectedly: %s", fatalMessage)
			}
			if err != nil {
				t.Fatalf("Run() returned unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Run() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRun_OptionalInputMergedIntoRequiredInput(t *testing.T) {
	promotedRef := taskid.NewTaskReference[string]("harness-test.promoted")
	task := coretask.Define(taskid.NewDefaultImplementationID[string]("harness-test.promoted-task"), func(b *coretask.Binder) func(ctx context.Context) (string, error) {
		promoted := coretask.UseOptional(b, promotedRef.Ref(taskid.ScopeActiveGraph))
		coretask.After(b, promotedRef)
		return func(ctx context.Context) (string, error) {
			value, found := promoted.Get(ctx)
			return fmt.Sprintf("promoted=%s,%t", value, found), nil
		}
	})

	testCases := []struct {
		name             string
		inputs           []InputValue
		want             string
		wantFatalSubstrs []string
	}{
		{
			name:   "given value is read",
			inputs: []InputValue{Given(promotedRef, "foo")},
			want:   "promoted=foo,true",
		},
		{
			name:             "omitted value fails as a missing required input",
			inputs:           nil,
			wantFatalSubstrs: []string{"missing required input ref:harness-test.promoted (string)"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, fatalMessage, err := runWithFatalRecorder(t, task, tc.inputs...)
			if len(tc.wantFatalSubstrs) > 0 {
				for _, want := range tc.wantFatalSubstrs {
					if !strings.Contains(fatalMessage, want) {
						t.Errorf("Run() failure message = %q, want substring %q", fatalMessage, want)
					}
				}
				return
			}
			if fatalMessage != "" {
				t.Fatalf("Run() failed the test unexpectedly: %s", fatalMessage)
			}
			if err != nil {
				t.Fatalf("Run() returned unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Run() = %q, want %q", got, tc.want)
			}
		})
	}
}
