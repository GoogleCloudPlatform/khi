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

import (
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestConstantLogThrottle(t *testing.T) {
	throttle := NewConstantLogThrottle(10)
	for i := 0; i < 9; i++ {
		if throttle.ThrottleStatus("foo") != StatusNoThrottle {
			t.Errorf("key foo shouldn't be throttled yet")
		}
		if throttle.ThrottleStatus("") != StatusNoThrottle {
			t.Errorf("key \"\" shouldn't be throttled never")
		}
	}
	if throttle.ThrottleStatus("foo") != StatusJustBeforeThrottle {
		t.Errorf("key foo should be throttled from next")
	}
	if throttle.ThrottleStatus("foo") != StatusThrottled {
		t.Errorf("foo should be throttled")
	}
	if throttle.ThrottleStatus("bar") != StatusNoThrottle {
		t.Errorf("key foo shouldn't be throttled yet")
	}
	if throttle.ThrottleStatus("") != StatusNoThrottle {
		t.Errorf("key \"\" shouldn't be throttled never")
	}
	if throttle.ThrottleStatus("") != StatusNoThrottle {
		t.Errorf("key \"\" shouldn't be throttled never")
	}
}
