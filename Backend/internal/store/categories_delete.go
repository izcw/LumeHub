package store

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"lumehub/internal/model"
)

var ErrCategoryDeleteTargetNotFound = errors.New("category delete target not found")

func (s *Store) moveFolderItemsToTrashUnlocked(major model.Category, sub model.Subcategory) (int, error) {
	folderKey := sub.FolderKey
	items, err := s.readItemsUnlocked(folderKey)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}

	trashItems, err := s.readTrashUnlocked()
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, it := range items {
		entry := model.TrashEntry{
			FolderKey: folderKey,
			DeletedAt: now,
			Item:      it,
		}
		applyTrashCategoryMeta(&entry, major, sub)
		trashItems = append(trashItems, entry)
	}
	if err := s.writeTrashUnlocked(trashItems); err != nil {
		return 0, err
	}
	if err := s.writeItemsUnlocked(folderKey, nil); err != nil {
		return 0, err
	}
	return len(items), nil
}

// DeleteCategorySub 删除子分类（画廊）；目录内条目移入回收站，磁盘文件保留。
func (s *Store) DeleteCategorySub(majorID, subID int) (*model.CategoriesDoc, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.purgeExpiredTrashUnlocked(time.Now().UTC()); err != nil {
		return nil, 0, err
	}

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return nil, 0, err
	}

	mi := -1
	for i := range doc.Categories {
		if doc.Categories[i].ID == majorID {
			mi = i
			break
		}
	}
	if mi < 0 {
		return nil, 0, ErrCategoryDeleteTargetNotFound
	}

	sj := -1
	for i := range doc.Categories[mi].Subcategories {
		if doc.Categories[mi].Subcategories[i].ID == subID {
			sj = i
			break
		}
	}
	if sj < 0 {
		return nil, 0, ErrCategoryDeleteTargetNotFound
	}

	sub := doc.Categories[mi].Subcategories[sj]
	n, err := s.moveFolderItemsToTrashUnlocked(doc.Categories[mi], sub)
	if err != nil {
		return nil, 0, err
	}

	subs := doc.Categories[mi].Subcategories
	doc.Categories[mi].Subcategories = append(subs[:sj], subs[sj+1:]...)
	if err := s.writeCategoriesUnlocked(doc); err != nil {
		return nil, 0, err
	}

	s.cleanupUploadSessionsForFolderUnlocked(sub.FolderKey)
	s.touchStorageUsedAfterMutationUnlocked()
	return doc, n, nil
}

// DeleteCategoryMajor 删除大分类（导航）；其下所有画廊条目移入回收站。
func (s *Store) DeleteCategoryMajor(majorID int) (*model.CategoriesDoc, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.purgeExpiredTrashUnlocked(time.Now().UTC()); err != nil {
		return nil, 0, err
	}

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return nil, 0, err
	}

	mi := -1
	for i := range doc.Categories {
		if doc.Categories[i].ID == majorID {
			mi = i
			break
		}
	}
	if mi < 0 {
		return nil, 0, ErrCategoryDeleteTargetNotFound
	}

	totalTrashed := 0
	major := doc.Categories[mi]
	for _, sub := range major.Subcategories {
		n, err := s.moveFolderItemsToTrashUnlocked(major, sub)
		if err != nil {
			return nil, 0, err
		}
		totalTrashed += n
		s.cleanupUploadSessionsForFolderUnlocked(sub.FolderKey)
	}

	doc.Categories = append(doc.Categories[:mi], doc.Categories[mi+1:]...)
	if err := s.writeCategoriesUnlocked(doc); err != nil {
		return nil, 0, err
	}

	s.touchStorageUsedAfterMutationUnlocked()
	return doc, totalTrashed, nil
}

func (s *Store) cleanupUploadSessionsForFolderUnlocked(folderKey string) {
	root := uploadSessionsRoot(s.dataDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		meta, err := readUploadSessionMeta(dir)
		if err != nil {
			continue
		}
		if meta.FolderKey == folderKey {
			_ = os.RemoveAll(dir)
		}
	}
}
