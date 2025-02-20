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

package progress

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestIndeterminateUpdator(t *testing.T) {
	progress := NewTaskProgress("foo")
	updator := NewIndeterminateUpdator(progress, 1000*time.Millisecond)
	err := updator.Start("working")
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}
	time.Sleep(1500 * time.Millisecond)
	if diff := cmp.Diff(&TaskProgress{
		Id:            "foo",
		Label:         "foo",
		Message:       "working.",
		Percentage:    0,
		Indeterminate: true,
	}, progress, cmpopts.IgnoreUnexported(TaskProgress{})); diff != "" {
		t.Errorf("The result status is not in the expected status\n%s", diff)
	}
	err = updator.Done()
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}
	msg := progress.Message
	time.Sleep(1000 * time.Millisecond)
	if msg != progress.Message {
		t.Errorf("The progress is keeping updated")
	}
}
