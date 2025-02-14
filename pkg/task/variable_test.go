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
	"context"
	"testing"
	"time"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestVariableSetToBeReleasedWhenItWasSet(t *testing.T) {
	variableSet := NewVariableSet(map[string]any{})
	go func() {
		<-time.After(time.Second)
		variableSet.Set("foo-key", "bar")
	}()
	val, err := variableSet.Wait(context.Background(), "foo-key")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if val != "bar" {
		t.Errorf("variable `foo-key` wasn't resolved to `bar`")
	}
}

func TestVariableSetToBeCancelledWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	variableSet := NewVariableSet(map[string]any{})
	go func() {
		<-time.After(time.Second)
		cancel()
	}()
	_, err := variableSet.Wait(ctx, "foo-key")
	if err != context.Canceled {
		t.Errorf("unexpected error %v", err)
	}
	_, err = variableSet.Wait(ctx, "foo-key")
	if err != context.Canceled {
		t.Errorf("unexpected error %v", err)
	}
}

func TestVariableSetShouldReturnErrorWhenTheVariableSetTwice(t *testing.T) {
	variableSet := NewVariableSet(map[string]any{})
	err1 := variableSet.Set("foo-key", "bar1")
	err2 := variableSet.Set("foo-key", "bar2")
	if err1 != nil {
		t.Errorf("err1 is expected to be nil. but an error occured.")
	}
	if err2 == nil {
		t.Errorf("err2 is expected to be an error. but no error occured.")
	}
}

func TestGetTypedVariableFromTaskVariable(t *testing.T) {
	vs := NewVariableSet(map[string]any{})
	err := vs.Set("foo", time.Date(2023, time.April, 1, 1, 1, 1, 1, time.UTC))
	if err != nil {
		t.Errorf("unexpected error\n%s", err)
	}
	result, err := GetTypedVariableFromTaskVariable[time.Time](vs, "foo", time.Time{})
	if err != nil {
		t.Errorf("unexpected error\n%s", err)
	}
	if result.String() != time.Date(2023, time.April, 1, 1, 1, 1, 1, time.UTC).String() {
		t.Errorf("not matching with the expected value\n%s", err)
	}
}

func TestGetTypedVariableFromTaskVariableWithInvalidType(t *testing.T) {
	vs := NewVariableSet(map[string]any{})
	err := vs.Set("foo", time.Date(2023, time.April, 1, 1, 1, 1, 1, time.UTC))
	if err != nil {
		t.Errorf("unexpected error\n%s", err)
	}
	_, err = GetTypedVariableFromTaskVariable[*time.Time](vs, "foo", nil)
	if err.Error() != "the given value 2023-04-01 01:01:01.000000001 +0000 UTC in foo couldn't be converted to *time.Time" {
		t.Errorf("the result error is not matching with the expected error\n%s", err)
	}
}
