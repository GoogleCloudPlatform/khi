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

package k8s

import (
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestGenerateDefaultMergeConfig(t *testing.T) {
	resolver, err := GenerateDefaultMergeConfig()
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	fmt.Println(resolver)
}

func TestBuilder(t *testing.T) {
	builder := appsv1.SchemeBuilder
	fmt.Println(builder)
}
