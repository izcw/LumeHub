package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"lumehub/internal/model"
)

const DefaultStorageQuotaBytes = 30 << 30 // 30 GiB

var ErrStorageQuotaExceeded = errors.New("storage quota exceeded")

func (s *Store) storagePath() string {
	return filepath.Join(s.dataDir, "storage.json")
}

func (s *Store) readStorageDocUnlocked() (*model.StorageDoc, error) {
	raw, err := os.ReadFile(s.storagePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			doc := &model.StorageDoc{
				Version:    1,
				QuotaBytes: DefaultStorageQuotaBytes,
				UsedBytes:  0,
			}
			if err := s.writeStorageDocUnlocked(doc); err != nil {
				return nil, err
			}
			return doc, nil
		}
		return nil, err
	}
	var doc model.StorageDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if doc.Version <= 0 {
		doc.Version = 1
	}
	if doc.QuotaBytes <= 0 {
		doc.QuotaBytes = DefaultStorageQuotaBytes
	}
	if doc.UsedBytes < 0 {
		doc.UsedBytes = 0
	}
	return &doc, nil
}

func (s *Store) writeStorageDocUnlocked(doc *model.StorageDoc) error {
	if doc.Version <= 0 {
		doc.Version = 1
	}
	if doc.QuotaBytes <= 0 {
		doc.QuotaBytes = DefaultStorageQuotaBytes
	}
	if doc.UsedBytes < 0 {
		doc.UsedBytes = 0
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storagePath(), raw, 0o644)
}

func (s *Store) measureStorageBytesUnlocked() (int64, error) {
	var total int64
	roots := []string{
		filepath.Join(s.dataDir, "resource"),
		filepath.Join(s.dataDir, "upload_sessions"),
	}
	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return 0, err
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				total += info.Size()
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

func (s *Store) persistMeasuredStorageUnlocked() (int64, error) {
	used, err := s.measureStorageBytesUnlocked()
	if err != nil {
		return 0, err
	}
	doc, err := s.readStorageDocUnlocked()
	if err != nil {
		return 0, err
	}
	doc.UsedBytes = used
	doc.CalculatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.writeStorageDocUnlocked(doc); err != nil {
		return 0, err
	}
	return used, nil
}

// RecalculateStorageUsed 扫描磁盘并更新 storage.json 中的 usedBytes。
func (s *Store) RecalculateStorageUsed() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persistMeasuredStorageUnlocked()
}

// StorageStatus 返回当前配额与用量（不触发全盘扫描）。
func (s *Store) StorageStatus() (model.StorageStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readStorageDocUnlocked()
	if err != nil {
		return model.StorageStatus{}, err
	}
	return storageStatusFromDoc(doc), nil
}

func storageStatusFromDoc(doc *model.StorageDoc) model.StorageStatus {
	avail := doc.QuotaBytes - doc.UsedBytes
	if avail < 0 {
		avail = 0
	}
	pct := 0.0
	if doc.QuotaBytes > 0 {
		pct = float64(doc.UsedBytes) / float64(doc.QuotaBytes) * 100
		if pct > 100 {
			pct = 100
		}
	}
	return model.StorageStatus{
		QuotaBytes:     doc.QuotaBytes,
		UsedBytes:      doc.UsedBytes,
		AvailableBytes: avail,
		UsedPercent:    pct,
		CalculatedAt:   doc.CalculatedAt,
	}
}

// PatchStorageQuota 更新配额上限（字节）。
func (s *Store) PatchStorageQuota(quotaBytes int64) (model.StorageStatus, error) {
	if quotaBytes <= 0 {
		return model.StorageStatus{}, os.ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readStorageDocUnlocked()
	if err != nil {
		return model.StorageStatus{}, err
	}
	doc.QuotaBytes = quotaBytes
	if err := s.writeStorageDocUnlocked(doc); err != nil {
		return model.StorageStatus{}, err
	}
	return storageStatusFromDoc(doc), nil
}

// checkStorageQuotaUnlocked 校验新增字节后是否仍在配额内；additionalBytes<=0 时仅检查当前是否已满。
func (s *Store) checkStorageQuotaUnlocked(additionalBytes int64) error {
	used, err := s.measureStorageBytesUnlocked()
	if err != nil {
		return err
	}
	doc, err := s.readStorageDocUnlocked()
	if err != nil {
		return err
	}
	if additionalBytes > 0 && used+additionalBytes > doc.QuotaBytes {
		return ErrStorageQuotaExceeded
	}
	if additionalBytes <= 0 && used >= doc.QuotaBytes {
		return ErrStorageQuotaExceeded
	}
	return nil
}

// touchStorageUsedAfterMutationUnlocked 在资源增删后刷新用量缓存。
func (s *Store) touchStorageUsedAfterMutationUnlocked() {
	_, _ = s.persistMeasuredStorageUnlocked()
}
