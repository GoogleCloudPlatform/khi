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
	"fmt"
	"sync"
)

// Engine manages the initialization, execution, and termination lifecycle of KHI.
type Engine struct {
	ctx              *InitContext
	cancel           context.CancelFunc
	mu               sync.Mutex
	runHooks         []func(ctx context.Context) error
	terminationHooks []func() error
	terminateOnce    sync.Once
	terminateErr     error
}

// NewEngine creates a new Engine wrapping the provided parent context.
func NewEngine(parent context.Context) *Engine {
	ctx, cancel := context.WithCancel(parent)
	engine := &Engine{
		cancel: cancel,
	}
	engine.ctx = newInitContext(ctx, engine)
	return engine
}

// Context returns the InitContext managed by this engine.
func (e *Engine) Context() *InitContext {
	return e.ctx
}

func (e *Engine) registerRunHook(hook func(ctx context.Context) error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.runHooks = append(e.runHooks, hook)
}

func (e *Engine) registerTerminateHook(hook func() error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.terminationHooks = append(e.terminationHooks, hook)
}

// Init resolves and executes all registered Initializers in topological order.
func (e *Engine) Init() error {
	registryMu.RLock()
	inits := append([]*Initializer(nil), initializers...)
	registryMu.RUnlock()

	sorted, err := sortInitializers(inits)
	if err != nil {
		return fmt.Errorf("failed resolving initializer dependencies: %w", err)
	}

	Set(e.ctx, ResolvedInitializersKey, sorted)

	for _, init := range sorted {
		if err := init.Init(e.ctx); err != nil {
			return fmt.Errorf("initializer %s failed: %w", init.ID, err)
		}
	}
	return nil
}

// Run executes all registered OnRun hooks.
func (e *Engine) Run() error {
	e.mu.Lock()
	hooks := append([]func(context.Context) error(nil), e.runHooks...)
	e.mu.Unlock()

	for _, hook := range hooks {
		if err := hook(e.ctx.Context); err != nil {
			return err
		}
	}
	return nil
}

// Terminate cancels the context and executes all registered termination hooks in reverse order.
func (e *Engine) Terminate() error {
	e.cancel()
	e.terminateOnce.Do(func() {
		e.mu.Lock()
		hooks := e.terminationHooks
		e.terminationHooks = nil
		e.mu.Unlock()

		var errs []error
		for i := len(hooks) - 1; i >= 0; i-- {
			if err := hooks[i](); err != nil {
				errs = append(errs, err)
			}
		}
		e.terminateErr = errors.Join(errs...)
	})
	return e.terminateErr
}
