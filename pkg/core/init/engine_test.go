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

package coreinit

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEngine_Lifecycle(t *testing.T) {
	t.Run("runs initializers, executes onRun hooks, and terminates in reverse order", func(t *testing.T) {
		ResetInitializersForTest()
		defer ResetInitializersForTest()

		var events []string

		RegisterInitializer(&Initializer{
			ID: "step-1",
			Init: func(ctx *InitContext) error {
				events = append(events, "init:1")
				ctx.OnRun(func(runCtx context.Context) error {
					events = append(events, "run:1")
					return nil
				})
				ctx.OnTerminate(func() error {
					events = append(events, "terminate:1")
					return nil
				})
				return nil
			},
		})

		RegisterInitializer(&Initializer{
			ID:           "step-2",
			Dependencies: []InitializerID{"step-1"},
			Init: func(ctx *InitContext) error {
				events = append(events, "init:2")
				ctx.OnRun(func(runCtx context.Context) error {
					events = append(events, "run:2")
					return nil
				})
				ctx.OnTerminate(func() error {
					events = append(events, "terminate:2")
					return nil
				})
				return nil
			},
		})

		engine := NewEngine(context.Background())

		if err := engine.Init(); err != nil {
			t.Fatalf("Init() failed: %v", err)
		}

		// Verify that resolved initializers are stored in context
		resolved, found := Get(engine.Context(), ResolvedInitializersKey)
		if !found {
			t.Errorf("expected ResolvedInitializersKey to be stored in context")
		} else if len(resolved) != 2 {
			t.Errorf("expected 2 resolved initializers, got %d", len(resolved))
		}

		if err := engine.Run(); err != nil {
			t.Fatalf("Run() failed: %v", err)
		}

		if err := engine.Terminate(); err != nil {
			t.Fatalf("Terminate() failed: %v", err)
		}

		// OnTerminate must run in reverse order (terminate:2 then terminate:1)
		wantEvents := []string{
			"init:1",
			"init:2",
			"run:1",
			"run:2",
			"terminate:2",
			"terminate:1",
		}

		if diff := cmp.Diff(wantEvents, events); diff != "" {
			t.Errorf("lifecycle event order mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("returns error when an initializer fails", func(t *testing.T) {
		ResetInitializersForTest()
		defer ResetInitializersForTest()

		expectedErr := errors.New("boom")
		RegisterInitializer(&Initializer{
			ID: "failing-step",
			Init: func(ctx *InitContext) error {
				return expectedErr
			},
		})

		engine := NewEngine(context.Background())
		err := engine.Init()
		if err == nil {
			t.Fatalf("expected Init() to return an error")
		}
	})

	t.Run("returns error when an onRun hook fails", func(t *testing.T) {
		ResetInitializersForTest()
		defer ResetInitializersForTest()

		expectedErr := errors.New("run failure")
		RegisterInitializer(&Initializer{
			ID: "failing-run-step",
			Init: func(ctx *InitContext) error {
				ctx.OnRun(func(runCtx context.Context) error {
					return expectedErr
				})
				return nil
			},
		})

		engine := NewEngine(context.Background())
		if err := engine.Init(); err != nil {
			t.Fatalf("Init() failed: %v", err)
		}

		err := engine.Run()
		if !errors.Is(err, expectedErr) {
			t.Errorf("Run() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("aggregates errors during termination", func(t *testing.T) {
		ResetInitializersForTest()
		defer ResetInitializersForTest()

		err1 := errors.New("term-1 failed")
		err2 := errors.New("term-2 failed")

		RegisterInitializer(&Initializer{
			ID: "term-errors",
			Init: func(ctx *InitContext) error {
				ctx.OnTerminate(func() error {
					return err1
				})
				ctx.OnTerminate(func() error {
					return err2
				})
				return nil
			},
		})

		engine := NewEngine(context.Background())
		if err := engine.Init(); err != nil {
			t.Fatalf("Init() failed: %v", err)
		}

		err := engine.Terminate()
		if !errors.Is(err, err1) || !errors.Is(err, err2) {
			t.Errorf("Terminate() expected aggregated error containing err1 and err2, got %v", err)
		}
	})

	t.Run("is idempotent when calling Terminate multiple times", func(t *testing.T) {
		ResetInitializersForTest()
		defer ResetInitializersForTest()

		callCount := 0
		termErr := errors.New("term error")
		RegisterInitializer(&Initializer{
			ID: "idempotent-term",
			Init: func(ctx *InitContext) error {
				ctx.OnTerminate(func() error {
					callCount++
					return termErr
				})
				return nil
			},
		})

		engine := NewEngine(context.Background())
		if err := engine.Init(); err != nil {
			t.Fatalf("Init() failed: %v", err)
		}

		errFirst := engine.Terminate()
		if !errors.Is(errFirst, termErr) {
			t.Errorf("first Terminate() expected %v, got %v", termErr, errFirst)
		}

		errSecond := engine.Terminate()
		if !errors.Is(errSecond, termErr) {
			t.Errorf("second Terminate() expected %v, got %v", termErr, errSecond)
		}

		if callCount != 1 {
			t.Errorf("expected termination hook to be called exactly once, got %d", callCount)
		}
	})
}
