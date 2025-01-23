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

package testutil

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var ForceUpdateGolden = false

// VerifyWithGolden test if the given body is matching with the golden file saved for each tests under /test/golden/
func VerifyWithGolden(t *testing.T, verificationTargetName string, body string) {
	t.Helper()
	_, updateGolden := os.LookupEnv("UPDATE_GOLDEN")
	goldenName := t.Name() + "-" + verificationTargetName
	goldenPath := fmt.Sprintf("test/golden/%s", goldenName)
	if updateGolden || ForceUpdateGolden {
		file, err := os.OpenFile(goldenPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		_, err = file.Write([]byte(body))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Golden file updated: %s", goldenPath)
	} else {
		golden := MustReadText(goldenPath)
		if diff := cmp.Diff(golden, body); diff != "" {
			t.Errorf("input is not matching with the golden (-want,+got):\n%s", diff)
		}
	}
}
