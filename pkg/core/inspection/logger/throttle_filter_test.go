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
	"log/slog"
	"strings"
	"testing"
)

func TestThrottleFilter(t *testing.T) {
	const maxPerKind = 2
	const logKind = "test-kind"

	buf, childHandler := newTestHandler()
	throttleFilter := NewThrottleFilter(maxPerKind, childHandler)
	logger := slog.New(throttleFilter)

	// --- Test Case 1: First log, should not be throttled ---
	t.Run("First log should not be throttled", func(t *testing.T) {
		buf.Reset()
		logger.Info("message 1", slog.String(LogKindAttrKey, logKind))
		output := buf.String()
		expected := `level=INFO msg="message 1" log-kind=test-kind`
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, but got %q", expected, output)
		}
		if strings.Contains(output, "Similar logs will be omitted") {
			t.Errorf("did not expect warning message, but got one in %q", output)
		}
	})

	// --- Test Case 2: Second log, should have warning ---
	t.Run("Second log should include a warning", func(t *testing.T) {
		buf.Reset()
		logger.Info("message 2", slog.String(LogKindAttrKey, logKind))
		output := buf.String()
		if !strings.Contains(output, "Similar logs will be omitted") {
			t.Errorf("expected output to contain warning, but got %q", output)
		}
		if !strings.Contains(output, "message 2") {
			t.Errorf("expected output to contain original message, but got %q", output)
		}
	})

	// --- Test Case 3: Third log, should be throttled ---
	t.Run("Third log should be throttled", func(t *testing.T) {
		buf.Reset()
		logger.Info("message 3", slog.String(LogKindAttrKey, logKind))
		output := buf.String()
		if output != "" {
			t.Errorf("expected buffer to be empty, but got: %q", output)
		}
	})

	// --- Test Case 4: Log with a different kind, should not be throttled ---
	t.Run("Log with a different kind should not be throttled", func(t *testing.T) {
		buf.Reset()
		logger.Info("another message", slog.String(LogKindAttrKey, "another-kind"))
		output := buf.String()
		expected := `level=INFO msg="another message" log-kind=another-kind`
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, but got %q", expected, output)
		}
	})

	// --- Test Case 5: Log with no kind, should not be throttled ---
	t.Run("Log with no kind should not be throttled", func(t *testing.T) {
		buf.Reset()
		logger.Info("no kind message")
		output := buf.String()
		expected := `level=INFO msg="no kind message"`
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, but got %q", expected, output)
		}
	})
}

func TestThrottleFilter_WithAttrs(t *testing.T) {
	const maxPerKind = 2
	const logKind = "test-kind-attrs"

	buf, childHandler := newTestHandler()
	baseHandler := NewThrottleFilter(maxPerKind, childHandler)
	handlerWithAttrs := baseHandler.WithAttrs([]slog.Attr{slog.String("attr_key", "attr_value")})
	logger := slog.New(handlerWithAttrs)

	t.Run("first log should pass with attrs", func(t *testing.T) {
		buf.Reset()
		logger.Info("message 1", slog.String(LogKindAttrKey, logKind))
		output := buf.String()
		expected := `level=INFO msg="message 1" attr_key=attr_value log-kind=test-kind-attrs`
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, but got %q", expected, output)
		}
	})
}

func TestThrottleFilter_WithGroup(t *testing.T) {
	const maxPerKind = 2
	const logKind = "test-kind-group"

	buf, childHandler := newTestHandler()
	baseHandler := NewThrottleFilter(maxPerKind, childHandler)
	handlerWithGroup := baseHandler.WithGroup("group_name")
	logger := slog.New(handlerWithGroup)

	t.Run("first log should pass with group", func(t *testing.T) {
		buf.Reset()
		logger.Info("message 1", slog.String(LogKindAttrKey, logKind))
		output := buf.String()
		expected := `level=INFO msg="message 1" group_name.log-kind=test-kind-group`
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, but got %q", expected, output)
		}
	})
}
