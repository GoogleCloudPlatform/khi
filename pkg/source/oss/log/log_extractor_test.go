// Copyright 2025 Google LLC
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
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/adapter"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func logFromYAML(yaml string) (*log.LogEntity, error) {
	readerFactory := structure.NewReaderFactory(&structuredatastore.OnMemoryStructureDataStore{})
	reader, err := readerFactory.NewReader(adapter.Yaml(yaml))
	if err != nil {
		return nil, err
	}
	return log.NewLogEntity(reader, &OSSAuditLogFieldExtractor{}), nil
}

func TestOSSAuditLogFieldExtractor(t *testing.T) {
	testutil.InitTestIO()

	sampleLog := testutil.MustReadText("test/logs/oss/test-log.yaml")
	log, err := logFromYAML(sampleLog)

	if err != nil {
		t.Errorf("failed to read sample log as yaml")
	}

	t.Run("Get DisplayID", func(t *testing.T) {
		id := log.DisplayId()
		if id != "7a816f5c-b093-4f2f-8124-0c6083e41cd4" {
			t.Errorf("wrong ID, got: %s, expected: 7a816f5c-b093-4f2f-8124-0c6083e41cd4", id)
		}
	})

	t.Run("Get ID", func(t *testing.T) {
		id := log.ID()
		if id != "7a816f5c-b093-4f2f-8124-0c6083e41cd4" {
			t.Errorf("wrong ID, got: %s, expected: 7a816f5c-b093-4f2f-8124-0c6083e41cd4", id)
		}
	})

	t.Run("Get LogBody", func(t *testing.T) {
		logBody := log.LogBody()
		if logBody == "" {
			t.Error("log body should not be empty")
		}
		// Test that the log body contains key elements
		if !strings.Contains(logBody, "auditID") || !strings.Contains(logBody, "7a816f5c-b093-4f2f-8124-0c6083e41cd4") {
			t.Error("log body missing expected content")
		}
	})

	t.Run("Get MainMessage", func(t *testing.T) {
		message, err := log.MainMessage()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if message != "" {
			t.Errorf("expected empty message, got: %s", message)
		}
	})

	t.Run("Get Severity", func(t *testing.T) {
		severity, err := log.Severity()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if severity != enum.SeverityUnknown {
			t.Errorf("wrong severity, got: %v, expected: %v", severity, enum.SeverityUnknown)
		}
	})

	t.Run("Get Timestamp", func(t *testing.T) {
		timestamp := log.Timestamp()
		expected, _ := time.Parse(time.RFC3339Nano, "2025-04-01T13:33:59.457732Z")
		if !timestamp.Equal(expected) {
			t.Errorf("wrong timestamp, got: %v, expected: %v", timestamp, expected)
		}
	})
}
