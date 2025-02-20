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

package structuredata

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	corev1 "k8s.io/api/core/v1"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestReadPodManifest(t *testing.T) {
	testutil.InitTestIO()
	podYaml := testutil.MustReadText("test/k8s/sample_pod.yaml")
	sd, err := DataFromYaml(podYaml)
	if err != nil {
		t.Fatal(err)
	}
	var pod corev1.Pod
	err = ReadReflectK8sManifest(sd, &pod)
	if err != nil {
		t.Fatal(err)
	}
	if pod.UID != "7899f560-3d56-4831-a381-2691c28ea3e5" {
		t.Errorf("parsed pod UID is not matching with the expected value")
	}
}
