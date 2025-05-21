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

package parser_test

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

// ParseFromYamlLogFile returns the parsed ChangeSet from the yaml log file at the given path with specified parser.
func ParseFromYamlLogFile(testFile string, parser parser.Parser, builder *history.Builder, readers ...log.FieldSetReader) (*history.ChangeSet, error) {
	testutil.InitTestIO()
	yamlStr := testutil.MustReadText(testFile)
	l := testlog.MustLogFromYAML(yamlStr)
	for _, reader := range readers {
		err := l.SetFieldSetReader(reader)
		if err != nil {
			return nil, err
		}
	}
	cs := history.NewChangeSet(l)
	err := parser.Parse(context.Background(), l, cs, builder)
	if err != nil {
		return nil, err
	}
	return cs, nil
}
