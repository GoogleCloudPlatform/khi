// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

// GeneratedRule corresponds to a single rule set for depguard.
type GeneratedRule struct {
	// RuleName will be the key for the rule set in .golangci.yaml.
	// e.g., "common-dependencies", "pkg-no-scripts"
	RuleName string

	// TargetFiles is a list of glob patterns for files/packages to which this rule applies.
	// This corresponds to the `files` directive in depguard.
	TargetFiles []string

	// DeniedPkgs is a list of packages to be denied.
	// This corresponds to the `deny` directive in depguard.
	DeniedPkgs []map[string]string // Slice of maps, where each map is {"pkg": "path", "desc": "description"}

	// AllowedPkgs is a list of packages to be allowed.
	// This corresponds to the `allow` directive in depguard.
	AllowedPkgs []string
}

// NewGeneratedRule creates a new rule with a given name and target files.
func NewGeneratedRule(ruleName string, targetFiles []string) *GeneratedRule {
	return &GeneratedRule{
		RuleName:    ruleName,
		TargetFiles: targetFiles,
		DeniedPkgs:  []map[string]string{},
		AllowedPkgs: []string{},
	}
}

// AddDeny adds a package to the deny list for this rule.
func (r *GeneratedRule) AddDeny(pkgs []string, desc string) *GeneratedRule {
	for _, pkg := range pkgs {
		r.DeniedPkgs = append(r.DeniedPkgs, map[string]string{"pkg": pkg, "desc": desc})
	}
	return r
}

// AddAllow adds packages to the allow list for this rule.
func (r *GeneratedRule) AddAllow(pkgs []string) *GeneratedRule {
	r.AllowedPkgs = append(r.AllowedPkgs, pkgs...)
	return r
}

// DepGuardRule represents a single depguard rule's configuration.
type DepGuardRule struct {
	Files       []string            `yaml:"files"`
	DeniedPkgs  []map[string]string `yaml:"deny,omitempty"`
	AllowedPkgs []string            `yaml:"allow,omitempty"`
}

// DepGuardRuleSet maps rule names to their respective configurations.
type DepGuardRuleSet map[string]DepGuardRule
