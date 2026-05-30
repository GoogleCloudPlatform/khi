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

package logger

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
)

func TestKHILogFormatHandler_Handle(t *testing.T) {
	taskID := "test-task-123"
	ctxWithTaskID := khictx.WithValue(context.Background(), core_contract.TaskImplementationIDContextKey, taskid.NewDefaultImplementationID[struct{}](taskID).(taskid.UntypedTaskImplementationID))

	testCases := []struct {
		name           string
		handler        *KHILogFormatHandler
		ctx            context.Context
		record         slog.Record
		expectedOutput string
	}{
		{
			name:           "With TaskID, With Color, Level Info",
			handler:        NewKHIFormatLogger(new(bytes.Buffer), true),
			ctx:            ctxWithTaskID,
			record:         slog.NewRecord(time.Now(), slog.LevelInfo, "info message", 0),
			expectedOutput: fmt.Sprintf("%s%s >%s %s %s\n", "\033[91m", taskID+"#default", reset, "\033[96mINFO\033[0m", "\033[96minfo message\033[0m"),
		},
		{
			name:           "With TaskID, No Color, Level Warn",
			handler:        NewKHIFormatLogger(new(bytes.Buffer), false),
			ctx:            ctxWithTaskID,
			record:         slog.NewRecord(time.Now(), slog.LevelWarn, "warn message", 0),
			expectedOutput: fmt.Sprintf("%s > WARN warn message\n", taskID+"#default"),
		},
		{
			name:           "Without TaskID, With Color, Level Error",
			handler:        NewKHIFormatLogger(new(bytes.Buffer), true),
			ctx:            context.Background(),
			record:         slog.NewRecord(time.Now(), slog.LevelError, "error message", 0),
			expectedOutput: fmt.Sprintf("global > %s %s\n", "\033[97;101mERROR\033[0m", "\033[97;101merror message\033[0m"),
		},
		{
			name:           "Without TaskID, No Color, Level Debug",
			handler:        NewKHIFormatLogger(new(bytes.Buffer), false),
			ctx:            context.Background(),
			record:         slog.NewRecord(time.Now(), slog.LevelDebug, "debug message", 0),
			expectedOutput: "global > DEBUG debug message\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := tc.handler.out.(*bytes.Buffer)
			buf.Reset()

			err := tc.handler.Handle(tc.ctx, tc.record)
			if err != nil {
				t.Fatalf("Handle() returned an unexpected error: %v", err)
			}

			if got := buf.String(); got != tc.expectedOutput {
				t.Errorf("mismatched log output:\ngot:  %q\nwant: %q", got, tc.expectedOutput)
			}
		})
	}
}

func TestKHILogFormatHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handlerSource := NewKHIFormatLogger(&buf, false)
	handler := handlerSource.WithAttrs([]slog.Attr{slog.String("component", "parser")})

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "message", 0)
	record.AddAttrs(slog.Int("count", 10))
	ctx := context.Background()

	if err := handler.Handle(ctx, record); err != nil {
		t.Fatalf("Handle() failed: %v", err)
	}

	expectedOutput := "global > INFO message component=parser count=10\n"
	if got := buf.String(); got != expectedOutput {
		t.Errorf("mismatched log output:\ngot:  %q\nwant: %q", got, expectedOutput)
	}
}

func TestKHILogFormatHandler_WithAttrsAppends(t *testing.T) {
	var buf bytes.Buffer
	handlerSource := NewKHIFormatLogger(&buf, false)
	handler := handlerSource.
		WithAttrs([]slog.Attr{slog.String("component", "parser")}).
		WithAttrs([]slog.Attr{slog.String("phase", "read")})

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "message", 0)
	ctx := context.Background()

	if err := handler.Handle(ctx, record); err != nil {
		t.Fatalf("Handle() failed: %v", err)
	}

	expectedOutput := "global > INFO message component=parser phase=read\n"
	if got := buf.String(); got != expectedOutput {
		t.Errorf("mismatched log output:\ngot:  %q\nwant: %q", got, expectedOutput)
	}
}

func TestKHILogFormatHandler_WithGroup(t *testing.T) {
	handler1 := NewKHIFormatLogger(new(bytes.Buffer), false)
	handler2 := handler1.WithGroup("my-group")

	// The implementation is a no-op, so it should return the same handler instance.
	if handler1 != handler2 {
		t.Errorf("WithGroup() should be a no-op and return the same handler instance")
	}
}

func TestKHILogFormatHandler_Enabled(t *testing.T) {
	handler := NewKHIFormatLogger(new(bytes.Buffer), false)
	ctx := context.Background()

	levels := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}
	for _, level := range levels {
		if !handler.Enabled(ctx, level) {
			t.Errorf("Enabled() should return true for level %s, but it returned false", level)
		}
	}
}
