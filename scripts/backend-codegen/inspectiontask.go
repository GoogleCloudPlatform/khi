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
)

// InspectionTaskPackage represents a Go package that defines an inspection task.
type InspectionTaskPackage struct {
	// PackageRootFolderPath is the file system path to the root of the package.
	PackageRootFolderPath string
	// PackageImportPathBase is the base import path for the package.
	PackageImportPathBase string
	// PackageNamePrefix is the prefix for the package name, derived from its directory name.
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

// ImplPackageName returns the package name for the implementation part of the task.
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
		// All packages having `pkg/task/inspection/<package_name>/impl` folder require registration.
		implPath := filepath.Join(pkg.PackageRootFolderPath, "impl")
		info, err := os.Stat(implPath)
		switch {
		case err == nil && info.IsDir():
			if _, ok := DoNotRegisterPackagePaths[pkg.PackageRootFolderPath]; !ok {
				requiredPackages = append(requiredPackages, pkg)
			}
		case err == nil && !info.IsDir():
			return nil, fmt.Errorf("expected %s is a directory name but it's not a dir", implPath)
		case os.IsNotExist(err):
			continue
		default:
			return nil, err
		}
	}
	return requiredPackages, nil
}

// findAllInspectionTaskPackages returns a list of all packages that are direct children
// of the inspection task package root directory.
func (f *InspectionTaskPackageFinder) findAllInspectionTaskPackages() ([]InspectionTaskPackage, error) {
	entries, err := os.ReadDir(f.InspectionTaskPackageRootFilePath)
	if err != nil {
		return nil, err
	}

	var packages []InspectionTaskPackage
	for _, entry := range entries {
		if entry.IsDir() {
			dirName := entry.Name()
			pkgRootPath := filepath.Join(f.InspectionTaskPackageRootFilePath, dirName)
			// The package name prefix is the same name of its directory name
			packages = append(packages, InspectionTaskPackage{
				PackageRootFolderPath: pkgRootPath,
				PackageImportPathBase: filepath.ToSlash(filepath.Join(f.RepositoryPackageName, pkgRootPath)),
				PackageNamePrefix:     dirName,
			})
		}
	}
	return packages, nil
}
