package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// removeOrphanResourceFolderUnlocked 在导航未登记、回收站无条目且 items.json 为空时，删除 resource/{folderKey}。
func (s *Store) removeOrphanResourceFolderUnlocked(folderKey string) {
	folderKey = strings.TrimSpace(folderKey)
	if folderKey == "" {
		return
	}
	if _, reserved := ReservedResourceFolderKeys[folderKey]; reserved {
		return
	}

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return
	}
	if _, _, ok := lookupInDoc(doc, folderKey); ok {
		return
	}

	trashItems, err := s.readTrashUnlocked()
	if err != nil {
		return
	}
	for _, entry := range trashItems {
		if entry.FolderKey == folderKey {
			return
		}
	}

	items, err := s.readItemsUnlocked(folderKey)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}
	if len(items) > 0 {
		return
	}

	resourceDir := filepath.Join(s.dataDir, "resource", folderKey)
	_ = os.RemoveAll(resourceDir)
}

func (s *Store) removeOrphanResourceFoldersUnlocked(folderKeys ...string) {
	seen := make(map[string]struct{}, len(folderKeys))
	for _, fk := range folderKeys {
		fk = strings.TrimSpace(fk)
		if fk == "" {
			continue
		}
		if _, ok := seen[fk]; ok {
			continue
		}
		seen[fk] = struct{}{}
		s.removeOrphanResourceFolderUnlocked(fk)
	}
}

// CleanupOrphanResourceFolders 扫描 resource/ 并删除已无导航、无回收站条目、无资源的孤儿目录。
func (s *Store) CleanupOrphanResourceFolders() (removed []string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	root := filepath.Join(s.dataDir, "resource")
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return nil, err
	}
	trashItems, err := s.readTrashUnlocked()
	if err != nil {
		return nil, err
	}
	trashByFolder := map[string]struct{}{}
	for _, entry := range trashItems {
		trashByFolder[entry.FolderKey] = struct{}{}
	}

	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		fk := strings.TrimSpace(ent.Name())
		if fk == "" || fk == "system" {
			continue
		}
		if _, _, ok := lookupInDoc(doc, fk); ok {
			continue
		}
		if _, inTrash := trashByFolder[fk]; inTrash {
			continue
		}
		items, readErr := s.readItemsUnlocked(fk)
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			return removed, readErr
		}
		if len(items) > 0 {
			continue
		}
		dir := filepath.Join(root, fk)
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			return removed, rmErr
		}
		removed = append(removed, fk)
	}
	if len(removed) > 0 {
		s.touchStorageUsedAfterMutationUnlocked()
	}
	return removed, nil
}
