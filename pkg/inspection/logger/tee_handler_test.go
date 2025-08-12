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
	"log/slog"
	"strings"
	"testing"
)

func TestTeeHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	// Create two child handlers, each writing to a different buffer.
	handler1 := slog.NewTextHandler(&buf1, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{} // Remove time for stable comparison
			}
			return a
		},
	})
	handler2 := slog.NewTextHandler(&buf2, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{} // Remove time for stable comparison
			}
			return a
		},
	})

	// Create the TeeHandler with the two child handlers.
	teeHandler := NewTeeHandler(handler1, handler2)
	logger := slog.New(teeHandler)

	// --- Test Case 1: Basic log ---
	t.Run("Basic log should be written to both handlers", func(t *testing.T) {
		buf1.Reset()
		buf2.Reset()

		logger.Info("hello world")

		output1 := buf1.String()
		output2 := buf2.String()

		expected := `level=INFO msg="hello world"`
		if !strings.Contains(output1, expected) {
			t.Errorf("handler 1: expected output to contain %q, but got %q", expected, output1)
		}
		if output1 != output2 {
			t.Errorf("handler 1 and 2 outputs do not match:\n1: %q\n2: %q", output1, output2)
		}
	})

	// --- Test Case 2: Log with attributes ---
	t.Run("Log with attributes should be written to both handlers", func(t *testing.T) {
		buf1.Reset()
		buf2.Reset()

		loggerWithAttrs := logger.With(slog.String("user", "kakeru"))
		loggerWithAttrs.Warn("a warning message")

		output1 := buf1.String()
		output2 := buf2.String()

		expected := `level=WARN msg="a warning message" user=kakeru`
		if !strings.Contains(output1, expected) {
			t.Errorf("handler 1: expected output to contain %q, but got %q", expected, output1)
		}
		if output1 != output2 {
			t.Errorf("handler 1 and 2 outputs do not match:\n1: %q\n2: %q", output1, output2)
		}
	})

	// --- Test Case 3: Log with a group ---
	t.Run("Log with a group should be written to both handlers", func(t *testing.T) {
		buf1.Reset()
		buf2.Reset()

		loggerWithGroup := logger.WithGroup("request")
		loggerWithGroup.Error("something failed", slog.Int("status", 500))

		output1 := buf1.String()
		output2 := buf2.String()

		expected := `level=ERROR msg="something failed" request.status=500`
		if !strings.Contains(output1, expected) {
			t.Errorf("handler 1: expected output to contain %q, but got %q", expected, output1)
		}
		if output1 != output2 {
			t.Errorf("handler 1 and 2 outputs do not match:\n1: %q\n2: %q", output1, output2)
		}
	})
}
