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

package report

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

//go:embed templates/report.html.tmpl
var reportHTMLTemplateSource string

var reportHTMLTemplate = template.Must(template.New("vrt-report").Parse(reportHTMLTemplateSource))

type htmlTemplateData struct {
	PayloadJSON template.JS
}

// WriteHTMLReport renders an interactive visual diff viewer to outPath.
//
// Images are referenced via report-relative paths (before/, after/, diff/) so the HTML stays lightweight
// and renders all statuses including unchanged snapshots via external image loading.
func WriteHTMLReport(outPath string, rep ComparisonReport) error {
	rawJSON, err := json.Marshal(rep)
	if err != nil {
		return fmt.Errorf("failed to marshal HTML report payload: %w", err)
	}
	// Escape closing script tags inside JSON text so embedded error strings cannot break the script element.
	safeJSON := bytes.ReplaceAll(rawJSON, []byte("</"), []byte(`<\/`))

	var buf bytes.Buffer
	if err := reportHTMLTemplate.Execute(&buf, htmlTemplateData{
		PayloadJSON: template.JS(safeJSON),
	}); err != nil {
		return fmt.Errorf("failed to render HTML report template: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("failed to create HTML report dir: %w", err)
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to write HTML report %s: %w", outPath, err)
	}
	return nil
}
