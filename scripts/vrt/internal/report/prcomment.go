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
	"path"
	"strings"
)

// Sticky PR comment HTML marker and Check Run log envelope delimiters.
const (
	PRCommentMarker     = "<!-- khi-vrt-report -->"
	EnvelopeBeginMarker = "KHI_VRT_B64_BEGIN"
	EnvelopeEndMarker   = "KHI_VRT_B64_END"

	defaultMaxCommentBytes = 24000
	defaultMaxListedItems  = 15
	defaultMaxInlineImages = 6
)

// PRCommentMeta holds CI links and display options for rendering the sticky GitHub PR comment.
type PRCommentMeta struct {
	// CommitSHA is the candidate commit SHA.
	CommitSHA string
	// BaseSHA is the baseline merge-base commit SHA.
	BaseSHA string
	// BaseBranch is the target base branch name (e.g., "main").
	BaseBranch string
	// BuildURL links to the Cloud Build execution log.
	BuildURL string
	// ReportBaseURL is the GCS URL prefix of the uploaded report directory (without trailing slash).
	ReportBaseURL string
	// InlineImages embeds composite diff thumbnails directly in Markdown when true (use for public GCS buckets).
	InlineImages bool
	// SkippedReason explains why VRT was skipped when non-empty (e.g., no frontend files changed).
	SkippedReason string
	// MaxBytes overrides the maximum comment length before truncation (0 uses 24000).
	MaxBytes int
}

// FormatPRCommentBody renders the sticky GitHub PR comment Markdown body with marker, summary, links, and footer.
func FormatPRCommentBody(rep ComparisonReport, meta PRCommentMeta) string {
	var buf bytes.Buffer
	buf.WriteString(PRCommentMarker)
	buf.WriteByte('\n')

	if strings.TrimSpace(meta.SkippedReason) != "" {
		buf.WriteString("## Frontend Visual Regression Report — ⏭️ SKIPPED\n")
		buf.WriteByte('\n')
		fmt.Fprintf(&buf, "%s\n", strings.TrimSpace(meta.SkippedReason))
		return finalizeComment(buf.String(), meta)
	}

	writeCommentHeader(&buf, rep, meta)
	writeErrorSection(&buf, rep)
	writeChangedSection(&buf, rep, meta)
	writeAddedOrRemovedSection(&buf, "Added Stories", StatusAdded, rep, meta)
	writeAddedOrRemovedSection(&buf, "Removed Stories", StatusRemoved, rep, meta)

	return finalizeComment(buf.String(), meta)
}

