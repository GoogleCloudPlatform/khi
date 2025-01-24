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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

type commonLogFieldExtractorStub struct {
	displayId            string
	id                   string
	message              string
	severity             enum.Severity
	timestamp            time.Time
	callCountDisplayId   int
	callCountId          int
	callCountMainMessage int
	callCountSeverity    int
	callCountTimestamp   int
}

// LogBody implements CommonLogFieldExtractor.
func (c *commonLogFieldExtractorStub) LogBody(l *LogEntity) string {
	panic("unimplemented")
}

// DisplayID implements CommonLogFieldExtractor.
func (c *commonLogFieldExtractorStub) DisplayID(l *LogEntity) string {
	c.callCountDisplayId++
	return c.displayId
}

// ID implements CommonLogFieldExtractor.
func (c *commonLogFieldExtractorStub) ID(l *LogEntity) string {
	c.callCountId++
	return c.id
}

// MainMessage implements CommonLogFieldExtractor.
func (c *commonLogFieldExtractorStub) MainMessage(l *LogEntity) (string, error) {
	c.callCountMainMessage++
	return c.message, nil
}

// Severity implements CommonLogFieldExtractor.
func (c *commonLogFieldExtractorStub) Severity(l *LogEntity) (enum.Severity, error) {
	c.callCountSeverity++
	return c.severity, nil
}

// Timestamp implements CommonLogFieldExtractor.
func (c *commonLogFieldExtractorStub) Timestamp(l *LogEntity) time.Time {
	c.callCountTimestamp++
	return c.timestamp
}

var _ CommonLogFieldExtractor = (*commonLogFieldExtractorStub)(nil)

func TestCachedLogFieldExtractor_DisplayID(t *testing.T) {
	stub := &commonLogFieldExtractorStub{
		displayId: "displayId",
	}
	cached := NewCachedLogFieldExtractor(stub)
	if cached.DisplayID(nil) != "displayId" {
		t.Errorf("DisplayID() = %v, want %v", cached.DisplayID(nil), "displayId")
	}
	if cached.DisplayID(nil) != "displayId" {
		t.Errorf("2nd DisplayID() = %v, want %v", cached.DisplayID(nil), "displayId")
	}
	if stub.callCountDisplayId != 1 {
		t.Errorf("DisplayID() = %v, want %v", stub.callCountDisplayId, 1)
	}
}

func TestCachedLogFieldExtractor_ID(t *testing.T) {
	stub := &commonLogFieldExtractorStub{
		id: "id",
	}
	cached := NewCachedLogFieldExtractor(stub)
	if cached.ID(nil) != "id" {
		t.Errorf("ID() = %v, want %v", cached.ID(nil), "id")
	}
	if cached.ID(nil) != "id" {
		t.Errorf("2nd ID() = %v, want %v", cached.ID(nil), "id")
	}
	if stub.callCountId != 1 {
		t.Errorf("ID() = %v, want %v", stub.callCountId, 1)
	}
}

func TestCachedLogFieldExtractor_MainMessage(t *testing.T) {
	stub := &commonLogFieldExtractorStub{
		message: "message",
	}
	cached := NewCachedLogFieldExtractor(stub)
	message, err := cached.MainMessage(nil)
	if err != nil {
		t.Errorf("MainMessage() = %v, want %v", err, nil)
	}
	if message != "message" {
		t.Errorf("MainMessage() = %v, want %v", message, "message")
	}
}

func TestCachedLogFieldExtractor_Severity(t *testing.T) {
	stub := &commonLogFieldExtractorStub{
		severity: enum.SeverityInfo,
	}
	cached := NewCachedLogFieldExtractor(stub)
	severity, err := cached.Severity(nil)
	if err != nil {
		t.Errorf("Severity() = %v, want %v", err, nil)
	}
	if severity != enum.SeverityInfo {
		t.Errorf("Severity() = %v, want %v", severity, enum.SeverityInfo)
	}
}

func TestCachedLogFieldExtractor_Timestamp(t *testing.T) {
	stub := &commonLogFieldExtractorStub{
		timestamp: time.Date(2023, 10, 1, 12, 30, 0, 0, time.UTC),
	}
	cached := NewCachedLogFieldExtractor(stub)
	timestamp := cached.Timestamp(nil)
	if timestamp.UTC().Format(time.RFC3339) != "2023-10-01T12:30:00Z" {
		t.Errorf("Timestamp() = %v, want %v", timestamp.UTC().Format(time.RFC3339), "2023-10-01T12:30:00Z")
	}
	timestamp = cached.Timestamp(nil)
	if timestamp.UTC().Format(time.RFC3339) != "2023-10-01T12:30:00Z" {
		t.Errorf("2nd Timestamp() = %v, want %v", timestamp.UTC().Format(time.RFC3339), "2023-10-01T12:30:00Z")
	}
	if stub.callCountTimestamp != 1 {
		t.Errorf("Timestamp() = %v, want %v", stub.callCountTimestamp, 1)
	}
}
