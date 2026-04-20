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

package khifilev6

import (
	"testing"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/google/go-cmp/cmp"
)

func TestInternPool_Intern(t *testing.T) {
	idGen := &IDGenerator{}
	pool := NewInternPool(idGen)

	testCases := []struct {
		name   string
		input  string
		wantID uint32
	}{
		{
			name:   "first string",
			input:  "foo",
			wantID: 1,
		},
		{
			name:   "second string",
			input:  "bar",
			wantID: 2,
		},
		{
			name:   "duplicate string",
			input:  "foo",
			wantID: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := pool.InternString(tc.input)
			if ref.id != tc.wantID {
				t.Errorf("Intern(%q) ID = %d, want %d", tc.input, ref.id, tc.wantID)
			}
			got := ref.Resolve()
			if diff := cmp.Diff(tc.input, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInternPool_ResolveStringFromID(t *testing.T) {
	idGen := &IDGenerator{}
	pool := NewInternPool(idGen)
	ref1 := pool.InternString("foo")
	ref2 := pool.InternString("bar")

	testCases := []struct {
		name string
		id   uint32
		want string
	}{
		{
			name: "resolve foo",
			id:   ref1.id,
			want: "foo",
		},
		{
			name: "resolve bar",
			id:   ref2.id,
			want: "bar",
		},
		{
			name: "invalid ID 0",
			id:   0,
			want: "",
		},
		{
			name: "invalid ID large",
			id:   999,
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := pool.resolveStringFromID(tc.id)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("resolveStringFromID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInternStringRef_ToProto(t *testing.T) {
	idGen := &IDGenerator{}
	pool := NewInternPool(idGen)
	ref := pool.InternString("foo")

	got := ref.ToProto()

	wantId := ref.id
	wantVal := "foo"
	want := &pb.InternString{
		Id:    &wantId,
		Value: &wantVal,
	}

	if got.GetId() != want.GetId() {
		t.Errorf("ToProto().GetId() = %d, want %d", got.GetId(), want.GetId())
	}
	if got.GetValue() != want.GetValue() {
		t.Errorf("ToProto().GetValue() = %q, want %q", got.GetValue(), want.GetValue())
	}
}

func TestInternPool_SortedRefs(t *testing.T) {
	idGen := &IDGenerator{}
	pool := NewInternPool(idGen)
	pool.InternString("c")
	pool.InternString("a")
	pool.InternString("b")

	var refs []*InternStringRef
	for ref := range pool.SortedStringRefs() {
		refs = append(refs, ref)
	}

	testCases := []struct {
		name string
		idx  int
		want string
	}{
		{
			name: "first is a",
			idx:  0,
			want: "a",
		},
		{
			name: "second is b",
			idx:  1,
			want: "b",
		},
		{
			name: "third is c",
			idx:  2,
			want: "c",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.idx >= len(refs) {
				t.Fatalf("Index %d out of range", tc.idx)
			}
			got := refs[tc.idx].Resolve()
			if got != tc.want {
				t.Errorf("refs[%d].Resolve() = %q, want %q", tc.idx, got, tc.want)
			}
		})
	}
}

func TestInternPool_InternFieldSet(t *testing.T) {
	idGen := &IDGenerator{}
	pool := NewInternPool(idGen)

	testCases := []struct {
		name   string
		input  []string
		want   []string
		wantID uint32
	}{
		{
			name:   "first set",
			input:  []string{"a", "b"},
			want:   []string{"a", "b"},
			wantID: 1,
		},
		{
			name:   "second set",
			input:  []string{"c"},
			want:   []string{"c"},
			wantID: 2,
		},
		{
			name:   "duplicate set",
			input:  []string{"a", "b"},
			want:   []string{"a", "b"},
			wantID: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := pool.InternFieldSet(tc.input)
			if ref.id != tc.wantID {
				t.Errorf("InternFieldSet(%v) ID = %d, want %d", tc.input, ref.id, tc.wantID)
			}
			got := ref.Resolve()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Resolve() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInternPool_FieldSetRefs(t *testing.T) {
	idGen := &IDGenerator{}
	pool := NewInternPool(idGen)

	pool.InternFieldSet([]string{"a", "b"})
	pool.InternFieldSet([]string{"c"})
	pool.InternFieldSet([]string{"a", "c"})

	var refs []*FieldPathSetRef
	for ref := range pool.FieldSetRefs() {
		refs = append(refs, ref)
	}

	testCases := []struct {
		name string
		idx  int
		want []string
	}{
		{
			name: "first is [a, b]",
			idx:  0,
			want: []string{"a", "b"},
		},
		{
			name: "second is [c]",
			idx:  1,
			want: []string{"c"},
		},
		{
			name: "third is [a, c]",
			idx:  2,
			want: []string{"a", "c"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.idx >= len(refs) {
				t.Fatalf("Index %d out of range", tc.idx)
			}
			got := refs[tc.idx].Resolve()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("refs[%d].Resolve() mismatch (-want +got):\n%s", tc.idx, diff)
			}
		})
	}
}
