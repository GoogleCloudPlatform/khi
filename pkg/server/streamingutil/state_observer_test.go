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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestPollSnapshot(t *testing.T) {
	testCases := []struct {
		name        string
		snapshot    SnapshotProvider[string]
		mapResponse func(string) *int
		wantVal     int
		wantErr     bool
	}{
		{
			name: "successful snapshot",
			snapshot: func(ctx context.Context) (string, error) {
				return "hello", nil
			},
			mapResponse: func(s string) *int {
				v := len(s)
				return &v
			},
			wantVal: 5,
			wantErr: false,
		},
		{
			name: "error snapshot",
			snapshot: func(ctx context.Context) (string, error) {
				return "", errors.New("snapshot failure")
			},
			mapResponse: func(s string) *int {
				v := len(s)
				return &v
			},
			wantVal: 0,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := PollSnapshot(context.Background(), tc.snapshot, tc.mapResponse)
			if (err != nil) != tc.wantErr {
				t.Fatalf("PollSnapshot() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.wantVal, *resp.Msg); diff != "" {
					t.Errorf("PollSnapshot() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestStreamWatch_Subscription(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "timer finishes cycle cleanly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			snapshot := func(ctx context.Context) (string, error) {
				return "initial", nil
			}
			ch := make(chan string, 1)
			subscribe := func(ctx context.Context) (<-chan string, func()) {
				return ch, func() {}
			}

			// We don't have connect.ServerStream mock easily, but we test the context termination
			err := StreamWatch[string, string](
				ctx,
				nil, // nil stream since mapResponse returns false so stream.Send won't be called
				100*time.Millisecond,
				10*time.Millisecond,
				snapshot,
				subscribe,
				func(s string) (*string, bool) {
					return nil, false
				},
			)
			if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
				t.Errorf("StreamWatch() expected context error, got %v", err)
			}
		})
	}
}
