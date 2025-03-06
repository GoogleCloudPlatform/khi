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
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestGZipCacheItemStorageProvider(t *testing.T) {
	parent := newTestOnMemoryCacheItemStorageProvider()
	gzip := NewGZipCacheItemStorageProvider(parent)
	original := []byte("hello world")
	err := gzip.Set("test", original)
	if err != nil {
		t.Errorf("Set failed: %s", err)
	}
	result, err := gzip.Get("test")
	if err != nil {
		t.Errorf("Get failed: %s", err)
	}
	if string(result) != string(original) {
		t.Errorf("Unexpected result: expected:%s,actual:%s", string(original), string(result))
	}
}

func BenchmarkGZipCacheItemStorageProvider__Set(b *testing.B) {
	parent := newTestOnMemoryCacheItemStorageProvider()
	gzip := NewGZipCacheItemStorageProvider(parent)
	comp := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		rand.Read(comp)
		gzip.Set("test", comp)
	}
}

func BenchmarkGZipCacheItemStorageProvider__Get(b *testing.B) {
	parent := newTestOnMemoryCacheItemStorageProvider()
	gzip := NewGZipCacheItemStorageProvider(parent)
	comp := make([]byte, 1024)
	rand.Read(comp)
	gzip.Set("test", comp)
	for i := 0; i < b.N; i++ {
		gzip.Get("test")
	}
}
