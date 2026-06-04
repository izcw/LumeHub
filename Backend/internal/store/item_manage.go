package store

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lumehub/internal/model"
)

func sanitizeLinkStem(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", os.ErrInvalid
	}
	if strings.Contains(raw, "/") || strings.Contains(raw, "\\") || strings.Contains(raw, "..") {
		return "", os.ErrInvalid
	}
	base := filepath.Base(raw)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" || stem == "." {
		return "", os.ErrInvalid
	}
	return stem, nil
}

func linkStemFromName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func composeLinkName(stem, ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return stem + ext
}

func sanitizeItemLinkName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	stem, err := sanitizeLinkStem(name)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(filepath.Base(name))
	if ext == "" {
		ext = ".jpg"
	}
	return composeLinkName(stem, ext), nil
}

func itemResourcePaths(resourceDir string, it model.Item) []string {
	var out []string
	add := func(rel string) {
		rel = strings.TrimSpace(filepath.ToSlash(rel))
		if rel == "" {
			return
		}
		out = append(out, filepath.Join(resourceDir, filepath.FromSlash(rel)))
	}
	add(it.Filename)
	add(it.EditedFilename)
	add(it.Thumbnail)
	add(it.RawFilename)
	return out
}

func editedRelForItem(itemID, uploadName string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(uploadName)))
	if ext == "" {
		ext = ".jpg"
	}
	return filepath.ToSlash(filepath.Join("edited", itemID+ext))
}

func (s *Store) regenerateItemThumbnail(resourceDir string, sourceRel string, item *model.Item) error {
	sourceRel = strings.TrimSpace(filepath.ToSlash(sourceRel))
	if sourceRel == "" {
		return nil
	}
	srcPath := filepath.Join(resourceDir, filepath.FromSlash(sourceRel))
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if thumbRel := GenerateGalleryThumbnail(resourceDir, sourceRel, srcPath, data, int64(len(data))); thumbRel != "" {
		item.Thumbnail = thumbRel
		return nil
	}

	img, ok := decodeGalleryRasterForThumbnail(data, sourceRel)
	if !ok {
		item.Thumbnail = ""
		return nil
	}
	thumbRel := ThumbRelForOriginal(sourceRel)
	if thumbRel == "" {
		// 编辑版等非 original/ 路径：thumb/edited/{id}.jpg
		base := filepath.Base(sourceRel)
		stem := strings.TrimSuffix(base, filepath.Ext(base))
		if stem == "" {
			stem = "edited"
		}
		ext := strings.ToLower(filepath.Ext(sourceRel))
		if ext == ".png" {
			thumbRel = filepath.ToSlash(filepath.Join("thumb", "edited", stem+".png"))
		} else {
			thumbRel = filepath.ToSlash(filepath.Join("thumb", "edited", stem+".jpg"))
		}
	}
	item.Thumbnail = thumbRel
	thumbPath := filepath.Join(resourceDir, filepath.FromSlash(thumbRel))
	if err := os.MkdirAll(filepath.Dir(thumbPath), 0o755); err != nil {
		return err
	}
	return writeGalleryThumbnail(sourceRel, data, int64(len(data)), img, thumbPath)
}

