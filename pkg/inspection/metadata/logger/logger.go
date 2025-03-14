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
	"os"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var LoggerMetadataKey = metadata.NewMetadataKey[*Logger]("log")

// similarLogThrottlingLogCount is the count of similar logs to start preventing output.
var similarLogThrottlingLogCount = 10

var _ slog.Handler = (*TaskSlogHandler)(nil)

type TaskSlogHandler struct {
	enableStdout  bool
	stdoutHandler slog.Handler
	stringHandler slog.Handler
	minLogLevel   slog.Level
	throttle      LogThrottler
}

type SerializableLogItem struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Log  string `json:"log"`
}

// Enabled implements slog.Handler.
func (t *TaskSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if level < t.minLogLevel {
		return false
	}
	if !t.enableStdout {
		return t.stringHandler.Enabled(ctx, level)
	}
	return t.stdoutHandler.Enabled(ctx, level) || t.stringHandler.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (t *TaskSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	throttleStatus := t.throttle.ThrottleStatus(t.getLogKind(r))
	if throttleStatus == StatusThrottled {
		return nil
	}
	if throttleStatus == StatusJustBeforeThrottle {
		r = r.Clone()
		r.Message += "\n  (Over 10 similar logs shown for this task. Similar logs will be omitted from next.)"
	}
	if t.enableStdout {
		t.stdoutHandler.Handle(ctx, r)
	}
	if r.Level >= slog.LevelInfo {
		// store string log only for >= info logs.
		return t.stringHandler.Handle(ctx, r)
	}
	return nil
}

// WithAttrs implements slog.Handler.
func (t *TaskSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TaskSlogHandler{
		minLogLevel:   t.minLogLevel,
		enableStdout:  t.enableStdout,
		stdoutHandler: t.stdoutHandler.WithAttrs(attrs),
		stringHandler: t.stringHandler.WithAttrs(attrs),
		throttle:      t.throttle,
	}
}

// WithGroup implements slog.Handler.
func (t *TaskSlogHandler) WithGroup(name string) slog.Handler {
	return &TaskSlogHandler{
		minLogLevel:   t.minLogLevel,
		enableStdout:  t.enableStdout,
		stdoutHandler: t.stdoutHandler.WithGroup(name),
		stringHandler: t.stringHandler.WithGroup(name),
		throttle:      t.throttle,
	}
}

// getLogKind returns the log kind from attrs in slog.Record
func (t *TaskSlogHandler) getLogKind(r slog.Record) string {
	kind := ""
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == logger.LogKindAttrKey {
			kind = a.Value.String()
			return false
		}
		return true
	})
	return kind
}

type TaskLogger struct {
	id         string
	name       string
	logHandler slog.Handler
	logBuffer  *bytes.Buffer
}

func (t *TaskLogger) Read() string {
	return t.logBuffer.String()
}

func (t *TaskLogger) AsSerializableLogItem() *SerializableLogItem {
	return &SerializableLogItem{
		Id:   t.id,
		Name: t.name,
		Log:  t.Read(),
	}
}

type Logger struct {
	loggers []*TaskLogger
}

var _ metadata.Metadata = (*Logger)(nil)

// Labels implements metadata.Metadata.
func (*Logger) Labels() *typedmap.ReadonlyTypedMap {
	return task.NewLabelSet(
		metadata.IncludeInRunResult(),
	)
}

// ToSerializable implements metadata.Metadata.
func (l *Logger) ToSerializable() interface{} {
	result := make([]*SerializableLogItem, 0)
	for _, l := range l.loggers {
		result = append(result, l.AsSerializableLogItem())
	}
	return result
}

func (l *Logger) MakeTaskLogger(ctx context.Context, minLevel slog.Level) *TaskLogger {
	stdoutWithColor := true
	if parameters.Debug.NoColor != nil && *parameters.Debug.NoColor {
		stdoutWithColor = false
	}
	iidAny := ctx.Value("iid")
	if iid, convertible := iidAny.(string); convertible {
		tidAny := ctx.Value("tid")
		if tid, convertible := tidAny.(taskid.TaskImplementationId); convertible {
			ridAny := ctx.Value("rid")
			if rid, convertible := ridAny.(string); convertible {
				lb := new(bytes.Buffer)
				th := &TaskSlogHandler{
					minLogLevel:   minLevel,
					enableStdout:  true,
					stdoutHandler: logger.NewKHIFormatLogger(os.Stdout, stdoutWithColor),
					stringHandler: logger.NewKHIFormatLogger(lb, false),
					throttle:      NewConstantLogThrottle(similarLogThrottlingLogCount),
				}
				tl := &TaskLogger{
					id:         tid.String(),
					name:       tid.String(),
					logHandler: th,
					logBuffer:  lb,
				}
				logger.RegisterTaskLogger(iid, tid.String(), rid, th)
				l.loggers = append(l.loggers, tl)
				return tl
			} else {
				slog.Error("given context is not associated with any run id")
			}
		} else {
			slog.Error("given context is not associated with any task id")
		}
	} else {
		slog.Error("given context is not associated with any inspection id")
	}
	return nil
}

func NewLogger() *Logger {
	return &Logger{
		loggers: make([]*TaskLogger, 0),
	}
}
