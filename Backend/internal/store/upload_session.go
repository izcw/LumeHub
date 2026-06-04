package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lumehub/internal/model"
)

const (
	uploadSessionChunkDefault = 4 << 20
	uploadSessionMaxChunk     = 16 << 20
	uploadSessionStale        = 36 * time.Hour
)

var (
	ErrUploadBadChunk     = errors.New("invalid chunk")
	ErrUploadHashMismatch = errors.New("sha256 mismatch")
)

type uploadSessionMeta struct {
	FolderKey  string `json:"folderKey"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	ChunkSize  int64  `json:"chunkSize"`
	Created    string `json:"created"`
	FileSHA256 string `json:"fileSha256,omitempty"`
}

// UploadSessionStatus GET /upload/session/:id 返回。
type UploadSessionStatus struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ChunkSize   int64  `json:"chunkSize"`
	TotalChunks int    `json:"totalChunks"`
	Received    []int  `json:"received"`
}

func uploadSessionsRoot(dataDir string) string {
	return filepath.Join(dataDir, "upload_sessions")
}

func (s *Store) uploadSessionDir(sessionID string) string {
	return filepath.Join(uploadSessionsRoot(s.dataDir), sessionID)
}

func readUploadSessionMeta(sessionDir string) (*uploadSessionMeta, error) {
	raw, err := os.ReadFile(filepath.Join(sessionDir, "meta.json"))
	if err != nil {
		return nil, err
	}
	var m uploadSessionMeta
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func writeUploadSessionMeta(sessionDir string, m *uploadSessionMeta) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(sessionDir, "meta.json"), raw, 0o644)
}

func uploadNumChunks(total, chunkSize int64) int {
	if chunkSize <= 0 || total <= 0 {
		return 0
	}
	return int((total + chunkSize - 1) / chunkSize)
}

func uploadChunkExpectedSize(total, chunkSize int64, index, nChunks int) int64 {
	if index < 0 || index >= nChunks {
		return -1
	}
	if index == nChunks-1 {
		if r := total % chunkSize; r != 0 {
			return r
		}
		return chunkSize
	}
	return chunkSize
}

func chunkPartPath(sessionDir string, index int) string {
	return filepath.Join(sessionDir, fmt.Sprintf("%d.part", index))
}

func (s *Store) cleanupStaleUploadSessionsUnlocked() {
	root := uploadSessionsRoot(s.dataDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-uploadSessionStale)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		meta, err := readUploadSessionMeta(dir)
		if err != nil {
			_ = os.RemoveAll(dir)
			continue
		}
		t, perr := time.Parse(time.RFC3339, meta.Created)
		if perr != nil || t.Before(cutoff) {
			_ = os.RemoveAll(dir)
		}
	}
}

// CreateUploadSession 开启可断点续传上传会话。
func (s *Store) CreateUploadSession(folderKey, filename string, totalSize int64, fileSHA256Hex string) (sessionID string, chunkSize int64, err error) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()

	s.cleanupStaleUploadSessionsUnlocked()

	if totalSize <= 0 {
		return "", 0, os.ErrInvalid
	}
	if err := s.checkStorageQuotaUnlocked(totalSize); err != nil {
		return "", 0, err
	}
	doc, err := s.ReadCategories()
	if err != nil {
		return "", 0, err
	}
	if _, _, ok := lookupInDoc(doc, folderKey); !ok {
		return "", 0, os.ErrNotExist
	}

	sessionID = randomHex(16)
	dir := s.uploadSessionDir(sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, err
	}

	chunkSize = int64(uploadSessionChunkDefault)
	if chunkSize > uploadSessionMaxChunk {
		chunkSize = uploadSessionMaxChunk
	}
	if totalSize < chunkSize && totalSize > 0 {
		chunkSize = totalSize
	}

	hashLower := strings.ToLower(strings.TrimSpace(fileSHA256Hex))
	if len(hashLower) != 0 && len(hashLower) != 64 {
		_ = os.RemoveAll(dir)
		return "", 0, os.ErrInvalid
	}

	meta := uploadSessionMeta{
		FolderKey:  folderKey,
		Filename:   filename,
		Size:       totalSize,
		ChunkSize:  chunkSize,
		Created:    time.Now().UTC().Format(time.RFC3339),
		FileSHA256: hashLower,
	}
	if err := writeUploadSessionMeta(dir, &meta); err != nil {
		_ = os.RemoveAll(dir)
		return "", 0, err
	}
	return sessionID, chunkSize, nil
}

// UploadSessionGetStatus 列出已收到的分片索引。
func (s *Store) UploadSessionGetStatus(sessionID, folderKey string) (UploadSessionStatus, error) {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()

	dir := s.uploadSessionDir(sessionID)
	meta, err := readUploadSessionMeta(dir)
	if err != nil {
		return UploadSessionStatus{}, err
	}
	if meta.FolderKey != folderKey {
		return UploadSessionStatus{}, os.ErrNotExist
	}
	n := uploadNumChunks(meta.Size, meta.ChunkSize)
	recv := make([]int, 0, n)
	for i := 0; i < n; i++ {
		p := chunkPartPath(dir, i)
		if fi, err := os.Stat(p); err == nil && fi.Size() > 0 {
			want := uploadChunkExpectedSize(meta.Size, meta.ChunkSize, i, n)
			if fi.Size() == want {
				recv = append(recv, i)
			}
		}
	}
	sort.Ints(recv)
	return UploadSessionStatus{
		Filename:    meta.Filename,
		Size:        meta.Size,
		ChunkSize:   meta.ChunkSize,
		TotalChunks: n,
		Received:    recv,
	}, nil
}

// WriteUploadChunk 写入单个分片（覆盖同序号）。
func (s *Store) WriteUploadChunk(sessionID, folderKey string, index int, r io.Reader) error {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()

	dir := s.uploadSessionDir(sessionID)
	meta, err := readUploadSessionMeta(dir)
	if err != nil {
		return err
	}
	if meta.FolderKey != folderKey {
		return os.ErrNotExist
	}
	n := uploadNumChunks(meta.Size, meta.ChunkSize)
	if n <= 0 || index < 0 || index >= n {
		return ErrUploadBadChunk
	}
	want := uploadChunkExpectedSize(meta.Size, meta.ChunkSize, index, n)
	if want <= 0 {
		return ErrUploadBadChunk
	}

	tmpDir := dir
	f, err := os.CreateTemp(tmpDir, fmt.Sprintf("c-%d-*.tmp", index))
	if err != nil {
		return err
	}
	tmpPath := f.Name()
	nbytes, copyErr := io.Copy(f, io.LimitReader(r, want+1))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if nbytes != want {
		_ = os.Remove(tmpPath)
		return ErrUploadBadChunk
	}
	final := chunkPartPath(dir, index)
	if err := os.Rename(tmpPath, final); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func sha256FileHex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func mergeSessionChunks(sessionDir string, meta *uploadSessionMeta, mergedPath string) (hexSum string, err error) {
	n := uploadNumChunks(meta.Size, meta.ChunkSize)
	for i := 0; i < n; i++ {
		want := uploadChunkExpectedSize(meta.Size, meta.ChunkSize, i, n)
		p := chunkPartPath(sessionDir, i)
		fi, err := os.Stat(p)
		if err != nil || fi.Size() != want {
			return "", fmt.Errorf("%w %d", ErrUploadBadChunk, i)
		}
	}
	tmp := mergedPath + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	mw := io.MultiWriter(out, h)
	var total int64
	for i := 0; i < n; i++ {
		p := chunkPartPath(sessionDir, i)
		in, err := os.Open(p)
		if err != nil {
			_ = out.Close()
			_ = os.Remove(tmp)
			return "", err
		}
		k, err := io.Copy(mw, in)
		_ = in.Close()
		if err != nil {
			_ = out.Close()
			_ = os.Remove(tmp)
			return "", err
		}
		total += k
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if total != meta.Size {
		_ = os.Remove(tmp)
		return "", ErrUploadBadChunk
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if err := os.Rename(tmp, mergedPath); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return sum, nil
}

// CompleteUploadSession 合并分片、校验哈希并入资源库；成功后删除会话目录。
func (s *Store) CompleteUploadSession(sessionID, folderKey, optionalClientSHA256 string) (model.Item, string, error) {
	dir := s.uploadSessionDir(sessionID)

	s.sessMu.Lock()
	meta, err := readUploadSessionMeta(dir)
	if err != nil {
		s.sessMu.Unlock()
		return model.Item{}, "", err
	}
	if meta.FolderKey != folderKey {
		s.sessMu.Unlock()
		return model.Item{}, "", os.ErrNotExist
	}

	mergedPath := filepath.Join(dir, "merged.bin")
	var sum string
	if fi, statErr := os.Stat(mergedPath); statErr == nil && fi.Size() == meta.Size {
		sum, err = sha256FileHex(mergedPath)
		if err != nil {
			s.sessMu.Unlock()
			return model.Item{}, "", err
		}
	} else {
		sum, err = mergeSessionChunks(dir, meta, mergedPath)
		if err != nil {
			s.sessMu.Unlock()
			return model.Item{}, "", err
		}
	}

	if want := strings.ToLower(strings.TrimSpace(meta.FileSHA256)); want != "" && !strings.EqualFold(sum, want) {
		s.sessMu.Unlock()
		return model.Item{}, "", ErrUploadHashMismatch
	}
	if c := strings.ToLower(strings.TrimSpace(optionalClientSHA256)); c != "" && !strings.EqualFold(sum, c) {
		s.sessMu.Unlock()
		return model.Item{}, "", ErrUploadHashMismatch
	}

	localCopy := filepath.Join(dir, "finalize.bin")
	if err := os.Rename(mergedPath, localCopy); err != nil {
		if err := copyFile(mergedPath, localCopy); err != nil {
			s.sessMu.Unlock()
			return model.Item{}, "", err
		}
		_ = os.Remove(mergedPath)
	}
	s.sessMu.Unlock()

	item, err := s.SaveUploadedFileFromLocalPath(meta.FolderKey, meta.Filename, localCopy)

	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	if err != nil {
		return model.Item{}, "", err
	}
	_ = os.RemoveAll(dir)
	return item, sum, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// DeleteUploadSession 取消上传并清理临时文件。
func (s *Store) DeleteUploadSession(sessionID, folderKey string) error {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	dir := s.uploadSessionDir(sessionID)
	meta, err := readUploadSessionMeta(dir)
	if err != nil {
		return err
	}
	if meta.FolderKey != folderKey {
		return os.ErrNotExist
	}
	return os.RemoveAll(dir)
}
