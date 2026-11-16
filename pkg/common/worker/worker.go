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

package worker

import (
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/errorreport"
)

// Pool enables running a goroutine with max parallel count limit.
type Pool struct {
	semaphore chan struct{}
	waitGroup *sync.WaitGroup
}

// NewPool creates a Pool allowing up to maxParallelCount concurrent goroutines.
func NewPool(maxParallelCount int) *Pool {
	return &Pool{
		semaphore: make(chan struct{}, maxParallelCount),
		waitGroup: &sync.WaitGroup{},
	}
}

// Run schedules f to execute in the pool, blocking if maxParallelCount goroutines are already running.
func (t *Pool) Run(f func()) {
	t.waitGroup.Add(1)
	t.semaphore <- struct{}{}
	go func() {
		defer errorreport.CheckAndReportPanic()
		defer func() {
			<-t.semaphore
			t.waitGroup.Done()
		}()
		f()
	}()
}

// Wait blocks until all goroutines started via Run have completed.
func (t *Pool) Wait() {
	t.waitGroup.Wait()
}
