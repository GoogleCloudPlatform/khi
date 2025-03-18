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

package task

import (
	"testing"
	"time"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestGetTypedVariableFromTaskVariable(t *testing.T) {
	vs := NewVariableSet(map[string]any{})
	err := vs.Set("foo", time.Date(2023, time.April, 1, 1, 1, 1, 1, time.UTC))
	if err != nil {
		t.Errorf("unexpected error\n%s", err)
	}
	result, err := GetTypedVariableFromTaskVariable(vs, "foo", time.Time{})
	if err != nil {
		t.Errorf("unexpected error\n%s", err)
	}
	if result.String() != time.Date(2023, time.April, 1, 1, 1, 1, 1, time.UTC).String() {
		t.Errorf("not matching with the expected value\n%s", err)
	}
}
