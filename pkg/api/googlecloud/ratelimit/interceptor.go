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
	"fmt"
	"log/slog"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryClientInterceptor returns a grpc.UnaryClientInterceptor that throttles RPCs using an AdaptiveRateLimiter.
func UnaryClientInterceptor(limiter *AdaptiveRateLimiter) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			if status.Code(err) == codes.ResourceExhausted {
				prevRate := limiter.CurrentRate()
				limiter.OnResourceExhausted()
				newRate := limiter.CurrentRate()
				slog.DebugContext(ctx, fmt.Sprintf("rate limit encountered for method %s, decreased rate from %.2f QPS to %.2f QPS", method, prevRate, newRate))
			}
			return err
		}

		limiter.OnSuccess()
		return nil
	}
}

// PoolUnaryClientInterceptor returns a grpc.UnaryClientInterceptor that resolves an AdaptiveRateLimiter from the pool based on the ResourceContainer.
func PoolUnaryClientInterceptor(pool *Pool, container googlecloud.ResourceContainer) grpc.UnaryClientInterceptor {
	limiter := pool.GetLimiter(container)
	return UnaryClientInterceptor(limiter)
}
