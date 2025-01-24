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

	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	log_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/log"
)

// ParseFromYamlLogFile returns the parsed ChangeSet from the yaml log file at the given path with specified parser.
func ParseFromYamlLogFile(testFile string, parser parser.Parser, builder *history.Builder, variables *task.VariableSet) (*history.ChangeSet, error) {
	testutil.InitTestIO()
	yamlStr := testutil.MustReadText(testFile)
	l := log_test.MustLogEntity(yamlStr)
	cs := history.NewChangeSet(l)
	err := parser.Parse(context.Background(), l, cs, builder, variables)
	if err != nil {
		return nil, err
	}
	return cs, nil
}
