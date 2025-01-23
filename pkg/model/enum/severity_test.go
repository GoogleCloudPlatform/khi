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

package enum

import (
	"fmt"
	"testing"
)

func TestSeverityMetadataIsFilled(t *testing.T) {
	for i := 0; i <= int(severityUnusedEnd); i++ {
		if _, ok := Severities[Severity(i)]; !ok {
			t.Errorf("SeverityMetadata[%d] is not filled", i)
		}
	}
}

func TestSeverityMetadataIsValid(t *testing.T) {
	for i := 0; i <= int(severityUnusedEnd); i++ {
		if severity, ok := Severities[Severity(i)]; ok {
			t.Run(fmt.Sprintf("%d-%s", i, Severities[Severity(i)].Label), func(t *testing.T) {
				if severity.EnumKeyName == "" {
					t.Errorf("EnumKeyName in `%s(%d)` is empty", severity.Label, i)
				}
				if severity.LabelColor == "" {
					t.Errorf("LabelColor in `%s(%d)` is empty", severity.Label, i)
				}
				if severity.BackgroundColor == "" {
					t.Errorf("LabelBackgroundColor in `%s(%d)` is empty", severity.Label, i)
				}
			})
		}
	}
}
