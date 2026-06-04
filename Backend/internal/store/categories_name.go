package store

import (
	"errors"
	"strings"
)

type CategoryNamePatch struct {
	Scope   string `json:"scope"` // major | sub
	MajorID int    `json:"majorId"`
	SubID   *int   `json:"subId,omitempty"`
	Name    string `json:"name"`
}

var ErrCategoryNameTargetNotFound = errors.New("category name target not found")
var ErrCategoryNameEmpty = errors.New("category name empty")

// PatchCategoryNames 更新大分类或二级分类名称。
func (s *Store) PatchCategoryNames(patches []CategoryNamePatch) error {
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
		name := strings.TrimSpace(p.Name)
		if name == "" {
			return ErrCategoryNameEmpty
		}
		switch p.Scope {
		case "major":
			mi := findMajorIndex(doc, p.MajorID)
			if mi < 0 {
				return ErrCategoryNameTargetNotFound
			}
			doc.Categories[mi].Name = name
		case "sub":
			if p.SubID == nil {
				return ErrCategoryNameTargetNotFound
			}
			mi, sj := findSubIndex(doc, p.MajorID, *p.SubID)
			if mi < 0 || sj < 0 {
				return ErrCategoryNameTargetNotFound
			}
			doc.Categories[mi].Subcategories[sj].Name = name
		default:
			return ErrCategoryNameTargetNotFound
		}
	}

	return s.writeCategoriesUnlocked(doc)
}
