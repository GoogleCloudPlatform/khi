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
	"sync"
	"time"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
)

// ProgressUpdatorOnTickFunc is a function type that is called by ProgressUpdator
// on each tick to update the task progress.
type ProgressUpdatorOnTickFunc = func(tp *inspectionmetadata.TaskProgressMetadata)

// ProgressUpdator periodically updates a TaskProgress object at a specified
// interval. It uses a callback function to perform the update logic on each tick.
// A ProgressUpdator instance is intended for a single-use lifecycle and cannot be
// reused once started.
type ProgressUpdator struct {
	Progress *inspectionmetadata.TaskProgressMetadata
	Interval time.Duration
	OnTick   ProgressUpdatorOnTickFunc
	context  context.Context
	cancel   func()
	wg       sync.WaitGroup
}

// NewProgressUpdator creates and initializes a new ProgressUpdator.
func NewProgressUpdator(progress *inspectionmetadata.TaskProgressMetadata, interval time.Duration, onTick ProgressUpdatorOnTickFunc) *ProgressUpdator {
	return &ProgressUpdator{
		Progress: progress,
		Interval: interval,
		OnTick:   onTick,
	}
}

// Start begins the periodic updates. It invokes the OnTick callback immediately
// and then continues to call it at the specified interval until Done is called.
// It returns an error if the updator has already been started. A ProgressUpdator
// instance cannot be reused or restarted once started.
func (p *ProgressUpdator) Start(ctx context.Context) error {
	if p.context != nil {
		return fmt.Errorf("this updator is already started")
	}
	p.OnTick(p.Progress)
	cancellable, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.context = cancellable
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-cancellable.Done():
				return
			case <-ticker.C:
				select {
				case <-cancellable.Done():
					return
				default:
				}
				p.OnTick(p.Progress)
			}
		}
	}()
	return nil
}

// Done stops the periodic updates and waits for the worker goroutine to complete.
// It returns an error if the updator was not started. Calling Done multiple times
// is idempotent.
func (p *ProgressUpdator) Done() error {
	if p.context == nil {
		return fmt.Errorf("this updator is not yet started")
	}
	p.cancel()
	p.wg.Wait()
	return nil
}
