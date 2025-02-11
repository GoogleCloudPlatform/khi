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

package parser

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/errorreport"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"golang.org/x/sync/errgroup"
)

var PARSER_MAX_THREADS = 16

// Parser is common interfaces across all of log parsers in KHI
type Parser interface {
	// GetParserName Returns it's own parser name. It must be unique by each instances.
	GetParserName() string

	// TargetLogType returns the log type which this parser should mainly parse and generate revisions or events for.
	TargetLogType() enum.LogType

	// Parse a log. Return an error to decide skip to parse the log and delegate later parsers.
	Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder, variables *task.VariableSet) error

	// Description returns comprehensive description of the parser.
	// Parser tasks are registered as a `feature task` and the description is shown on the frontend.
	Description() string

	// LogTask returns the task Id generating []*log.LogEntity
	LogTask() string

	// Dependencies returns the list of task Ids excluding the log task
	Dependencies() []string

	// Grouper returns LogGrouper that groups logs into multiple sets. These sets are sorted individually and parsed in parallel, then merged later.
	Grouper() grouper.LogGrouper
}

func NewParserTaskFromParser(taskId string, parser Parser, isDefaultFeature bool, labelOpts ...task.LabelOpt) task.Definition {
	return inspection_task.NewInspectionProcessor(taskId, append(parser.Dependencies(), parser.LogTask(), inspection_task.BuilderGeneratorTaskID), func(ctx context.Context, taskMode int, v *task.VariableSet, tp *progress.TaskProgress) (any, error) {
		if taskMode == inspection_task.TaskModeDryRun {
			slog.DebugContext(ctx, "Skipping task because this is dry run mode")
			return struct{}{}, nil
		}
		builder, err := inspection_task.GetHistoryBuilderFromTaskVariable(v)
		if err != nil {
			return nil, err
		}
		logs, err := task.GetTypedVariableFromTaskVariable[[]*log.LogEntity](v, parser.LogTask(), nil)
		if err != nil {
			return nil, err
		}
		preparedLogCount := atomic.Int32{}
		updator := progress.NewProgressUpdator(tp, time.Second, func(tp *progress.TaskProgress) {
			current := preparedLogCount.Load()
			tp.Percentage = float32(current) / float32(len(logs))
			tp.Message = fmt.Sprintf("%d/%d", current, len(logs))
		})
		updator.Start(ctx)
		err = builder.PrepareParseLogs(ctx, logs, func() {
			preparedLogCount.Add(1)
		})
		if err != nil {
			return nil, err
		}
		grouper := parser.Grouper()
		groups := grouper.Group(logs)
		groupNames := []string{}
		for key := range groups {
			groupNames = append(groupNames, key)
		}
		limitChannel := make(chan struct{}, PARSER_MAX_THREADS)
		logCounterChannel := make(chan struct{})
		currentGroup := 0
		wg := errgroup.Group{}
		parserStarted := false
		threadCount := 0
		doneThreadCount := atomic.Int32{}

		// gorutine to update progress of parseing logs.
		go func() {
			parsedLogCount := 0
			cancellable, cancel := context.WithCancel(ctx)
			defer cancel()
			go func() {
				lastLogCount := 0
				for {
					select {
					case <-cancellable.Done():
						cancel()
						return
					case <-time.After(time.Second):
						if !parserStarted {
							updator.Done()
							parserStarted = true
						}
						tp.Update(float32(parsedLogCount)/float32(len(logs)), fmt.Sprintf("%d lps(concurrency %d/%d)", parsedLogCount-lastLogCount, doneThreadCount.Load(), threadCount))
						lastLogCount = parsedLogCount
					}
				}
			}()
			for {
				select {
				case <-logCounterChannel:
					parsedLogCount += 1
				case <-ctx.Done():
					return
				}
			}
		}()

		for {
			if currentGroup >= len(groupNames) {
				break
			}
			select {
			case <-ctx.Done():
				close(limitChannel)
			default:
				limitChannel <- struct{}{}
				groupedLogs := groups[groupNames[currentGroup]]
				threadCount += 1
				wg.Go(func() error { // TODO: replace this with pkg/common/worker/pool
					defer errorreport.CheckAndReportPanic()
					err = builder.ParseLogsByGroups(ctx, groupedLogs, func(logIndex int, l *log.LogEntity) *history.ChangeSet {
						cs := history.NewChangeSet(l)
						err := parser.Parse(ctx, l, cs, builder, v)
						logCounterChannel <- struct{}{}
						if err != nil {
							yaml, err2 := l.Fields.ToYaml("")
							if err2 != nil {
								yaml = "ERROR!! failed to dump in yaml"
							}
							slog.WarnContext(ctx, fmt.Sprintf("parser end with an error\n%s", err))
							slog.DebugContext(ctx, yaml)
							return nil
						}
						return cs
					})
					<-limitChannel
					return err
				})
				currentGroup += 1
				doneThreadCount.Add(1)
			}
		}
		err = wg.Wait()
		if err != nil {
			return nil, err
		}
		close(logCounterChannel)
		return struct{}{}, nil
	},
		append([]task.LabelOpt{
			inspection_task.FeatureTaskLabel(parser.GetParserName(), parser.Description(), parser.TargetLogType(), isDefaultFeature),
		}, labelOpts...)...)
}
