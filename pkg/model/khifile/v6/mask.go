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

package khifilev6

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	pbv6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"google.golang.org/protobuf/proto"
)

const (
	// internPoolProgressReportStride defines how many strings are processed between progress updates and cancellation checks.
	internPoolProgressReportStride = 8192

	// internPoolReplaceProgressWeight is the share of an intern pool chunk's progress allocated to string replacement before compression.
	internPoolReplaceProgressWeight = 0.85
)

// MaskRule specifies a substring replacement rule applied across KHI metadata and string pools.
type MaskRule struct {
	Target      string
	Replacement string
}

// DefaultMaskReplacement formats the 1-based sequential placeholder string ("<masked_1>", "<masked_2>", ...)
// used when a masking rule does not specify a custom replacement.
func DefaultMaskReplacement(index int) string {
	if index < 1 {
		index = 1
	}
	return fmt.Sprintf("<masked_%d>", index)
}

// MaskReplacer performs longest-match-first single-pass substring masking while preserving null field-path delimiters.
type MaskReplacer struct {
	replacer *strings.Replacer
}

// NewMaskReplacer constructs a MaskReplacer from the provided rules, ordering targets by descending rune length.
func NewMaskReplacer(rules []MaskRule) *MaskReplacer {
	if len(rules) == 0 {
		return &MaskReplacer{}
	}

	dedup := make(map[string]string, len(rules))
	for _, r := range rules {
		if r.Target == "" {
			continue
		}
		dedup[r.Target] = r.Replacement
	}
	if len(dedup) == 0 {
		return &MaskReplacer{}
	}

	normalized := make([]MaskRule, 0, len(dedup))
	for target, replacement := range dedup {
		normalized = append(normalized, MaskRule{
			Target:      target,
			Replacement: replacement,
		})
	}

	slices.SortFunc(normalized, func(a, b MaskRule) int {
		runeDiff := utf8.RuneCountInString(b.Target) - utf8.RuneCountInString(a.Target)
		if runeDiff != 0 {
			return runeDiff
		}
		byteDiff := len(b.Target) - len(a.Target)
		if byteDiff != 0 {
			return byteDiff
		}
		return cmp.Compare(a.Target, b.Target)
	})

	pairs := make([]string, 0, len(normalized)*2)
	for _, r := range normalized {
		pairs = append(pairs, r.Target, r.Replacement)
	}

	return &MaskReplacer{
		replacer: strings.NewReplacer(pairs...),
	}
}

// HasRules reports whether at least one non-empty target masking rule is configured.
func (m *MaskReplacer) HasRules() bool {
	return m != nil && m.replacer != nil
}

// Replace applies all masking rules in a single pass while preserving null byte separators in interned field paths.
func (m *MaskReplacer) Replace(value string) string {
	if m == nil || m.replacer == nil || value == "" {
		return value
	}
	if !strings.Contains(value, "\x00") {
		return m.replacer.Replace(value)
	}
	segments := strings.Split(value, "\x00")
	for i, seg := range segments {
		if seg != "" {
			segments[i] = m.replacer.Replace(seg)
		}
	}
	return strings.Join(segments, "\x00")
}

// formatMaskedInspectionName masks an inspection title and appends a "(Masked)" suffix when rules are active.
func formatMaskedInspectionName(rawName string, replacer *MaskReplacer) string {
	masked := strings.TrimSpace(replacer.Replace(rawName))
	if !replacer.HasRules() {
		return masked
	}
	if masked == "" {
		return "Masked Inspection"
	}
	if strings.HasSuffix(masked, "(Masked)") {
		return masked
	}
	return masked + " (Masked)"
}

