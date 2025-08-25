// Copyright 2025 Google LLC
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
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"strings"
	"text/template"
)

const (
	templatePath     = "scripts/backend-codegen/templates/inspection_registration.go.tpl"
	outputFilePath   = "pkg/generated/zzz_inspection_registration.generated.go"
	inspectionPkgDir = "pkg/task/inspection"
	goModPath        = "go.mod"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
	log.Printf("Successfully generated %s\n", outputFilePath)
}

func run() error {
	repoPackageName, err := getRepoPackageName(goModPath)
	if err != nil {
		return fmt.Errorf("failed to get repository package name: %w", err)
	}

	finder := &InspectionTaskPackageFinder{
		InspectionTaskPackageRootFilePath: inspectionPkgDir,
		RepositoryPackageName:             repoPackageName,
	}

	packages, err := finder.FindAllRequireRegistration()
	if err != nil {
		return fmt.Errorf("failed to find packages requiring registration: %w", err)
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, packages); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	formattedSource, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	if err := os.WriteFile(outputFilePath, formattedSource, 0644); err != nil {
		return fmt.Errorf("failed to write generated file: %w", err)
	}

	return nil
}

func getRepoPackageName(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("could not open go.mod file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning go.mod file: %w", err)
	}

	return "", fmt.Errorf("module directive not found in go.mod")
}
