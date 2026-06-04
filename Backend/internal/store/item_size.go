package store

import (
	"os"
	"path/filepath"
	"strings"

	"lumehub/internal/model"
)

func itemCanonicalResourceRel(it model.Item) string {
	return strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(it.Filename)), "/")
}

func itemDisplayResourceRel(it model.Item) string {
	if it.UseEdited {
		ed := strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(it.EditedFilename)), "/")
		if ed != "" {
			return ed
		}
	}
	return itemCanonicalResourceRel(it)
}

func (s *Store) statResourceFileSize(folderKey, rel string) int64 {
	rel = strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(rel)), "/")
	if rel == "" {
		return 0
	}
	p := filepath.Join(s.dataDir, "resource", folderKey, filepath.FromSlash(rel))
	st, err := os.Stat(p)
	if err != nil || st.Size() <= 0 {
		return 0
	}
	return st.Size()
}

// EffectiveItemFileSize 返回条目主资源字节数；优先从磁盘 stat（展示文件 → 原版），避免 items.json 中过期的 fileSize。
func (s *Store) EffectiveItemFileSize(folderKey string, it model.Item) int64 {
	for _, rel := range []string{itemDisplayResourceRel(it), itemCanonicalResourceRel(it)} {
		if size := s.statResourceFileSize(folderKey, rel); size > 0 {
			return size
		}
	}
	if it.FileSize > 0 {
		return it.FileSize
	}
	return 0
}
