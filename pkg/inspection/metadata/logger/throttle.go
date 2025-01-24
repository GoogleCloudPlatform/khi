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

type LogThrottleStatus = int

var (
	StatusNoThrottle         LogThrottleStatus = 0
	StatusJustBeforeThrottle LogThrottleStatus = 1
	StatusThrottled          LogThrottleStatus = 2
)

// LogThrottler is an interface to decide if the new record should be printed or not.
// It mainly for reducing log volume addressing same kind log.
type LogThrottler interface {
	// ThrottleStatus returns true only when it should be handled. This must be called once for a single log entry.
	ThrottleStatus(logKind string) LogThrottleStatus
}

type ConstantLogThrottle struct {
	counter         *common.ConcurrentCounter
	MaxCountPerKind int
}

func NewConstantLogThrottle(maxCountPerKind int) LogThrottler {
	return ConstantLogThrottle{
		counter:         common.NewDefaultConcurrentCounter(common.NewSuffixShardingProvider(16, 1)),
		MaxCountPerKind: maxCountPerKind,
	}
}

func (c ConstantLogThrottle) ThrottleStatus(logKind string) LogThrottleStatus {
	if logKind == "" {
		return StatusNoThrottle
	}
	cnt := c.counter.Incr(logKind)
	if cnt == c.MaxCountPerKind {
		return StatusJustBeforeThrottle
	} else if cnt > c.MaxCountPerKind {
		return StatusThrottled
	} else {
		return StatusNoThrottle
	}
}
