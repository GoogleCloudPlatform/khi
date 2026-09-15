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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// InspectionTaskPackage represents a Go package that defines an inspection task.
type InspectionTaskPackage struct {
	// PackageRootFolderPath is the file system path to the root of the package.
	PackageRootFolderPath string
	// PackageImportPathBase is the base import path for the package.
	PackageImportPathBase string
	// PackageNamePrefix is the prefix for the package name, derived from its directory path.
	PackageNamePrefix string
}

// ContractPackageName returns the package name for the contract part of the task.
func (p *InspectionTaskPackage) ContractPackageName() string {
	return fmt.Sprintf("%s_contract", p.PackageNamePrefix)
}

// ContractPackageImportPath returns the full Go import path for the contract package.
func (p *InspectionTaskPackage) ContractPackageImportPath() string {
	return fmt.Sprintf("%s/contract", p.PackageImportPathBase)
}

// ImplPackageName returns the package alias name for the implementation part of the task.
func (p *InspectionTaskPackage) ImplPackageName() string {
	return fmt.Sprintf("%s_impl", p.PackageNamePrefix)
}

// ImplPackageImportPath returns the full Go import path for the implementation package.
func (p *InspectionTaskPackage) ImplPackageImportPath() string {
	return fmt.Sprintf("%s/impl", p.PackageImportPathBase)
}

// DoNotRegisterPackagePaths is a set of package root folder paths that should be excluded
// from the automatic registration process.
var DoNotRegisterPackagePaths = map[string]struct{}{
	"pkg/task/inspection/inspectioncore": {},
}

// InspectionTaskPackageFinder is responsible for finding inspection task packages
// within the project structure.
type InspectionTaskPackageFinder struct {
	// PackageRootFilePath is the root directory of the project's Go packages (e.g., "pkg").
	PackageRootFilePath string
	// InspectionTaskPackageRootFilePath is the specific directory where inspection task packages reside.
	InspectionTaskPackageRootFilePath string
	// RepositoryPackageName is the Go module name for the repository (e.g., "github.com/GoogleCloudPlatform/khi").
	RepositoryPackageName string
}

// FindAllRequireRegistration finds all inspection task packages that have an 'impl' subdirectory
// and are not in the DoNotRegisterPackagePaths list. These packages require registration.
func (f *InspectionTaskPackageFinder) FindAllRequireRegistration() ([]InspectionTaskPackage, error) {
	allPackages, err := f.findAllInspectionTaskPackages()
	if err != nil {
		return nil, err
	}

	var requiredPackages []InspectionTaskPackage
	for _, pkg := range allPackages {
		if _, ok := DoNotRegisterPackagePaths[pkg.PackageRootFolderPath]; !ok {
			requiredPackages = append(requiredPackages, pkg)
		}
	}
	return requiredPackages, nil
}

// findAllInspectionTaskPackages recursively walks the inspection task package root directory
// and returns all packages that contain an 'impl' subdirectory.
func (f *InspectionTaskPackageFinder) findAllInspectionTaskPackages() ([]InspectionTaskPackage, error) {
	var packages []InspectionTaskPackage
	err := filepath.WalkDir(f.InspectionTaskPackageRootFilePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || path == f.InspectionTaskPackageRootFilePath {
			return nil
		}
		if d.Name() == "impl" {
			pkgRootPath := filepath.ToSlash(filepath.Dir(path))
			relPath, err := filepath.Rel(f.InspectionTaskPackageRootFilePath, pkgRootPath)
			if err != nil {
				return err
			}
			aliasPrefix := strings.ReplaceAll(filepath.ToSlash(relPath), "/", "_")
			packages = append(packages, InspectionTaskPackage{
				PackageRootFolderPath: pkgRootPath,
				PackageImportPathBase: filepath.ToSlash(filepath.Join(f.RepositoryPackageName, pkgRootPath)),
				PackageNamePrefix:     aliasPrefix,
			})
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(packages, func(a, b InspectionTaskPackage) int {
		return strings.Compare(a.PackageImportPathBase, b.PackageImportPathBase)
	})
	return packages, nil
}
