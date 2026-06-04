package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lumehub/internal/model"
)

const trashRetentionDays = 30

func (s *Store) trashPath() string {
	return filepath.Join(s.dataDir, "trash.json")
}

func (s *Store) readTrashUnlocked() ([]model.TrashEntry, error) {
	raw, err := os.ReadFile(s.trashPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc model.TrashDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return doc.Items, nil
}

func (s *Store) writeTrashUnlocked(items []model.TrashEntry) error {
	doc := model.TrashDoc{Items: items}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.trashPath(), raw, 0o644)
}

func trashExpiresAt(deletedAt string) (time.Time, bool) {
	deletedAt = strings.TrimSpace(deletedAt)
	if deletedAt == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, deletedAt)
	if err != nil {
		return time.Time{}, false
	}
	return t.Add(trashRetentionDays * 24 * time.Hour), true
}

func isTrashExpired(deletedAt string, now time.Time) bool {
	exp, ok := trashExpiresAt(deletedAt)
	if !ok {
		return false
	}
	return !now.Before(exp)
}

// PurgeExpiredTrash 永久删除超过保留期的回收站条目及其磁盘文件。
func (s *Store) PurgeExpiredTrash() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.purgeExpiredTrashUnlocked(time.Now().UTC())
}

func (s *Store) purgeExpiredTrashUnlocked(now time.Time) (int, error) {
	items, err := s.readTrashUnlocked()
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}
	kept := make([]model.TrashEntry, 0, len(items))
	purged := 0
	purgedFolders := make(map[string]struct{})
	for _, entry := range items {
		if isTrashExpired(entry.DeletedAt, now) {
			resourceDir := filepath.Join(s.dataDir, "resource", entry.FolderKey)
			for _, p := range itemResourcePaths(resourceDir, entry.Item) {
				_ = os.Remove(p)
			}
			purgedFolders[entry.FolderKey] = struct{}{}
			purged++
			continue
		}
		kept = append(kept, entry)
	}
	if purged == 0 {
		return 0, nil
	}
	if err := s.writeTrashUnlocked(kept); err != nil {
		return 0, err
	}
	for fk := range purgedFolders {
		s.removeOrphanResourceFolderUnlocked(fk)
	}
	s.touchStorageUsedAfterMutationUnlocked()
	return purged, nil
}

// MoveCategoryItemToTrash 将条目从分类移入回收站（保留磁盘文件）。
func (s *Store) MoveCategoryItemToTrash(folderKey, itemID string) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return os.ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.purgeExpiredTrashUnlocked(time.Now().UTC()); err != nil {
		return err
	}

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return err
	}
	mi, sj, ok := lookupInDoc(doc, folderKey)
	if !ok {
		return os.ErrNotExist
	}
	major := doc.Categories[mi]
	sub := doc.Categories[mi].Subcategories[sj]

	items, err := s.readItemsUnlocked(folderKey)
	if err != nil {
		return err
	}

	idx := -1
	var target model.Item
	for i, it := range items {
		if it.ID == itemID {
			idx = i
			target = it
			break
		}
	}
	if idx < 0 {
		return os.ErrNotExist
	}

	items = append(items[:idx], items[idx+1:]...)
	if err := s.writeItemsUnlocked(folderKey, items); err != nil {
		return err
	}

	trashItems, err := s.readTrashUnlocked()
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	entry := model.TrashEntry{
		FolderKey: folderKey,
		DeletedAt: now,
		Item:      target,
	}
	applyTrashCategoryMeta(&entry, major, sub)
	trashItems = append(trashItems, entry)
	return s.writeTrashUnlocked(trashItems)
}

// ListTrashItems 返回回收站条目（会先清理过期项）。
func (s *Store) ListTrashItems() ([]model.TrashEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.purgeExpiredTrashUnlocked(time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.readTrashUnlocked()
}

// ClearAllTrash 永久删除回收站全部条目及其磁盘文件。
func (s *Store) ClearAllTrash() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.readTrashUnlocked()
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}

	for _, entry := range items {
		resourceDir := filepath.Join(s.dataDir, "resource", entry.FolderKey)
		for _, p := range itemResourcePaths(resourceDir, entry.Item) {
			_ = os.Remove(p)
		}
	}

	if err := s.writeTrashUnlocked(nil); err != nil {
		return 0, err
	}
	for _, entry := range items {
		s.removeOrphanResourceFolderUnlocked(entry.FolderKey)
	}
	s.touchStorageUsedAfterMutationUnlocked()
	return len(items), nil
}

