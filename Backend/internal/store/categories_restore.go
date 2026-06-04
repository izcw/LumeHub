package store

import (
	"os"
	"strings"

	"lumehub/internal/model"
)

func applyTrashCategoryMeta(entry *model.TrashEntry, major model.Category, sub model.Subcategory) {
	entry.MajorName = major.Name
	entry.SubName = sub.Name
	entry.MajorKey = major.Key
	entry.MajorID = major.ID
	entry.SubID = sub.ID
	entry.SubSort = sub.Sort
	entry.Layout = sub.Layout
	entry.ItemSortBy = sub.ItemSortBy
	entry.Public = sub.Public
	entry.Encrypted = sub.Encrypted
	entry.EncryptedPasswordHash = sub.EncryptedPasswordHash
}

func majorIDTaken(doc *model.CategoriesDoc, id int) bool {
	for _, c := range doc.Categories {
		if c.ID == id {
			return true
		}
	}
	return false
}

func subIDTaken(doc *model.CategoriesDoc, id int) bool {
	for _, c := range doc.Categories {
		for _, su := range c.Subcategories {
			if su.ID == id {
				return true
			}
		}
	}
	return false
}

func findMajorIndexForTrashRestore(doc *model.CategoriesDoc, entry model.TrashEntry) int {
	if entry.MajorID > 0 {
		for i := range doc.Categories {
			if doc.Categories[i].ID == entry.MajorID {
				return i
			}
		}
	}
	majorKey := strings.TrimSpace(entry.MajorKey)
	if majorKey != "" {
		for i := range doc.Categories {
			if doc.Categories[i].Key == majorKey {
				return i
			}
		}
	}
	majorName := strings.TrimSpace(entry.MajorName)
	if majorName != "" {
		for i := range doc.Categories {
			if doc.Categories[i].Name == majorName {
				return i
			}
		}
	}
	return -1
}

func subcategoryFromTrashEntry(entry model.TrashEntry, doc *model.CategoriesDoc, major *model.Category) model.Subcategory {
	pub := true
	if entry.Public != nil {
		pub = *entry.Public
	}
	layout := entry.Layout
	if layout.Mode == "" {
		layout = model.Layout{Mode: "grid", Columns: "auto"}
	}
	itemSortBy := strings.TrimSpace(entry.ItemSortBy)
	if itemSortBy == "" {
		itemSortBy = "uploaded_at"
	}
	subID := entry.SubID
	if subID <= 0 || subIDTaken(doc, subID) {
		_, maxSub := maxCategoryAndSubIDs(doc)
		subID = maxSub + 1
	}
	sortVal := entry.SubSort
	if sortVal <= 0 {
		sortVal = nextSubSortInMajor(major)
	}
	return model.Subcategory{
		ID:                    subID,
		Sort:                  sortVal,
		Name:                  strings.TrimSpace(entry.SubName),
		FolderKey:             entry.FolderKey,
		Layout:                layout,
		ItemSortBy:            itemSortBy,
		Public:                &pub,
		Encrypted:             entry.Encrypted,
		EncryptedPasswordHash: entry.EncryptedPasswordHash,
	}
}

func fallbackMajorKey(entry model.TrashEntry) string {
	if k := strings.TrimSpace(entry.MajorKey); k != "" {
		return k
	}
	return entry.FolderKey
}

// ensureCategoryForTrashRestoreUnlocked 在恢复回收站条目时，若原画廊/导航已被删除则按元数据重建。
func (s *Store) ensureCategoryForTrashRestoreUnlocked(doc *model.CategoriesDoc, entry model.TrashEntry) error {
	if _, _, ok := lookupInDoc(doc, entry.FolderKey); ok {
		return nil
	}
	if strings.TrimSpace(entry.SubName) == "" {
		return os.ErrNotExist
	}

	mi := findMajorIndexForTrashRestore(doc, entry)
	if mi >= 0 {
		major := &doc.Categories[mi]
		sub := subcategoryFromTrashEntry(entry, doc, major)
		major.Subcategories = append(major.Subcategories, sub)
	} else {
		maxCat, _ := maxCategoryAndSubIDs(doc)
		majorID := entry.MajorID
		if majorID <= 0 || majorIDTaken(doc, majorID) {
			majorID = maxCat + 1
		}
		pub := true
		if entry.Public != nil {
			pub = *entry.Public
		}
		major := model.Category{
			ID:            majorID,
			Sort:          nextMajorSort(doc),
			Name:          strings.TrimSpace(entry.MajorName),
			Key:           fallbackMajorKey(entry),
			Public:        &pub,
			Subcategories: []model.Subcategory{},
		}
		if major.Name == "" {
			major.Name = entry.SubName
		}
		sub := subcategoryFromTrashEntry(entry, doc, &major)
		major.Subcategories = append(major.Subcategories, sub)
		doc.Categories = append(doc.Categories, major)
	}

	if err := s.ensureResourceFolderAndItems(entry.FolderKey); err != nil {
		return err
	}
	return s.writeCategoriesUnlocked(doc)
}
