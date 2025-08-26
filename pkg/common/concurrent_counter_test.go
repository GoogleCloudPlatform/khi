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

package common

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentCounter(t *testing.T) {
	ThreadCount := 100
	ItemCount := 100
	counter := NewDefaultConcurrentCounter(NewSuffixShardingProvider(16, 1))
	wg := sync.WaitGroup{}
	for tc := 0; tc < ThreadCount; tc++ {
		wg.Add(1)
		go func() {
			for ic := 0; ic < ItemCount; ic++ {
				counter.Incr(fmt.Sprintf("item-%d", ic))
			}
			wg.Done()
		}()
	}
	wg.Wait()
	for ic := 0; ic < ItemCount; ic++ {
		cnt := counter.Get(fmt.Sprintf("item-%d", ic))
		if cnt != ItemCount {
			t.Errorf("expected %d, got %d", ItemCount, cnt)
		}
	}
}