// SetItemThumbnailFromReader 用客户端提供的封面图覆盖条目缩略图（适用于 HEVC 等服务端无法抽帧的视频）。
func (s *Store) SetItemThumbnailFromReader(folderKey, itemID string, src io.Reader, payloadSize int64) (model.Item, error) {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" || src == nil {
		return model.Item{}, os.ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if payloadSize > 0 {
		if err := s.checkStorageQuotaUnlocked(payloadSize); err != nil {
			return model.Item{}, err
		}
	}

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return model.Item{}, err
	}
	if _, _, ok := lookupInDoc(doc, folderKey); !ok {
		return model.Item{}, os.ErrNotExist
	}

	items, err := s.readItemsUnlocked(folderKey)
	if err != nil {
		return model.Item{}, err
	}
	idx := -1
	var item model.Item
	for i, it := range items {
		if it.ID == itemID {
			idx = i
			item = it
			break
		}
	}
	if idx < 0 {
		return model.Item{}, os.ErrNotExist
	}
	origRel := strings.TrimSpace(filepath.ToSlash(item.Filename))
	if !strings.HasPrefix(origRel, "original/") {
		return model.Item{}, os.ErrInvalid
	}
	if !isVideoOriginalExt(strings.ToLower(filepath.Ext(origRel))) {
		return model.Item{}, os.ErrInvalid
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return model.Item{}, err
	}
	if len(data) == 0 {
		return model.Item{}, os.ErrInvalid
	}
	img, ok := decodeGalleryRasterForThumbnail(data, origRel)
	if !ok || img == nil {
		return model.Item{}, os.ErrInvalid
	}

	resourceDir := filepath.Join(s.dataDir, "resource", folderKey)
	thumbRel := ThumbRelForOriginal(origRel)
	if thumbRel == "" {
		return model.Item{}, os.ErrInvalid
	}
	thumbPath := filepath.Join(resourceDir, filepath.FromSlash(thumbRel))
	if err := writeGalleryThumbnail(origRel, data, int64(len(data)), img, thumbPath); err != nil {
		return model.Item{}, err
	}
	item.Thumbnail = thumbRel
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	items[idx] = item
	if err := s.writeItemsUnlocked(folderKey, items); err != nil {
		return model.Item{}, err
	}
	s.touchStorageUsedAfterMutationUnlocked()
	return item, nil
}

// DeleteCategoryItem 将条目移入回收站（软删除，保留磁盘文件）。
func (s *Store) DeleteCategoryItem(folderKey, itemID string) error {
	return s.MoveCategoryItemToTrash(folderKey, itemID)
}

type PatchCategoryItemInput struct {
	SetLinkName     bool
	LinkName        string
	SetTitle        bool
	Title           string
	HasTags         bool
	Tags            []string
	File            io.Reader
	Filename        string
	SaveAsEdited    bool
	ReplaceOriginal bool
	RevertEdited    bool
}

