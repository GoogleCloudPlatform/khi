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

package khictx

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// GetValue retrieves a typed value from the context.
// Returns error if the value doesn't exist or can't be converted to type T.
func GetValue[T any](ctx context.Context, key typedmap.TypedKey[T]) (T, error) {
	var zero T
	valueAny := ctx.Value(key)
	if valueAny == nil {
		return zero, fmt.Errorf("value not found for key: %s", key.Key())
	}

	value, convertible := valueAny.(T)
	if !convertible {
		return zero, fmt.Errorf("value of type %T cannot be converted to type %T", valueAny, *new(T))
	}
	return value, nil
}

// WithValue returns a new context with the provided value stored under the given key.
// Warns when nil pointer values are stored as this may be confused with "not found" cases.
func WithValue[T any](ctx context.Context, key typedmap.TypedKey[T], value T) context.Context {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		slog.ErrorContext(ctx, fmt.Sprintf("WithValue got a nil pointer value for key %s of type %T. Assigning a nil value on context is discouraged.", key.Key(), *new(T)))
	}
	return context.WithValue(ctx, key, value)
}

// MustGetValue retrieves a typed value from the context.
// Panics if the value doesn't exist or can't be converted to type T.
func MustGetValue[T any](ctx context.Context, key typedmap.TypedKey[T]) T {
	value, err := GetValue(ctx, key)
	if err != nil {
		panic(fmt.Sprintf("MustGetValue failed: %v", err))
	}
	return value
}
