// Copyright 2026 Google LLC
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
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// ProgressCallback receives streaming progress percentage (in [0.0, 1.0] or a mapped range) and a message.
type ProgressCallback func(progressPercentage float64, message string) error

// ProgressOptions defines configuration for progress reporting in parallel operations.
type ProgressOptions struct {
	// Interval is the interval between periodic progress updates. Defaults to 1s if <= 0.
	Interval time.Duration
	// MessageFmt is the template for progress messages (e.g. "Processing (%d/%d)...").
	MessageFmt string
	// MinProgress is the starting progress percentage for this step (e.g. 0.0).
	MinProgress float64
	// MaxProgress is the ending progress percentage for this step (e.g. 1.0).
	MaxProgress float64
	// NumWorkers overrides the number of parallel workers. Defaults to GOMAXPROCS if <= 0.
	NumWorkers int
}

// ParallelChunkMap partitions the given slice into chunks across worker goroutines,
// executes workerFunc concurrently with errorgroup, and periodically reports progress.
func ParallelChunkMap[T any, R any](
	ctx context.Context,
	items []T,
	workerFunc func(ctx context.Context, workerIdx int, chunk []T, onProcessed func(int)) (R, error),
	onProgress ProgressCallback,
	opts ProgressOptions,
) ([]R, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	numWorkers := opts.NumWorkers
	if numWorkers <= 0 {
		numWorkers = runtime.GOMAXPROCS(0)
	}
	if numWorkers > len(items) {
		numWorkers = len(items)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	interval := opts.Interval
	if interval <= 0 {
		interval = time.Second
	}

	minProg := opts.MinProgress
	maxProg := opts.MaxProgress
	if maxProg <= minProg {
		maxProg = 1.0
	}

	totalItems := int64(len(items))
	var processedCount atomic.Int64

	formatProgress := func(done int64) (float64, string) {
		fraction := float64(done) / float64(totalItems)
		if fraction > 1.0 {
			fraction = 1.0
		}
		pct := minProg + fraction*(maxProg-minProg)
		var msg string
		if opts.MessageFmt != "" {
			if strings.Count(opts.MessageFmt, "%d") >= 2 {
				msg = fmt.Sprintf(opts.MessageFmt, done, totalItems)
			} else {
				msg = opts.MessageFmt
			}
		}
		return pct, msg
	}

	var stopReporter func()
	if onProgress != nil {
		stopCh := make(chan struct{})
		doneCh := make(chan struct{})

		go func() {
			defer close(doneCh)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-stopCh:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					pct, msg := formatProgress(processedCount.Load())
					_ = onProgress(pct, msg)
				}
			}
		}()

		stopReporter = func() {
			close(stopCh)
			<-doneCh
			pct, msg := formatProgress(processedCount.Load())
			_ = onProgress(pct, msg)
		}
	} else {
		stopReporter = func() {}
	}

	chunkSize := (len(items) + numWorkers - 1) / numWorkers
	if actualWorkers := (len(items) + chunkSize - 1) / chunkSize; actualWorkers < numWorkers {
		numWorkers = actualWorkers
	}
	results := make([]R, numWorkers)

	g, gCtx := errgroup.WithContext(ctx)
	for w := 0; w < numWorkers; w++ {
		workerIdx := w
		start := workerIdx * chunkSize
		end := start + chunkSize
		if end > len(items) {
			end = len(items)
		}
		if start >= end {
			continue
		}

		chunk := items[start:end]
		g.Go(func() error {
			res, err := workerFunc(gCtx, workerIdx, chunk, func(n int) {
				processedCount.Add(int64(n))
			})
			if err != nil {
				return err
			}
			results[workerIdx] = res
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		stopReporter()
		return nil, err
	}
	stopReporter()

	return results, nil
}
