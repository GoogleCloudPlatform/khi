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
	"context"
	"log/slog"
)

// TeeHandler is a slog.Handler that duplicates log records to multiple child handlers,
// similar to the Unix `tee` command.
type TeeHandler struct {
	childHandlers []slog.Handler
}

// NewTeeHandler creates a new TeeHandler that writes to the provided handlers.
func NewTeeHandler(handlers ...slog.Handler) *TeeHandler {
	return &TeeHandler{handlers}
}

// Enabled implements slog.Handler.
func (t *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range t.childHandlers {
		// If any
		// of a child logger is enabled, task logger should be enabled.
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle implements slog.Handler.
func (t *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range t.childHandlers {
		err := handler.Handle(ctx, r)
		if err != nil {
			return err
		}
	}
	return nil
}

// WithAttrs implements slog.Handler.
func (t *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	childLoggersWithAttrs := make([]slog.Handler, len(t.childHandlers))
	for i, child := range t.childHandlers {
		childLoggersWithAttrs[i] = child.WithAttrs(attrs)
	}
	return &TeeHandler{
		childHandlers: childLoggersWithAttrs,
	}
}

// WithGroup implements slog.Handler.
func (t *TeeHandler) WithGroup(name string) slog.Handler {
	childLoggersWithGroup := make([]slog.Handler, len(t.childHandlers))
	for i, child := range t.childHandlers {
		childLoggersWithGroup[i] = child.WithGroup(name)
	}
	return &TeeHandler{
		childHandlers: childLoggersWithGroup,
	}
}

// TeeHandler implements slog.Handler
var _ slog.Handler = (*TeeHandler)(nil)
