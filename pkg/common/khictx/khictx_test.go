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

package khictx_test

import (
	"context"
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

func TestGetValue(t *testing.T) {
	// Define test keys
	stringKey := typedmap.NewTypedKey[string]("string-key")
	intKey := typedmap.NewTypedKey[int]("int-key")

	// Test value exists and is of correct type
	t.Run("value exists and is of correct type", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, stringKey, "test-value")

		value, err := khictx.GetValue(ctx, stringKey)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if value != "test-value" {
			t.Errorf("Expected value 'test-value', got '%s'", value)
		}
	})

	// Test value does not exist
	t.Run("value does not exist", func(t *testing.T) {
		ctx := context.Background()

		_, err := khictx.GetValue(ctx, stringKey)
		if err == nil {
			t.Error("Expected error for non-existent value, got nil")
		}
	})

	// Test value exists but is of wrong type
	t.Run("value exists but is of wrong type", func(t *testing.T) {
		ctx := context.Background()
		// Store a string value
		ctx = context.WithValue(ctx, intKey, "not-an-int")

		// Try to get it as an int
		_, err := khictx.GetValue(ctx, intKey)
		if err == nil {
			t.Error("Expected type conversion error, got nil")
		}
	})
}

func TestWithValue(t *testing.T) {
	// Define test keys
	stringKey := typedmap.NewTypedKey[string]("string-key")
	pointerKey := typedmap.NewTypedKey[*string]("pointer-key")

	// Test normal value storage and retrieval
	t.Run("normal value storage", func(t *testing.T) {
		ctx := context.Background()
		ctx = khictx.WithValue(ctx, stringKey, "test-value")

		value, err := khictx.GetValue(ctx, stringKey)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if value != "test-value" {
			t.Errorf("Expected value 'test-value', got '%s'", value)
		}
	})

	// Test nil pointer value (functionality works, but logs a warning)
	// We're just testing the value is still stored correctly
	t.Run("nil pointer value", func(t *testing.T) {
		ctx := context.Background()
		var nilPointer *string = nil

		ctx = khictx.WithValue(ctx, pointerKey, nilPointer)

		value, err := khictx.GetValue(ctx, pointerKey)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if value != nil {
			t.Errorf("Expected nil value, got %v", value)
		}
	})
}

func TestMustGetValue(t *testing.T) {
	// Define test key
	stringKey := typedmap.NewTypedKey[string]("string-key")

	// Test successful retrieval
	t.Run("successful retrieval", func(t *testing.T) {
		ctx := context.Background()
		ctx = khictx.WithValue(ctx, stringKey, "test-value")

		value := khictx.MustGetValue(ctx, stringKey)
		if value != "test-value" {
			t.Errorf("Expected value 'test-value', got '%s'", value)
		}
	})

	// Test panic on missing value
	t.Run("panic on missing value", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected MustGetValue to panic on missing value, but it didn't")
			}
		}()

		ctx := context.Background()
		_ = khictx.MustGetValue(ctx, stringKey) // This should panic
	})
}
