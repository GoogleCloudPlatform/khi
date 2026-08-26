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
	"time"

	"connectrpc.com/connect"
)

// SnapshotProvider returns the current state snapshot.
type SnapshotProvider[T any] func(ctx context.Context) (T, error)

// SubscriptionProvider returns a channel for real-time events and a cleanup function.
type SubscriptionProvider[T any] func(ctx context.Context) (<-chan T, func())

// StreamWatch executes a standard stream cycle (up to cycleDuration).
// It streams updates from either a subscription channel or periodic polling.
func StreamWatch[T any, Resp any](
	ctx context.Context,
	stream *connect.ServerStream[Resp],
	cycleDuration time.Duration,
	pollInterval time.Duration,
	snapshot SnapshotProvider[T],
	subscribe SubscriptionProvider[T],
	mapResponse func(T) (*Resp, bool),
) error {
	initial, err := snapshot(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if resp, ok := mapResponse(initial); ok && resp != nil {
		if err := stream.Send(resp); err != nil {
			return err
		}
	}

	timer := time.NewTimer(cycleDuration)
	defer timer.Stop()

	if subscribe != nil {
		ch, unsub := subscribe(ctx)
		defer unsub()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			case ev, ok := <-ch:
				if !ok {
					return nil
				}
				if resp, ok := mapResponse(ev); ok && resp != nil {
					if err := stream.Send(resp); err != nil {
						return err
					}
				}
			}
		}
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		case <-ticker.C:
			item, err := snapshot(ctx)
			if err != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
			if resp, ok := mapResponse(item); ok && resp != nil {
				if err := stream.Send(resp); err != nil {
					return err
				}
			}
		}
	}
}

// PollSnapshot resolves the current snapshot and constructs a Unary connect.Response.
func PollSnapshot[T any, Resp any](
	ctx context.Context,
	snapshot SnapshotProvider[T],
	mapResponse func(T) *Resp,
) (*connect.Response[Resp], error) {
	item, err := snapshot(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(mapResponse(item)), nil
}