// PatchCategoryItem 更新元数据；File 可写入 edited/（默认）或覆盖 original/。
func (s *Store) PatchCategoryItem(folderKey, itemID string, in PatchCategoryItemInput) (model.Item, error) {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return model.Item{}, os.ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return model.Item{}, err
	}
	if _, _, ok := lookupInDoc(doc, folderKey); !ok {
		return model.Item{}, os.ErrNotExist
	}

	items, err := s.readItemsUnlocked(folderKey)
	if err != nil {
		return model.Item{}, err
	}

	idx := -1
	for i, it := range items {
		if it.ID == itemID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return model.Item{}, os.ErrNotExist
	}

	item := items[idx]
	resourceDir := filepath.Join(s.dataDir, "resource", folderKey)

	if in.RevertEdited {
		item.UseEdited = false
		oldThumb := item.Thumbnail
		if err := s.regenerateItemThumbnail(resourceDir, item.Filename, &item); err != nil {
			return model.Item{}, err
		}
		if oldThumb != "" && oldThumb != item.Thumbnail {
			_ = os.Remove(filepath.Join(resourceDir, filepath.FromSlash(oldThumb)))
		}
		item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		items[idx] = item
		if err := s.writeItemsUnlocked(folderKey, items); err != nil {
			return model.Item{}, err
		}
		return item, nil
	}

	if in.SetLinkName {
		stem, err := sanitizeLinkStem(in.LinkName)
		if err != nil {
			return model.Item{}, err
		}
		ext := filepath.Ext(filepath.Base(item.Filename))
		if ext == "" {
			ext = ".jpg"
		}
		ln := composeLinkName(stem, ext)
		stemLower := strings.ToLower(stem)
		for _, maj := range doc.Categories {
			for _, sub := range maj.Subcategories {
				otherFK := strings.TrimSpace(sub.FolderKey)
				if otherFK == "" {
					continue
				}
				others, err := s.readItemsUnlocked(otherFK)
				if err != nil {
					continue
				}
				for _, other := range others {
					if other.ID == itemID && otherFK == folderKey {
						continue
					}
					otherStem := strings.ToLower(linkStemFromName(other.LinkName))
					if otherStem == "" {
						otherStem = strings.ToLower(linkStemFromName(filepath.Base(other.Filename)))
					}
					if otherStem == stemLower {
						return model.Item{}, os.ErrExist
					}
				}
			}
		}
		item.LinkName = ln
	}

	if in.SetTitle {
		item.Title = strings.TrimSpace(in.Title)
	}

	if in.HasTags {
		clean := make([]string, 0, len(in.Tags))
		seen := make(map[string]struct{}, len(in.Tags))
		for _, t := range in.Tags {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			key := strings.ToLower(t)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			clean = append(clean, t)
		}
		item.Tags = clean
	}

	if in.File != nil {
		data, err := io.ReadAll(in.File)
		if err != nil {
			return model.Item{}, err
		}
		if len(data) == 0 {
			return model.Item{}, os.ErrInvalid
		}
		// For replacements, only check the net additional bytes (new - old).
		additionalBytes := int64(len(data))
		if in.ReplaceOriginal {
			oldSize := s.statResourceFileSize(folderKey, strings.TrimSpace(filepath.ToSlash(item.Filename)))
			additionalBytes -= oldSize
			if additionalBytes < 0 {
				additionalBytes = 0
			}
		}
		if err := s.checkStorageQuotaUnlocked(additionalBytes); err != nil {
			return model.Item{}, err
		}

		saveAsEdited := in.SaveAsEdited && !in.ReplaceOriginal
		var dstRel string
		if saveAsEdited {
			editedDir := filepath.Join(resourceDir, "edited")
			if err := os.MkdirAll(editedDir, 0o755); err != nil {
				return model.Item{}, err
			}
			dstRel = editedRelForItem(itemID, in.Filename)
			item.EditedFilename = dstRel
			item.UseEdited = true
		} else {
			dstRel = strings.TrimSpace(filepath.ToSlash(item.Filename))
			if dstRel == "" {
				return model.Item{}, os.ErrInvalid
			}
			// Replace-original: clear stale edited version so display uses the new original.
			if oldEdited := strings.TrimSpace(item.EditedFilename); oldEdited != "" {
				_ = os.Remove(filepath.Join(resourceDir, filepath.FromSlash(oldEdited)))
				item.EditedFilename = ""
			}
			item.UseEdited = false
		}

		dstPath := filepath.Join(resourceDir, filepath.FromSlash(dstRel))
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return model.Item{}, err
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			return model.Item{}, err
		}
		// 仅覆盖 original/ 时更新主资源大小；写入 edited/ 不覆盖（避免元数据显示成小编辑文件体积）
		if !saveAsEdited {
			item.FileSize = int64(len(data))
		} else if item.FileSize <= 0 {
			if size := s.statResourceFileSize(folderKey, itemCanonicalResourceRel(item)); size > 0 {
				item.FileSize = size
			}
		}

		oldThumb := item.Thumbnail
		if err := s.regenerateItemThumbnail(resourceDir, dstRel, &item); err != nil {
			return model.Item{}, err
		}
		if oldThumb != "" && oldThumb != item.Thumbnail {
			_ = os.Remove(filepath.Join(resourceDir, filepath.FromSlash(oldThumb)))
		}
	}

	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	items[idx] = item
	if err := s.writeItemsUnlocked(folderKey, items); err != nil {
		return model.Item{}, err
	}
	if in.File != nil {
		s.touchStorageUsedAfterMutationUnlocked()
	}
	return item, nil
}

func appendUniqueTag(tags []string, tag string) []string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return tags
	}
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return tags
		}
	}
	return append(tags, tag)
}

