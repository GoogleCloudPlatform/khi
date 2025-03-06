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

package cache

import (
	"crypto/rand"
	"fmt"
	"sync"
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestTouchKeyAndGetRemoved(t *testing.T) {
	testCases := []struct {
		Name                string
		TouchKeys           []string
		ExpectedRemovedKeys []string
		Capacity            int
		ExpectedLastSize    int
	}{{
		Name:                "simple non duplicated elements within capacity",
		TouchKeys:           []string{"a", "b", "c"},
		ExpectedRemovedKeys: []string{"", "", ""},
		Capacity:            4,
		ExpectedLastSize:    3,
	}, {
		Name:                "simple non duplicated elements exceeding capacity",
		TouchKeys:           []string{"a", "b", "c"},
		ExpectedRemovedKeys: []string{"", "", "a"},
		Capacity:            2,
		ExpectedLastSize:    2,
	},
		{
			Name:                "duplicated elements non exceeding capacity",
			TouchKeys:           []string{"a", "a", "a"},
			ExpectedRemovedKeys: []string{"", "", ""},
			Capacity:            1,
			ExpectedLastSize:    1,
		}, {
			Name:                "duplicated elements non exceeding capacity",
			TouchKeys:           []string{"a", "b", "c", "a", "d", "c", "b"},
			ExpectedRemovedKeys: []string{"", "", "", "", "b", "", "a"},
			Capacity:            3,
			ExpectedLastSize:    3,
		}}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			chain := NewLRUCacheItemReleaseStrategy(tc.Capacity)
			if len(tc.TouchKeys) != len(tc.ExpectedRemovedKeys) {
				t.Errorf("expected %d keys, got %d", len(tc.TouchKeys), len(tc.ExpectedRemovedKeys))
			}
			for i := 0; i < len(tc.TouchKeys); i++ {
				removedKey := chain.TouchAndGetRemovedKey(tc.TouchKeys[i])
				if removedKey != tc.ExpectedRemovedKeys[i] {
					t.Errorf("expected %s, got %s", tc.ExpectedRemovedKeys[i], removedKey)
				}
			}
			if chain.size != tc.ExpectedLastSize {
				t.Errorf("expected %d, got %d", tc.ExpectedLastSize, chain.size)
			}
			if chain.size != len(chain.refs) {
				t.Errorf(
					"expected %d, got %d",
					len(chain.refs),
					chain.size,
				)
			}
		})
	}
}

func TestLRU_MultiThread(t *testing.T) {
	THREAD := 1000
	COUNT := 1000
	chain := NewLRUCacheItemReleaseStrategy(10)
	wg := sync.WaitGroup{}
	wg.Add(THREAD)
	for i := 0; i < THREAD; i++ {
		go func() {
			randBytes := []byte{0}
			for j := 0; j < COUNT; j++ {
				rand.Read(randBytes)
				ri := int(randBytes[0])
				chain.TouchAndGetRemovedKey(fmt.Sprintf("%d", ri))
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkLRU10(b *testing.B) {
	chain := NewLRUCacheItemReleaseStrategy(10)
	elements := []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	}
	for i := 0; i < b.N; i++ {
		chain.TouchAndGetRemovedKey(elements[i%len(elements)])
	}
}

func BenchmarkLRU1(b *testing.B) {
	chain := NewLRUCacheItemReleaseStrategy(1)
	elements := []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	}
	for i := 0; i < b.N; i++ {
		chain.TouchAndGetRemovedKey(elements[i%len(elements)])
	}
}

func BenchmarkLRU1Random(b *testing.B) {
	chain := NewLRUCacheItemReleaseStrategy(1)
	elements := []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	}
	randBytes := []byte{0}
	for i := 0; i < b.N; i++ {
		rand.Read(randBytes)
		ri := int(randBytes[0])
		chain.TouchAndGetRemovedKey(elements[ri%len(elements)])
	}
}

func BenchmarkLRU10Random(b *testing.B) {
	chain := NewLRUCacheItemReleaseStrategy(10)
	elements := []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	}
	randBytes := []byte{0}
	for i := 0; i < b.N; i++ {
		rand.Read(randBytes)
		ri := int(randBytes[0])
		chain.TouchAndGetRemovedKey(elements[ri%len(elements)])
	}
}

func BenchmarkLRU24Random(b *testing.B) {
	chain := NewLRUCacheItemReleaseStrategy(24)
	elements := []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	}
	randBytes := []byte{0}
	for i := 0; i < b.N; i++ {
		rand.Read(randBytes)
		ri := int(randBytes[0])
		chain.TouchAndGetRemovedKey(elements[ri%len(elements)])
	}
}
