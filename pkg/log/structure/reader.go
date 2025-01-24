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

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredata"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
	"k8s.io/apimachinery/pkg/runtime"
)

type Reader struct {
	dataRef structuredatastore.StructureDataStorageRef
}

func NewReader(store structuredatastore.StructureDataStorageRef) *Reader {
	return &Reader{store}
}

// Reader navigates a hierarchical data structure using the provided path and optional filters.
// If the path resolves to multiple elements (typically array elements), an array of
// corresponding Readers is returned. Filters allow you to selectively include
// elements at each array level encountered during the path navigation.
// This function won't work with a path including `.` in the route. Use `ReaderFromArrayRoute` for the case.
//
// The 'filters' argument accepts a variable number of ReaderFilter arrays. Each array
// contains filters corresponding to a specific array level within the path.
//
// Example Paths:
//
//	"data.items[].name" // Access 'name' in each item within the 'items'
//	"config"           // Access root-level 'config'
func (r *Reader) Reader(path string, filters ...[]ReaderFilter) ([]*Reader, error) {
	if path == "" {
		return []*Reader{r}, nil
	}
	route := strings.Split(path, ".")
	return r.ReaderFromArrayRoute(route, filters...)
}

// ChildReader get a Reader from direct children of this node.
func (r *Reader) ChildReader(childName string) (*Reader, error) {
	readers, err := r.ReaderFromArrayRoute([]string{childName})
	if err != nil {
		return nil, err
	}
	if len(readers) == 0 {
		return nil, fmt.Errorf("child %s was not found", childName)
	}
	return readers[0], nil
}

func (r *Reader) ReaderFromArrayRoute(route []string, filters ...[]ReaderFilter) ([]*Reader, error) {
	current := []*Reader{r}
	arrayPathIndex := 0
	for i := 0; i < len(route); i++ {
		next := []*Reader{}
		for _, currentElement := range current {
			key := route[i]
			sd, err := currentElement.dataRef.Get()
			if err != nil {
				return nil, err
			}
			if strings.HasSuffix(key, "[]") {
				sequenceElementKey := strings.TrimSuffix(key, "[]")
				sequenceAny, err := sd.Value(sequenceElementKey)
				if err != nil {
					continue
				}
				if sequence, convertible := sequenceAny.(structuredata.StructureData); convertible {
					elemType, err := sequence.Type()
					if err != nil {
						return nil, err
					}
					if elemType != structuredata.StructuredTypeArray {
						continue
					}
					keys, err := sequence.Keys()
					if err != nil {
						return nil, err
					}
					for _, key := range keys {
						anyData, err := sequence.Value(key)
						if err != nil {
							return nil, err
						}
						if child, convertible := anyData.(structuredata.StructureData); convertible {
							childReader, err := getEphemeralReader(child)
							if err != nil {
								return nil, err
							}
							if arrayPathIndex < len(filters) {
								ignore := false
								for _, filter := range filters[arrayPathIndex] {
									nonIgnore, err := filter.Filter(childReader)
									if err != nil {
										return nil, err
									}
									if !nonIgnore {
										ignore = true
										break
									}
								}
								if ignore {
									continue
								}
							}
							next = append(next, childReader)
						}
					}
				}
				arrayPathIndex += 1
			} else {
				nextAny, err := sd.Value(key)
				if err != nil {
					continue
				}
				if nextElem, ok := nextAny.(structuredata.StructureData); ok {
					reader, err := getEphemeralReader(nextElem)
					if err != nil {
						return nil, err
					}
					next = append(next, reader)
				} else {
					return nil, err
				}
			}

		}
		current = next
	}
	return current, nil
}

// ReaderSingle calls the Reader method with expecting it to return a single reader as the result.
func (r *Reader) ReaderSingle(path string) (*Reader, error) {
	readers, err := r.Reader(path)
	if err != nil {
		return nil, err
	}
	if len(readers) == 0 {
		return nil, fmt.Errorf("path `%s` not found", path)
	}
	if len(readers) > 1 {
		return nil, fmt.Errorf("multiple readers are returned for %s", path)
	}
	return readers[0], nil
}

