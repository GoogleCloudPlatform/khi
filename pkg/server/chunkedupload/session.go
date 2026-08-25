package chunkedupload

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/ttlcleaner"
)

var (
	// ErrSessionNotFound is returned when a session token is not found or has expired.
	ErrSessionNotFound = errors.New("chunk upload session not found or expired")
	// ErrInvalidTotalSize is returned when the specified total file size is non-positive.
	ErrInvalidTotalSize = errors.New("total file size must be positive")
	// ErrEmptyChunkData is returned when an uploaded chunk contains empty data payload.
	ErrEmptyChunkData = errors.New("chunk data must not be empty")
	// ErrInvalidOffset is returned when the chunk byte offset is invalid.
	ErrInvalidOffset = errors.New("invalid chunk byte offset")
	// ErrChunkSizeTooLarge is returned when a single chunk exceeds the maximum permitted size limit.
	ErrChunkSizeTooLarge = errors.New("chunk size exceeds maximum allowed limit")
)

const (
	// DefaultSuggestedChunkSize is the recommended chunk payload size (25MB).
	DefaultSuggestedChunkSize = 25 * 1024 * 1024
	// DefaultMaxChunkSize is the maximum allowed single chunk payload (32MB).
	DefaultMaxChunkSize = 32 * 1024 * 1024
	// DefaultSessionTTL is the duration after which an inactive chunk upload session expires.
	DefaultSessionTTL = 30 * time.Minute
	// DefaultCleanupInterval is the frequency at which the cleaner checks for expired sessions.
	DefaultCleanupInterval = 1 * time.Minute
)

// ByteRange represents an uploaded byte interval [Start, End).
type ByteRange struct {
	Start int64
	End   int64
}

// ChunkSession represents an in-progress chunked upload session.
type ChunkSession struct {
	Token          string
	FileName       string
	TotalSize      int64
	ReceivedBytes  int64
	ReceivedRanges []ByteRange
	TempFilePath   string
	TempFile       *os.File
	ExpiresAt      time.Time
	mu             sync.Mutex
}

// ChunkSessionOption is a functional option for configuring ChunkSessionManager.
type ChunkSessionOption func(*ChunkSessionManager)

// WithSessionTTL configures the session expiration TTL.
func WithSessionTTL(ttl time.Duration) ChunkSessionOption {
	return func(m *ChunkSessionManager) {
		m.ttl = ttl
	}
}

// WithMaxChunkSize configures the maximum permitted single chunk size in bytes.
func WithMaxChunkSize(maxChunkSize int64) ChunkSessionOption {
	return func(m *ChunkSessionManager) {
		m.maxChunkSize = maxChunkSize
	}
}

// WithSuggestedChunkSize configures the recommended chunk size in bytes.
func WithSuggestedChunkSize(suggestedChunkSize int64) ChunkSessionOption {
	return func(m *ChunkSessionManager) {
		m.suggestedChunkSize = suggestedChunkSize
	}
}

// WithTokenGenerator configures the token generator for session tokens.
func WithTokenGenerator(generator idgenerator.IDGenerator) ChunkSessionOption {
	return func(m *ChunkSessionManager) {
		m.tokenGenerator = generator
	}
}

// ChunkSessionManager manages chunked upload sessions and temporary storage.
type ChunkSessionManager struct {
	uploadDir          string
	tokenGenerator     idgenerator.IDGenerator
	sessions           map[string]*ChunkSession
	ttl                time.Duration
	maxChunkSize       int64
	suggestedChunkSize int64
	cleaner            *ttlcleaner.TTLCleaner[string]
	mu                 sync.RWMutex
}

var _ ttlcleaner.ExpirableTarget[string] = (*ChunkSessionManager)(nil)

// NewChunkSessionManager creates a new ChunkSessionManager instance.
func NewChunkSessionManager(uploadDir string, opts ...ChunkSessionOption) *ChunkSessionManager {
	if uploadDir == "" {
		uploadDir = os.TempDir()
	}
	m := &ChunkSessionManager{
		uploadDir:          uploadDir,
		tokenGenerator:     idgenerator.NewPrefixIDGenerator("chunk-upload-"),
		sessions:           make(map[string]*ChunkSession),
		ttl:                DefaultSessionTTL,
		maxChunkSize:       DefaultMaxChunkSize,
		suggestedChunkSize: DefaultSuggestedChunkSize,
	}
	for _, opt := range opts {
		opt(m)
	}
	m.cleaner = ttlcleaner.NewTTLCleaner[string](m, DefaultCleanupInterval)
	m.cleaner.Start()
	return m
}

// Close stops the background cleaner and cleans up active sessions.
func (m *ChunkSessionManager) Close() {
	if m.cleaner != nil {
		m.cleaner.Stop()
	}
}

// SuggestedChunkSize returns the recommended chunk size in bytes.
func (m *ChunkSessionManager) SuggestedChunkSize() int64 {
	return m.suggestedChunkSize
}

