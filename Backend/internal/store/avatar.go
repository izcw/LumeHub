package store

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"lumehub/internal/model"
)

var avatarExts = []string{".png", ".jpg", ".jpeg", ".webp"}

const (
	avatarMaxBytes      = 10 << 20 // 10MB
	avatarRasterMaxEdge = 512     // 头像长边上限（列表缩略图逻辑的更小版本）
)

var (
	ErrAvatarUnsupportedType = errors.New("unsupported avatar image type")
	ErrAvatarTooLarge        = errors.New("avatar file too large")
)

func (s *Store) avatarDir() string {
	return filepath.Join(s.dataDir, "system", "avatar")
}

func normalizeAvatarExt(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp":
		if ext == ".jpeg" {
			return ".jpg", nil
		}
		return ext, nil
	default:
		return "", ErrAvatarUnsupportedType
	}
}

func avatarStemsForAccount(acc model.Account) []string {
	stems := []string{strings.TrimSpace(acc.ID)}
	if e := strings.TrimSpace(acc.Email); e != "" {
		stems = append(stems, e)
	}
	if u := strings.TrimSpace(acc.Username); u != "" {
		stems = append(stems, u)
	}
	return stems
}

func (s *Store) removeAvatarFilesUnlocked(stems ...string) error {
	base := s.avatarDir()
	seen := make(map[string]struct{})
	for _, stem := range stems {
		stem = strings.TrimSpace(stem)
		if stem == "" {
			continue
		}
		if _, ok := seen[stem]; ok {
			continue
		}
		seen[stem] = struct{}{}
		for _, ext := range avatarExts {
			p := filepath.Join(base, stem+ext)
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

// AccountAvatarFilePath 返回 data/system/avatar 下与账号匹配的头像文件路径（邮箱、用户名或 id + 常见扩展名）。
func (s *Store) AccountAvatarFilePath(acc model.Account) (string, bool) {
	base := s.avatarDir()
	try := func(stem string) (string, bool) {
		if stem == "" {
			return "", false
		}
		for _, ext := range avatarExts {
			p := filepath.Join(base, stem+ext)
			fi, err := os.Stat(p)
			if err == nil && !fi.IsDir() {
				return p, true
			}
		}
		return "", false
	}
	if p, ok := try(strings.TrimSpace(acc.Email)); ok {
		return p, true
	}
	if p, ok := try(strings.TrimSpace(acc.Username)); ok {
		return p, true
	}
	if p, ok := try(strings.TrimSpace(acc.ID)); ok {
		return p, true
	}
	return "", false
}

func writeAccountAvatarImage(destPath string, payload []byte, filename string) error {
	img, ok := decodeGalleryRasterForThumbnail(payload, filename)
	if !ok {
		return ErrAvatarUnsupportedType
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	longPx := thumbnailLongEdgePx(img)
	maxEdge := pickThumbnailMaxEdge(w, h, int64(len(payload)))
	if maxEdge > avatarRasterMaxEdge {
		maxEdge = avatarRasterMaxEdge
	}
	q := pickJPEGQuality(longPx, maxEdge)
	return writeRasterThumbnail(destPath, img, maxEdge, q)
}

// SaveAccountAvatar 将上传图片压缩后保存到 data/system/avatar/{id}.jpg，并更新账号头像 URL。
func (s *Store) SaveAccountAvatar(id string, r io.Reader, filename string, size int64) (*model.Account, error) {
	if size > avatarMaxBytes {
		return nil, ErrAvatarTooLarge
	}
	if _, err := normalizeAvatarExt(filename); err != nil {
		return nil, err
	}

	payload, err := io.ReadAll(io.LimitReader(r, avatarMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > avatarMaxBytes {
		return nil, ErrAvatarTooLarge
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readAccountsUnlocked()
	if err != nil {
		return nil, err
	}
	found := -1
	for i := range doc.Accounts {
		if doc.Accounts[i].ID == id {
			found = i
			break
		}
	}
	if found < 0 {
		return nil, os.ErrNotExist
	}
	acc := doc.Accounts[found]

	if err := os.MkdirAll(s.avatarDir(), 0o755); err != nil {
		return nil, err
	}
	if err := s.removeAvatarFilesUnlocked(avatarStemsForAccount(acc)...); err != nil {
		return nil, err
	}

	dest := filepath.Join(s.avatarDir(), id+".jpg")
	if err := writeAccountAvatarImage(dest, payload, filename); err != nil {
		return nil, err
	}

	doc.Accounts[found].Avatar = "/api/avatar/" + id
	if err := s.writeAccountsUnlocked(doc); err != nil {
		return nil, err
	}
	c := doc.Accounts[found]
	return &c, nil
}

// ClearAccountAvatar 删除账号在 data/system/avatar 下的头像文件，并清空 avatar 字段。
func (s *Store) ClearAccountAvatar(id string) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readAccountsUnlocked()
	if err != nil {
		return nil, err
	}
	found := -1
	for i := range doc.Accounts {
		if doc.Accounts[i].ID == id {
			found = i
			break
		}
	}
	if found < 0 {
		return nil, os.ErrNotExist
	}
	acc := doc.Accounts[found]
	if err := s.removeAvatarFilesUnlocked(avatarStemsForAccount(acc)...); err != nil {
		return nil, err
	}
	doc.Accounts[found].Avatar = ""
	if err := s.writeAccountsUnlocked(doc); err != nil {
		return nil, err
	}
	c := doc.Accounts[found]
	return &c, nil
}
