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

func TestProjectResourceContainer(t *testing.T) {
	const projectID = "foo"
	p := Project(projectID)

	if gotType := p.GetType(); gotType != ResourceContainerProject {
		t.Errorf("GetType() = %v, want %v", gotType, ResourceContainerProject)
	}

	const wantIdentifier = "projects/foo"
	if gotIdentifier := p.Identifier(); gotIdentifier != wantIdentifier {
		t.Errorf("Identifier() = %q, want %q", gotIdentifier, wantIdentifier)
	}

	if gotProjectID := p.ProjectID(); gotProjectID != projectID {
		t.Errorf("ProjectID() = %q, want %q", gotProjectID, projectID)
	}
}
