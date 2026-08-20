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

package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryClientInterceptor(t *testing.T) {
	testCases := []struct {
		name          string
		initialQPS    float64
		invokerErr    error
		cancelCtx     bool
		wantErr       bool
		wantInvoked   bool
		wantFinalRate float64
	}{
		{
			name:          "success invokes call and increases rate",
			initialQPS:    1.0,
			invokerErr:    nil,
			cancelCtx:     false,
			wantErr:       false,
			wantInvoked:   true,
			wantFinalRate: 1.5, // 1.0 + 0.5
		},
		{
			name:          "resource exhausted decreases rate",
			initialQPS:    10.0,
			invokerErr:    status.Error(codes.ResourceExhausted, "quota exceeded"),
			cancelCtx:     false,
			wantErr:       true,
			wantInvoked:   true,
			wantFinalRate: 5.0, // 10.0 * 0.5
		},
		{
			name:          "other error propagates without rate modification",
			initialQPS:    10.0,
			invokerErr:    status.Error(codes.Internal, "internal server error"),
			cancelCtx:     false,
			wantErr:       true,
			wantInvoked:   true,
			wantFinalRate: 10.0,
		},
		{
			name:          "cancelled context aborts before invoker",
			initialQPS:    10.0,
			invokerErr:    nil,
			cancelCtx:     true,
			wantErr:       true,
			wantInvoked:   false,
			wantFinalRate: 10.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewAdaptiveRateLimiter(AdaptiveRateLimiterConfig{
				InitialQPS:       tc.initialQPS,
				MinQPS:           1.0,
				MaxQPS:           100.0,
				IncreaseStep:     0.5,
				DecreaseFactor:   0.5,
				DecreaseCooldown: 1500 * time.Millisecond,
				Burst:            1,
			})

			interceptor := UnaryClientInterceptor(limiter)

			ctx, cancel := context.WithCancel(context.Background())
			if tc.cancelCtx {
				cancel()
			} else {
				defer cancel()
			}

			invoked := false
			invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				invoked = true
				return tc.invokerErr
			}

			err := interceptor(ctx, "/google.logging.v2.LoggingServiceV2/ListLogEntries", nil, nil, nil, invoker)
			if (err != nil) != tc.wantErr {
				t.Errorf("interceptor() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if invoked != tc.wantInvoked {
				t.Errorf("invoker called = %v, wantInvoked = %v", invoked, tc.wantInvoked)
			}
			if diff := cmp.Diff(tc.wantFinalRate, limiter.CurrentRate()); diff != "" {
				t.Errorf("CurrentRate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUnaryClientInterceptor_NonStatusError(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(AdaptiveRateLimiterConfig{
		InitialQPS: 10.0,
		MinQPS:     0.1,
		MaxQPS:     100.0,
	})
	interceptor := UnaryClientInterceptor(limiter)

	customErr := errors.New("network failure")
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return customErr
	}

	err := interceptor(context.Background(), "/test", nil, nil, nil, invoker)
	if !errors.Is(err, customErr) {
		t.Errorf("interceptor() error = %v, want %v", err, customErr)
	}
	if diff := cmp.Diff(10.0, limiter.CurrentRate()); diff != "" {
		t.Errorf("CurrentRate() mismatch (-want +got):\n%s", diff)
	}
}