// sanitizeSuggestedFilename strips angle brackets from "<masked_N>" placeholders and replaces filesystem-unsafe characters.
func sanitizeSuggestedFilename(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r == '<' || r == '>':
			continue
		case r < 0x20 || strings.ContainsRune(`:"/\|?*`, r):
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// formatMaskedSuggestedFilename masks a suggested .khi filename and ensures a filesystem-safe, non-empty "-masked.khi" filename.
func formatMaskedSuggestedFilename(rawFilename string, replacer *MaskReplacer) string {
	if !replacer.HasRules() {
		return strings.TrimSpace(rawFilename)
	}
	masked := sanitizeSuggestedFilename(replacer.Replace(rawFilename))
	if masked == "" || strings.EqualFold(masked, ".khi") {
		return "masked-inspection.khi"
	}
	lower := strings.ToLower(masked)
	if strings.HasSuffix(lower, "-masked.khi") {
		return masked
	}
	if strings.HasSuffix(lower, ".khi") {
		base := strings.TrimSpace(masked[:len(masked)-4])
		if base == "" {
			return "masked-inspection.khi"
		}
		return base + "-masked.khi"
	}
	return masked + "-masked.khi"
}

// MaskExportProgressCallback receives progress updates in the [0.0, 1.0] range during masked KHI archive export.
type MaskExportProgressCallback func(progress float64, message string) error

// maskCountingReader wraps an io.Reader and tracks the total number of compressed bytes read.
type maskCountingReader struct {
	r         io.Reader
	bytesRead int64
}

func (c *maskCountingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.bytesRead += int64(n)
	return n, err
}

// ExportMaskedKHIFile streams a source KHI v6 archive into w while rewriting metadata and interned string pools.
// Log, timeline, and style chunks are copied without decompression.
func ExportMaskedKHIFile(
	ctx context.Context,
	r io.Reader,
	totalSize int64,
	w io.Writer,
	replacer *MaskReplacer,
	onProgress MaskExportProgressCallback,
) error {
	lastProgress := 0.0
	emitProgress := func(progress float64, message string) error {
		if onProgress == nil {
			return nil
		}
		if progress < lastProgress {
			progress = lastProgress
		}
		if progress > 1.0 {
			progress = 1.0
		}
		lastProgress = progress
		return onProgress(progress, message)
	}
	byteFraction := func(bytes int64) float64 {
		if totalSize <= 0 {
			return lastProgress
		}
		ratio := float64(bytes) / float64(totalSize)
		if ratio > 1.0 {
			ratio = 1.0
		}
		return ratio
	}

	if err := emitProgress(0.0, "Reading KHI archive header..."); err != nil {
		return err
	}

	cr := &maskCountingReader{r: r}
	reader, err := NewReader(cr)
	if err != nil {
		return fmt.Errorf("failed to open KHI reader: %w", err)
	}
	writer, err := NewWriter(w)
	if err != nil {
		return fmt.Errorf("failed to initialize KHI writer: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		chunkStartBytes := cr.bytesRead
		rawChunk, err := reader.NextRawChunk()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read source chunk: %w", err)
		}
		chunkEndBytes := cr.bytesRead

		startFrac := byteFraction(chunkStartBytes)
		endFrac := byteFraction(chunkEndBytes)

		if !replacer.HasRules() {
			if err := copyPassthroughChunk(rawChunk, writer, endFrac, "Copying KHI archive chunks...", emitProgress); err != nil {
				return err
			}
			continue
		}

		switch rawChunk.Type {
		case ChunkTypeMetadata:
			if err := writeMaskedMetadataChunk(rawChunk, writer, replacer, endFrac, emitProgress); err != nil {
				return err
			}
		case ChunkTypeInternPool, ChunkTypeServerInternPool:
			if err := writeMaskedInternPoolChunk(
				ctx,
				rawChunk,
				writer,
				replacer,
				startFrac,
				endFrac,
				emitProgress,
			); err != nil {
				return err
			}
		default:
			if err := copyPassthroughChunk(rawChunk, writer, endFrac, "Copying log and timeline chunks...", emitProgress); err != nil {
				return err
			}
		}
	}

	return emitProgress(1.0, "Completed masked KHI archive stream.")
}

// writeMaskedMetadataChunk rewrites inspection titles, suggested filenames, and query strings in a metadata chunk.
func writeMaskedMetadataChunk(
	rawChunk *RawChunk,
	writer *Writer,
	replacer *MaskReplacer,
	endFrac float64,
	emitProgress MaskExportProgressCallback,
) error {
	var metaChunk pbv6.MetadataChunk
	if err := rawChunk.DecompressWith(func(data []byte) error {
		return proto.Unmarshal(data, &metaChunk)
	}); err != nil {
		return fmt.Errorf("failed to decode metadata chunk: %w", err)
	}
	for _, item := range metaChunk.Metadata {
		if header := item.GetHeader(); header != nil {
			header.InspectionName = proto.String(formatMaskedInspectionName(header.GetInspectionName(), replacer))
			header.SuggestedFilename = proto.String(formatMaskedSuggestedFilename(header.GetSuggestedFilename(), replacer))
		}
		if queryMeta := item.GetQuery(); queryMeta != nil {
			for _, q := range queryMeta.Queries {
				if q == nil {
					continue
				}
				q.Name = proto.String(replacer.Replace(q.GetName()))
				q.Query = proto.String(replacer.Replace(q.GetQuery()))
			}
		}
	}
	if err := writer.WriteChunk(ChunkTypeMetadata, &metaChunk); err != nil {
		return fmt.Errorf("failed to write masked metadata chunk: %w", err)
	}
	return emitProgress(endFrac, "Masking inspection metadata...")
}

// writeMaskedInternPoolChunk masks all strings in an intern pool chunk and writes the compressed chunk.
func writeMaskedInternPoolChunk(
	ctx context.Context,
	rawChunk *RawChunk,
	writer *Writer,
	replacer *MaskReplacer,
	startFrac float64,
	endFrac float64,
	emitProgress MaskExportProgressCallback,
) error {
	var poolChunk pbv6.InterningPoolChunk
	if err := rawChunk.DecompressWith(func(data []byte) error {
		return proto.Unmarshal(data, &poolChunk)
	}); err != nil {
		return fmt.Errorf("failed to decode intern pool chunk: %w", err)
	}

	poolLabel := "Masking string pool"
	if rawChunk.Type == ChunkTypeServerInternPool {
		poolLabel = "Masking structured log strings"
	}
	totalStrings := len(poolChunk.Strings)

	for i, str := range poolChunk.Strings {
		if i%internPoolProgressReportStride == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if totalStrings > 0 {
				frac := startFrac + (endFrac-startFrac)*internPoolReplaceProgressWeight*(float64(i)/float64(totalStrings))
				if err := emitProgress(frac, fmt.Sprintf("%s (%d / %d strings)...", poolLabel, i, totalStrings)); err != nil {
					return err
				}
			}
		}
		if str == nil || str.Value == nil {
			continue
		}
		maskedVal := replacer.Replace(*str.Value)
		str.Value = &maskedVal
	}
	if err := emitProgress(startFrac+(endFrac-startFrac)*internPoolReplaceProgressWeight, fmt.Sprintf("Compressing masked string pool (%d strings)...", totalStrings)); err != nil {
		return err
	}
	if err := writer.WriteChunk(rawChunk.Type, &poolChunk); err != nil {
		return fmt.Errorf("failed to write masked intern pool chunk: %w", err)
	}
	return emitProgress(endFrac, fmt.Sprintf("%s (%d / %d strings)...", poolLabel, totalStrings, totalStrings))
}

// copyPassthroughChunk writes a raw chunk without decompression and reports progress.
func copyPassthroughChunk(
	rawChunk *RawChunk,
	writer *Writer,
	endFrac float64,
	message string,
	emitProgress MaskExportProgressCallback,
) error {
	if err := writer.WriteRawChunk(rawChunk); err != nil {
		return fmt.Errorf("failed to copy raw chunk type %d: %w", rawChunk.Type, err)
	}
	return emitProgress(endFrac, message)
}
