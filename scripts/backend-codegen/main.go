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
	"path/filepath"
	"strings"
	"text/template"
)

const (
	inspectionRegistrationTemplatePath = "scripts/backend-codegen/templates/zzz_register_inspection.go.tpl"
	testflagTemplatePath               = "scripts/backend-codegen/templates/zzz_testflag_test.go.tpl"
	inspectionRegistrationOutputPath   = "pkg/generated/zzz_register_inspection.go"
	testFlagOutputFileName             = "zzz_testflag_test.go"
	inspectionPkgDir                   = "pkg/task/inspection"
	testPkgDir                         = "pkg"
	goModPath                          = "go.mod"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	if err := generateInspectionRegistration(); err != nil {
		return fmt.Errorf("failed to generate inspection registration: %w", err)
	}
	if err := generateTestflags(); err != nil {
		return fmt.Errorf("failed to generate testflags: %w", err)
	}
	return nil
}

func generateInspectionRegistration() error {
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

	tmpl, err := template.ParseFiles(inspectionRegistrationTemplatePath)
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

	if err := os.WriteFile(inspectionRegistrationOutputPath, formattedSource, 0644); err != nil {
		return fmt.Errorf("failed to write generated file: %w", err)
	}
	return nil
}

func generateTestflags() error {
	finder := NewTestPackageFinder(testPkgDir)
	packages, err := finder.Find()
	if err != nil {
		return fmt.Errorf("failed to find test packages: %w", err)
	}

	tmpl, err := template.ParseFiles(testflagTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	for _, pkg := range packages {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, pkg); err != nil {
			return fmt.Errorf("failed to execute template for %s: %w", pkg.DirectoryPath, err)
		}

		formattedSource, err := format.Source(buf.Bytes())
		if err != nil {
			return fmt.Errorf("failed to format generated code for %s: %w", pkg.DirectoryPath, err)
		}

		outputPath := filepath.Join(pkg.DirectoryPath, testFlagOutputFileName)
		if err := os.WriteFile(outputPath, formattedSource, 0644); err != nil {
			return fmt.Errorf("failed to write generated file for %s: %w", pkg.DirectoryPath, err)
		}
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
