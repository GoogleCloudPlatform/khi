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
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestSeverityFilter(t *testing.T) {
	var buf bytes.Buffer
	childHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		// Remove time to make assertions stable
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})

	// Set the minimum severity to INFO
	severityFilter := NewSeverityFilter(slog.LevelInfo, childHandler)

	testCases := []struct {
		name          string
		level         slog.Level
		message       string
		expectBlocked bool
	}{
		{
			name:          "DEBUG log should be blocked",
			level:         slog.LevelDebug,
			message:       "this is a debug message",
			expectBlocked: true,
		},
		{
			name:          "INFO log should pass",
			level:         slog.LevelInfo,
			message:       "this is an info message",
			expectBlocked: false,
		},
		{
			name:          "WARN log should pass",
			level:         slog.LevelWarn,
			message:       "this is a warning message",
			expectBlocked: false,
		},
		{
			name:          "ERROR log should pass",
			level:         slog.LevelError,
			message:       "this is an error message",
			expectBlocked: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()

			// Check the Enabled method first
			if severityFilter.Enabled(context.Background(), tc.level) == tc.expectBlocked {
				t.Errorf("Enabled() returned %v, expected %v", !tc.expectBlocked, tc.expectBlocked)
			}

			// Handle the record
			record := slog.NewRecord(time.Time{}, tc.level, tc.message, 0)
			if err := severityFilter.Handle(context.Background(), record); err != nil {
				t.Fatalf("Handle() returned an unexpected error: %v", err)
			}

			output := buf.String()

			if tc.expectBlocked {
				if output != "" {
					t.Errorf("expected buffer to be empty for blocked log, but got: %q", output)
				}
			} else {
				expectedMsg := `msg="` + tc.message + `"`
				if !strings.Contains(output, expectedMsg) {
					t.Errorf("expected output to contain %q, but got %q", expectedMsg, output)
				}
				expectedLevel := `level=` + tc.level.String()
				if !strings.Contains(output, expectedLevel) {
					t.Errorf("expected output to contain %q, but got %q", expectedLevel, output)
				}
			}
		})
	}
}
