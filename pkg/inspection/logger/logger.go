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
	"log/slog"
	"os"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/task/contextkey"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var globalLogHandler *globalLoggerHandler = nil

// A slog.Handler for KHI.
// This should route write requests to the corresponding slog.Handler.

type globalLoggerHandler struct {
	defaultHandler slog.Handler
	handlersLock   *sync.Mutex
	handlers       *map[string]slog.Handler
	attrs          []slog.Attr
	group          string
}

var _ slog.Handler = (*globalLoggerHandler)(nil)

// Enabled implements slog.Handler.
func (g *globalLoggerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return g.getHandler(ctx).Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (g *globalLoggerHandler) Handle(ctx context.Context, r slog.Record) error {
	return g.getHandler(ctx).Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (g *globalLoggerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &globalLoggerHandler{
		defaultHandler: g.defaultHandler,
		handlersLock:   g.handlersLock,
		handlers:       g.handlers,
		attrs:          append(append([]slog.Attr{}, g.attrs...), attrs...),
		group:          g.group,
	}
}

// WithGroup implements slog.Handler.
func (g *globalLoggerHandler) WithGroup(name string) slog.Handler {
	return &globalLoggerHandler{
		defaultHandler: g.defaultHandler,
		handlersLock:   g.handlersLock,
		handlers:       g.handlers,
		attrs:          g.attrs,
		group:          name,
	}
}

func (g *globalLoggerHandler) getHandler(ctx context.Context) slog.Handler {
	baseHandler := g.routeHandler(ctx)
	if g.group != "" {
		baseHandler = baseHandler.WithGroup(g.group)
	}
	if len(g.attrs) > 0 {
		baseHandler = baseHandler.WithAttrs(g.attrs)
	}
	return baseHandler
}

func (g *globalLoggerHandler) routeHandler(ctx context.Context) slog.Handler {
	tid, err := khictx.GetValue(ctx, task_contextkey.TaskImplementationIDContextKey)
	if err == nil {
		iid, err := khictx.GetValue(ctx, inspection_task_contextkey.InspectionTaskInspectionID)
		if err == nil {
			rid, err := khictx.GetValue(ctx, inspection_task_contextkey.InspectionTaskRunID)
			if err == nil {
				g.handlersLock.Lock()
				defer g.handlersLock.Unlock()
				loggerId := fmt.Sprintf("%s-%s-%s", iid, tid.String(), rid)
				if handler, found := (*g.handlers)[loggerId]; found {
					return handler
				} else {
					slog.Warn(fmt.Sprintf("No logger found for logger id:%s", loggerId))
					return g.defaultHandler
				}
			}
		}
	}
	return g.defaultHandler
}

func (g *globalLoggerHandler) RegisterTaskLogger(inspectionId string, taskId taskid.UntypedTaskImplementationID, runId string, handler slog.Handler) {
	g.handlersLock.Lock()
	defer g.handlersLock.Unlock()
	loggerId := fmt.Sprintf("%s-%s-%s", inspectionId, taskId.String(), runId)
	if _, found := (*g.handlers)[loggerId]; found {
		slog.Warn(fmt.Sprintf("duplicated logger found for %s. Ignoreing...", loggerId))
	} else {
		(*g.handlers)[loggerId] = handler
	}
}
func (g *globalLoggerHandler) UnregisterTaskLogger(inspectionId string, taskId taskid.UntypedTaskImplementationID, runId string, handler slog.Handler) {
	g.handlersLock.Lock()
	defer g.handlersLock.Unlock()
	loggerId := fmt.Sprintf("%s-%s-%s", inspectionId, taskId.String(), runId)
	delete((*g.handlers), loggerId)
}

func InitGlobalKHILogger() {
	globalLogHandler = localInitInspectionLogger(NewKHIFormatLogger(os.Stdout, true))
	slog.SetDefault(slog.New(globalLogHandler))
}

func localInitInspectionLogger(defaultHandler slog.Handler) *globalLoggerHandler {
	handler := &globalLoggerHandler{
		defaultHandler: defaultHandler,
		handlersLock:   &sync.Mutex{},
		handlers:       &map[string]slog.Handler{},
		attrs:          []slog.Attr{},
		group:          "",
	}
	return handler
}

func RegisterTaskLogger(inspectionId string, taskId taskid.UntypedTaskImplementationID, runId string, handler slog.Handler) {
	globalLogHandler.RegisterTaskLogger(inspectionId, taskId, runId, handler)
}