// WriteCommentEnvelope writes body as a Base64-encoded block delimited by EnvelopeBeginMarker and EnvelopeEndMarker.
func WriteCommentEnvelope(w io.Writer, body string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	fmt.Fprintln(w)
	fmt.Fprintln(w, EnvelopeBeginMarker)
	fmt.Fprintln(w, encoded)
	fmt.Fprintln(w, EnvelopeEndMarker)
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

func writeCommentHeader(buf *bytes.Buffer, rep ComparisonReport, meta PRCommentMeta) {
	verdict := "✅ PASS — No visual changes"
	switch {
	case rep.HasErrors():
		verdict = fmt.Sprintf("❌ ERROR — %d story error(s)", rep.Summary.Errored)
	case rep.HasVisualDiffs():
		verdict = fmt.Sprintf("🔍 VISUAL DIFF — %d changed, %d added, %d removed", rep.Summary.Changed, rep.Summary.Added, rep.Summary.Removed)
	}
	fmt.Fprintf(buf, "## Frontend Visual Regression Report — %s\n", verdict)
	buf.WriteByte('\n')

	if meta.ReportBaseURL != "" {
		reportURL := joinReportURL(meta.ReportBaseURL, ReportHTMLFileName)
		fmt.Fprintf(buf, "**Interactive Diff Viewer:** [Open HTML Report (GCS)](%s)", reportURL)
		if !meta.InlineImages {
			buf.WriteString(" _(requires authenticated GCP project access)_")
		}
		buf.WriteString("\n")
		buf.WriteByte('\n')
	}

	buf.WriteString(`| Total | Changed | Added | Removed | Errored | Unchanged | Skipped |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
`)
	fmt.Fprintf(
		buf,
		"| %d | %d | %d | %d | %d | %d | %d |\n",
		rep.Summary.Total,
		rep.Summary.Changed,
		rep.Summary.Added,
		rep.Summary.Removed,
		rep.Summary.Errored,
		rep.Summary.Unchanged,
		rep.Summary.Skipped,
	)
}

func writeErrorSection(buf *bytes.Buffer, rep ComparisonReport) {
	errored := filterByStatus(rep.Items, StatusErrored)
	if len(errored) == 0 {
		return
	}
	buf.WriteByte('\n')
	buf.WriteString(`### ❌ Capture / Runtime Errors

| Story | Error |
| --- | --- |
`)
	for i, item := range errored {
		if i >= defaultMaxListedItems {
			buf.WriteByte('\n')
			fmt.Fprintf(buf, "_...and %d more errored stories._\n", len(errored)-i)
			break
		}
		fmt.Fprintf(buf, "| `%s` | %s |\n", escapeMarkdownCell(displayName(item)), escapeMarkdownCell(item.Error))
	}
}

func writeChangedSection(buf *bytes.Buffer, rep ComparisonReport, meta PRCommentMeta) {
	changed := filterByStatus(rep.Items, StatusChanged)
	if len(changed) == 0 {
		return
	}
	buf.WriteByte('\n')
	buf.WriteString(`### 🔍 Changed Stories

| Story | Diff | Dimensions | Links |
| --- | ---: | --- | --- |
`)
	for i, item := range changed {
		if i >= defaultMaxListedItems {
			buf.WriteByte('\n')
			fmt.Fprintf(buf, "_...and %d more changed stories (see HTML report for full list)._\n", len(changed)-i)
			break
		}
		dims := fmt.Sprintf("%dx%d", item.AfterWidth, item.AfterHeight)
		if item.SizeMismatch {
			dims = fmt.Sprintf("%dx%d → %dx%d", item.BeforeWidth, item.BeforeHeight, item.AfterWidth, item.AfterHeight)
		}
		links := formatStoryLinks(meta.ReportBaseURL, item)
		fmt.Fprintf(buf, "| `%s` | %.2f%% (%d px) | %s | %s |\n",
			escapeMarkdownCell(displayName(item)),
			item.DiffRatio*100,
			item.DiffPixels,
			dims,
			links,
		)
	}

	if meta.InlineImages && meta.ReportBaseURL != "" {
		limit := min(len(changed), defaultMaxInlineImages)
		buf.WriteByte('\n')
		buf.WriteString("<details>\n")
		fmt.Fprintf(buf, "<summary>Inline Before | Diff | After Previews (top %d changed)</summary>\n", limit)
		buf.WriteByte('\n')
		for _, item := range changed[:limit] {
			if item.CompositeRelPath == "" {
				continue
			}
			imgURL := joinReportURL(meta.ReportBaseURL, item.CompositeRelPath)
			fmt.Fprintf(buf, "**%s** (%.2f%%)\n", displayName(item), item.DiffRatio*100)
			buf.WriteByte('\n')
			fmt.Fprintf(buf, "![%s](%s)\n", displayName(item), imgURL)
			buf.WriteByte('\n')
		}
		buf.WriteString("</details>\n")
	}
}

func writeAddedOrRemovedSection(buf *bytes.Buffer, heading string, status Status, rep ComparisonReport, meta PRCommentMeta) {
	matched := filterByStatus(rep.Items, status)
	if len(matched) == 0 {
		return
	}
	buf.WriteByte('\n')
	buf.WriteString("<details>\n")
	fmt.Fprintf(buf, "<summary>%s (%d)</summary>\n", heading, len(matched))
	buf.WriteByte('\n')
	for i, item := range matched {
		if i >= defaultMaxListedItems {
			buf.WriteByte('\n')
			fmt.Fprintf(buf, "_...and %d more._\n", len(matched)-i)
			break
		}
		rel := item.AfterRelPath
		if status == StatusRemoved {
			rel = item.BeforeRelPath
		}
		if meta.ReportBaseURL != "" && rel != "" {
			fmt.Fprintf(buf, "- [`%s`](%s)\n", displayName(item), joinReportURL(meta.ReportBaseURL, rel))
		} else {
			fmt.Fprintf(buf, "- `%s`\n", displayName(item))
		}
	}
	buf.WriteByte('\n')
	buf.WriteString("</details>\n")
}

func formatStoryLinks(baseURL string, item StoryComparison) string {
	if baseURL == "" {
		return "—"
	}
	var links []string
	if item.CompositeRelPath != "" {
		links = append(links, fmt.Sprintf("[Composite](%s)", joinReportURL(baseURL, item.CompositeRelPath)))
	}
	if item.BeforeRelPath != "" {
		links = append(links, fmt.Sprintf("[Before](%s)", joinReportURL(baseURL, item.BeforeRelPath)))
	}
	if item.DiffRelPath != "" {
		links = append(links, fmt.Sprintf("[Diff](%s)", joinReportURL(baseURL, item.DiffRelPath)))
	}
	if item.AfterRelPath != "" {
		links = append(links, fmt.Sprintf("[After](%s)", joinReportURL(baseURL, item.AfterRelPath)))
	}
	if len(links) == 0 {
		return "—"
	}
	return strings.Join(links, " · ")
}

func filterByStatus(items []StoryComparison, status Status) []StoryComparison {
	var out []StoryComparison
	for _, item := range items {
		if item.Status == status {
			out = append(out, item)
		}
	}
	return out
}

func displayName(item StoryComparison) string {
	if item.Title != "" && item.Name != "" {
		return item.Title + " / " + item.Name
	}
	return item.ID
}

func joinReportURL(baseURL string, relPath string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimPrefix(path.Clean("/"+relPath), "/")
}

func escapeMarkdownCell(s string) string {
	cleaned := strings.ReplaceAll(s, "\r\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "|", `\|`)
	if len(cleaned) > 200 {
		return cleaned[:197] + "..."
	}
	return cleaned
}

func finalizeComment(body string, meta PRCommentMeta) string {
	footer := formatCommentFooter(meta)
	maxBytes := cmp.Or(meta.MaxBytes, defaultMaxCommentBytes)
	return truncateCommentBody(body, maxBytes-len(footer)) + footer
}

func formatCommentFooter(meta PRCommentMeta) string {
	var parts []string
	if sha := strings.TrimSpace(meta.CommitSHA); sha != "" {
		parts = append(parts, fmt.Sprintf("Commit: `%s`", sha[:min(len(sha), 12)]))
	}
	if base := strings.TrimSpace(meta.BaseSHA); base != "" {
		label := fmt.Sprintf("Base: `%s`", base[:min(len(base), 12)])
		if branch := strings.TrimSpace(meta.BaseBranch); branch != "" {
			label = fmt.Sprintf("Base (`%s`): `%s`", branch, base[:min(len(base), 12)])
		}
		parts = append(parts, label)
	}
	if url := strings.TrimSpace(meta.BuildURL); url != "" {
		parts = append(parts, fmt.Sprintf("[Cloud Build Logs](%s)", url))
	}
	if len(parts) == 0 {
		return ""
	}
	return `
---
_` + strings.Join(parts, " · ") + "_\n"
}

func truncateCommentBody(body string, maxBytes int) string {
	const suffix = `

_(Report truncated to fit GitHub Check Run payload — open the GCS HTML report for full details.)_
`
	if maxBytes <= len(suffix) || len(body) <= maxBytes {
		return body
	}
	prefix := strings.ToValidUTF8(body[:maxBytes-len(suffix)], "")
	return prefix + suffix
}
