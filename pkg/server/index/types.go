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

package index

// IndexTagGenerator generete list of tags to be injected in the index.html.
type IndexTagGenerator interface {
	// GenerateTags returns the tags injected in the header.
	GenerateTags() []string
}

var indexTagGenerators []IndexTagGenerator = make([]IndexTagGenerator, 0)

func AddTagGenerator(generator IndexTagGenerator) {
	indexTagGenerators = append(indexTagGenerators, generator)
}

func GenerateTags() []string {
	result := make([]string, 0)
	for _, generator := range indexTagGenerators {
		result = append(result, generator.GenerateTags()...)
	}
	return result

}
