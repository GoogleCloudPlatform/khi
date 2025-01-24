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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/parser/yaml/yamlutil"
	"gopkg.in/yaml.v3"
)

func BaseYaml(yamlStr string) TestLogOpt {
	return func(original *yaml.Node) (*yaml.Node, error) {
		if original != nil {
			return nil, fmt.Errorf("BaseYaml expects no previous TestLogOpt is given. But an instance of node was given")
		}
		if yamlStr == "" {
			return yamlutil.NewEmptyMapNode(), nil
		}
		var node yaml.Node
		err := yaml.Unmarshal([]byte(yamlStr), &node)
		if err != nil {
			return nil, err
		}
		return &node, err
	}
}
