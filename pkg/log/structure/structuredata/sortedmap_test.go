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

package structuredata

import (
	"fmt"
	"math/rand"
	"testing"

	"gopkg.in/yaml.v3"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func shufleFields(fields []string) []string {
	for i := len(fields) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		fields[i], fields[j] = fields[j], fields[i]
	}
	return fields
}

func TestUnorderedMarshal(t *testing.T) {
	sm := newSortedMap()
	fields := []string{}
	for i := 0; i < 10; i++ {
		fields = append(fields, fmt.Sprintf("field-%d", i))
	}
	shufleFields(fields)
	yamlStr := ""
	for i, field := range fields {
		yamlStr += fmt.Sprintf("%s: %d\n", field, i)
		sm.AddNextField(field, i)
	}

	result, err := yaml.Marshal(sm)
	if err != nil {
		t.Fatal(err)
	}

	resultYaml := string(result)
	if yamlStr != resultYaml {
		t.Errorf("Result is not matching with the input YAML data\nEXPECTED:\n\n%s\n\nACTUAL:\n\n%s", yamlStr, resultYaml)
	}
}

func TestUnorderedMarshalWithNilField(t *testing.T) {
	sm := newSortedMap()
	fields := []string{}
	for i := 0; i < 10; i++ {
		fields = append(fields, fmt.Sprintf("field-%d", i))
	}
	shufleFields(fields)
	yamlStr := ""
	for _, field := range fields {
		yamlStr += fmt.Sprintf("%s: null\n", field)
		sm.AddNextField(field, nil)
	}

	result, err := yaml.Marshal(sm)
	if err != nil {
		t.Fatal(err)
	}

	resultYaml := string(result)
	if yamlStr != resultYaml {
		t.Errorf("Result is not matching with the input YAML data\nEXPECTED:\n\n%s\n\nACTUAL:\n\n%s", yamlStr, resultYaml)
	}
}
