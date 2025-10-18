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
	"errors"
	"testing"

	"google.golang.org/api/option"
)

func TestNewClientFactory(t *testing.T) {
	nopContextModifier := func(ctx context.Context, container ResourceContainer) (context.Context, error) {
		return ctx, nil
	}
	nopOptionsModifier := func(opts []option.ClientOption, container ResourceContainer) ([]option.ClientOption, error) {
		return opts, nil
	}

	testCases := []struct {
		name    string
		options []ClientFactoryOption
		wantErr bool
		want    *ClientFactory
	}{
		{
			name:    "No options",
			options: nil,
			want:    &ClientFactory{},
		},
		{
			name: "With options",
			options: []ClientFactoryOption{
				func(s *ClientFactory) error {
					s.ContextModifiers = append(s.ContextModifiers, nopContextModifier)
					return nil
				},
				func(s *ClientFactory) error {
					s.ClientOptions = append(s.ClientOptions, nopOptionsModifier)
					return nil
				},
			},
			want: &ClientFactory{
				ContextModifiers: []ClientFactoryContextModifiers{nopContextModifier},
				ClientOptions:    []ClientFactoryOptionsModifiers{nopOptionsModifier},
			},
		},
		{
			name: "Option returns error",
			options: []ClientFactoryOption{
				func(s *ClientFactory) error {
					return errors.New("option error")
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory, err := NewClientFactory(tc.options...)
			if (err != nil) != tc.wantErr {
				t.Errorf("NewClientFactory() error = %v, expectError %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && (len(factory.ContextModifiers) != len(tc.want.ContextModifiers) || len(factory.ClientOptions) != len(tc.want.ClientOptions)) {
				t.Errorf("NewClientFactory() = %v, want %v", factory, tc.want)
			}
		})
	}
}

func TestClientFactory_context(t *testing.T) {
	modifier1 := func(ctx context.Context, c ResourceContainer) (context.Context, error) {
		//lint:ignore SA1029 this is only used for test
		return context.WithValue(ctx, "key1", "value1"), nil
	}
	modifier2 := func(ctx context.Context, c ResourceContainer) (context.Context, error) {
		//lint:ignore SA1029 this is only used for test
		return context.WithValue(ctx, "key2", "value2"), nil
	}
	errorModifier := func(ctx context.Context, c ResourceContainer) (context.Context, error) {
		return nil, errors.New("context modifier error")
	}

	testCases := []struct {
		name        string
		factory     *ClientFactory
		expectError bool
		expectedCtx map[interface{}]interface{}
	}{
		{
			name:    "No modifiers",
			factory: &ClientFactory{},
		},
		{
			name: "Multiple modifiers",
			factory: &ClientFactory{
				ContextModifiers: []ClientFactoryContextModifiers{modifier1, modifier2},
			},
			expectedCtx: map[interface{}]interface{}{"key1": "value1", "key2": "value2"},
		},
		{
			name: "Modifier returns error",
			factory: &ClientFactory{
				ContextModifiers: []ClientFactoryContextModifiers{errorModifier},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, err := tc.factory.context(context.Background(), Project("test-project"))
			if (err != nil) != tc.expectError {
				t.Errorf("context() error = %v, expectError %v", err, tc.expectError)
				return
			}
			if !tc.expectError {
				for k, v := range tc.expectedCtx {
					if ctx.Value(k) != v {
						t.Errorf("context() did not set value for key %v. Got %v, want %v", k, ctx.Value(k), v)
					}
				}
			}
		})
	}
}

func TestClientFactory_options(t *testing.T) {
	modifier1 := func(opts []option.ClientOption, c ResourceContainer) ([]option.ClientOption, error) {
		return append(opts, option.WithAPIKey("key1")), nil
	}
	modifier2 := func(opts []option.ClientOption, c ResourceContainer) ([]option.ClientOption, error) {
		return append(opts, option.WithAPIKey("key2")), nil
	}
	errorModifier := func(opts []option.ClientOption, c ResourceContainer) ([]option.ClientOption, error) {
		return nil, errors.New("options modifier error")
	}

	testCases := []struct {
		name          string
		factory       *ClientFactory
		expectError   bool
		expectedCount int
	}{
		{
			name:          "No modifiers",
			factory:       &ClientFactory{},
			expectedCount: 0,
		},
		{
			name: "Multiple modifiers",
			factory: &ClientFactory{
				ClientOptions: []ClientFactoryOptionsModifiers{modifier1, modifier2},
			},
			expectedCount: 2,
		},
		{
			name: "Modifier returns error",
			factory: &ClientFactory{
				ClientOptions: []ClientFactoryOptionsModifiers{errorModifier},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := tc.factory.options(Project("test-project"), []ClientFactoryOptionsModifiers{})
			if (err != nil) != tc.expectError {
				t.Errorf("options() error = %v, expectError %v", err, tc.expectError)
				return
			}
			if !tc.expectError && len(opts) != tc.expectedCount {
				t.Errorf("options() returned %d options, want %d", len(opts), tc.expectedCount)
			}
		})
	}
}
