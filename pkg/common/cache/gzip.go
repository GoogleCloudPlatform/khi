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
	"bytes"
	"compress/gzip"
)

type GZipCacheItemStorageProvider struct {
	parent CacheItemStorageProvider
}

func NewGZipCacheItemStorageProvider(parent CacheItemStorageProvider) *GZipCacheItemStorageProvider {
	return &GZipCacheItemStorageProvider{parent: parent}
}

// Get implements CacheItemStorageProvider.
func (g *GZipCacheItemStorageProvider) Get(key string) ([]byte, error) {
	compressedData, err := g.parent.Get(key)
	if err != nil {
		return nil, err
	}
	gr, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(gr)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Set implements CacheItemStorageProvider.
func (g *GZipCacheItemStorageProvider) Set(key string, value []byte) error {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write(value)
	if err != nil {
		return err
	}
	err = gw.Close()
	if err != nil {
		return err
	}
	return g.parent.Set(key, buf.Bytes())
}

var _ CacheItemStorageProvider = (*GZipCacheItemStorageProvider)(nil)
