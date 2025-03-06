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

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestLogTypeMetadataIsFilled(t *testing.T) {
	for i := 0; i <= int(logTypeUnusedEnd); i++ {
		if _, ok := LogTypes[LogType(i)]; !ok {
			t.Errorf("LogTypeMetadata[%d] is not filled", i)
		}
	}
}

func TestLogTypeMetadataIsValid(t *testing.T) {
	for i := 0; i <= int(logTypeUnusedEnd); i++ {
		if logType, ok := LogTypes[LogType(i)]; ok {
			t.Run(fmt.Sprintf("%d-%s", i, logType.EnumKeyName), func(t *testing.T) {
				if logType.EnumKeyName == "" {
					t.Errorf("EnumKeyName in `%s(%d)` is empty", logType.Label, i)
				}
				if logType.LabelBackgroundColor == "" {
					t.Errorf("LabelBackgroundColor in `%s(%d)` is empty", logType.Label, i)
				}
				if logType.Label == "" {
					t.Errorf("Label in `%s(%d)` is empty", logType.Label, i)
				}
			})
		}
	}
}
