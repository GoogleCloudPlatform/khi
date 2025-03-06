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
	"crypto/rand"
	"sync"
	"testing"
	"time"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

const keyLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randStr(digit uint32) string {
	b := make([]byte, digit)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	result := ""
	for _, v := range b {
		result += string(keyLetters[int(v)%len(keyLetters)])
	}
	return result
}

func benchmarkShardingMap(n int, threadCount int, shardCount int) {
	shardMap := NewShardingMap[struct{}](NewSuffixShardingProvider(shardCount, 1))
	wg := sync.WaitGroup{}
	for t := 0; t < threadCount; t++ {
		wg.Add(1)
		go func() {
			for i := 0; i < n; i++ {
				key := randStr(10)
				m := shardMap.AcquireShard(key)
				m[key] = struct{}{}
				<-time.After(time.Millisecond)
				shardMap.ReleaseShard(key)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkShardingMap1(b *testing.B) {
	benchmarkShardingMap(b.N, 100, 1)
}

func BenchmarkShardingMap4(b *testing.B) {
	benchmarkShardingMap(b.N, 100, 4)
}

func BenchmarkShardingMap16(b *testing.B) {
	benchmarkShardingMap(b.N, 100, 16)
}
