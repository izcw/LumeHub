package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"lumehub/internal/model"
)

var folderKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_]{1,62}$`)

// ReservedResourceFolderKeys 与 data/resource 下保留目录冲突时禁止作为相册 folderKey。
var ReservedResourceFolderKeys = map[string]struct{}{
	"system": {},
	"search": {},
}

var ErrFolderKeyInvalid = errors.New("folderKey invalid")
var ErrFolderKeyTaken = errors.New("folderKey already exists")
var ErrMajorNotFound = errors.New("major category not found")

func normalizeFolderKey(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func maxCategoryAndSubIDs(doc *model.CategoriesDoc) (maxCat, maxSub int) {
	for _, c := range doc.Categories {
		if c.ID > maxCat {
			maxCat = c.ID
		}
		for _, su := range c.Subcategories {
			if su.ID > maxSub {
				maxSub = su.ID
			}
		}
	}
	return
}

func nextMajorSort(doc *model.CategoriesDoc) int {
	m := 0
	for _, c := range doc.Categories {
		if c.Sort > m {
			m = c.Sort
		}
	}
	return m + 1
}

func nextSubSortInMajor(cat *model.Category) int {
	m := 0
	for _, su := range cat.Subcategories {
		if su.Sort > m {
			m = su.Sort
		}
	}
	return m + 1
}

func folderKeyTaken(doc *model.CategoriesDoc, fk string) bool {
	for _, c := range doc.Categories {
		for _, su := range c.Subcategories {
			if su.FolderKey == fk {
				return true
			}
		}
	}
	return false
}

func (s *Store) ensureResourceFolderAndItems(folderKey string) error {
	dir := filepath.Join(s.dataDir, "resource", folderKey)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	itemsPath := filepath.Join(dir, "items.json")
	if _, statErr := os.Stat(itemsPath); statErr == nil {
		return nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	raw := []byte("{\n  \"items\": []\n}\n")
	return os.WriteFile(itemsPath, raw, 0o644)
}

// CreateCategoryMajor 新增大分类；若传入 subName 与 folderKey，则同步创建第一个子分类。
func (s *Store) CreateCategoryMajor(majorName, subName, folderKey string, public bool) (*model.CategoriesDoc, error) {
	fk := normalizeFolderKey(folderKey)
	subName = strings.TrimSpace(subName)
	needCreateSub := subName != "" || fk != ""
	if needCreateSub {
		if subName == "" {
			return nil, errors.New("name required")
		}
		if !folderKeyPattern.MatchString(fk) {
			return nil, fmt.Errorf("%w: use 2–63 chars, a-z 0-9 underscore, start with letter or digit", ErrFolderKeyInvalid)
		}
		if _, reserved := ReservedResourceFolderKeys[fk]; reserved {
			return nil, fmt.Errorf("%w: reserved name %q", ErrFolderKeyInvalid, fk)
		}
	}
	majorName = strings.TrimSpace(majorName)
	if majorName == "" {
		return nil, errors.New("name required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return nil, err
	}
	if needCreateSub && folderKeyTaken(doc, fk) {
		return nil, ErrFolderKeyTaken
	}
	if needCreateSub {
		if err := s.ensureResourceFolderAndItems(fk); err != nil {
			return nil, err
		}
	}
	maxCat, maxSub := maxCategoryAndSubIDs(doc)
	pub := public
	newMajor := model.Category{
		ID:            maxCat + 1,
		Sort:          nextMajorSort(doc),
		Name:          majorName,
		Public:        &pub,
		Encrypted:     false,
		Subcategories: []model.Subcategory{},
	}
	if needCreateSub {
		newMajor.Subcategories = append(newMajor.Subcategories, model.Subcategory{
			ID:         maxSub + 1,
			Sort:       1,
			Name:       subName,
			FolderKey:  fk,
			Layout:     model.Layout{Mode: "grid", Columns: "auto"},
			ItemSortBy: "uploaded_at",
			Public:     &pub,
			Encrypted:  false,
		})
	}
	doc.Categories = append(doc.Categories, newMajor)
	if err := s.writeCategoriesUnlocked(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// CreateCategorySub 在指定大分类下新增子分类，并创建 resource 目录与空 items.json。
func (s *Store) CreateCategorySub(majorID int, subName, folderKey string, public bool) (*model.CategoriesDoc, error) {
	fk := normalizeFolderKey(folderKey)
	if !folderKeyPattern.MatchString(fk) {
		return nil, fmt.Errorf("%w: use 2–63 chars, a-z 0-9 underscore, start with letter or digit", ErrFolderKeyInvalid)
	}
	if _, reserved := ReservedResourceFolderKeys[fk]; reserved {
		return nil, fmt.Errorf("%w: reserved name %q", ErrFolderKeyInvalid, fk)
	}
	subName = strings.TrimSpace(subName)
	if subName == "" {
		return nil, errors.New("name required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return nil, err
	}
	if folderKeyTaken(doc, fk) {
		return nil, ErrFolderKeyTaken
	}
	mi := -1
	for i := range doc.Categories {
		if doc.Categories[i].ID == majorID {
			mi = i
			break
		}
	}
	if mi < 0 {
		return nil, ErrMajorNotFound
	}
	if err := s.ensureResourceFolderAndItems(fk); err != nil {
		return nil, err
	}
	_, maxSub := maxCategoryAndSubIDs(doc)
	pub := public
	sub := model.Subcategory{
		ID:         maxSub + 1,
		Sort:       nextSubSortInMajor(&doc.Categories[mi]),
		Name:       subName,
		FolderKey:  fk,
		Layout:     model.Layout{Mode: "grid", Columns: "auto"},
		ItemSortBy: "uploaded_at",
		Public:     &pub,
		Encrypted:  false,
	}
	doc.Categories[mi].Subcategories = append(doc.Categories[mi].Subcategories, sub)
	if err := s.writeCategoriesUnlocked(doc); err != nil {
		return nil, err
	}
	return doc, nil
}
