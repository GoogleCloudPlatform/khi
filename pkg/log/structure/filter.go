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

package structure

import "fmt"

type ReaderFilter interface {
	Filter(reader *Reader) (bool, error)
}

type FieldEqualityFilter struct {
	fieldName string
	equalTo   any
}

// Filter implements ReaderFilter.
func (f *FieldEqualityFilter) Filter(reader *Reader) (bool, error) {
	readers, err := reader.Reader(f.fieldName)
	if err != nil {
		return false, nil
	}
	if len(readers) > 1 {
		return false, fmt.Errorf("filter target is expected to be a single value")
	}
	if len(readers) == 0 {
		return false, nil
	}
	value, err := readers[0].readScalar()
	if err != nil {
		return false, err
	}
	return value == f.equalTo, nil
}

func EqualFilter(fieldName string, equalTo any) ReaderFilter {
	return &FieldEqualityFilter{
		fieldName: fieldName,
		equalTo:   equalTo,
	}
}
