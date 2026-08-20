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

package workbench

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

// ProgressCallback receives streaming progress updates during dataset loading.
type ProgressCallback func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error

// countingReader wraps an io.Reader and tracks the total number of bytes read.
type countingReader struct {
	r         io.Reader
	bytesRead int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.bytesRead += int64(n)
	return n, err
}

// formatByteSize formats a byte count into a human-readable string with units (B, KB, MB, GB, etc.).
func formatByteSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

type rawChunkTask struct {
	seq            int
	compressedSize int64
	chunk          *khifilev6model.RawChunk
}

type parsedChunkResult struct {
	seq            int
	compressedSize int64
	chunkType      khifilev6model.ChunkType
	metadata       *khifilev6.MetadataChunk
	internPool     *khifilev6.InterningPoolChunk
	style          *khifilev6.TimelineStyleChunk
	log            *khifilev6.LogChunk
	timeline       *khifilev6.TimelineChunk
	err            error
}

// NewFromReader creates and initializes a Workbench instance by parsing chunks from the given reader in parallel.
func NewFromReader(
	ctx context.Context,
	id string,
	inspectionID string,
	reader io.Reader,
	totalSize int64,
	onProgress ProgressCallback,
) (*Workbench, error) {
	if onProgress == nil {
		onProgress = func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error {
			return nil
		}
	}

	cr := &countingReader{r: reader}
	khiReader, err := khifilev6model.NewReader(cr)
	if err != nil {
		return nil, fmt.Errorf("failed to create KHI file reader: %w", err)
	}

	wb := NewWorkbench(id, inspectionID)

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers < 1 {
		numWorkers = 1
	}

	tasks := make(chan rawChunkTask, numWorkers*2)
	results := make(chan parsedChunkResult, numWorkers*2)

	g, gCtx := errgroup.WithContext(ctx)

	// Goroutine 1: Sequential I/O reader from stream
	g.Go(func() error {
		defer close(tasks)
		seq := 0
		for {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}

			rawChunk, err := khiReader.NextRawChunk()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to read next chunk: %w", err)
			}

			compressedSize := int64(len(rawChunk.Data)) + 8
			select {
			case tasks <- rawChunkTask{seq: seq, compressedSize: compressedSize, chunk: rawChunk}:
				seq++
			case <-gCtx.Done():
				return gCtx.Err()
			}
		}
	})

	// Goroutine 2..N+1: Parallel worker pool for decompression and proto unmarshaling
	var workerWg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for task := range tasks {
				select {
				case <-gCtx.Done():
					return
				default:
				}

				decompressed, err := task.chunk.Decompress()
				if err != nil {
					select {
					case results <- parsedChunkResult{seq: task.seq, err: fmt.Errorf("failed to decompress chunk #%d: %w", task.seq, err)}:
					case <-gCtx.Done():
					}
					return
				}

				res := parsedChunkResult{seq: task.seq, compressedSize: task.compressedSize, chunkType: decompressed.Type}
				switch decompressed.Type {
				case khifilev6model.ChunkTypeMetadata:
					var meta khifilev6.MetadataChunk
					if err := proto.Unmarshal(decompressed.Data, &meta); err != nil {
						res.err = fmt.Errorf("failed to unmarshal metadata chunk #%d: %w", task.seq, err)
					} else {
						res.metadata = &meta
					}
				case khifilev6model.ChunkTypeInternPool, khifilev6model.ChunkTypeServerInternPool:
					var pool khifilev6.InterningPoolChunk
					if err := proto.Unmarshal(decompressed.Data, &pool); err != nil {
						res.err = fmt.Errorf("failed to unmarshal intern pool chunk #%d: %w", task.seq, err)
					} else {
						res.internPool = &pool
					}
				case khifilev6model.ChunkTypeTimelineStyle:
					var style khifilev6.TimelineStyleChunk
					if err := proto.Unmarshal(decompressed.Data, &style); err != nil {
						res.err = fmt.Errorf("failed to unmarshal timeline style chunk #%d: %w", task.seq, err)
					} else {
						res.style = &style
					}
				case khifilev6model.ChunkTypeLog:
					var logChunk khifilev6.LogChunk
					if err := proto.Unmarshal(decompressed.Data, &logChunk); err != nil {
						res.err = fmt.Errorf("failed to unmarshal log chunk #%d: %w", task.seq, err)
					} else {
						res.log = &logChunk
					}
				case khifilev6model.ChunkTypeTimeline:
					var timelineChunk khifilev6.TimelineChunk
					if err := proto.Unmarshal(decompressed.Data, &timelineChunk); err != nil {
						res.err = fmt.Errorf("failed to unmarshal timeline chunk #%d: %w", task.seq, err)
					} else {
						res.timeline = &timelineChunk
					}
				}

				select {
				case results <- res:
				case <-gCtx.Done():
					return
				}
			}
		}()
	}

	go func() {
		workerWg.Wait()
		close(results)
	}()

	// Goroutine N+2: In-order chunk collector and workbench ingester
	g.Go(func() error {
		nextExpectedSeq := 0
		buffered := make(map[int]parsedChunkResult)
		var processedBytes int64

		for res := range results {
			if res.err != nil {
				return res.err
			}
			buffered[res.seq] = res
			for {
				nextRes, ok := buffered[nextExpectedSeq]
				if !ok {
					break
				}
				delete(buffered, nextExpectedSeq)
				if err := wb.ingestParsedChunk(&nextRes); err != nil {
					return err
				}
				processedBytes += nextRes.compressedSize
				if totalSize > 0 {
					cur := processedBytes
					if cur > totalSize {
						cur = totalSize
					}
					pct := 10.0 + (float64(cur)/float64(totalSize))*40.0
					if pct > 50.0 {
						pct = 50.0
					}
					msg := fmt.Sprintf("Parsing dataset chunks (%s / %s)...", formatByteSize(cur), formatByteSize(totalSize))
					if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_PARSING_CHUNKS, pct, msg); err != nil {
						return err
					}
				}
				nextExpectedSeq++
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_INDEXING_DATA, 0.0, "Building base search index..."); err != nil {
		return nil, err
	}
	searchIndex, err := wb.BuildBaseSearchIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to build base search index: %w", err)
	}
	if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_INDEXING_DATA, 100.0, "Base search index ready."); err != nil {
		return nil, err
	}
	wb.searchIndex = searchIndex
	wb.logChunks = nil
	wb.timelineChunks = nil

	return wb, nil
}

func (w *Workbench) ingestParsedChunk(res *parsedChunkResult) error {
	switch res.chunkType {
	case khifilev6model.ChunkTypeMetadata:
		if res.metadata != nil {
			w.metadataChunks = append(w.metadataChunks, res.metadata)
		}
	case khifilev6model.ChunkTypeInternPool, khifilev6model.ChunkTypeServerInternPool:
		if res.internPool != nil {
			w.internPool.IngestChunk(res.internPool)
		}
	case khifilev6model.ChunkTypeTimelineStyle:
		if res.style != nil {
			w.styleChunk = res.style
		}
	case khifilev6model.ChunkTypeLog:
		if res.log != nil {
			w.logChunks = append(w.logChunks, res.log)
		}
	case khifilev6model.ChunkTypeTimeline:
		if res.timeline != nil {
			w.timelineChunks = append(w.timelineChunks, res.timeline)
		}
	}
	return nil
}
