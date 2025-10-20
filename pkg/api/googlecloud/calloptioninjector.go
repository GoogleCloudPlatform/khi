// Copyright 2025 Google LLC
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

package googlecloud

import (
	"context"
	"net/http"
)

// CallOptionInjector injects call options for a given resource target.
type CallOptionInjector struct {
	options []CallOptionInjectorOption
}

// NewCallOptionInjector creates a new CallOptionInjector with the given options.
func NewCallOptionInjector(options ...CallOptionInjectorOption) *CallOptionInjector {
	return &CallOptionInjector{options: options}
}

// CallOptionInjectorOption defines an interface for options that can inject
// call options into a context or raw HTTP header.
type CallOptionInjectorOption interface {
	// ApplyToCallContext applies the call option to the given context.
	// This is typically used for clients from `cloud.google.com/go`.
	ApplyToCallContext(ctx context.Context, container ResourceContainer) context.Context
	// ApplyToRawHTTPHeader applies the call option to the given HTTP header.
	// This is typically used for clients from `google.golang.org/api`.
	ApplyToRawHTTPHeader(header http.Header, container ResourceContainer)
}

// headerProvider is an interface for types that can provide an HTTP header.
type headerProvider interface {
	Header() http.Header
}

// InjectToCallContext injects call options into given context.
// This is used for any clients under cloud.google.com/go
func (c *CallOptionInjector) InjectToCallContext(ctx context.Context, container ResourceContainer) context.Context {
	for _, option := range c.options {
		ctx = option.ApplyToCallContext(ctx, container)
	}
	return ctx
}

// InjectToCall injects call options into the given call request.
// This is used for any clients under google.golang.org/api
func (c *CallOptionInjector) InjectToCall(call headerProvider, container ResourceContainer) {
	for _, option := range c.options {
		option.ApplyToRawHTTPHeader(call.Header(), container)
	}
}
