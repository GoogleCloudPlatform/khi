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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/crazy3lf/colorconv"
)

const SCSS_FILE_LOCATION = "./web/src/app/generated.scss"
const SCSS_TEMPLATE = "./scripts/frontend-codegen/templates/generated.scss.gtpl"

const GENERATED_TS_FILE_LOCATION = "./web/src/app/generated.ts"
const GENERATED_TS_TEMPLATE = "./scripts/frontend-codegen/templates/generated.ts.gtpl"

const USED_ICON_FILES_LOCATION = "./scripts/msdf-generator/zzz_generated_used_icons.json"

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

type usedIconSetting struct {
	Icons []string `json:"icons"`
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

	for _, revisionState := range enum.RevisionStates {
		color, err := colorconv.HexToColor(revisionState.BackgroundColor)
		if err != nil {
			panic(err)
		}
		h, s, l := colorconv.ColorToHSL(color)
		dl := l * 0.8
		input.RevisionStateDarkColors[revisionState.CSSSelector] = fmt.Sprintf("hsl(%fdeg %f%% %f%%)", h, s*100, dl*100)
	}

	scssTemplate := loadTemplate("color-scss", SCSS_TEMPLATE)
	var scssTemplateResult bytes.Buffer
	err := scssTemplate.Execute(&scssTemplateResult, input)
	if err != nil {
		panic(err)
	}
	mustWriteFile(SCSS_FILE_LOCATION, scssTemplateResult.String())

	var legendTemplateResult bytes.Buffer
	legendTemplate := loadTemplate("logtypes-ts", GENERATED_TS_TEMPLATE)
	err = legendTemplate.Execute(&legendTemplateResult, input)
	if err != nil {
		panic(err)
	}
	mustWriteFile(GENERATED_TS_FILE_LOCATION, legendTemplateResult.String())

	// Generate icons.json storeing all the icons used in revision state to generate the icon font atlas.
	var icons = map[string]struct{}{}
	for _, revisonState := range enum.RevisionStates {
		icons[revisonState.Icon] = struct{}{}
	}
	iconSetting := usedIconSetting{
		Icons: []string{},
	}
	for icon := range icons {
		if icon == "" {
			continue
		}
		iconSetting.Icons = append(iconSetting.Icons, icon)
	}
	iconsJson, err := json.Marshal(iconSetting)
	if err != nil {
		panic(err)
	}
	mustWriteFile(USED_ICON_FILES_LOCATION, string(iconsJson))
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
