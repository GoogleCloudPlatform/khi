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

package rawlogs

import (
	"math/rand"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestFilesystemRepository(t *testing.T) {
	t.Run("sort data in correct order", func(t *testing.T) {
		r, err := NewFilesystemRepository()
		if err != nil {
			t.Error("an error was raised", err)
			return
		}
		r.Write(time.Unix(2, 0), []byte{2, 0, 0})
		r.Write(time.Unix(1, 0), []byte{1, 0, 0})
		r.Write(time.Unix(3, 0), []byte{3, 0, 0})
		r.Write(time.Unix(4, 0), []byte{4, 0, 0})
		r.Write(time.Unix(3, 0), []byte{3, 0, 0})
		r.Write(time.Unix(5, 0), []byte{5, 0, 0})

		iter := r.IterateInSortedOrder()

		resultInOrder := make([]byte, 0)
		for iter.HasNext() {
			data, err := iter.Next()
			if err != nil {
				t.Error("an error was raised", err)
				return
			}
			if len(data) != 3 {
				t.Error("data length mismatch,", len(data))
			}
			resultInOrder = append(resultInOrder, data[0])
		}

		if diff := cmp.Diff([]byte{1, 2, 3, 3, 4, 5}, resultInOrder); diff != "" {
			t.Error("-expected,+actual", diff)
		}
		r.Dispose()
	})

	t.Run("large data test", func(t *testing.T) {
		sizePerItem := 1024 * 1024 // 1MB
		itemCount := 1024 * 10     // 10GB
		r, err := NewFilesystemRepository()
		validationBytes := make([]byte, 0)
		if err != nil {
			t.Error("an error was raised", err)
			return
		}

		for i := 0; i < itemCount; i++ {
			dataBuffer := make([]byte, sizePerItem)
			dataBuffer[0] = byte(rand.Uint32() % 256)
			validationBytes = append(validationBytes, dataBuffer[0])
			err := r.Write(time.UnixMicro(int64(itemCount-i)), dataBuffer)
			if err != nil {
				t.Error("an error was raised", err)
				return
			}
		}
		iter := r.IterateInSortedOrder()

		for i := itemCount - 1; i >= 0; i-- {
			result, err := iter.Next()
			if err != nil {
				t.Error("an error was raised", err)
				return
			}

			if len(result) != sizePerItem {
				t.Error("data length mismatch,", len(result))
				return
			}

			if result[0] != validationBytes[i] {
				t.Error("validation byte mismatch", result[0], validationBytes[i])
				return
			}
		}
		r.Dispose()
	})
}
