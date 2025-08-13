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

package progressutil

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
)

// IndeterminateUpdator updates a progress bar for a task that cannot report
// its progress as a percentage. It shows a message with an animated indicator
// to signify that the task is running.
type IndeterminateUpdator struct {
	Progress *progress.TaskProgress
	Interval time.Duration
	context  context.Context
	cancel   func()
}

// NewIndeterminateUpdator creates and initializes a new IndeterminateUpdator.
// It marks the associated TaskProgress as indeterminate.
func NewIndeterminateUpdator(progress *progress.TaskProgress, interval time.Duration) *IndeterminateUpdator {
	progress.Indeterminate = true
	return &IndeterminateUpdator{
		Progress: progress,
		Interval: interval,
	}
}

// Start starts updating progress bar.
// It returns an error if the updator is already started.
func (i *IndeterminateUpdator) Start(msg string) error {
	if i.context != nil {
		return fmt.Errorf("this updator is already used")
	}
	cancellable, cancel := context.WithCancel(context.Background())
	i.Progress.Message = msg
	i.context = cancellable
	i.cancel = cancel
	go func() {
		for itr := 1; true; itr++ {
			select {
			case <-i.context.Done():
				i.Progress.Indeterminate = false
				return
			case <-time.After(i.Interval):
				i.Progress.Message = fmt.Sprintf("%s%s", msg, i.workingIndicator(itr))
				itr++
			}
		}
	}()
	return nil
}

// Done stops updating progress bar.
// It returns an error if the updator is not yet started.
func (i *IndeterminateUpdator) Done() error {
	if i.context == nil {
		return fmt.Errorf("this updator is not yet started")
	}
	i.cancel()
	return nil
}

func (i *IndeterminateUpdator) workingIndicator(itr int) string {
	dots := ""
	for i := 0; i < itr%20; i++ {
		dots += "."
	}
	return dots
}
