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

package inspectiontaskbase

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

func mustNewLogFromYAML(t *testing.T, yaml string) *log.Log {
	t.Helper()
	l, err := log.NewLogFromYAMLString(yaml)
	if err != nil {
		t.Fatalf("failed to create log from YAML: %v", err)
	}
	return l
}
