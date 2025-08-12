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

// SeverityFilter is a slog.Handler that filters log records based on a minimum severity level.
type SeverityFilter struct {
	childLogHandler slog.Handler
	minSeverity     slog.Level
}

// NewSeverityFilter creates a new SeverityFilter.
// It wraps a child handler and only passes log records to it that meet the minimum severity level.
func NewSeverityFilter(minSeverity slog.Level, childHandler slog.Handler) slog.Handler {
	return &SeverityFilter{childHandler, minSeverity}
}

// Enabled implements slog.Handler.
func (s *SeverityFilter) Enabled(ctx context.Context, l slog.Level) bool {
	if l < s.minSeverity {
		return false
	}
	return s.childLogHandler.Enabled(ctx, l)
}

// Handle implements slog.Handler.
func (s *SeverityFilter) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < s.minSeverity {
		return nil
	}
	return s.childLogHandler.Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (s *SeverityFilter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SeverityFilter{
		childLogHandler: s.childLogHandler.WithAttrs(attrs),
		minSeverity:     s.minSeverity,
	}
}

// WithGroup implements slog.Handler.
func (s *SeverityFilter) WithGroup(name string) slog.Handler {
	return &SeverityFilter{
		childLogHandler: s.childLogHandler.WithGroup(name),
		minSeverity:     s.minSeverity,
	}
}

var _ slog.Handler = (*SeverityFilter)(nil)
