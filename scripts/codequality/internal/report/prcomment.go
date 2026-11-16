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
	"cmp"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// Sticky PR comment HTML marker and Check Run log envelope delimiters.
const (
	PRCommentMarker     = "<!-- khi-codequality-report -->"
	EnvelopeBeginMarker = "KHI_CODEQUALITY_B64_BEGIN"
	EnvelopeEndMarker   = "KHI_CODEQUALITY_B64_END"
)

// Cap raw Markdown at 24 KB (~32 KB Base64) so the envelope always fits inside
// the 65,535-char tail of GitHub Check Run output.text.
const defaultMaxCommentBytes = 24000

// PRCommentMeta holds optional CI metadata rendered in the sticky PR comment footer.
type PRCommentMeta struct {
	// CommitSHA is the Git commit SHA evaluated by this run.
	CommitSHA string
	// BuildURL links to the Cloud Build or CI execution logs.
	BuildURL string
	// MaxBytes overrides the maximum comment byte length before truncation (0 uses 24000).
	MaxBytes int
}

// FormatPRCommentBody renders the sticky GitHub PR comment Markdown body with marker and footer.
func FormatPRCommentBody(baseLabel string, reports []*rule.Report, meta PRCommentMeta) string {
	var buf bytes.Buffer
	buf.WriteString(PRCommentMarker)
	buf.WriteByte('\n')

	if len(reports) == 0 {
		fmt.Fprintf(&buf, "## Code Quality Report (`%s`) — ✅ PASS\n\nNo Go or TypeScript files were modified in this change.\n", baseLabel)
	} else {
		for i, rep := range reports {
			if i > 0 {
				buf.WriteByte('\n')
			}
			RenderGitHubSummary(&buf, rep)
		}
	}

	footer := formatCommentFooter(meta)
	maxBytes := cmp.Or(meta.MaxBytes, defaultMaxCommentBytes)
	body := truncateCommentBody(buf.String(), maxBytes-len(footer))
	return body + footer
}

// WriteCommentEnvelope writes body as a Base64-encoded block delimited by EnvelopeBeginMarker and EnvelopeEndMarker.
func WriteCommentEnvelope(w io.Writer, body string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	fmt.Fprintf(w, "\n%s\n%s\n%s\n", EnvelopeBeginMarker, encoded, EnvelopeEndMarker)
}

// ExtractCommentEnvelope decodes the last Base64 comment envelope found inside logText.
func ExtractCommentEnvelope(logText string) (string, bool, error) {
	beginIdx := strings.LastIndex(logText, EnvelopeBeginMarker)
	if beginIdx < 0 {
		return "", false, nil
	}
	rest := logText[beginIdx+len(EnvelopeBeginMarker):]
	endIdx := strings.Index(rest, EnvelopeEndMarker)
	if endIdx < 0 {
		return "", false, fmt.Errorf("unterminated %s block", EnvelopeBeginMarker)
	}
	compact := strings.Join(strings.Fields(rest[:endIdx]), "")
	decoded, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return "", false, fmt.Errorf("failed to decode comment envelope base64: %w", err)
	}
	return string(decoded), true, nil
}

func formatCommentFooter(meta PRCommentMeta) string {
	var parts []string
	if sha := strings.TrimSpace(meta.CommitSHA); sha != "" {
		parts = append(parts, fmt.Sprintf("Commit: `%s`", sha[:min(len(sha), 12)]))
	}
	if url := strings.TrimSpace(meta.BuildURL); url != "" {
		parts = append(parts, fmt.Sprintf("[Cloud Build Logs](%s)", url))
	}
	if len(parts) == 0 {
		return ""
	}
	return "\n---\n_" + strings.Join(parts, " · ") + "_\n"
}

func truncateCommentBody(body string, maxBytes int) string {
	const suffix = "\n\n_(Report truncated to fit GitHub Check Run payload — see Cloud Build logs for full output.)_\n"
	if maxBytes <= len(suffix) || len(body) <= maxBytes {
		return body
	}
	prefix := strings.ToValidUTF8(body[:maxBytes-len(suffix)], "")
	return prefix + suffix
}
