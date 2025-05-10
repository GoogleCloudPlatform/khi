// Copyright 2025 Google LLC
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

package testlog

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/structurev2"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
)

// MustLogFromYAML returns a log.Log instance from given YAML string.
// This method is for testing only.
func MustLogFromYAML(text string, fieldReaders ...log.FieldSetReader) *log.Log {
	yamlNode, err := structurev2.FromYAML(text)
	if err != nil {
		panic(err.Error())
	}
	l := log.NewLog(structurev2.NewNodeReader(yamlNode))
	for _, fieldReader := range fieldReaders {
		err := l.SetFieldSetReader(fieldReader)
		if err != nil {
			panic(err.Error())
		}
	}
	return l
}

func NewEmptyLogWithID(id string) *log.Log {
	l := log.NewLog(structurev2.NewNodeReader(structurev2.NewEmptyMapNode()))
	l.ID = id
	return l
}
