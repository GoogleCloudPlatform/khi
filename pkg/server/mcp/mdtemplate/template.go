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

package mdtemplate

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"strings"
	"text/template"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Code encloses s in markdown backticks, handling embedded backticks per CommonMark.
func Code(s string) string {
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	maxRun := 0
	currentRun := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '`' {
			currentRun++
			if currentRun > maxRun {
				maxRun = currentRun
			}
		} else {
			currentRun = 0
		}
	}
	delim := strings.Repeat("`", maxRun+1)
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return delim + " " + s + " " + delim
	}
	return delim + s + delim
}

// Cell escapes pipes and converts newlines to spaces for markdown table cells.
func Cell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// FormatTime formats t in UTC RFC3339, or returns an empty string if t is zero.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// Percent formats ratio as a rounded integer percentage.
func Percent(ratio float64) string {
	return fmt.Sprintf("%.0f%%", math.Round(ratio*100))
}

// Fence wraps body in a markdown fenced code block with the given language.
func Fence(lang, body string) string {
	maxRun := 0
	currentRun := 0
	for i := 0; i < len(body); i++ {
		if body[i] == '`' {
			currentRun++
			if currentRun > maxRun {
				maxRun = currentRun
			}
		} else {
			currentRun = 0
		}
	}
	fenceLen := max(3, maxRun+1)
	fence := strings.Repeat("`", fenceLen)
	trimmedBody := strings.TrimSuffix(strings.TrimSuffix(body, "\n"), "\r")
	return fence + lang + "\n" + trimmedBody + "\n" + fence
}

// PageFooter formats the pagination footer containing nextPageToken, or returns an empty string if empty.
func PageFooter(nextPageToken string) string {
	if nextPageToken == "" {
		return ""
	}
	return fmt.Sprintf("pageToken: %q", nextPageToken)
}

// DefaultFuncMap returns the default template functions for markdown templates.
func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"code":       Code,
		"cell":       Cell,
		"time":       FormatTime,
		"percent":    Percent,
		"fence":      Fence,
		"pageFooter": PageFooter,
	}
}

// Set wraps a parsed set of markdown text templates.
type Set struct {
	tmpl *template.Template
}

// Parse parses templates from fsys matching pattern with extraFuncs and DefaultFuncMap.
func Parse(fsys fs.FS, pattern string, extraFuncs ...template.FuncMap) (*Set, error) {
	tmpl := template.New("mdtemplate").Funcs(DefaultFuncMap())
	for _, extra := range extraFuncs {
		tmpl.Funcs(extra)
	}
	parsed, err := tmpl.ParseFS(fsys, pattern)
	if err != nil {
		return nil, err
	}
	return &Set{tmpl: parsed}, nil
}

// MustParse parses templates from fsys matching pattern and panics on error.
func MustParse(fsys fs.FS, pattern string, extraFuncs ...template.FuncMap) *Set {
	s, err := Parse(fsys, pattern, extraFuncs...)
	if err != nil {
		panic(err)
	}
	return s
}

// Render executes templateName with data and trims trailing \r and \n.
func (s *Set) Render(templateName string, data any) (string, error) {
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\r\n"), nil
}

// ToolResult renders templateName with data and wraps it into an MCP CallToolResult.
func (s *Set) ToolResult(templateName string, data any) (*mcp.CallToolResult, any, error) {
	text, err := s.Render(templateName, data)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, nil, nil
}

// ErrorToolResult renders templateName with data and wraps it into an MCP CallToolResult with IsError set to true.
func (s *Set) ErrorToolResult(templateName string, data any) (*mcp.CallToolResult, any, error) {
	res, extra, err := s.ToolResult(templateName, data)
	if err != nil {
		return nil, nil, err
	}
	res.IsError = true
	return res, extra, nil
}

// ResourceResult renders templateName with data and wraps it into an MCP ReadResourceResult.
func (s *Set) ResourceResult(uri, templateName string, data any) (*mcp.ReadResourceResult, error) {
	text, err := s.Render(templateName, data)
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: "text/markdown",
				Text:     text,
			},
		},
	}, nil
}

// FormatError formats an error code and optional bullet details into markdown.
func FormatError(code string, bullets ...string) string {
	if len(bullets) == 0 {
		return "Error: " + code
	}
	var sb strings.Builder
	sb.WriteString("Error: ")
	sb.WriteString(code)
	sb.WriteString("\n\n")
	for i, bullet := range bullets {
		sb.WriteString("- ")
		sb.WriteString(bullet)
		if i < len(bullets)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// ErrorResult formats an error and wraps it into an MCP CallToolResult with IsError set to true.
func ErrorResult(code string, bullets ...string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: FormatError(code, bullets...),
			},
		},
	}, nil, nil
}

type pageTokenPayload struct {
	Offset int `json:"offset"`
}

// EncodePageToken serializes offset into a base64url encoded page token.
func EncodePageToken(offset int) string {
	data, _ := json.Marshal(pageTokenPayload{Offset: offset})
	return base64.RawURLEncoding.EncodeToString(data)
}

// DecodePageToken decodes a base64url encoded page token and returns the offset.
func DecodePageToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("invalid page token encoding: %w", err)
	}
	var payload pageTokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0, fmt.Errorf("invalid page token payload: %w", err)
	}
	if payload.Offset < 0 {
		return 0, fmt.Errorf("invalid negative page token offset: %d", payload.Offset)
	}
	return payload.Offset, nil
}

// PageResult contains the paginated slice of items and pagination metadata.
type PageResult[T any] struct {
	// Items contains the items for the current page.
	Items []T
	// Start is the 1-based display index of the first item on the current page, or 0 if Items is empty.
	Start int
	// End is the 1-based inclusive display index of the last item on the current page, or 0 if Items is empty.
	End int
	// Total is the total number of items across all pages.
	Total int
	// NextPageToken is the opaque token for fetching the next page, or empty on the final page.
	NextPageToken string
}

// Paginate slices items according to pagination parameters and returns a PageResult.
func Paginate[T any](items []T, pageSize, defaultPageSize, maxPageSize int, pageToken string) (PageResult[T], error) {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if maxPageSize > 0 && pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	offset, err := DecodePageToken(pageToken)
	if err != nil {
		return PageResult[T]{}, err
	}
	total := len(items)
	startOffset := offset
	if startOffset > total {
		startOffset = total
	}
	endOffset := startOffset + pageSize
	if endOffset > total {
		endOffset = total
	}
	sliced := items[startOffset:endOffset]
	var start, end int
	if len(sliced) > 0 {
		start = startOffset + 1
		end = endOffset
	}
	var nextToken string
	if endOffset < total {
		nextToken = EncodePageToken(endOffset)
	}
	return PageResult[T]{
		Items:         sliced,
		Start:         start,
		End:           end,
		Total:         total,
		NextPageToken: nextToken,
	}, nil
}
