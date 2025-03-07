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
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/crazy3lf/colorconv"
)

const (
	sassFileLocation        = "./web/src/app/generated.sass"
	sassTemplateLocation    = "./scripts/frontend-codegen/templates/generated.sass.gtpl"
	generatedTSFileLocation = "./web/src/app/generated.ts"
	generatedTSTemplate     = "./scripts/frontend-codegen/templates/generated.ts.gtpl"
)

var templateFuncMap = template.FuncMap{
	"ToLower": strings.ToLower,
	"ToUpper": strings.ToUpper,
}

type TemplateInput struct {
	ParentRelationships     map[enum.ParentRelationship]enum.ParentRelationshipFrontendMetadata
	Severities              map[enum.Severity]enum.SeverityFrontendMetadata
	LogTypes                map[enum.LogType]enum.LogTypeFrontendMetadata
	RevisionStates          map[enum.RevisionState]enum.RevisionStateFrontendMetadata
	Verbs                   map[enum.RevisionVerb]enum.RevisionVerbFrontendMetadata
	LogTypeDarkColors       map[string]string
	RevisionStateDarkColors map[string]string
}

func main() {
	input := TemplateInput{
		ParentRelationships:     enum.ParentRelationships,
		Severities:              enum.Severities,
		LogTypes:                enum.LogTypes,
		RevisionStates:          enum.RevisionStates,
		Verbs:                   enum.RevisionVerbs,
		LogTypeDarkColors:       generateDarkColors(enum.LogTypes, "LabelBackgroundColor"),
		RevisionStateDarkColors: generateDarkColors(enum.RevisionStates, "BackgroundColor"),
	}

	generateFiles(sassFileLocation, sassTemplateLocation, input)
	generateFiles(generatedTSFileLocation, generatedTSTemplate, input)
}

func generateDarkColors[T comparable](items map[T]enum.ColorMetadata, colorField string) map[string]string {
	colors := make(map[string]string)
	for _, item := range items {
		hexColor := item.GetColor(colorField)
		color, err := colorconv.HexToColor(hexColor)
		if err != nil {
			log.Printf("Грешка при конвертиране на цвят (%s): %v", hexColor, err)
			continue
		}
		h, s, l := colorconv.ColorToHSL(color)
		colors[item.GetLabel()] = formatHSL(h, s, l)
	}
	return colors
}

func formatHSL(h, s, l float64) string {
	// Ако l е 0, задаваме стойност 0.8; в противен случай намаляваме светлинността с 20%
	if l == 0.0 {
		l = 0.8
	} else {
		l = l * 0.8
	}
	return fmt.Sprintf("hsl(%fdeg %f%% %f%%)", h, s*100, l)
}

func generateFiles(filePath, templatePath string, data TemplateInput) {
	tpl, err := loadTemplate(templatePath)
	if err != nil {
		log.Fatalf("Неуспешно зареждане на шаблона %q: %v", templatePath, err)
	}

	var output bytes.Buffer
	if err := tpl.Execute(&output, data); err != nil {
		log.Fatalf("Грешка при изпълнение на шаблона %q: %v", templatePath, err)
	}

	if err := os.WriteFile(filePath, output.Bytes(), 0644); err != nil {
		log.Fatalf("Грешка при запис на файл %q: %v", filePath, err)
	}
}

func loadTemplate(templateLocation string) (*template.Template, error) {
	file, err := os.Open(templateLocation)
	if err != nil {
		return nil, fmt.Errorf("отваряне на шаблон: %w", err)
	}
	defer file.Close()

	templateContent, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("четене на шаблон: %w", err)
	}

	tpl, err := template.New(templateLocation).Funcs(templateFuncMap).Parse(string(templateContent))
	if err != nil {
		return nil, fmt.Errorf("парсиране на шаблон: %w", err)
	}
	return tpl, nil
}
