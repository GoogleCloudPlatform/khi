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

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// RuleWriter defines the interface for writing generated depguard rules.
type RuleWriter interface {
	Write(rules ...*GeneratedRule) error
}

// FileSystemRuleWriter implements RuleWriter to write rules to a YAML file.
type FileSystemRuleWriter struct {
	// Path is the destination file path for the YAML output.
	Path string
}

// Write serializes the given rules into YAML format and merges them into the specified file.
// It preserves the existing content of the file and only updates the 'rules' section
// under 'linters-settings.depguard'.
func (w *FileSystemRuleWriter) Write(rules ...*GeneratedRule) error {
	f, err := os.Open(w.Path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to open file %s: %w", w.Path, err)
	}
	if f != nil {
		defer f.Close()
	}

	var root yaml.Node
	// If the file exists and is not empty, decode it.
	if err == nil {
		if err := yaml.NewDecoder(f).Decode(&root); err != nil && err != io.EOF {
			return fmt.Errorf("failed to decode yaml from %s: %w", w.Path, err)
		}
	} else {
		// If the file does not exist, initialize with a basic map structure.
		root.Kind = yaml.DocumentNode
		mapNode := yaml.Node{Kind: yaml.MappingNode}
		root.Content = []*yaml.Node{&mapNode}
	}

	rulesNode, err := w.createRulesNode(rules)
	if err != nil {
		return fmt.Errorf("failed to create rules node: %w", err)
	}

	path := []string{"linters", "settings", "depguard", "rules"}
	targetNode, err := findOrCreateNode(&root, path)
	if err != nil {
		return fmt.Errorf("failed to find or create node path: %w", err)
	}

	*targetNode = *rulesNode

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&root); err != nil {
		return fmt.Errorf("failed to encode yaml: %w", err)
	}

	if err := os.WriteFile(w.Path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", w.Path, err)
	}

	log.Printf("Successfully updated depguard rules in %s", w.Path)
	return nil
}

func (w *FileSystemRuleWriter) createRulesNode(rules []*GeneratedRule) (*yaml.Node, error) {
	rulesMap := make(DepGuardRuleSet)
	for _, rule := range rules {
		rulesMap[rule.RuleName] = DepGuardRule{
			Files:       rule.TargetFiles,
			DeniedPkgs:  rule.DeniedPkgs,
			AllowedPkgs: rule.AllowedPkgs,
		}
	}

	var node yaml.Node
	if err := node.Encode(rulesMap); err != nil {
		return nil, fmt.Errorf("failed to encode rules map to yaml.Node: %w", err)
	}
	return &node, nil
}

func findOrCreateNode(root *yaml.Node, path []string) (*yaml.Node, error) {
	current := root
	if current.Kind == yaml.DocumentNode {
		if len(current.Content) == 0 {
			current.Content = append(current.Content, &yaml.Node{Kind: yaml.MappingNode})
		}
		current = current.Content[0]
	}

	for _, key := range path {
		found := false
		for i := 0; i < len(current.Content); i += 2 {
			keyNode := current.Content[i]
			if keyNode.Value == key {
				current = current.Content[i+1]
				found = true
				break
			}
		}
		if !found {
			keyNode := &yaml.Node{}
			keyNode.Encode(key)

			valueNode := &yaml.Node{Kind: yaml.MappingNode}

			current.Content = append(current.Content, keyNode, valueNode)
			current = valueNode
		}
	}
	return current, nil
}
