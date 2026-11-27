// Copyright 2026 Google LLC
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

package advisorycheck

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func detectGoCommentSuppressions(
	targetFset *token.FileSet,
	targetFile *ast.File,
	_ *token.FileSet,
	baseFile *ast.File,
	in goFileInput,
) []rule.QAItem {
	baseCount := 0
	if baseFile != nil {
		baseCount = len(collectGoSuppressions(nil, baseFile))
	}
	targetSuppressions := collectGoSuppressions(targetFset, targetFile)
	if len(targetSuppressions) <= baseCount {
		return nil
	}
	var items []rule.QAItem
	for _, sup := range targetSuppressions {
		if !isLineInScope(sup.line, in.changedLines) {
			continue
		}
		item := makeAdvisoryItem(checkSuppressionAdded, in.relPath, sup.line, "<comment>", "newly added quality or linter suppression comment")
		item.Evidence = map[string]any{"marker": sup.subject, "before_count": baseCount, "after_count": len(targetSuppressions)}
		item.ReviewPrompt = "Verify that this CODEQUALITY_ALLOW or nolint suppression is strictly justified and not hiding an avoidable defect."
		items = append(items, item)
	}
	return items
}

func collectGoSuppressions(fset *token.FileSet, file *ast.File) []lineSubject {
	var out []lineSubject
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			marker := matchGoSuppressionMarker(c.Text)
			if marker == "" {
				continue
			}
			ln := 1
			if fset != nil {
				ln = fset.Position(c.Pos()).Line
			}
			out = append(out, lineSubject{line: ln, subject: marker})
		}
	}
	return out
}

func matchGoSuppressionMarker(commentText string) string {
	for _, rawLine := range strings.Split(commentText, "\n") {
		trimmedRaw := strings.TrimSpace(rawLine)
		body := stripCommentLineDelimiters(trimmedRaw)
		if hasAllowDirectivePrefix(body) {
			return "CODEQUALITY_ALLOW"
		}
		if hasGoNolintDirective(trimmedRaw, body) {
			return "nolint"
		}
	}
	return ""
}

func hasGoNolintDirective(trimmedRaw string, body string) bool {
	// Detect naked //nolint even when followed by non-standard text after whitespace.
	if after, ok := strings.CutPrefix(trimmedRaw, "//nolint"); ok {
		return after == "" || after[0] == ':' || after[0] == '/' || after[0] == ' ' || after[0] == '\t'
	}
	after, ok := strings.CutPrefix(body, "nolint")
	if !ok {
		return false
	}
	rest := strings.TrimLeft(after, " \t")
	return rest == "" || strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, "//")
}
