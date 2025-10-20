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
	"testing"
)

// mockCallOptionInjectorOption is a mock implementation of CallOptionInjectorOption for testing.
type mockCallOptionInjectorOption struct {
	applyToCallContextCalled   bool
	applyToRawHTTPHeaderCalled bool
}

func (m *mockCallOptionInjectorOption) ApplyToCallContext(ctx context.Context, container ResourceContainer) context.Context {
	m.applyToCallContextCalled = true
	//lint:ignore SA1029 This is only used for testing.
	return context.WithValue(ctx, "test_key", "test_value")
}

func (m *mockCallOptionInjectorOption) ApplyToRawHTTPHeader(header http.Header, container ResourceContainer) {
	m.applyToRawHTTPHeaderCalled = true
	header.Set("Test-Header", "Test-Value")
}

type mockHeaderProviderClient struct {
	header http.Header
}

// Header implements headerProvider.
func (m *mockHeaderProviderClient) Header() http.Header {
	return m.header
}

var _ headerProvider = (*mockHeaderProviderClient)(nil)

func TestNewCallOptionInjector(t *testing.T) {
	option1 := &mockCallOptionInjectorOption{}
	option2 := &mockCallOptionInjectorOption{}
	injector := NewCallOptionInjector(option1, option2)

	if len(injector.options) != 2 {
		t.Errorf("Expected 2 options, but got %d", len(injector.options))
	}
}

func TestCallOptionInjector_InjectToCallContext(t *testing.T) {
	option1 := &mockCallOptionInjectorOption{}
	option2 := &mockCallOptionInjectorOption{}
	injector := NewCallOptionInjector(option1, option2)

	ctx := context.Background()
	newCtx := injector.InjectToCallContext(ctx, Project("foobar"))

	if !option1.applyToCallContextCalled {
		t.Error("Expected ApplyToCallContext to be called on option1, but it was not")
	}
	if !option2.applyToCallContextCalled {
		t.Error("Expected ApplyToCallContext to be called on option2, but it was not")
	}
	if newCtx.Value("test_key") != "test_value" {
		t.Error("Expected context to have value 'test_value' for key 'test_key'")
	}
}

func TestCallOptionInjector_InjectToCall(t *testing.T) {
	option1 := &mockCallOptionInjectorOption{}
	option2 := &mockCallOptionInjectorOption{}
	injector := NewCallOptionInjector(option1, option2)

	hp := &mockHeaderProviderClient{
		header: make(http.Header),
	}
	injector.InjectToCall(hp, Project("foobar"))

	if !option1.applyToRawHTTPHeaderCalled {
		t.Error("Expected ApplyToRawHTTPHeader to be called on option1, but it was not")
	}
	if !option2.applyToRawHTTPHeaderCalled {
		t.Error("Expected ApplyToRawHTTPHeader to be called on option2, but it was not")
	}
	if hp.Header().Get("Test-Header") != "Test-Value" {
		t.Error("Expected header to have 'Test-Value' for 'Test-Header'")
	}
}
