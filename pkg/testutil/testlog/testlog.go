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

package testlog

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/structurev2"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
)

type TestLogOpt = func(original structurev2.Node) (structurev2.Node, error)

// TestLog is a type to generate mock log data effectively for test.
type TestLog struct {
	opts []TestLogOpt
}

func New(opts ...TestLogOpt) *TestLog {
	return &TestLog{
		opts: opts,
	}
}

// With instantiate a new TestLog with the given additional options and the current options.
func (b *TestLog) With(additionalOpts ...TestLogOpt) *TestLog {
	opts := []TestLogOpt{}
	opts = append(opts, b.opts...)
	opts = append(opts, additionalOpts...)
	return &TestLog{
		opts: opts,
	}
}

func (b *TestLog) BuildReader() (*structurev2.NodeReader, error) {
	var node structurev2.Node
	var err error
	for _, opt := range b.opts {
		node, err = opt(node)
		if err != nil {
			return nil, err
		}
	}
	return structurev2.NewNodeReader(node), nil
}

func (b *TestLog) MustBuildYamlString() string {
	reader, err := b.BuildReader()
	if err != nil {
		panic(err)
	}
	serializedRaw, err := reader.Serialize("", &structurev2.YAMLNodeSerializer{})
	if err != nil {
		panic(err)
	}
	return string(serializedRaw)
}

func (b *TestLog) MustBuildLogEntity(fieldSetReaders ...log.FieldSetReader) *log.Log {
	reader, err := b.BuildReader()
	if err != nil {
		panic(err.Error())
	}
	l := log.NewLog(reader)
	for _, fieldSetReader := range fieldSetReaders {
		err := l.SetFieldSetReader(fieldSetReader)
		if err != nil {
			panic(err.Error())
		}
	}
	return l
}
