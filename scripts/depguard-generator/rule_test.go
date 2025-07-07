// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGeneratedRule_Builder(t *testing.T) {
	rule := NewGeneratedRule("test-rule", []string{"pkg/a/*"}).
		AddDeny([]string{"pkg/b"}, "reason b").
		AddDeny([]string{"pkg/c", "pkg/d"}, "reason c and d")

	expected := &GeneratedRule{
		RuleName:    "test-rule",
		TargetFiles: []string{"pkg/a/*"},
		DeniedPkgs: []map[string]string{
			{"pkg": "pkg/b", "desc": "reason b"},
			{"pkg": "pkg/c", "desc": "reason c and d"},
			{"pkg": "pkg/d", "desc": "reason c and d"},
		},
		AllowedPkgs: []string{},
	}

	if diff := cmp.Diff(expected, rule); diff != "" {
		t.Errorf("GeneratedRule mismatch (-want +got):\n%s", diff)
	}
}
