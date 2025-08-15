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

package logger

import "github.com/GoogleCloudPlatform/khi/pkg/common"

type logThrottleStatus = int

var (
	statusNoThrottle         logThrottleStatus = 0
	statusJustBeforeThrottle logThrottleStatus = 1
	statusThrottled          logThrottleStatus = 2
)

// logThrottler is an interface to decide if a new log record should be printed.
// It is used to reduce the volume of similar, repetitive logs.
type logThrottler interface {
	// ThrottleStatus determines if a log of a given kind should be processed.
	// It must be called once for each log entry.
	ThrottleStatus(logKind string) logThrottleStatus
}

// constantCountLogThrottler is an implementation of logThrottler that allows
// a constant number of logs per kind before it starts throttling.
type constantCountLogThrottler struct {
	counter         *common.ConcurrentCounter
	MaxCountPerKind int
}

func newConstantCountLogThrottler(maxCountPerKind int) logThrottler {
	return constantCountLogThrottler{
		counter:         common.NewDefaultConcurrentCounter(common.NewSuffixShardingProvider(16, 1)),
		MaxCountPerKind: maxCountPerKind,
	}
}

func (c constantCountLogThrottler) ThrottleStatus(logKind string) logThrottleStatus {
	if logKind == "" {
		return statusNoThrottle
	}
	cnt := c.counter.Incr(logKind)
	switch {
	case cnt == c.MaxCountPerKind:
		return statusJustBeforeThrottle
	case cnt > c.MaxCountPerKind:
		return statusThrottled
	default:
		return statusNoThrottle
	}
}