// ClearTrashFolder 永久删除指定 folderKey 下全部回收站条目及其磁盘文件。
func (s *Store) ClearTrashFolder(folderKey string) (int, error) {
	folderKey = strings.TrimSpace(folderKey)
	if folderKey == "" {
		return 0, os.ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	trashItems, err := s.readTrashUnlocked()
	if err != nil {
		return 0, err
	}

	kept := make([]model.TrashEntry, 0, len(trashItems))
	deleted := 0
	for _, entry := range trashItems {
		if entry.FolderKey != folderKey {
			kept = append(kept, entry)
			continue
		}
		resourceDir := filepath.Join(s.dataDir, "resource", entry.FolderKey)
		for _, p := range itemResourcePaths(resourceDir, entry.Item) {
			_ = os.Remove(p)
		}
		deleted++
	}
	if deleted == 0 {
		return 0, os.ErrNotExist
	}
	if err := s.writeTrashUnlocked(kept); err != nil {
		return 0, err
	}
	s.removeOrphanResourceFolderUnlocked(folderKey)
	s.touchStorageUsedAfterMutationUnlocked()
	return deleted, nil
}

// PermanentDeleteTrashItem 从回收站永久删除条目及其磁盘文件。
func (s *Store) PermanentDeleteTrashItem(folderKey, itemID string) error {
	itemID = strings.TrimSpace(itemID)
	folderKey = strings.TrimSpace(folderKey)
	if itemID == "" || folderKey == "" {
		return os.ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	trashItems, err := s.readTrashUnlocked()
	if err != nil {
		return err
	}

	idx := -1
	var target model.TrashEntry
	for i, entry := range trashItems {
		if entry.FolderKey == folderKey && entry.Item.ID == itemID {
			idx = i
			target = entry
			break
		}
	}
	if idx < 0 {
		return os.ErrNotExist
	}

	resourceDir := filepath.Join(s.dataDir, "resource", folderKey)
	paths := itemResourcePaths(resourceDir, target.Item)

	trashItems = append(trashItems[:idx], trashItems[idx+1:]...)
	if err := s.writeTrashUnlocked(trashItems); err != nil {
		return err
	}

	for _, p := range paths {
		_ = os.Remove(p)
	}
	s.removeOrphanResourceFolderUnlocked(folderKey)
	s.touchStorageUsedAfterMutationUnlocked()
	return nil
}

// RestoreTrashItem 将回收站条目还原到原分类（磁盘文件保持不变）。
// 返回值 categoryRecreated 表示原画廊/导航已被删除并在恢复时重建。
func (s *Store) RestoreTrashItem(folderKey, itemID string) (categoryRecreated bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, categoryRecreated, err = s.restoreTrashEntriesUnlocked(folderKey, func(entry model.TrashEntry) bool {
		return entry.Item.ID == strings.TrimSpace(itemID)
	})
	return categoryRecreated, err
}

// RestoreTrashFolder 将指定 folderKey 下全部回收站条目一次性还原。
func (s *Store) RestoreTrashFolder(folderKey string) (restored int, categoryRecreated bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.restoreTrashEntriesUnlocked(folderKey, nil)
}

func (s *Store) restoreTrashEntriesUnlocked(
	folderKey string,
	match func(model.TrashEntry) bool,
) (int, bool, error) {
	folderKey = strings.TrimSpace(folderKey)
	if folderKey == "" {
		return 0, false, os.ErrInvalid
	}

	trashItems, err := s.readTrashUnlocked()
	if err != nil {
		return 0, false, err
	}

	var picked []model.TrashEntry
	kept := make([]model.TrashEntry, 0, len(trashItems))
	for _, entry := range trashItems {
		if entry.FolderKey == folderKey && (match == nil || match(entry)) {
			picked = append(picked, entry)
			continue
		}
		kept = append(kept, entry)
	}
	if len(picked) == 0 {
		return 0, false, os.ErrNotExist
	}
	if match != nil && len(picked) != 1 {
		return 0, false, os.ErrNotExist
	}

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return 0, false, err
	}
	categoryRecreated := false
	if _, _, ok := lookupInDoc(doc, folderKey); !ok {
		categoryRecreated = true
		if err := s.ensureCategoryForTrashRestoreUnlocked(doc, picked[0]); err != nil {
			return 0, false, err
		}
	}

	items, err := s.readItemsUnlocked(folderKey)
	if err != nil {
		return 0, false, err
	}
	if match != nil {
		for _, it := range items {
			if it.ID == picked[0].Item.ID {
				return 0, false, os.ErrExist
			}
		}
	}

	existing := make(map[string]struct{}, len(items))
	for _, it := range items {
		existing[it.ID] = struct{}{}
	}

	restored := 0
	for _, entry := range picked {
		if _, dup := existing[entry.Item.ID]; dup {
			continue
		}
		items = append(items, entry.Item)
		existing[entry.Item.ID] = struct{}{}
		restored++
	}
	if restored == 0 && match != nil {
		return 0, false, os.ErrExist
	}

	if err := s.writeTrashUnlocked(kept); err != nil {
		return 0, false, err
	}
	if err := s.writeItemsUnlocked(folderKey, items); err != nil {
		return 0, false, err
	}
	return restored, categoryRecreated, nil
}
