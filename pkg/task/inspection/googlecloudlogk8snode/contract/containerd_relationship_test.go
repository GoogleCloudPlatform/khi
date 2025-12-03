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

package googlecloudlogk8snode_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/google/go-cmp/cmp"
)

func TestPodSandboxIDInfo_ResourcePath(t *testing.T) {
	info := &PodSandboxIDInfo{
		PodNamespace: "test-namespace",
		PodName:      "test-pod",
	}
	want := resourcepath.Pod("test-namespace", "test-pod")
	got := info.ResourcePath()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ResourcePath() mismatch (-want +got):\n%s", diff)
	}
}
