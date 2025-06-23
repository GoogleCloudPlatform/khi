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

package typedmap

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFilter(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	predicate := func(i int) bool {
		return i%2 == 0
	}
	expected := []int{2, 4, 6}

	result := Filter(items, predicate)

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("Filter() mismatch (-want +got):\n%s", diff)
	}
}

func TestWhereFieldEquals(t *testing.T) {
	type testStruct struct {
		labels *ReadonlyTypedMap
	}
	key := NewTypedKey[string]("color")
	items := []testStruct{
		{labels: NewTypedMap().AsReadonly()},
		{labels: func() *ReadonlyTypedMap {
			m := NewTypedMap()
			Set(m, key, "red")
			return m.AsReadonly()
		}()},
		{labels: func() *ReadonlyTypedMap {
			m := NewTypedMap()
			Set(m, key, "blue")
			return m.AsReadonly()
		}()},
	}

	getMap := func(s testStruct) *ReadonlyTypedMap { return s.labels }
	predicate := WhereFieldEquals(getMap, key, "red")
	result := Filter(items, predicate)

	if len(result) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(result))
	}
	if val, _ := Get(result[0].labels, key); val != "red" {
		t.Errorf("Expected item with color red, but got %v", val)
	}
}

func TestWhereFieldContainsElement(t *testing.T) {
	type testStruct struct {
		labels *ReadonlyTypedMap
	}
	key := NewTypedKey[[]string]("tags")
	items := []testStruct{
		{labels: NewTypedMap().AsReadonly()},
		{labels: func() *ReadonlyTypedMap {
			m := NewTypedMap()
			Set(m, key, []string{"a", "b"})
			return m.AsReadonly()
		}()},
		{labels: func() *ReadonlyTypedMap {
			m := NewTypedMap()
			Set(m, key, []string{"b", "c"})
			return m.AsReadonly()
		}()},
	}

	getMap := func(s testStruct) *ReadonlyTypedMap { return s.labels }
	predicate := WhereFieldContainsElement(getMap, key, "a")
	result := Filter(items, predicate)

	if len(result) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(result))
	}
	if val, _ := Get(result[0].labels, key); !cmp.Equal(val, []string{"a", "b"}) {
		t.Errorf("Expected item with tags ['a', 'b'], but got %v", val)
	}
}

func TestWhereFieldIsEnabled(t *testing.T) {
	type testStruct struct {
		labels *ReadonlyTypedMap
	}
	key := NewTypedKey[bool]("enabled")
	items := []testStruct{
		{labels: NewTypedMap().AsReadonly()},
		{labels: func() *ReadonlyTypedMap {
			m := NewTypedMap()
			Set(m, key, true)
			return m.AsReadonly()
		}()},
		{labels: func() *ReadonlyTypedMap {
			m := NewTypedMap()
			Set(m, key, false)
			return m.AsReadonly()
		}()},
	}

	getMap := func(s testStruct) *ReadonlyTypedMap { return s.labels }
	predicate := WhereFieldIsEnabled(getMap, key)
	result := Filter(items, predicate)

	if len(result) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(result))
	}
	if val, _ := Get(result[0].labels, key); !val {
		t.Errorf("Expected enabled item, but it was disabled")
	}
}
