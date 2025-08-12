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

package log

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/task/core/contract/taskid"
	metadata_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/metadata"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestConformance(t *testing.T) {
	logMetadata := NewLogMetadata()
	metadata_test.ConformanceMetadataTypeTest(t, logMetadata)
}

func TestLogMetadata(t *testing.T) {
	logMetadata := NewLogMetadata()

	// task2 comes before task1 alphabetically to test sorting
	taskID1 := taskid.NewDefaultImplementationID[any]("task2")
	taskID2 := taskid.NewDefaultImplementationID[any]("task1")

	// 1. Get buffer for a new task
	buffer1 := logMetadata.GetTaskLogBuffer(taskID1)
	if buffer1.Len() != 0 {
		t.Errorf("expected a new empty buffer, but got buffer with length %d", buffer1.Len())
	}

	// 2. Write to the buffer
	logMessage1 := "hello from task 2"
	buffer1.WriteString(logMessage1)

	// 3. Get buffer for the same task again
	buffer1Again := logMetadata.GetTaskLogBuffer(taskID1)
	if buffer1Again != buffer1 {
		t.Error("expected the same buffer instance, but got a different one")
	}
	if buffer1Again.String() != logMessage1 {
		t.Errorf("expected buffer to contain %q, but got %q", logMessage1, buffer1Again.String())
	}

	// 4. Get buffer for another new task
	buffer2 := logMetadata.GetTaskLogBuffer(taskID2)
	if buffer2.Len() != 0 {
		t.Errorf("expected a new empty buffer for task1, but got buffer with length %d", buffer2.Len())
	}
	logMessage2 := "hello from task 1"
	buffer2.WriteString(logMessage2)

	// 5. Test ToSerializable
	serializable := logMetadata.ToSerializable()
	items, ok := serializable.([]SerializableLogItem)
	if !ok {
		t.Fatalf("ToSerializable() did not return []SerializableLogItem, got %T", serializable)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 log items, but got %d", len(items))
	}

	// Check content and order of the serialized items
	if items[0].Id != taskID2.String() {
		t.Errorf("expected first item to have ID %q, but got %q", taskID2.String(), items[0].Id)
	}
	if items[0].Log != logMessage2 {
		t.Errorf("expected first item to have log %q, but got %q", logMessage2, items[0].Log)
	}

	if items[1].Id != taskID1.String() {
		t.Errorf("expected second item to have ID %q, but got %q", taskID1.String(), items[1].Id)
	}
	if items[1].Log != logMessage1 {
		t.Errorf("expected second item to have log %q, but got %q", logMessage1, items[1].Log)
	}
}
