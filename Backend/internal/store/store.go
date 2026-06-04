package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"lumehub/internal/model"
)

func lookupInDoc(doc *model.CategoriesDoc, folderKey string) (majorIdx, subIdx int, ok bool) {
	for mi := range doc.Categories {
		for sj := range doc.Categories[mi].Subcategories {
			if doc.Categories[mi].Subcategories[sj].FolderKey == folderKey {
				return mi, sj, true
			}
		}
	}
	return -1, -1, false
}

type Store struct {
	dataDir string
	mu      sync.Mutex
	sessMu  sync.Mutex
}

func New(dataDir string) *Store {
	return &Store{dataDir: dataDir}
}

// DataDir 返回数据根目录（含 resource 等子目录）。
func (s *Store) DataDir() string {
	return s.dataDir
}

// ThumbRelForOriginal 根据 original/ 下相对路径生成规范缩略图相对路径：同主干名；PNG 为 .png，其余栅格图为 .jpg。
func ThumbRelForOriginal(originalRel string) string {
	originalRel = strings.TrimSpace(originalRel)
	if originalRel == "" {
		return ""
	}
	base := filepath.Base(filepath.ToSlash(originalRel))
	if base == "" || base == "." {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(base))
	stem := strings.TrimSuffix(base, ext)
	if stem == "" {
		return ""
	}
	thumbExt := ".jpg"
	if ext == ".png" {
		thumbExt = ".png"
	}
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".avif",
		".mp4", ".m4v", ".mov":
		return filepath.ToSlash(filepath.Join("thumb", stem+thumbExt))
	case ".webm", ".mkv", ".avi", ".wmv", ".flv":
		// 非 H.264 MP4 容器：暂不自动生成缩略图
		return ""
	default:
		return ""
	}
}

// EffectiveThumbnailRel 返回 resource/{folderKey} 下实际存在的缩略图相对路径（兼容 .jpeg 配置、canonical .jpg/.png、旧版 *-thumbnail.jpg）。
func (s *Store) EffectiveThumbnailRel(folderKey, thumbRel string) string {
	thumbRel = strings.TrimSpace(thumbRel)
	if thumbRel == "" {
		return ""
	}
	slash := filepath.ToSlash(thumbRel)
	try := func(rel string) bool {
		p := filepath.Join(s.dataDir, "resource", folderKey, filepath.FromSlash(rel))
		_, err := os.Stat(p)
		return err == nil
	}
	if try(slash) {
		return slash
	}
	if !strings.HasPrefix(strings.ToLower(slash), "thumb/") {
		return slash
	}
	base := filepath.Base(slash)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" {
		return slash
	}
	candidates := []string{
		filepath.ToSlash(filepath.Join("thumb", stem+".jpg")),
		filepath.ToSlash(filepath.Join("thumb", stem+".png")),
		filepath.ToSlash(filepath.Join("thumb", stem+".jpeg")),
		filepath.ToSlash(filepath.Join("thumb", stem+"-thumbnail.jpg")),
	}
	seen := map[string]struct{}{slash: {}}
	for _, c := range candidates {
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		if try(c) {
			return c
		}
	}
	return slash
}

func (s *Store) categoriesPath() string {
	return filepath.Join(s.dataDir, "categories.json")
}

func (s *Store) itemsPath(folderKey string) string {
	return filepath.Join(s.dataDir, "resource", folderKey, "items.json")
}

func (s *Store) ReadCategories() (*model.CategoriesDoc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.categoriesPath())
	if err != nil {
		return nil, err
	}
	var doc model.CategoriesDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Store) writeCategoriesUnlocked(doc *model.CategoriesDoc) error {
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.categoriesPath(), raw, 0o644)
}

func (s *Store) ReadItems(folderKey string) ([]model.Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.readItemsUnlocked(folderKey)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) readItemsUnlocked(folderKey string) ([]model.Item, error) {
	p := s.itemsPath(folderKey)
	raw, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc model.ItemsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return doc.Items, nil
}

func (s *Store) writeItemsUnlocked(folderKey string, items []model.Item) error {
	doc := model.ItemsDoc{Items: items}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.itemsPath(folderKey), raw, 0o644)
}

func (s *Store) UpdateCategoryLayout(folderKey string, layout model.Layout) (*model.Subcategory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return nil, err
	}
	foundMajor, foundSub, ok := lookupInDoc(doc, folderKey)
	if !ok {
		return nil, os.ErrNotExist
	}
	doc.Categories[foundMajor].Subcategories[foundSub].Layout = layout
	if err := s.writeCategoriesUnlocked(doc); err != nil {
		return nil, err
	}
	c := doc.Categories[foundMajor].Subcategories[foundSub]
	return &c, nil
}

func (s *Store) UpdateCategoryItemOrder(
	folderKey string,
	orderedItemIDs []string,
	masonryPlacement map[string]model.MasonryPosition,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return err
	}
	mi, sj, ok := lookupInDoc(doc, folderKey)
	if !ok {
		return os.ErrNotExist
	}

	items, err := s.readItemsUnlocked(folderKey)
	if err != nil {
		return err
	}

	byID := make(map[string]model.Item, len(items))
	for _, it := range items {
		if it.ID == "" {
			continue
		}
		byID[it.ID] = it
	}

	used := make(map[string]struct{}, len(orderedItemIDs))
	ordered := make([]model.Item, 0, len(items))
	for _, id := range orderedItemIDs {
		if id == "" {
			continue
		}
		if _, exists := used[id]; exists {
			continue
		}
		it, ok := byID[id]
		if !ok {
			continue
		}
		used[id] = struct{}{}
		ordered = append(ordered, it)
	}
	for _, it := range items {
		if it.ID == "" {
			continue
		}
		if _, ok := used[it.ID]; ok {
			continue
		}
		ordered = append(ordered, it)
	}

	for i := range ordered {
		ordered[i].Sort = (i + 1) * 10
		if masonryPlacement != nil {
			if pos, ok := masonryPlacement[ordered[i].ID]; ok && pos.Col >= 0 && pos.Row >= 0 {
				ordered[i].MasonryCol = pos.Col + 1
				ordered[i].MasonryRow = pos.Row + 1
			}
		}
	}

	if err := s.writeItemsUnlocked(folderKey, ordered); err != nil {
		return err
	}

	doc.Categories[mi].Subcategories[sj].ItemSortBy = "sort"
	return s.writeCategoriesUnlocked(doc)
}

func (s *Store) readCategoriesUnlocked() (*model.CategoriesDoc, error) {
	raw, err := os.ReadFile(s.categoriesPath())
	if err != nil {
		return nil, err
	}
	var doc model.CategoriesDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Store) GetCategoryByFolderKey(folderKey string) (*model.Subcategory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return nil, err
	}
	mi, sj, ok := lookupInDoc(doc, folderKey)
	if !ok {
		return nil, os.ErrNotExist
	}
	c := doc.Categories[mi].Subcategories[sj]
	return &c, nil
}

// LookupFolderInCategories 在 categories.json 中查找 folderKey 所属大分类与二级。
func (s *Store) LookupFolderInCategories(folderKey string) (model.Category, model.Subcategory, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return model.Category{}, model.Subcategory{}, false
	}
	mi, sj, ok := lookupInDoc(doc, folderKey)
	if !ok {
		return model.Category{}, model.Subcategory{}, false
	}
	return doc.Categories[mi], doc.Categories[mi].Subcategories[sj], true
}
