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

package structured

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type testStructA struct {
	Foo string
	Bar int
}

type testStructB struct {
	Baz bool
}

func TestMockNode_GetMock(t *testing.T) {
	testTime := time.Date(2026, 8, 30, 15, 0, 0, 0, time.UTC)
	dataA := testStructA{Foo: "hello", Bar: 42}
	dataB := &testStructB{Baz: true}

	mockNode := NewMockNode(dataA, dataB, testTime)
	reader := NewNodeReader(mockNode)

	t.Run("retrieve value type when registered as value", func(t *testing.T) {
		got, ok := GetMock[testStructA](reader)
		if !ok {
			t.Fatalf("GetMock[testStructA]() returned ok=false")
		}
		if diff := cmp.Diff(dataA, got); diff != "" {
			t.Errorf("GetMock[testStructA]() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("retrieve pointer type when registered as value", func(t *testing.T) {
		got, ok := GetMock[*testStructA](reader)
		if !ok {
			t.Fatalf("GetMock[*testStructA]() returned ok=false")
		}
		if diff := cmp.Diff(&dataA, got); diff != "" {
			t.Errorf("GetMock[*testStructA]() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("retrieve pointer type when registered as pointer", func(t *testing.T) {
		got, ok := GetMock[*testStructB](reader)
		if !ok {
			t.Fatalf("GetMock[*testStructB]() returned ok=false")
		}
		if diff := cmp.Diff(dataB, got); diff != "" {
			t.Errorf("GetMock[*testStructB]() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("retrieve value type when registered as pointer", func(t *testing.T) {
		got, ok := GetMock[testStructB](reader)
		if !ok {
			t.Fatalf("GetMock[testStructB]() returned ok=false")
		}
		if diff := cmp.Diff(*dataB, got); diff != "" {
			t.Errorf("GetMock[testStructB]() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("retrieve primitive time.Time", func(t *testing.T) {
		got, ok := GetMock[time.Time](reader)
		if !ok {
			t.Fatalf("GetMock[time.Time]() returned ok=false")
		}
		if diff := cmp.Diff(testTime, got); diff != "" {
			t.Errorf("GetMock[time.Time]() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("returns false for unregistered type", func(t *testing.T) {
		_, ok := GetMock[string](reader)
		if ok {
			t.Errorf("GetMock[string]() expected ok=false, got true")
		}
	})

	t.Run("returns false for non-mock reader", func(t *testing.T) {
		nonMockReader := NewNodeReader(NewEmptyMapNode())
		_, ok := GetMock[testStructA](nonMockReader)
		if ok {
			t.Errorf("GetMock() on nonMockReader expected ok=false, got true")
		}
	})

	t.Run("returns false for nil reader", func(t *testing.T) {
		_, ok := GetMock[testStructA](nil)
		if ok {
			t.Errorf("GetMock() on nil reader expected ok=false, got true")
		}
	})
}
