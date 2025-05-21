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

package componentparser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
)

var ErrParserNoMatchingWithLog = errors.New("Parser didn't match with the given log")

type SchedulerComponentParser struct{}

// Process implements ControlPlaneComponentParser.
func (s *SchedulerComponentParser) Process(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) (bool, error) {
	path, err := s.podRelatedLogsToResourcePath(ctx, l)
	if err == nil {
		cs.RecordEvent(path)
	}
	return true, nil
}

// ShouldProcess implements ControlPlaneComponentParser.
func (s *SchedulerComponentParser) ShouldProcess(component_name string) bool {
	return component_name == "scheduler"
}

func (s *SchedulerComponentParser) podRelatedLogsToResourcePath(ctx context.Context, l *log.Log) (resourcepath.ResourcePath, error) {
	mainMessageFieldSet := log.MustGetFieldSet(l, &log.MainMessageFieldSet{})
	hasPodField := mainMessageFieldSet.HasKLogField("pod")
	if hasPodField {
		pod, err := mainMessageFieldSet.KLogField("pod")
		if err != nil {
			return resourcepath.ResourcePath{}, ErrParserNoMatchingWithLog
		}
		splittedPodName := strings.Split(pod, "/")
		if len(splittedPodName) != 2 {
			slog.WarnContext(ctx, fmt.Sprintf("Unexpected pod klog format: %s", pod))
			return resourcepath.ResourcePath{}, ErrParserNoMatchingWithLog
		}
		return resourcepath.Pod(splittedPodName[0], splittedPodName[1]), nil
	}
	return resourcepath.ResourcePath{}, ErrParserNoMatchingWithLog
}

var _ ControlPlaneComponentParser = (*SchedulerComponentParser)(nil)