// AttachItemCompanionRaw 为已有条目附加伴生文件（如实况图 MOV）。
func (s *Store) AttachItemCompanionRaw(folderKey, itemID, rawFilename string, raw io.Reader) (model.Item, error) {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" || raw == nil {
		return model.Item{}, os.ErrInvalid
	}
	rawExt := strings.ToLower(filepath.Ext(strings.TrimSpace(rawFilename)))
	if rawExt == "" {
		rawExt = ".mov"
	}
	if !IsLiveVideoCompanionExt(rawExt) {
		return model.Item{}, os.ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return model.Item{}, err
	}
	if _, _, ok := lookupInDoc(doc, folderKey); !ok {
		return model.Item{}, os.ErrNotExist
	}

	items, err := s.readItemsUnlocked(folderKey)
	if err != nil {
		return model.Item{}, err
	}

	idx := -1
	var item model.Item
	for i, it := range items {
		if it.ID == itemID {
			idx = i
			item = it
			break
		}
	}
	if idx < 0 {
		return model.Item{}, os.ErrNotExist
	}

	resourceDir := filepath.Join(s.dataDir, "resource", folderKey)
	if strings.TrimSpace(item.RawFilename) != "" {
		oldPath := filepath.Join(resourceDir, filepath.FromSlash(item.RawFilename))
		_ = os.Remove(oldPath)
	}

	rawStoredPath := filepath.ToSlash(filepath.Join("original", itemID+"_raw"+rawExt))
	rawData, err := io.ReadAll(raw)
	if err != nil {
		return model.Item{}, err
	}
	if err := s.checkStorageQuotaUnlocked(int64(len(rawData))); err != nil {
		return model.Item{}, err
	}
	if err := os.WriteFile(filepath.Join(resourceDir, filepath.FromSlash(rawStoredPath)), rawData, 0o644); err != nil {
		return model.Item{}, err
	}

	item.RawFilename = rawStoredPath
	if strings.TrimSpace(item.GroupID) == "" {
		item.GroupID = "grp_" + itemID
	}
	item.Tags = appendUniqueTag(item.Tags, "live")
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	items[idx] = item
	if err := s.writeItemsUnlocked(folderKey, items); err != nil {
		return model.Item{}, err
	}
	s.touchStorageUsedAfterMutationUnlocked()
	return item, nil
}

func moveItemResourceFiles(srcDir, dstDir string, it model.Item) error {
	for _, abs := range itemResourcePaths(srcDir, it) {
		info, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(srcDir, abs)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstDir, rel)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		if err := os.Rename(abs, dstPath); err != nil {
			return err
		}
	}
	return nil
}

// MoveCategoryItem 将条目从当前二级目录转移到同一大分类下的另一二级目录。
func (s *Store) MoveCategoryItem(fromFolderKey, toFolderKey, itemID string) error {
	fromFolderKey = strings.TrimSpace(fromFolderKey)
	toFolderKey = strings.TrimSpace(toFolderKey)
	itemID = strings.TrimSpace(itemID)
	if fromFolderKey == "" || toFolderKey == "" || itemID == "" {
		return os.ErrInvalid
	}
	if fromFolderKey == toFolderKey {
		return os.ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return err
	}
	srcMajorIdx, _, srcOk := lookupInDoc(doc, fromFolderKey)
	dstMajorIdx, _, dstOk := lookupInDoc(doc, toFolderKey)
	if !srcOk || !dstOk {
		return os.ErrNotExist
	}
	if srcMajorIdx != dstMajorIdx {
		return os.ErrInvalid
	}

	srcItems, err := s.readItemsUnlocked(fromFolderKey)
	if err != nil {
		return err
	}
	idx := -1
	var item model.Item
	for i, it := range srcItems {
		if it.ID == itemID {
			idx = i
			item = it
			break
		}
	}
	if idx < 0 {
		return os.ErrNotExist
	}

	dstItems, err := s.readItemsUnlocked(toFolderKey)
	if err != nil {
		return err
	}
	for _, other := range dstItems {
		if other.ID == itemID {
			return os.ErrExist
		}
		if item.LinkName != "" && other.LinkName != "" &&
			strings.EqualFold(item.LinkName, other.LinkName) {
			return os.ErrExist
		}
	}

	srcDir := filepath.Join(s.dataDir, "resource", fromFolderKey)
	dstDir := filepath.Join(s.dataDir, "resource", toFolderKey)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	if err := moveItemResourceFiles(srcDir, dstDir, item); err != nil {
		return err
	}

	srcItems = append(srcItems[:idx], srcItems[idx+1:]...)
	if err := s.writeItemsUnlocked(fromFolderKey, srcItems); err != nil {
		return err
	}

	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	dstItems = append(dstItems, item)
	if err := s.writeItemsUnlocked(toFolderKey, dstItems); err != nil {
		return err
	}
	return nil
}
