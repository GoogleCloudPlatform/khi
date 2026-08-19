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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

var (
	// ResolvedInitializersKey stores the topologically resolved Initializers.
	ResolvedInitializersKey = typedmap.NewTypedKey[[]*Initializer]("khi.google.com/init/resolved-initializers")
)

// InitContext is passed to Initializers to share state and register lifecycle hooks.
type InitContext struct {
	context.Context
	data   *typedmap.TypedMap
	engine *Engine
}

// newInitContext creates a new InitContext.
func newInitContext(ctx context.Context, engine *Engine) *InitContext {
	return &InitContext{
		Context: ctx,
		data:    typedmap.NewTypedMap(),
		engine:  engine,
	}
}

// Get retrieves a typed value from InitContext.
func Get[T any](ctx *InitContext, key typedmap.TypedKey[T]) (T, bool) {
	return typedmap.Get(ctx.data, key)
}

// MustGet retrieves a typed value or panics if not found.
func MustGet[T any](ctx *InitContext, key typedmap.TypedKey[T]) T {
	val, ok := Get(ctx, key)
	if !ok {
		panic(fmt.Sprintf("required init context key %q not found", key.Key()))
	}
	return val
}

// Set stores a typed value in InitContext.
func Set[T any](ctx *InitContext, key typedmap.TypedKey[T], val T) {
	typedmap.Set(ctx.data, key, val)
}

// GetOrDefault retrieves a typed value or returns defaultValue if not found.
func GetOrDefault[T any](ctx *InitContext, key typedmap.TypedKey[T], defaultValue T) T {
	val, ok := Get(ctx, key)
	if !ok {
		return defaultValue
	}
	return val
}

// OnRun registers a runtime hook with the Engine.
func (c *InitContext) OnRun(hook func(ctx context.Context) error) {
	c.engine.registerRunHook(hook)
}

// OnTerminate registers a shutdown hook with the Engine.
func (c *InitContext) OnTerminate(hook func() error) {
	c.engine.registerTerminateHook(hook)
}
