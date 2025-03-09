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

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/crazy3lf/colorconv"
)

const SASS_FILE_LOCATION = "./web/src/app/generated.sass"
const SASS_TEMPLATE = "./scripts/frontend-codegen/templates/generated.sass.gtpl"

const GENERATED_TS_FILE_LOCATION = "./web/src/app/generated.ts"
const GENERATED_TS_TEMPLATE = "./scripts/frontend-codegen/templates/generated.ts.gtpl"

var templateFuncMap = template.FuncMap{
	"ToLower": strings.ToLower,
	"ToUpper": strings.ToUpper,
}

type templateInput struct {
	ParentRelationships     map[enum.ParentRelationship]enum.ParentRelationshipFrontendMetadata
	Severities              map[enum.Severity]enum.SeverityFrontendMetadata
	LogTypes                map[enum.LogType]enum.LogTypeFrontendMetadata
	RevisionStates          map[enum.RevisionState]enum.RevisionStateFrontendMetadata
	Verbs                   map[enum.RevisionVerb]enum.RevisionVerbFrontendMetadata
	LogTypeDarkColors       map[string]string
	RevisionStateDarkColors map[string]string
}

func main() {
	var input templateInput = templateInput{
		RevisionStates:          enum.RevisionStates,
		ParentRelationships:     enum.ParentRelationships,
		Severities:              enum.Severities,
		LogTypes:                enum.LogTypes,
		Verbs:                   enum.RevisionVerbs,
		LogTypeDarkColors:       map[string]string{},
		RevisionStateDarkColors: map[string]string{},
	}

	for _, logType := range enum.LogTypes {
		color, err := colorconv.HexToColor(logType.LabelBackgroundColor)
		if err != nil {
			panic(err)
		}
		h, s, l := colorconv.ColorToHSL(color)
		dl := l * 0.8
		if l == 0.0 { // only applicable for #000
			dl = 0.8
		}
		input.LogTypeDarkColors[logType.Label] = fmt.Sprintf("hsl(%fdeg %f%% %f%%)", h, s*100, dl*100)
	}

	for _, revisonState := range enum.RevisionStates {
		color, err := colorconv.HexToColor(revisonState.BackgroundColor)
		if err != nil {
			panic(err)
		}
		h, s, l := colorconv.ColorToHSL(color)
		dl := l * 0.8
		input.RevisionStateDarkColors[revisonState.CSSSelector] = fmt.Sprintf("hsl(%fdeg %f%% %f%%)", h, s*100, dl*100)
	}

	sassTemplate := loadTemplate("color-sass", SASS_TEMPLATE)
	var sassTemplateResult bytes.Buffer
	err := sassTemplate.Execute(&sassTemplateResult, input)
	if err != nil {
		panic(err)
	}
	mustWriteFile(SASS_FILE_LOCATION, sassTemplateResult.String())

	var legendTemplateResult bytes.Buffer
	legendTemplate := loadTemplate("logtypes-ts", GENERATED_TS_TEMPLATE)
	err = legendTemplate.Execute(&legendTemplateResult, input)
	if err != nil {
		panic(err)
	}
	mustWriteFile(GENERATED_TS_FILE_LOCATION, legendTemplateResult.String())

}

func loadTemplate(templateName string, templateLocation string) *template.Template {
	file, err := os.Open(templateLocation)
	if err != nil {
		panic(err)
	}
	templateContent, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	tpl, err := template.New(templateName).Funcs(templateFuncMap).Parse(string(templateContent))
	if err != nil {
		panic(err)
	}
	return tpl
}

func mustWriteFile(filePath string, data string) {
	perm32, _ := strconv.ParseUint("0644", 8, 32)
	err := os.WriteFile(filePath, []byte(data), os.FileMode(perm32))
	if err != nil {
		panic(err)
	}
}
