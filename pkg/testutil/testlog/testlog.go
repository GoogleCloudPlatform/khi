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
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/adapter"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredata"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
	"gopkg.in/yaml.v3"
)

type TestLogOpt = func(original *yaml.Node) (*yaml.Node, error)

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

func (b *TestLog) BuildReader() (*structure.Reader, error) {
	var node *yaml.Node
	var err error
	for _, opt := range b.opts {
		node, err = opt(node)
		if err != nil {
			return nil, err
		}
	}
	sd, err := structuredata.DataFromYamlNode(node)
	if err != nil {
		return nil, err
	}
	directStore := adapter.Direct(sd)
	return directStore.GetReaderBackedByStore(&structuredatastore.OnMemoryStructureDataStore{})
}

func (b *TestLog) MustBuildYamlString() string {
	reader, err := b.BuildReader()
	if err != nil {
		panic(err)
	}
	yamlStr, err := reader.ToYaml("")
	if err != nil {
		panic(err)
	}
	return yamlStr
}

func (b *TestLog) MustBuildLogEntity(le log.CommonLogFieldExtractor) *log.LogEntity {
	reader, err := b.BuildReader()
	if err != nil {
		panic(err)
	}
	return log.NewLogEntity(reader, le)
}
