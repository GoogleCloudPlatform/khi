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

package coreinit

import (
	"errors"
	"testing"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/server"
	"github.com/google/go-cmp/cmp"
)

// mockInitExtension is a mock implementation of InitExtension for testing.
type mockInitExtension struct {
	beforeAll                     func() error
	configureParameterStore       func() error
	afterParsingParameters        func() error
	configureInspectionTaskServer func(taskServer *coreinspection.InspectionTaskServer) error
	configureKHIWebServerFactory  func(serverFactory *server.ServerFactory) error
	beforeTerminate               func() error
}

func (m *mockInitExtension) BeforeAll() error {
	if m.beforeAll != nil {
		return m.beforeAll()
	}
	return nil
}

func (m *mockInitExtension) ConfigureParameterStore() error {
	if m.configureParameterStore != nil {
		return m.configureParameterStore()
	}
	return nil
}

func (m *mockInitExtension) AfterParsingParameters() error {
	if m.afterParsingParameters != nil {
		return m.afterParsingParameters()
	}
	return nil
}

func (m *mockInitExtension) ConfigureInspectionTaskServer(taskServer *coreinspection.InspectionTaskServer) error {
	if m.configureInspectionTaskServer != nil {
		return m.configureInspectionTaskServer(taskServer)
	}
	return nil
}

func (m *mockInitExtension) ConfigureKHIWebServerFactory(serverFactory *server.ServerFactory) error {
	if m.configureKHIWebServerFactory != nil {
		return m.configureKHIWebServerFactory(serverFactory)
	}
	return nil
}

func (m *mockInitExtension) BeforeTerminate() error {
	if m.beforeTerminate != nil {
		return m.beforeTerminate()
	}
	return nil
}

func TestCallInitExtensionInternal(t *testing.T) {
	t.Run("should call extensions in sorted order", func(t *testing.T) {
		var callOrder []int
		extensions := map[int]InitExtension{
			2: &mockInitExtension{},
			1: &mockInitExtension{},
			3: &mockInitExtension{},
		}

		caller := func(e InitExtension) error {
			// Find the key for the given extension
			for k, v := range extensions {
				if v == e {
					callOrder = append(callOrder, k)
					break
				}
			}
			return nil
		}

		err := callInitExtensionInternal(extensions, caller)

		if err != nil {
			t.Fatalf("expected no error, but got %v", err)
		}
		if diff := cmp.Diff([]int{1, 2, 3}, callOrder); diff != "" {
			t.Errorf("call order mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should return error if caller returns an error", func(t *testing.T) {
		extensions := map[int]InitExtension{
			1: &mockInitExtension{},
			2: &mockInitExtension{},
		}
		expectedErr := errors.New("caller error")

		var callCount int
		caller := func(e InitExtension) error {
			callCount++
			return expectedErr
		}

		err := callInitExtensionInternal(extensions, caller)

		if err != expectedErr {
			t.Errorf("expected error %v, but got %v", expectedErr, err)
		}
		if callCount != 1 {
			t.Errorf("expected call count to be 1, but got %d", callCount)
		}
	})

	t.Run("should handle empty extensions map", func(t *testing.T) {
		extensions := map[int]InitExtension{}
		callerCalled := false
		caller := func(e InitExtension) error {
			callerCalled = true
			return nil
		}

		err := callInitExtensionInternal(extensions, caller)

		if err != nil {
			t.Fatalf("expected no error, but got %v", err)
		}
		if callerCalled {
			t.Error("caller should not have been called")
		}
	})
}
