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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/google/go-cmp/cmp"
)

func TestInitContext_GetSet(t *testing.T) {
	testKeyString := typedmap.NewTypedKey[string]("test-string-key")
	testKeyInt := typedmap.NewTypedKey[int]("test-int-key")

	testCases := []struct {
		name         string
		setup        func(ctx *InitContext)
		key          typedmap.TypedKey[string]
		wantVal      string
		wantFound    bool
		wantDefault  string
		wantPanicGet bool
	}{
		{
			name: "existing key retrieves value",
			setup: func(ctx *InitContext) {
				Set(ctx, testKeyString, "hello world")
			},
			key:          testKeyString,
			wantVal:      "hello world",
			wantFound:    true,
			wantDefault:  "hello world",
			wantPanicGet: false,
		},
		{
			name:         "missing key returns not found and default",
			setup:        func(ctx *InitContext) {},
			key:          testKeyString,
			wantVal:      "",
			wantFound:    false,
			wantDefault:  "fallback",
			wantPanicGet: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewEngine(context.Background())
			ctx := engine.Context()
			tc.setup(ctx)

			gotVal, gotFound := Get(ctx, tc.key)
			if gotFound != tc.wantFound {
				t.Errorf("Get() found = %v, want %v", gotFound, tc.wantFound)
			}
			if diff := cmp.Diff(tc.wantVal, gotVal); diff != "" {
				t.Errorf("Get() mismatch (-want +got):\n%s", diff)
			}

			gotDefault := GetOrDefault(ctx, tc.key, "fallback")
			if diff := cmp.Diff(tc.wantDefault, gotDefault); diff != "" {
				t.Errorf("GetOrDefault() mismatch (-want +got):\n%s", diff)
			}

			if tc.wantPanicGet {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected MustGet to panic on missing key")
					}
				}()
				MustGet(ctx, tc.key)
			} else {
				gotMust := MustGet(ctx, tc.key)
				if diff := cmp.Diff(tc.wantVal, gotMust); diff != "" {
					t.Errorf("MustGet() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}

	t.Run("typed key with int", func(t *testing.T) {
		engine := NewEngine(context.Background())
		ctx := engine.Context()
		Set(ctx, testKeyInt, 42)

		val, found := Get(ctx, testKeyInt)
		if !found {
			t.Errorf("expected key to be found")
		}
		if val != 42 {
			t.Errorf("Get() = %d, want 42", val)
		}
	})
}
