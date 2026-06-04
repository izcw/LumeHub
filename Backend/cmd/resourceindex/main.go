// resourceindex 根据 data/resource/{分类}/ 下的实际文件重写 items.json（符合最新目录规范）。
package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"lumehub/internal/config"
	"lumehub/internal/model"
	"lumehub/internal/store"
)

func main() {
	root := filepath.Join(config.DataDir(), "resource")
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		folderKey := e.Name()
		dir := filepath.Join(root, folderKey)
		if err := writeItemsJSON(folderKey, dir); err != nil {
			log.Printf("%s: %v", folderKey, err)
		} else {
			fmt.Println("ok", folderKey)
		}
	}
}

func writeItemsJSON(folderKey, dir string) error {
	names, err := collectFilenames(dir)
	if err != nil {
		return err
	}
	slices.Sort(names)

	items := make([]model.Item, 0, len(names))
	seq := 0
	for _, name := range names {
		full := filepath.Join(dir, name)
		st, err := os.Stat(full)
		uploadedAt, updatedAt := "", ""
		if err == nil {
			t := st.ModTime().UTC().Format(time.RFC3339Nano)
			uploadedAt, updatedAt = t, t
		}
		fileSize := int64(0)
		if err == nil {
			fileSize = st.Size()
		}
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
		isDev := strings.HasPrefix(filepath.ToSlash(name), "_dev/")
		if isDev {
			items = append(items, model.Item{
				ID:         idFromFilename(name),
				UploadedAt: uploadedAt,
				UpdatedAt:  updatedAt,
				Filename:   filepath.ToSlash(name),
				FileSize:   fileSize,
				Tags:       []string{ext},
			})
			continue
		}
		seq++
		id := idFromMeta(folderKey, name, uploadedAt, seq)
		linkName := id + "." + ext
		item := model.Item{
			ID:         id,
			Sort:       seq * 10,
			UploadedAt: uploadedAt,
			UpdatedAt:  updatedAt,
			Filename:   filepath.ToSlash(name),
			FileSize:   fileSize,
			LinkName:   linkName,
			Tags:       []string{ext},
		}
		if th := defaultThumbnailForName(dir, name); th != "" {
			item.Thumbnail = th
		} else if strings.HasPrefix(filepath.ToSlash(name), "original/") {
			if gen := store.GenerateGalleryThumbnail(dir, name, full, nil, fileSize); gen != "" {
				item.Thumbnail = gen
			}
		}
		items = append(items, item)
	}
	doc := model.ItemsDoc{Items: items}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	out := filepath.Join(dir, "items.json")
	return os.WriteFile(out, raw, 0o644)
}

func datePart(iso string) string {
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		return "1970-01-01"
	}
	return t.UTC().Format("20060102")
}

func idFromMeta(folderKey, filename, uploadedAt string, seq int) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%s|%s|%d", folderKey, filename, uploadedAt, seq)))
	token := fmt.Sprintf("%x", sum[:])[:12]
	return token + "_" + datePart(uploadedAt)
}

func defaultThumbnailForName(dir, name string) string {
	cand := store.ThumbRelForOriginal(filepath.ToSlash(name))
	if cand != "" {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(cand))); err == nil {
			return cand
		}
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".avif":
	default:
		return ""
	}
	baseName := filepath.Base(name)
	legacyBase := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if legacyBase == "" {
		return ""
	}
	legacy := filepath.Join("thumb", legacyBase+"-thumbnail.jpg")
	if _, err := os.Stat(filepath.Join(dir, legacy)); err == nil {
		return filepath.ToSlash(legacy)
	}
	return ""
}

func collectFilenames(dir string) ([]string, error) {
	var names []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if rel == "thumb" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Base(rel), "items.json") {
			return nil
		}
		names = append(names, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

func idFromFilename(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		return "file"
	}
	return base
}
