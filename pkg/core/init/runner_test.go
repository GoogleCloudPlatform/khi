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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSortInitializers(t *testing.T) {
	testCases := []struct {
		name        string
		inits       []*Initializer
		wantOrder   []InitializerID
		wantErr     bool
		errContains string
	}{
		{
			name: "linear dependency order",
			inits: []*Initializer{
				{ID: "c", Dependencies: []InitializerID{"b"}},
				{ID: "a"},
				{ID: "b", Dependencies: []InitializerID{"a"}},
			},
			wantOrder: []InitializerID{"a", "b", "c"},
		},
		{
			name: "before dependency order",
			inits: []*Initializer{
				{ID: "parse"},
				{ID: "custom-store", Before: []InitializerID{"parse"}},
			},
			wantOrder: []InitializerID{"custom-store", "parse"},
		},
		{
			name: "combined dependencies and before",
			inits: []*Initializer{
				{ID: "c", Dependencies: []InitializerID{"b"}},
				{ID: "a", Before: []InitializerID{"b"}},
				{ID: "b"},
			},
			wantOrder: []InitializerID{"a", "b", "c"},
		},
		{
			name: "detects circular dependency",
			inits: []*Initializer{
				{ID: "a", Dependencies: []InitializerID{"b"}},
				{ID: "b", Dependencies: []InitializerID{"a"}},
			},
			wantErr:     true,
			errContains: "circular dependency",
		},
		{
			name: "detects missing dependency",
			inits: []*Initializer{
				{ID: "a", Dependencies: []InitializerID{"missing"}},
			},
			wantErr:     true,
			errContains: "missing dependency",
		},
		{
			name: "detects missing before target",
			inits: []*Initializer{
				{ID: "a", Before: []InitializerID{"missing-target"}},
			},
			wantErr:     true,
			errContains: "missing before target",
		},
		{
			name: "detects duplicate initializer ID",
			inits: []*Initializer{
				{ID: "a"},
				{ID: "a"},
			},
			wantErr:     true,
			errContains: "duplicate initializer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sortInitializers(tc.inits)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var gotOrder []InitializerID
			for _, item := range got {
				gotOrder = append(gotOrder, item.ID)
			}
			if diff := cmp.Diff(tc.wantOrder, gotOrder); diff != "" {
				t.Errorf("order mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRegisterInitializer_PanicsOnBlankID(t *testing.T) {
	testCases := []struct {
		name        string
		initializer *Initializer
	}{
		{
			name:        "nil initializer",
			initializer: nil,
		},
		{
			name:        "empty ID",
			initializer: &Initializer{ID: ""},
		},
		{
			name:        "whitespace only ID",
			initializer: &Initializer{ID: "   "},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("expected panic for %s, but got none", tc.name)
				}
			}()
			RegisterInitializer(tc.initializer)
		})
	}
}

func TestResetInitializersForTest(t *testing.T) {
	ResetInitializersForTest()
	defer ResetInitializersForTest()

	RegisterInitializer(&Initializer{ID: "temp"})
	if len(initializers) != 1 {
		t.Errorf("expected 1 initializer, got %d", len(initializers))
	}

	ResetInitializersForTest()
	if len(initializers) != 0 {
		t.Errorf("expected 0 initializers after reset, got %d", len(initializers))
	}
}

func TestEngineInitExecution(t *testing.T) {
	ResetInitializersForTest()
	defer ResetInitializersForTest()

	var executionOrder []string
	RegisterInitializer(&Initializer{
		ID:           "second",
		Dependencies: []InitializerID{"first"},
		Init: func(ctx *InitContext) error {
			executionOrder = append(executionOrder, "second")
			return nil
		},
	})
	RegisterInitializer(&Initializer{
		ID: "first",
		Init: func(ctx *InitContext) error {
			executionOrder = append(executionOrder, "first")
			return nil
		},
	})

	engine := NewEngine(context.Background())
	if err := engine.Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	wantOrder := []string{"first", "second"}
	if diff := cmp.Diff(wantOrder, executionOrder); diff != "" {
		t.Errorf("execution order mismatch (-want +got):\n%s", diff)
	}
}
