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

package binarychunk

import (
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFileSystemBinaryWriter(t *testing.T) {
	t.Run("GetBinary must return a reader pointing at the head", func(t *testing.T) {
		w, err := NewFileSystemBinaryWriter("/tmp", 0, 100)
		if err != nil {
			t.Errorf("err was not a nil:%v", err)
		}
		data := []byte{0x01, 0x02, 0x03}

		_, err = w.Write(data)
		if err != nil {
			t.Errorf("err was not a nil:%v", err)
		}
		reader, err := w.GetBinary()
		if err != nil {
			t.Errorf("err was not a nil:%v", err)
		}

		readResult, err := io.ReadAll(reader)
		if err != nil {
			t.Errorf("err was not a nil:%v", err)
		}
		if diff := cmp.Diff(data, readResult); diff != "" {
			t.Errorf("+data, -received,%s", diff)
		}
	})

	t.Run("GetBinary must return the valid BinaryReference", func(t *testing.T) {
		w, err := NewFileSystemBinaryWriter("/tmp", 1234, 100)
		if err != nil {
			t.Errorf("err was not a nil:%v", err)
		}
		data1 := []byte{0x01, 0x02, 0x03}
		data2 := []byte{0x04, 0x05, 0x06, 0x07}
		expected1 := &BinaryReference{
			Buffer: 1234,
			Length: 3,
			Offset: 0,
		}
		expected2 := &BinaryReference{
			Buffer: 1234,
			Length: 4,
			Offset: 3,
		}

		result1, err := w.Write(data1)
		if err != nil {
			t.Errorf("err was not a nil:%v", err)
		}
		result2, err := w.Write(data2)
		if err != nil {
			t.Errorf("err was not a nil:%v", err)
		}

		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("+data, -received,%s", diff)
		}
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("+data, -received,%s", diff)
		}
	})

	t.Run("Read & Write must be safe with being called concurrently", func(t *testing.T) {
		w, err := NewFileSystemBinaryWriter("/tmp", 0, 1024*1024*1024)
		if err != nil {
			t.Errorf("unexpected error")
		}
		wg := sync.WaitGroup{}
		THREAD_COUNT := 1000
		for th := 0; th < THREAD_COUNT; th += 1 {
			wg.Add(1)
			thread_index := th
			go func() {
				wg2 := sync.WaitGroup{}
				for i := 0; i < 1000; i++ {
					wg2.Add(1)
					data := fmt.Sprintf("data-%d-%d", thread_index, i)
					ref, err := w.Write([]byte(data))
					if err != nil {
						t.Errorf("unexpected error:%s", err.Error())
					}
					go func(ref *BinaryReference, expected string) {
						result, err := w.Read(ref)
						if err != nil {
							t.Errorf("unexpected error:%s", err.Error())
						}
						actual := string(result)
						if actual != expected {
							t.Errorf("Read result unmatched with the expected value\nexpected:%s,actual:%s", expected, actual)
						}
						wg2.Done()
					}(ref, data)
				}
				wg.Done()
			}()
		}
		wg.Wait()

	})
}
