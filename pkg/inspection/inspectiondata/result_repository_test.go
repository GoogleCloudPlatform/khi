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

package inspectiondata

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestFileSystemResultRepository(t *testing.T) {
	testutil.InitTestIO()
	repo := NewFileSystemInspectionResultRepository("/tmp/test.json")
	t.Run("ReadInspectionResult can read inspection result written with WriteInspectionResult", func(t *testing.T) {
		testInspectionData := []byte{
			0x01, 0x02, 0x03, 0x04, 0x05,
		}

		writer, writeErr := repo.GetWriter()
		if writeErr != nil {
			t.Errorf("writeErr: want nil, got %s", writeErr)
		}
		writer.Write(testInspectionData)
		repo.Close()
		received, readErr := repo.GetReader()
		var readTarget = make([]byte, 5)
		_, err := received.Read(readTarget)
		if err != nil {
			t.Errorf("unexpected errir %s", err)
		}
		repo.Close()

		if readErr != nil {
			t.Errorf("readErr: want nil, got %s", readErr)
		}
		if diff := cmp.Diff(testInspectionData, readTarget); diff != "" {
			t.Errorf("+testInspectionData, -received,%s", diff)
		}
	})
}
