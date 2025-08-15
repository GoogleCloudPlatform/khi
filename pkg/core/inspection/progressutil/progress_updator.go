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

// ProgressUpdatorOnTickFunc is a function type that is called by ProgressUpdator
// on each tick to update the task progress.
type ProgressUpdatorOnTickFunc = func(tp *progress.TaskProgress)

// ProgressUpdator periodically updates a TaskProgress object at a specified
// interval. It uses a callback function to perform the update logic on each tick.
type ProgressUpdator struct {
	Progress *progress.TaskProgress
	Interval time.Duration
	OnTick   ProgressUpdatorOnTickFunc
	context  context.Context
	cancel   func()
}

// NewProgressUpdator creates and initializes a new ProgressUpdator.
func NewProgressUpdator(progress *progress.TaskProgress, interval time.Duration, onTick ProgressUpdatorOnTickFunc) *ProgressUpdator {
	return &ProgressUpdator{
		Progress: progress,
		Interval: interval,
		OnTick:   onTick,
	}
}

// Start begins the periodic updates. It invokes the OnTick callback immediately
// and then continues to call it at the specified interval until Done is called.
// It returns an error if the updator has already been started.
func (p *ProgressUpdator) Start(ctx context.Context) error {
	p.OnTick(p.Progress)
	cancellable, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.context = cancellable
	go func() {
		for itr := 1; true; itr++ {
			select {
			case <-p.context.Done():
				return
			case <-time.After(p.Interval):
				p.OnTick(p.Progress)
				itr++
			}
		}
	}()
	return nil
}

// Done stops the periodic updates.
// It returns an error if the updator was not started.
func (p *ProgressUpdator) Done() error {
	if p.context == nil {
		return fmt.Errorf("this updator is not yet started")
	}
	p.cancel()
	return nil
}