// Has returns true only when fields matching with the path is contained in the data
func (r *Reader) Has(path string) bool {
	readers, err := r.Reader(path)
	if err != nil {
		return false
	}
	return len(readers) > 0
}

// ReadString attempts to read and return a string value at the specified path.
// It returns an error if the path is not found, is not a string type, or if more than one reader matches the path.
func (r *Reader) ReadString(path string) (string, error) {
	reader, err := r.ReaderSingle(path)
	if err != nil {
		return "", err
	}
	return readScalarTyped[string](reader, "")
}

// ReadTimeAsString attempts to read the path as string or time.Time.
func (r *Reader) ReadTimeAsString(path string) (string, error) {
	reader, err := r.ReaderSingle(path)
	if err != nil {
		return "", nil
	}
	timeAsString, err := readScalarTyped[string](reader, "")
	if err == nil {
		return timeAsString, nil
	}
	timeAsTime, err := readScalarTyped[time.Time](reader, time.Time{})
	if err == nil {
		return timeAsTime.Format(time.RFC3339), nil
	}
	return "", fmt.Errorf("path %s can't be parsed as string or time.Time", path)
}

func (r *Reader) ReadInt(path string) (int, error) {
	reader, err := r.ReaderSingle(path)
	if err != nil {
		return 0, err
	}
	return readScalarTyped[int](reader, 0)
}

func (r *Reader) ReadStringOrDefault(path string, defaultValue string) string {
	val, err := r.ReadString(path)
	if err != nil {
		return defaultValue
	}
	return val
}

func (r *Reader) ReadIntOrDefault(path string, defaultValue int) int {
	val, err := r.ReadInt(path)
	if err != nil {
		return defaultValue
	}
	return val
}

func (r *Reader) ReadReflect(path string, target interface{}) error {
	reader, err := r.ReaderSingle(path)
	if err != nil {
		return err
	}
	sd, err := reader.dataRef.Get()
	if err != nil {
		return err
	}
	return structuredata.ReadReflect(sd, target)
}

func (r *Reader) ReadReflectK8sManifest(path string, target runtime.Object) error {
	reader, err := r.ReaderSingle(path)
	if err != nil {
		return err
	}
	sd, err := reader.dataRef.Get()
	if err != nil {
		return err
	}
	return structuredata.ReadReflectK8sManifest(sd, target)
}

func (r *Reader) ToYaml(path string) (string, error) {
	reader, err := r.ReaderSingle(path)
	if err != nil {
		return "", err
	}
	sd, err := reader.dataRef.Get()
	if err != nil {
		return "", err
	}
	return structuredata.ToYaml(sd)
}

func (r *Reader) ToJson(path string) (string, error) {
	reader, err := r.ReaderSingle(path)
	if err != nil {
		return "", err
	}
	sd, err := reader.dataRef.Get()
	if err != nil {
		return "", err
	}
	return structuredata.ToJson(sd)
}

func (r *Reader) readScalar() (any, error) {
	sd, err := r.dataRef.Get()
	if err != nil {
		return nil, err
	}
	strType, err := sd.Type()
	if err != nil {
		return nil, err
	}
	if strType != structuredata.StructuredTypeScalar {
		return nil, fmt.Errorf("current data is not a scalar")
	}
	return sd.Value("")
}

func getEphemeralReader(data structuredata.StructureData) (*Reader, error) {
	store := structuredatastore.OnMemoryStructureDataStore{}
	ref, err := store.StoreStructureData(data)
	if err != nil {
		return nil, err
	}
	return &Reader{
		dataRef: ref,
	}, nil
}

func readScalarTyped[T any](r *Reader, defaultValue T) (T, error) {
	raw, err := r.readScalar()
	if err != nil {
		return defaultValue, err
	}
	if converted, convertible := raw.(T); convertible {
		return converted, nil
	}
	return defaultValue, fmt.Errorf("the given value %v couldn't be converted to %T", raw, *new(T))
}
