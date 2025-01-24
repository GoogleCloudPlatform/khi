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
	"fmt"
	"io"
	"log/slog"

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var reset = "\033[0m"
var colors = []string{
	"\033[31m",
	"\033[32m",
	"\033[33m",
	"\033[34m",
	"\033[35m",
	"\033[36m",
	"\033[91m",
	"\033[92m",
	"\033[93m",
	"\033[94m",
	"\033[95m",
	"\033[96m",
}

type KHILogFormatHandler struct {
	out       io.Writer
	withColor bool
	attrs     []slog.Attr
}

// Enabled implements slog.Handler.
func (*KHILogFormatHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

// Handle implements slog.Handler.
func (lh *KHILogFormatHandler) Handle(ctx context.Context, r slog.Record) error {
	tidAny := ctx.Value("tid")
	var logLine string
	if tid, convertible := tidAny.(taskid.TaskImplementationId); convertible {
		logLine = fmt.Sprintf("%s%s >%s %s %s\n", lh.taskIdToColor(tid.String()), tid, lh.resetColor(), lh.wrapColorByLevel(r.Level, r.Level.String()), lh.wrapColorByLevel(r.Level, r.Message))
	} else {
		logLine = fmt.Sprintf("global > %s %s\n", lh.wrapColorByLevel(r.Level, r.Level.String()), lh.wrapColorByLevel(r.Level, r.Message))
	}
	lh.out.Write([]byte(logLine))
	return nil
}

// WithAttrs implements slog.Handler.
func (lh *KHILogFormatHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &KHILogFormatHandler{
		out:       lh.out,
		attrs:     attrs,
		withColor: lh.withColor,
	}
}

// WithGroup implements slog.Handler.
func (lh *KHILogFormatHandler) WithGroup(name string) slog.Handler {
	return lh // this is not supporting group
}

func (lh *KHILogFormatHandler) taskIdToColor(tid string) string {
	if !lh.withColor {
		return ""
	}
	sum := 0
	for c := range tid {
		sum = (sum + int(c)) % len(colors)
	}
	return colors[sum]
}

func (lh *KHILogFormatHandler) resetColor() string {
	if lh.withColor {
		return reset
	} else {
		return ""
	}
}

func (lh *KHILogFormatHandler) wrapColorByLevel(level slog.Level, msg string) string {
	if !lh.withColor {
		return msg
	}
	var colorBegin string
	if level.Level() == slog.LevelDebug {
		colorBegin = "\033[90m" //bright black
	}
	if level.Level() == slog.LevelInfo {
		colorBegin = "\033[96m" //bright cyan
	}
	if level.Level() == slog.LevelWarn {
		colorBegin = "\033[93m" //bright yellow
	}
	if level.Level() == slog.LevelError {
		colorBegin = "\033[97;101m" //bright yellow
	}
	return fmt.Sprintf("%s%s%s", colorBegin, msg, reset)
}

var _ slog.Handler = (*KHILogFormatHandler)(nil)

func NewKHIFormatLogger(out io.Writer, withColor bool) *KHILogFormatHandler {
	return &KHILogFormatHandler{
		out:       out,
		withColor: withColor,
	}
}
