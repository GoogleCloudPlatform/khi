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

package streamingutil

import (
	"context"
	"errors"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/ttlcleaner"
)

var jobIDGenerator = idgenerator.NewPrefixIDGenerator("job-")

// JobRunner defines the signature of a long-running operation that reports progress and returns a result.
type JobRunner[P any, R any] func(ctx context.Context, onProgress func(P) error) (R, error)

// Job represents an asynchronous in-flight or completed operation.
type Job[P any, R any] struct {
	ID             string
	mu             sync.RWMutex
	cancel         context.CancelFunc
	latestProgress P
	result         R
	err            error
	done           chan struct{}
	lastPollAt     time.Time
	expiresAt      time.Time
	hasProgress    bool
	isDone         bool
}

// AsyncJobManager manages execution, tracking, and polling for background jobs.
type AsyncJobManager[P any, R any] struct {
	mu             sync.Mutex
	jobs           map[string]*Job[P, R]
	currentJobID   string
	abandonTimeout time.Duration
	retentionTTL   time.Duration
	cleaner        *ttlcleaner.TTLCleaner[string]
}

var _ ttlcleaner.ExpirableTarget[string] = (*AsyncJobManager[any, any])(nil)

// Expirations returns a snapshot of active job expiration timestamps.
// If an in-progress job hasn't received any poll requests within abandonTimeout, its context is canceled.
func (m *AsyncJobManager[P, R]) Expirations() map[string]time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make(map[string]time.Time, len(m.jobs))
	now := time.Now()
	for id, job := range m.jobs {
		job.mu.RLock()
		if !job.isDone && now.Sub(job.lastPollAt) > m.abandonTimeout {
			job.cancel()
		}
		res[id] = job.expiresAt
		job.mu.RUnlock()
	}
	return res
}

// Evict removes and cancels an expired job from the manager.
func (m *AsyncJobManager[P, R]) Evict(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job, ok := m.jobs[key]; ok {
		job.cancel()
		delete(m.jobs, key)
	}
	return nil
}

// NewAsyncJobManager creates a new initialized AsyncJobManager.
func NewAsyncJobManager[P any, R any](abandonTimeout time.Duration, retentionTTL time.Duration) *AsyncJobManager[P, R] {
	m := &AsyncJobManager[P, R]{
		jobs:           make(map[string]*Job[P, R]),
		abandonTimeout: abandonTimeout,
		retentionTTL:   retentionTTL,
	}
	m.cleaner = ttlcleaner.NewTTLCleaner[string](m, 10*time.Second)
	m.cleaner.Start()
	return m
}

// Cancel terminates a running job if it exists and returns true if canceled.
func (m *AsyncJobManager[P, R]) Cancel(jobID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job, ok := m.jobs[jobID]; ok {
		job.cancel()
		return true
	}
	return false
}

// Close stops the background cleaner and cancels all active jobs.
func (m *AsyncJobManager[P, R]) Close() {
	m.cleaner.Stop()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, job := range m.jobs {
		job.cancel()
	}
	m.jobs = make(map[string]*Job[P, R])
}

// StreamJob executes a JobRunner synchronously within a streaming RPC, forwarding progress and the final result.
func StreamJob[P any, R any, Resp any](
	ctx context.Context,
	stream *connect.ServerStream[Resp],
	runner JobRunner[P, R],
	mapProgress func(P) *Resp,
	mapResult func(R) *Resp,
) error {
	res, err := runner(ctx, func(progress P) error {
		return stream.Send(mapProgress(progress))
	})
	if err != nil {
		return err
	}
	return stream.Send(mapResult(res))
}

// PollStatus represents the output returned to a polling RPC caller.
type PollStatus[P any, R any] struct {
	JobID       string
	IsDone      bool
	Progress    P
	HasProgress bool
	Result      R
	Err         error
}

// Poll handles starting a new job or retrieving the status of an existing job.
func (m *AsyncJobManager[P, R]) Poll(
	ctx context.Context,
	jobID string,
	fastWait time.Duration,
	runner JobRunner[P, R],
) (*PollStatus[P, R], error) {
	now := time.Now()
	m.mu.Lock()
	if jobID == "" {
		if prev, ok := m.jobs[m.currentJobID]; ok {
			prev.mu.RLock()
			isDone := prev.isDone
			prev.mu.RUnlock()
			if !isDone {
				prev.cancel()
			}
		}

		newID := jobIDGenerator.Generate()
		jobCtx, cancel := context.WithCancel(context.Background())
		job := &Job[P, R]{
			ID:         newID,
			cancel:     cancel,
			done:       make(chan struct{}),
			lastPollAt: now,
			expiresAt:  now.Add(m.abandonTimeout + m.retentionTTL),
		}
		m.jobs[newID] = job
		m.currentJobID = newID
		m.mu.Unlock()

		go func() {
			defer close(job.done)
			res, err := runner(jobCtx, func(p P) error {
				job.mu.Lock()
				job.latestProgress = p
				job.hasProgress = true
				job.mu.Unlock()
				return nil
			})
			job.mu.Lock()
			job.isDone = true
			job.result = res
			job.err = err
			job.expiresAt = time.Now().Add(m.retentionTTL)
			job.mu.Unlock()
		}()

		select {
		case <-job.done:
			return &PollStatus[P, R]{
				JobID:  newID,
				IsDone: true,
				Result: job.result,
				Err:    job.err,
			}, nil
		case <-time.After(fastWait):
			job.mu.RLock()
			defer job.mu.RUnlock()
			return &PollStatus[P, R]{
				JobID:       newID,
				IsDone:      false,
				Progress:    job.latestProgress,
				HasProgress: job.hasProgress,
			}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	job, exists := m.jobs[jobID]
	if !exists {
		m.mu.Unlock()
		return nil, connect.NewError(connect.CodeNotFound, errors.New("job not found or expired"))
	}
	m.mu.Unlock()

	job.mu.Lock()
	job.lastPollAt = now
	if !job.isDone {
		job.expiresAt = now.Add(m.abandonTimeout + m.retentionTTL)
	}
	job.mu.Unlock()

	job.mu.RLock()
	defer job.mu.RUnlock()
	return &PollStatus[P, R]{
		JobID:       jobID,
		IsDone:      job.isDone,
		Progress:    job.latestProgress,
		HasProgress: job.hasProgress,
		Result:      job.result,
		Err:         job.err,
	}, nil
}
