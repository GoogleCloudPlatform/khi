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

package server

import (
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/server/option"
	"github.com/gin-gonic/gin"
)

// mockOption is a helper for testing.
type mockOption struct {
	id    string
	order int
	apply func(e *gin.Engine) error
}

func (m *mockOption) ID() string {
	return m.id
}

func (m *mockOption) Order() int {
	return m.order
}

func (m *mockOption) Apply(engine *gin.Engine) error {
	if m.apply != nil {
		return m.apply(engine)
	}
	return nil
}

func TestServerFactory_AddOptions(t *testing.T) {
	factory := &ServerFactory{}
	opt1 := &mockOption{id: "opt1"}
	opt2 := &mockOption{id: "opt2"}

	factory.AddOptions(opt1)
	if len(factory.Options) != 1 || factory.Options[0].ID() != "opt1" {
		t.Errorf("AddOptions failed to add a single option. Got: %v", factory.Options)
	}

	factory.AddOptions(opt2)
	if len(factory.Options) != 2 || factory.Options[1].ID() != "opt2" {
		t.Errorf("AddOptions failed to add a second option. Got: %v", factory.Options)
	}
}

func TestServerFactory_CreateInstance(t *testing.T) {
	// Preserve original mode and restore after test
	originalMode := gin.Mode()
	defer gin.SetMode(originalMode)

	testCases := []struct {
		name        string
		factory     *ServerFactory
		mode        string
		expectErr   bool
		expectOrder []string
	}{
		{
			name: "successful creation with ordered options",
			factory: &ServerFactory{
				Options: []option.Option{
					&mockOption{id: "opt2", order: 2},
					&mockOption{id: "opt1", order: 1},
				},
			},
			mode:      gin.TestMode,
			expectErr: false,
		},
		{
			name: "creation fails when an option fails",
			factory: &ServerFactory{
				Options: []option.Option{
					&mockOption{id: "good-opt", order: 1},
					&mockOption{id: "bad-opt", order: 2, apply: func(e *gin.Engine) error {
						return errors.New("apply failed")
					}},
				},
			},
			mode:      gin.TestMode,
			expectErr: true,
		},
		{
			name:        "creation with no options",
			factory:     &ServerFactory{},
			mode:        gin.DebugMode,
			expectErr:   false,
			expectOrder: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine, err := tc.factory.CreateInstance(tc.mode)

			if (err != nil) != tc.expectErr {
				t.Fatalf("CreateInstance() error = %v, wantErr %v", err, tc.expectErr)
			}

			if tc.expectErr {
				if engine != nil {
					t.Error("CreateInstance() expected nil engine on error, but got one")
				}
			} else {
				if engine == nil {
					t.Fatal("CreateInstance() returned nil engine, but expected one")
				}
				if gin.Mode() != tc.mode {
					t.Errorf("gin mode not set correctly. got=%q, want=%q", gin.Mode(), tc.mode)
				}
			}
		})
	}
}
