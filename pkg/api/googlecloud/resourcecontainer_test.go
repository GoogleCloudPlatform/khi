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

package googlecloud

import "testing"

func TestProjectResourceContainer_GetType(t *testing.T) {
	gotType := Project("foo").GetType()

	if gotType != ResourceContainerProject {
		t.Errorf("GetType() = %v, want %v", gotType, ResourceContainerProject)
	}
}

func TestProjectResourceContainer_Identifier(t *testing.T) {
	gotIdentifier := Project("foo").Identifier()

	if gotIdentifier != "projects/foo" {
		t.Errorf("Identifier() = %q, want %q", gotIdentifier, "projects/foo")
	}
}

func TestProjectResourceContainer_ProjectID(t *testing.T) {
	gotProjectID := Project("foo").ProjectID()

	if gotProjectID != "foo" {
		t.Errorf("ProjectID() = %q, want %q", gotProjectID, "foo")
	}
}
