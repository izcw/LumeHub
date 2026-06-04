package store

import (
	"errors"
	"slices"
)

// ErrCategoryNavOrderInvalid 表示提交的 id 列表与当前分类结构不一致。
var ErrCategoryNavOrderInvalid = errors.New("invalid category nav order")

// SubOrderPatch 更新某个大类下全部二级的 sort（按 subIds 顺序赋值 1..n）。
type SubOrderPatch struct {
	MajorID int   `json:"majorId"`
	SubIDs  []int `json:"subIds"`
}

// PatchCategoriesNavOrder 将大类顺序与（可选的）若干大类下二级顺序写入 categories.json。
// primaryMajorIds 非空时：须为当前所有大类 id 的无重复全排列。
// subOrders 每项的 subIds 非空时：须为该 major 下全部二级 id 的无重复全排列。
func (s *Store) PatchCategoriesNavOrder(primaryMajorIds []int, subOrders []SubOrderPatch) error {
	if len(primaryMajorIds) == 0 && len(subOrders) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return err
	}

	if len(primaryMajorIds) > 0 {
		want := make([]int, 0, len(doc.Categories))
		for i := range doc.Categories {
			want = append(want, doc.Categories[i].ID)
		}
		if len(primaryMajorIds) != len(want) {
			return ErrCategoryNavOrderInvalid
		}
		got := slices.Clone(primaryMajorIds)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			return ErrCategoryNavOrderInvalid
		}
		byID := make(map[int]int, len(doc.Categories))
		for i := range doc.Categories {
			byID[doc.Categories[i].ID] = i
		}
		for rank, id := range primaryMajorIds {
			mi := byID[id]
			doc.Categories[mi].Sort = rank + 1
		}
	}

	for _, so := range subOrders {
		if len(so.SubIDs) == 0 {
			continue
		}
		mi := findMajorIndex(doc, so.MajorID)
		if mi < 0 {
			return ErrCategoryNavOrderInvalid
		}
		subs := doc.Categories[mi].Subcategories
		want := make([]int, 0, len(subs))
		for j := range subs {
			want = append(want, subs[j].ID)
		}
		if len(so.SubIDs) != len(want) {
			return ErrCategoryNavOrderInvalid
		}
		got := slices.Clone(so.SubIDs)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			return ErrCategoryNavOrderInvalid
		}
		bySub := make(map[int]int, len(subs))
		for j := range subs {
			bySub[subs[j].ID] = j
		}
		for rank, sid := range so.SubIDs {
			sj := bySub[sid]
			doc.Categories[mi].Subcategories[sj].Sort = rank + 1
		}
	}

	return s.writeCategoriesUnlocked(doc)
}
