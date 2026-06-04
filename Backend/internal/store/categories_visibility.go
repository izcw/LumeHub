package store

import (
	"errors"
	"strings"

	"lumehub/internal/model"
)

// CategoryVisibilityPatch 更新大分类或二级的公开 / 加密标记。
type CategoryVisibilityPatch struct {
	Scope     string `json:"scope"` // major | sub
	MajorID   int    `json:"majorId"`
	SubID     *int   `json:"subId,omitempty"`
	Public    *bool  `json:"public,omitempty"`
	Encrypted *bool  `json:"encrypted,omitempty"`
	// EncryptedPassword 设置或更新查看密码（明文入参，存储为哈希）。
	EncryptedPassword *string `json:"encryptedPassword,omitempty"`
}

var ErrCategoryVisibilityNotFound = errors.New("category visibility target not found")
var ErrCategoryEncryptedPasswordRequired = errors.New("encrypted password required")
var ErrCategoryEncryptedPasswordTooShort = errors.New("encrypted password too short")

// PatchCategoriesVisibility 将若干可见性修改写入 categories.json。
func (s *Store) PatchCategoriesVisibility(patches []CategoryVisibilityPatch) error {
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
		switch p.Scope {
		case "major":
			mi := findMajorIndex(doc, p.MajorID)
			if mi < 0 {
				return ErrCategoryVisibilityNotFound
			}
			if p.Public != nil {
				v := *p.Public
				doc.Categories[mi].Public = &v
			}
			if p.Encrypted != nil {
				doc.Categories[mi].Encrypted = *p.Encrypted
				if !*p.Encrypted {
					doc.Categories[mi].EncryptedPasswordHash = ""
				}
			}
			if p.EncryptedPassword != nil {
				plain := strings.TrimSpace(*p.EncryptedPassword)
				if len([]rune(plain)) < 4 {
					return ErrCategoryEncryptedPasswordTooShort
				}
				doc.Categories[mi].EncryptedPasswordHash = model.HashPasswordUTF8(plain)
			}
			if doc.Categories[mi].Encrypted && doc.Categories[mi].EncryptedPasswordHash == "" {
				return ErrCategoryEncryptedPasswordRequired
			}
		case "sub":
			if p.SubID == nil {
				return ErrCategoryVisibilityNotFound
			}
			mi, sj := findSubIndex(doc, p.MajorID, *p.SubID)
			if mi < 0 || sj < 0 {
				return ErrCategoryVisibilityNotFound
			}
			if p.Public != nil {
				v := *p.Public
				doc.Categories[mi].Subcategories[sj].Public = &v
			}
			if p.Encrypted != nil {
				doc.Categories[mi].Subcategories[sj].Encrypted = *p.Encrypted
				if !*p.Encrypted {
					doc.Categories[mi].Subcategories[sj].EncryptedPasswordHash = ""
				}
			}
			if p.EncryptedPassword != nil {
				plain := strings.TrimSpace(*p.EncryptedPassword)
				if len([]rune(plain)) < 4 {
					return ErrCategoryEncryptedPasswordTooShort
				}
				doc.Categories[mi].Subcategories[sj].EncryptedPasswordHash = model.HashPasswordUTF8(plain)
			}
			if doc.Categories[mi].Subcategories[sj].Encrypted && doc.Categories[mi].Subcategories[sj].EncryptedPasswordHash == "" {
				return ErrCategoryEncryptedPasswordRequired
			}
		default:
			return ErrCategoryVisibilityNotFound
		}
	}
	return s.writeCategoriesUnlocked(doc)
}

func findMajorIndex(doc *model.CategoriesDoc, majorID int) int {
	for i := range doc.Categories {
		if doc.Categories[i].ID == majorID {
			return i
		}
	}
	return -1
}

func findSubIndex(doc *model.CategoriesDoc, majorID, subID int) (majorIdx, subIdx int) {
	mi := findMajorIndex(doc, majorID)
	if mi < 0 {
		return -1, -1
	}
	for j := range doc.Categories[mi].Subcategories {
		if doc.Categories[mi].Subcategories[j].ID == subID {
			return mi, j
		}
	}
	return mi, -1
}
