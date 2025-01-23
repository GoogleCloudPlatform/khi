// Copyright 2024 Google LLC
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

package testlog

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTestLogWith(t *testing.T) {
	tl := New(BaseYaml(`foo: bar`))
	expectedTl1 := `foo: bar1
`
	expectedTl2 := `foo: bar2
`
	// With returns a new instance of TestLog and each instances are independent
	tl1 := tl.With(StringField("foo", "bar1"))
	tl2 := tl1.With(StringField("foo", "bar2"))

	tl1Yaml := tl1.MustBuildYamlString()
	tl2Yaml := tl2.MustBuildYamlString()
	if diff := cmp.Diff(tl1Yaml, expectedTl1); diff != "" {
		t.Errorf("Yaml string mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(tl2Yaml, expectedTl2); diff != "" {
		t.Errorf("Yaml string mismatch (-want +got):\n%s", diff)
	}
}
