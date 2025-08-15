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

// ThrottleFilter is a slog.Handler that filters log records to prevent excessive
// logging of similar messages.
type ThrottleFilter struct {
	childLogHandler slog.Handler
	throttler       logThrottler
}

// NewThrottleFilter creates a new ThrottleFilter.
// It wraps a child handler and throttles logs based on the maxPerKind count.
func NewThrottleFilter(maxPerKind int, childHandler slog.Handler) *ThrottleFilter {
	return &ThrottleFilter{
		childHandler,
		newConstantCountLogThrottler(maxPerKind),
	}
}

// Enabled implements slog.Handler.
func (t *ThrottleFilter) Enabled(ctx context.Context, level slog.Level) bool {
	return t.childLogHandler.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (t *ThrottleFilter) Handle(ctx context.Context, r slog.Record) error {
	throttleStatus := t.throttler.ThrottleStatus(t.getLogKind(r))
	switch throttleStatus {
	case statusThrottled:
		return nil
	case statusJustBeforeThrottle:
		r = r.Clone()
		r.Message += "\n  (Similar logs shown for this task. Similar logs will be omitted from next.)"
		return t.childLogHandler.Handle(ctx, r)
	default:
		return t.childLogHandler.Handle(ctx, r)
	}
}

// WithAttrs implements slog.Handler.
func (t *ThrottleFilter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ThrottleFilter{
		t.childLogHandler.WithAttrs(attrs),
		t.throttler,
	}
}

// WithGroup implements slog.Handler.
func (t *ThrottleFilter) WithGroup(name string) slog.Handler {
	return &ThrottleFilter{
		t.childLogHandler.WithGroup(name),
		t.throttler,
	}
}

// getLogKind returns the log kind from attrs in slog.Record
func (t *ThrottleFilter) getLogKind(r slog.Record) string {
	kind := ""
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == LogKindAttrKey {
			kind = a.Value.String()
			return false
		}
		return true
	})
	return kind
}

var _ slog.Handler = (*ThrottleFilter)(nil)
