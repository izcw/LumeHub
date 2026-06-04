package ordering

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"lumehub/internal/model"
)

// RankInt 将 JSON 中省略或为 0 的 sort 排到靠后（再按第二关键字排序）。
func RankInt(sort int) int {
	if sort == 0 {
		return 1 << 30
	}
	return sort
}

// SortCategoriesInPlace 按 sort 升序，其次 id 升序。
func SortCategoriesInPlace(cats []model.Category) {
	slices.SortFunc(cats, func(a, b model.Category) int {
		if c := cmp.Compare(RankInt(a.Sort), RankInt(b.Sort)); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

// SortItemsInPlace 按 sort 升序，其次 filename 字典序（自定义排序模式）。
func SortItemsInPlace(items []model.Item) {
	slices.SortFunc(items, func(a, b model.Item) int {
		if c := cmp.Compare(RankInt(a.Sort), RankInt(b.Sort)); c != 0 {
			return c
		}
		return strings.Compare(a.Filename, b.Filename)
	})
}

// SortSubcategoriesInPlace 按 sort 升序，其次 id 升序。
func SortSubcategoriesInPlace(subs []model.Subcategory) {
	slices.SortFunc(subs, func(a, b model.Subcategory) int {
		if c := cmp.Compare(RankInt(a.Sort), RankInt(b.Sort)); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

func parseItemTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// SortItemsByMode 按分类的资源排序策略排序（同一策略内新时间靠前）。
func SortItemsByMode(items []model.Item, mode string) {
	m := strings.TrimSpace(strings.ToLower(mode))
	switch m {
	case "updated_at":
		slices.SortFunc(items, func(a, b model.Item) int {
			ua, aOk := parseItemTime(a.UpdatedAt)
			ub, bOk := parseItemTime(b.UpdatedAt)
			if c := compareTimeDesc(ua, ub, aOk, bOk); c != 0 {
				return c
			}
			ia, iaOk := parseItemTime(a.UploadedAt)
			ib, ibOk := parseItemTime(b.UploadedAt)
			if c := compareTimeDesc(ia, ib, iaOk, ibOk); c != 0 {
				return c
			}
			return strings.Compare(a.Filename, b.Filename)
		})
	case "sort", "custom":
		SortItemsInPlace(items)
	default: // uploaded_at
		slices.SortFunc(items, func(a, b model.Item) int {
			ta, aOk := parseItemTime(a.UploadedAt)
			tb, bOk := parseItemTime(b.UploadedAt)
			if c := compareTimeDesc(ta, tb, aOk, bOk); c != 0 {
				return c
			}
			return strings.Compare(a.Filename, b.Filename)
		})
	}
}

// compareTimeDesc：时间新的在前；无时间排后。
func compareTimeDesc(a, b time.Time, aOk, bOk bool) int {
	if !aOk && !bOk {
		return 0
	}
	if !aOk {
		return 1
	}
	if !bOk {
		return -1
	}
	return cmp.Compare(b.UnixNano(), a.UnixNano())
}
