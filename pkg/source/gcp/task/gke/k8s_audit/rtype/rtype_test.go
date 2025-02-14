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

package rtype

import (
	"fmt"
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestTypesAreFilled(t *testing.T) {
	for i := 1; i <= RTypeUnusedEnd; i++ {
		t.Run(fmt.Sprintf("check-%d-filled", i), func(t *testing.T) {
			for _, value := range Types {
				if value == i {
					return
				}
			}
			t.Errorf("type(%d) is not included in the Types", i)
		})
	}
}
