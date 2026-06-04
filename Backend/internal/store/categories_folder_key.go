package store

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"lumehub/internal/model"
)

type CategoryFolderKeyPatch struct {
	MajorID   int  `json:"majorId"`
	SubID     *int `json:"subId,omitempty"`
	FolderKey string `json:"folderKey"`
}

var folderKeyRenamePattern = regexp.MustCompile(`^[A-Za-z_]{1,63}$`)

var ErrCategoryFolderKeyTargetNotFound = errors.New("category folderKey target not found")
var ErrCategoryFolderKeyEmpty = errors.New("category folderKey empty")
var ErrCategoryFolderKeyInvalid = errors.New("category folderKey invalid")
var ErrCategoryFolderKeyTaken = errors.New("category folderKey taken")

func folderKeyTakenExcept(doc *model.CategoriesDoc, fk string, majorID, subID int) bool {
	for _, c := range doc.Categories {
		for _, su := range c.Subcategories {
			if c.ID == majorID && su.ID == subID {
				continue
			}
			if strings.EqualFold(su.FolderKey, fk) {
				return true
			}
		}
	}
	return false
}

// PatchCategoryFolderKeys 更新二级分类 folderKey，并同步重命名 data/resource 目录。
func (s *Store) PatchCategoryFolderKeys(patches []CategoryFolderKeyPatch) error {
	if len(patches) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return err
	}

	for _, p := range patches {
		if p.SubID == nil {
			return ErrCategoryFolderKeyTargetNotFound
		}
		mi, sj := findSubIndex(doc, p.MajorID, *p.SubID)
		if mi < 0 || sj < 0 {
			return ErrCategoryFolderKeyTargetNotFound
		}

		next := strings.TrimSpace(p.FolderKey)
		if next == "" {
			return ErrCategoryFolderKeyEmpty
		}
		if !folderKeyRenamePattern.MatchString(next) {
			return ErrCategoryFolderKeyInvalid
		}
		if _, reserved := ReservedResourceFolderKeys[strings.ToLower(next)]; reserved {
			return ErrCategoryFolderKeyInvalid
		}
		if folderKeyTakenExcept(doc, next, p.MajorID, *p.SubID) {
			return ErrCategoryFolderKeyTaken
		}

		oldKey := doc.Categories[mi].Subcategories[sj].FolderKey
		if oldKey != next {
			oldDir := filepath.Join(s.dataDir, "resource", oldKey)
			newDir := filepath.Join(s.dataDir, "resource", next)
			if _, statErr := os.Stat(newDir); statErr == nil {
				return ErrCategoryFolderKeyTaken
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return statErr
			}
			if _, statErr := os.Stat(oldDir); statErr == nil {
				if err := os.Rename(oldDir, newDir); err != nil {
					return err
				}
			} else if errors.Is(statErr, os.ErrNotExist) {
				if err := s.ensureResourceFolderAndItems(next); err != nil {
					return err
				}
			} else {
				return statErr
			}
		}

		doc.Categories[mi].Subcategories[sj].FolderKey = next
	}

	return s.writeCategoriesUnlocked(doc)
}
