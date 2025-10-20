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
	"time"

	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
)

// DefaultRetryPolicy is the default retry policy widely used to access Google Cloud API.
var DefaultRetryPolicy = gax.WithRetry(func() gax.Retryer {
	return gax.OnCodes([]codes.Code{
		codes.Aborted,
		codes.Canceled,
		codes.Internal,
		codes.ResourceExhausted,
		codes.Unknown,
		codes.Unavailable,
		codes.DeadlineExceeded,
	}, gax.Backoff{
		Initial:    100 * time.Millisecond,
		Max:        5000 * time.Millisecond,
		Multiplier: 1.30,
	})
})

// NeverTimeout is gax.CallOption that never reaches the timeout.
var NeverTimeout = gax.WithTimeout(1<<63 - 1)