// StartSession initializes a new chunk upload session and creates a temporary file to store chunks.
func (m *ChunkSessionManager) StartSession(fileName string, totalSize int64) (*ChunkSession, error) {
	if totalSize <= 0 {
		return nil, ErrInvalidTotalSize
	}

	if err := os.MkdirAll(m.uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	tempFile, err := os.CreateTemp(m.uploadDir, "khi-chunk-*.part")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary chunk file: %w", err)
	}

	token := m.tokenGenerator.Generate()
	session := &ChunkSession{
		Token:          token,
		FileName:       fileName,
		TotalSize:      totalSize,
		ReceivedBytes:  0,
		ReceivedRanges: make([]ByteRange, 0),
		TempFilePath:   tempFile.Name(),
		TempFile:       tempFile,
		ExpiresAt:      time.Now().Add(m.ttl),
	}

	m.mu.Lock()
	m.sessions[token] = session
	m.mu.Unlock()

	return session, nil
}

// WriteChunk writes a chunk of data at the specified byte offset to the session's temporary file.
func (m *ChunkSessionManager) WriteChunk(token string, offset int64, data []byte) (int64, error) {
	m.mu.RLock()
	session, exists := m.sessions[token]
	m.mu.RUnlock()

	if !exists {
		return 0, ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if int64(len(data)) > m.maxChunkSize {
		return 0, ErrChunkSizeTooLarge
	}

	if offset < 0 {
		return 0, fmt.Errorf("%w: offset must be non-negative", ErrInvalidOffset)
	}

	if len(data) == 0 {
		return 0, ErrEmptyChunkData
	}

	if offset+int64(len(data)) > session.TotalSize {
		return 0, fmt.Errorf("%w: chunk end %d exceeds total size %d", ErrInvalidOffset, offset+int64(len(data)), session.TotalSize)
	}

	n, err := session.TempFile.WriteAt(data, offset)
	if err != nil {
		return 0, fmt.Errorf("failed to write chunk to temporary file: %w", err)
	}
	session.ReceivedRanges = append(session.ReceivedRanges, ByteRange{
		Start: offset,
		End:   offset + int64(n),
	})
	session.ReceivedBytes += int64(n)
	session.ExpiresAt = time.Now().Add(m.ttl)

	return session.ReceivedBytes, nil
}

// FinalizeSession validates chunk completeness, closes the temporary file, moves it to destinationPath (if provided),
// and removes the session from the manager. If destinationPath is empty, it closes the file and returns the temp file path.
func (m *ChunkSessionManager) FinalizeSession(token string, destinationPath string) (string, error) {
	m.mu.Lock()
	session, exists := m.sessions[token]
	if exists {
		delete(m.sessions, token)
	}
	m.mu.Unlock()

	if !exists {
		return "", ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if err := ValidateReceivedRanges(session.ReceivedRanges, session.TotalSize); err != nil {
		session.TempFile.Close()
		os.Remove(session.TempFilePath)
		return "", err
	}

	if err := session.TempFile.Close(); err != nil {
		os.Remove(session.TempFilePath)
		return "", fmt.Errorf("failed to close temporary upload file: %w", err)
	}

	if destinationPath != "" {
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
			os.Remove(session.TempFilePath)
			return "", fmt.Errorf("failed to create destination directory: %w", err)
		}
		if err := os.Rename(session.TempFilePath, destinationPath); err != nil {
			os.Remove(session.TempFilePath)
			return "", fmt.Errorf("failed to persist uploaded file: %w", err)
		}
		return destinationPath, nil
	}

	return session.TempFilePath, nil
}

// AbortSession closes and deletes the temporary file and removes the session from the manager.
func (m *ChunkSessionManager) AbortSession(token string) error {
	m.mu.Lock()
	session, exists := m.sessions[token]
	if exists {
		delete(m.sessions, token)
	}
	m.mu.Unlock()

	if !exists {
		return ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.TempFile.Close()
	os.Remove(session.TempFilePath)
	return nil
}

// Expirations implements ttlcleaner.ExpirableTarget.
func (m *ChunkSessionManager) Expirations() map[string]time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	expirations := make(map[string]time.Time, len(m.sessions))
	for token, s := range m.sessions {
		s.mu.Lock()
		expirations[token] = s.ExpiresAt
		s.mu.Unlock()
	}
	return expirations
}

// Evict implements ttlcleaner.ExpirableTarget.
func (m *ChunkSessionManager) Evict(token string) error {
	return m.AbortSession(token)
}

// ValidateReceivedRanges validates that the given byte ranges completely cover [0, expectedTotalSize)
// without gaps or overlaps after sorting by start offset.
func ValidateReceivedRanges(ranges []ByteRange, expectedTotalSize int64) error {
	if len(ranges) == 0 {
		return fmt.Errorf("incomplete upload: no data received")
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})

	if ranges[0].Start != 0 {
		return fmt.Errorf("incomplete upload: first chunk starts at offset %d, expected 0", ranges[0].Start)
	}

	for i := 0; i < len(ranges)-1; i++ {
		if ranges[i].End != ranges[i+1].Start {
			return fmt.Errorf("incomplete or overlapping upload: chunk %d ends at %d but next chunk starts at %d",
				i, ranges[i].End, ranges[i+1].Start)
		}
	}

	lastEnd := ranges[len(ranges)-1].End
	if lastEnd != expectedTotalSize {
		return fmt.Errorf("incomplete upload: received up to byte %d, expected %d", lastEnd, expectedTotalSize)
	}

	return nil
}
