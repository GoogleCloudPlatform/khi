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

package workbench

import (
	"testing"
)

func TestWorkbench_Lifecycle(t *testing.T) {
	wb := NewWorkbench("user-session-1", "inspection-100")

	if got := wb.ID(); got != "user-session-1" {
		t.Errorf("ID() = %q, want %q", got, "user-session-1")
	}

	if got := wb.InspectionID(); got != "inspection-100" {
		t.Errorf("InspectionID() = %q, want %q", got, "inspection-100")
	}

	if wb.IsClosed() {
		t.Errorf("expected new workbench not to be closed")
	}

	// Close marks as closed
	wb.Close()
	if !wb.IsClosed() {
		t.Errorf("expected workbench to be closed after Close()")
	}
}
