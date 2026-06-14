// Copyright 2025 Google LLC
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

package googlecloud

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DefaultRetryPolicy is the default retry policy widely used to access Google Cloud API.
var DefaultRetryPolicy = gax.WithRetry(NewDefaultRetryer)

// NewDefaultRetryer returns the default retryer.
func NewDefaultRetryer() gax.Retryer {
	return NewDefaultRetryerWithContext(context.Background())
}

// NewDefaultRetryerWithContext returns the default retryer with context.
func NewDefaultRetryerWithContext(ctx context.Context) gax.Retryer {
	r := gax.OnCodes([]codes.Code{
		codes.Aborted,
		codes.Canceled,
		codes.Internal,
		codes.ResourceExhausted,
		codes.Unknown,
		codes.Unavailable,
		codes.DeadlineExceeded,
		codes.Unauthenticated, // Non per-RPC cred failures are not retried, see defaultRetryer.Retry.
	}, gax.Backoff{
		Initial:    100 * time.Millisecond,
		Max:        5000 * time.Millisecond,
		Multiplier: 1.30,
	})
	return &defaultRetryer{Retryer: r, ctx: ctx}
}

// defaultRetryer wraps a gax.Retryer to customize the retry behavior for specific codes like Unauthenticated.
type defaultRetryer struct {
	gax.Retryer
	ctx context.Context
}

var _ gax.Retryer = (*defaultRetryer)(nil)

// Retry overrides the underlying gax.Retryer's Retry method to filter Unauthenticated errors.
func (r *defaultRetryer) Retry(err error) (time.Duration, bool) {
	// Temporal issue on Enterprise Certificate Proxy may result in Unauthenticated errors.
	// We retry on these errors since the root cause is likely a transient issue with ECP.
	if statusCode := status.Code(err); statusCode == codes.Unauthenticated {
		s, ok := status.FromError(err)
		if !ok || !strings.Contains(s.Message(), `per-RPC creds failed due to error:`) {
			return 0, false
		}
	}
	pause, shouldRetry := r.Retryer.Retry(err)
	if shouldRetry && r.ctx != nil {
		slog.DebugContext(r.ctx, "retrying Google Cloud API call due to transient error", "error", err, "pause", pause)
	}
	return pause, shouldRetry
}

// NeverTimeout is gax.CallOption that never reaches the timeout.
var NeverTimeout = gax.WithTimeout(1<<63 - 1)

// retryWithCountBudget retries for specific error codes but with count limit.
type retryWithCountBudget struct {
	gax.Retryer
	codes            []codes.Code
	initialDuration  time.Duration
	multiplier       float32
	resetDuration    time.Duration
	retryCount       int
	retryCountBudget int
	lastRetry        time.Time
}

// NewRetryWithCountBudget returns the instance of retryWithCountBudget.
func NewRetryWithCountBudget(codes []codes.Code, initialDuration time.Duration, multiplier float32, resetDuration time.Duration, retryCountBudget int, parentRetrier gax.Retryer) *retryWithCountBudget {
	return &retryWithCountBudget{
		Retryer:          parentRetrier,
		codes:            codes,
		initialDuration:  initialDuration,
		multiplier:       multiplier,
		resetDuration:    resetDuration,
		retryCount:       0,
		retryCountBudget: retryCountBudget,
		lastRetry:        time.Time{},
	}
}

// Retry implements gax.Retryer.
func (r *retryWithCountBudget) Retry(err error) (pause time.Duration, shouldRetry bool) {
	s, ok := status.FromError(err)
	if !ok {
		return 0, false
	}
	for _, code := range r.codes {
		if s.Code() == code {
			now := time.Now()
			if now.Sub(r.lastRetry) > r.resetDuration {
				r.retryCount = 0
			}
			r.lastRetry = now
			r.retryCount++
			if r.retryCount > r.retryCountBudget {
				return 0, false
			}
			return time.Duration(math.Pow(float64(r.multiplier), float64(r.retryCount-1)) * float64(r.initialDuration)), true
		}
	}
	if r.Retryer != nil {
		return r.Retryer.Retry(err)
	} else {
		return 0, false
	}
}

var _ gax.Retryer = (*retryWithCountBudget)(nil)
