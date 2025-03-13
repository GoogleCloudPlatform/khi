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

var (
	StringKey    = NewTypedKey[string]("string-key")
	StructKey    = NewTypedKey[Person]("struct-key")
	StructPtrKey = NewTypedKey[*Person]("struct-ptr-key")
)

type Person struct {
	Name string
	Age  int
}

type TestCase[T any] struct {
	name     string
	key      TypedKey[T]
	value    T
	wantOk   bool
	wantDiff string
}

func TestTypedMap(t *testing.T) {
	tm := NewTypedMap()

	stringValue := "Taro"
	person := Person{Name: "Hanako", Age: 25}

	Set(tm, StringKey, stringValue)
	Set(tm, StructKey, person)
	Set(tm, StructPtrKey, &person)

	t.Run("string values", func(t *testing.T) {
		cases := []TestCase[string]{
			{
				name:     "existing string key",
				key:      StringKey,
				value:    stringValue,
				wantOk:   true,
				wantDiff: "",
			},
			{
				name:     "non-existent string key",
				key:      NewTypedKey[string]("non-existent"),
				value:    "",
				wantOk:   false,
				wantDiff: "",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, ok := Get(tm, tc.key)
				if ok != tc.wantOk {
					t.Errorf("Get() ok = %v, wantOk %v", ok, tc.wantOk)
				}

				if ok {
					if diff := cmp.Diff(tc.value, got); diff != tc.wantDiff {
						t.Errorf("Get() mismatch (-want +got):\n%s", diff)
					}
				}
			})
		}
	})

	t.Run("struct values", func(t *testing.T) {
		cases := []TestCase[Person]{
			{
				name:     "existing struct key",
				key:      StructKey,
				value:    person,
				wantOk:   true,
				wantDiff: "",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, ok := Get(tm, tc.key)
				if ok != tc.wantOk {
					t.Errorf("Get() ok = %v, wantOk %v", ok, tc.wantOk)
				}

				if ok {
					if diff := cmp.Diff(tc.value, got); diff != tc.wantDiff {
						t.Errorf("Get() mismatch (-want +got):\n%s", diff)
					}
				}
			})
		}
	})

	t.Run("struct pointer values", func(t *testing.T) {
		cases := []TestCase[*Person]{
			{
				name:     "existing struct pointer key",
				key:      StructPtrKey,
				value:    &person,
				wantOk:   true,
				wantDiff: "",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, ok := Get(tm, tc.key)
				if ok != tc.wantOk {
					t.Errorf("Get() ok = %v, wantOk %v", ok, tc.wantOk)
				}

				if ok {
					if diff := cmp.Diff(*tc.value, *got); diff != tc.wantDiff {
						t.Errorf("Get() mismatch (-want +got):\n%s", diff)
					}
				}
			})
		}
	})
}

func TestReadonlyTypedMap(t *testing.T) {
	tm := NewTypedMap()
	Set(tm, StringKey, "ReadOnly")

	ro := tm.AsReadonly()

	t.Run("read operations on readonly map", func(t *testing.T) {
		want := "ReadOnly"
		got, ok := Get(ro, StringKey)
		if !ok {
			t.Errorf("Get() from readonly: ok = false, want true")
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Get() from readonly mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestGetOrDefault(t *testing.T) {
	tm := NewTypedMap()
	Set(tm, StringKey, "Value")

	cases := []struct {
		name         string
		key          TypedKey[string]
		defaultValue string
		want         string
	}{
		{
			name:         "existing key",
			key:          StringKey,
			defaultValue: "Default",
			want:         "Value",
		},
		{
			name:         "non-existent key",
			key:          NewTypedKey[string]("non-existent"),
			defaultValue: "Default",
			want:         "Default",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetOrDefault(tm, tc.key, tc.defaultValue)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GetOrDefault() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTypeAssertionFailure(t *testing.T) {
	tm := NewTypedMap()
	Set(tm, StringKey, "Taro")

	// Try to retrieve as different type with the same underlying key
	wrongKey := NewTypedKey[int]("string-key")
	_, ok := Get(tm, wrongKey)

	if ok {
		t.Error("Expected type assertion to fail, but it succeeded")
	}
}

func TestConcurrentAccess(t *testing.T) {
	tm := NewTypedMap()
	done := make(chan bool)
	const goroutines = 10000

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			keyName := string(rune('A' + id))
			countKey := NewTypedKey[string](keyName)
			value := string(rune('a' + id))
			Set(tm, countKey, value)

			got, ok := Get(tm, countKey)
			if !ok || got != value {
				t.Errorf("Concurrent access test failed: got %v, ok=%v, want %v", got, ok, value)
			}

			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
}

func TestTypedMapClone(t *testing.T) {
	tm := NewTypedMap()
	Set(tm, StringKey, "Original")
	Set(tm, StructKey, Person{Name: "John", Age: 30})

	cloned := tm.Clone()

	t.Run("cloned map has original values", func(t *testing.T) {
		got, ok := Get(cloned, StringKey)
		if !ok {
			t.Errorf("Get() from cloned: ok = false, want true")
		}
		if diff := cmp.Diff("Original", got); diff != "" {
			t.Errorf("Get() from cloned mismatch (-want +got):\n%s", diff)
		}

		person, ok := Get(cloned, StructKey)
		if !ok {
			t.Errorf("Get() person from cloned: ok = false, want true")
		}
		want := Person{Name: "John", Age: 30}
		if diff := cmp.Diff(want, person); diff != "" {
			t.Errorf("Get() person from cloned mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("changes to original don't affect clone", func(t *testing.T) {
		Set(tm, StringKey, "Modified")

		origValue, _ := Get(tm, StringKey)
		if origValue != "Modified" {
			t.Errorf("original value is %v, want %v", origValue, "Modified")
		}

		clonedValue, _ := Get(cloned, StringKey)
		if clonedValue != "Original" {
			t.Errorf("cloned value is %v, want %v", clonedValue, "Original")
		}
	})
}

func TestReadonlyTypedMapClone(t *testing.T) {
	tm := NewTypedMap()
	Set(tm, StringKey, "ReadOnly")
	Set(tm, StructKey, Person{Name: "Bob", Age: 40})

	ro := tm.AsReadonly()
	roClone := ro.Clone()

	t.Run("cloned readonly map has original values", func(t *testing.T) {
		got, ok := Get(roClone, StringKey)
		if !ok {
			t.Errorf("Get() from cloned readonly: ok = false, want true")
		}
		if diff := cmp.Diff("ReadOnly", got); diff != "" {
			t.Errorf("Get() from cloned readonly mismatch (-want +got):\n%s", diff)
		}

		person, ok := Get(roClone, StructKey)
		if !ok {
			t.Errorf("Get() person from cloned readonly: ok = false, want true")
		}
		want := Person{Name: "Bob", Age: 40}
		if diff := cmp.Diff(want, person); diff != "" {
			t.Errorf("Get() person from cloned readonly mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("changes to source don't affect readonly clone", func(t *testing.T) {
		Set(tm, StringKey, "Modified")

		srcValue, _ := Get(tm, StringKey)
		if srcValue != "Modified" {
			t.Errorf("source value is %v, want %v", srcValue, "Modified")
		}

		clonedValue, _ := Get(roClone, StringKey)
		if clonedValue != "ReadOnly" {
			t.Errorf("cloned readonly value is %v, want %v", clonedValue, "ReadOnly")
		}
	})
}
