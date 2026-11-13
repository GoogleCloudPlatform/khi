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

package langgo

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNormalizeDeclBytes(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "preserves blank identifier, selector fields, struct keys, and map string keys",
			input: `// Dedup removes duplicate strings.
func Dedup(items []string) []string {
	type localEntry struct{ Key string }
	seen := map[string]int{"seed": 1}
	for _, item := range items {
		_ = item
		seen[item] = 1
	}
	out := make([]string, 0, len(seen))
	var existingKey string
	for existingKey = range seen {
		out = append(out, existingKey)
	}
	return out
}`,
			want: `func targetFunc(v1 []string) []string {
	type localEntry struct{ Key string }
	v2 := map[string]int{"seed": 1}
	for _, v3 := range v1 {
		_ = v3
		v2[v3] = 1
	}
	v4 := make([]string, 0, len(v2))
	var v5 string
	for v5 = range v2 {
		v4 = append(v4, v5)
	}
	return v4
}`,
		},
		{
			name: "protects targetFunc, RecvType, selector, and struct field key against shadowing parameters",
			input: `func (RecvType *Box[T]) ShadowCheck(targetFunc T, Name string) Entry {
	_ = RecvType.Name
	return Entry{
		Name: Name,
	}
}`,
			want: `func (v1 *RecvType[v2]) targetFunc(v3 v2, v4 string) Entry {
	_ = v1.Name
	return Entry{
		Name: v4,
	}
}`,
		},
		{
			name: "normalizes multi-param generic receiver type and local const/var declarations",
			input: `func (d *TypedDict[K, V]) BuildEntry(label string) Entry {
	const offset = 1
	var count int
	count = len(label) + offset
	return Entry{
		Name: label,
		Size: count,
	}
}`,
			want: `func (v1 *RecvType[v2, v3]) targetFunc(v4 string) Entry {
	const v5 = 1
	var v6 int
	v6 = len(v4) + v5
	return Entry{
		Name: v4,
		Size: v6,
	}
}`,
		},
		{
			name: "protects targetTask against shadowing closure parameter",
			input: `var myTask, secondTask = taskbase.DefineTask(
	taskID,
	func(targetTask *Binder) TaskFunc {
		logs := coretask.Use(targetTask, refID)
		return func(ctx context.Context) error {
			_ = logs.Get(ctx)
			return nil
		}
	},
), nil`,
			want: `var targetTask, targetTask2 = taskbase.DefineTask(
	taskID,
	func(v1 *Binder) TaskFunc {
		v2 := coretask.Use(v1, refID)
		return func(v3 context.Context) error {
			_ = v2.Get(v3)
			return nil
		}
	},
), nil`,
		},
		{
			name:  "returns trimmed raw snippet on unparseable fragment",
			input: "  not valid go decl  ",
			want:  "not valid go decl",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeDeclBytes([]byte(tc.input))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("NormalizeDeclBytes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
