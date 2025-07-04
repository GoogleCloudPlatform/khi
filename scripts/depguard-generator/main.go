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
	"bytes"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	outputPath := flag.String("output", "../../.generated-golangci-depguard.yaml", "The path to the output YAML file.")
	flag.Parse()

	allPackages := mustListAllPackages()

	// All packages not end with _test.go must not depend pkg/testutil/*
	restrictTestUtil := NewGeneratedRule("no-testutil-in-non-test-files", []string{"**.go", "!**_test.go", "!**/pkg/testutil/*.go"})
	restrictTestUtil.AddDeny([]string{"github.com/GoogleCloudPlatform/khi/pkg/testutil"}, "Production code should not depend on test utilities.")

	// All packages under pkg/common must not depend other pacakge in this project
	restrictCommonDependingOther := NewGeneratedRule("common-cant-depend-other", []string{"pkg/common/**.go", "!pkg/common/**_test.go"})
	allNonCommonPackages := mustFilterPackages("^github.com/GoogleCloudPlatform/khi/pkg/common.*$", allPackages, true)

	restrictCommonDependingOther.AddDeny(allNonCommonPackages, "common package can't depend the other package")

	writer := &FileSystemRuleWriter{Path: *outputPath}
	if err := writer.Write(restrictTestUtil, restrictCommonDependingOther); err != nil {
		panic(fmt.Errorf("failed to write rules: %w", err))
	}
}

func mustFilterPackages(pattern string, sourceList []string, invert bool) []string {
	var matchedPackages []string

	for _, pkg := range sourceList {
		matched, err := regexp.MatchString(pattern, pkg)
		if err != nil {
			panic(fmt.Errorf("pattern match error on '%s': %w", pkg, err))
		}
		if matched == !invert {
			matchedPackages = append(matchedPackages, pkg)
		}
	}
	return matchedPackages
}

func mustListAllPackages() []string {
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}", "./...")
	// Assuming this script is run from the root of the khi repository.
	// For robustness, a root path could be passed in.
	cmd.Dir = "../.."

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Stderr when running `go list`: %s", stderr.String())
		panic(fmt.Errorf("failed to run 'go list': %w", err))
	}

	packages := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	return packages
}
