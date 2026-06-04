package store

import "errors"

type CategorySubMajorPatch struct {
	MajorID       int `json:"majorId"`
	SubID         int `json:"subId"`
	TargetMajorID int `json:"targetMajorId"`
}

var ErrCategorySubMajorInvalid = errors.New("category sub major patch invalid")
var ErrCategorySubMajorTargetNotFound = errors.New("category sub major target not found")

// PatchCategorySubMajors 将二级分类移动到另一大分类下。
func (s *Store) PatchCategorySubMajors(patches []CategorySubMajorPatch) error {
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
		if p.MajorID <= 0 || p.SubID <= 0 || p.TargetMajorID <= 0 {
			return ErrCategorySubMajorInvalid
		}
		if p.MajorID == p.TargetMajorID {
			continue
		}
		mi, sj := findSubIndex(doc, p.MajorID, p.SubID)
		if mi < 0 || sj < 0 {
			return ErrCategorySubMajorTargetNotFound
		}
		tmi := findMajorIndex(doc, p.TargetMajorID)
		if tmi < 0 {
			return ErrCategorySubMajorTargetNotFound
		}
		sub := doc.Categories[mi].Subcategories[sj]
		subs := doc.Categories[mi].Subcategories
		doc.Categories[mi].Subcategories = append(subs[:sj], subs[sj+1:]...)
		sub.Sort = nextSubSortInMajor(&doc.Categories[tmi])
		doc.Categories[tmi].Subcategories = append(doc.Categories[tmi].Subcategories, sub)
	}

	return s.writeCategoriesUnlocked(doc)
}
