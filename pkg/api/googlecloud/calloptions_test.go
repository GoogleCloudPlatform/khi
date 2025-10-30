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
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRetryWithCountBudget(t *testing.T) {
	testCases := []struct {
		desc           string
		currentRetrier *retryWithCountBudget
		nextError      error
		wantDuration   time.Duration
		wantRetry      bool
	}{
		{
			desc: "retry budget not exceeded",
			currentRetrier: NewRetryWithCountBudget(
				[]codes.Code{codes.Internal},
				100*time.Millisecond,
				2,
				1*time.Minute,
				3,
				nil,
			),
			nextError:    status.Error(codes.Internal, "internal error"),
			wantDuration: 100 * time.Millisecond,
			wantRetry:    true,
		},
		{
			desc: "retry budget not exceeded but with multiple time",
			currentRetrier: func() *retryWithCountBudget {
				r := NewRetryWithCountBudget(
					[]codes.Code{codes.Internal},
					100*time.Millisecond,
					2,
					1*time.Minute,
					3,
					nil,
				)
				r.Retry(status.Error(codes.Internal, "internal error"))
				<-time.After(100 * time.Millisecond)
				r.Retry(status.Error(codes.Internal, "internal error"))
				<-time.After(100 * time.Millisecond)
				return r
			}(),
			nextError:    status.Error(codes.Internal, "internal error"),
			wantDuration: 400 * time.Millisecond,
			wantRetry:    true,
		},
		{
			desc: "retry budget exceeded",
			currentRetrier: func() *retryWithCountBudget {
				r := NewRetryWithCountBudget(
					[]codes.Code{codes.Internal},
					100*time.Millisecond,
					2.0,
					1*time.Minute,
					1,
					nil,
				)
				r.Retry(status.Error(codes.Internal, "internal error"))
				return r
			}(),
			nextError:    status.Error(codes.Internal, "internal error"),
			wantDuration: 0,
			wantRetry:    false,
		},
		{
			desc: "retry budget reset after duration",
			currentRetrier: func() *retryWithCountBudget {
				r := NewRetryWithCountBudget(
					[]codes.Code{codes.Internal},
					100*time.Millisecond,
					1.3,
					1*time.Millisecond, // Small reset duration for testing
					1,
					nil,
				)
				r.Retry(status.Error(codes.Internal, "internal error"))
				<-time.After(2 * time.Millisecond)
				return r
			}(),
			nextError:    status.Error(codes.Internal, "internal error"),
			wantDuration: 100 * time.Millisecond,
			wantRetry:    true,
		},
		{
			desc: "not retriable code",
			currentRetrier: NewRetryWithCountBudget(
				[]codes.Code{codes.Internal},
				100*time.Millisecond,
				1.3,
				1*time.Minute,
				3,
				nil,
			),
			nextError:    status.Error(codes.NotFound, "not found"),
			wantDuration: 0,
			wantRetry:    false,
		},
		{
			desc: "non-grpc error",
			currentRetrier: NewRetryWithCountBudget(
				[]codes.Code{codes.Internal},
				100*time.Millisecond,
				1.3,
				1*time.Minute,
				3,
				nil,
			),
			nextError:    fmt.Errorf("non gRPC error"),
			wantDuration: 0,
			wantRetry:    false,
		},
		{
			desc: "parent retrier handles error",
			currentRetrier: NewRetryWithCountBudget(
				[]codes.Code{codes.Internal},
				100*time.Millisecond,
				1.3,
				1*time.Minute,
				1,
				NewRetryWithCountBudget([]codes.Code{codes.Unavailable},
					50*time.Millisecond,
					2,
					1*time.Minute,
					1,
					nil,
				),
			),
			nextError:    status.Error(codes.Unavailable, "service unavailable"),
			wantDuration: 50 * time.Millisecond,
			wantRetry:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			gotDuration, gotRetry := tc.currentRetrier.Retry(tc.nextError)
			if gotDuration != tc.wantDuration {
				t.Errorf("Retry() didn't returned expected duration. got=%v,want=%v", gotDuration, tc.wantDuration)
			}
			if gotRetry != tc.wantRetry {
				t.Errorf("Retry() didn't returned expected retry flag. got=%v,want=%v", gotRetry, tc.wantRetry)
			}
		})
	}
}
