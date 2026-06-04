package auth

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"lumehub/internal/model"
	"lumehub/internal/store"
)

// ResourceDir 校验静态资源路径：加密目录校验 ?k= / ?vg= / ?vp=；私密目录直链不拦截。
func ResourceDir(m *Manager, st *store.Store, root http.FileSystem) http.Handler {
	fs := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fs.ServeHTTP(w, r)
			return
		}
		p := path.Clean("/" + r.URL.Path)
		segs := strings.Split(strings.Trim(p, "/"), "/")
		if len(segs) == 0 || segs[0] == "." {
			http.NotFound(w, r)
			return
		}

		// 全局短链：/resource/{stem}（不含一级/二级目录）
		if len(segs) == 1 {
			if _, _, isFolder := st.LookupFolderInCategories(segs[0]); !isFolder {
				if fk, rel, ok := resolveGlobalLinkStem(st, segs[0]); ok {
					if !allowResourceAccess(m, st, r, w, fk) {
						return
					}
					serveResourcePath(w, r, fs, "/"+fk+"/"+rel)
					return
				}
			}
		}

		fk, tail, okPath := resolveFolderFromResourcePath(st, segs)
		if !okPath {
			fs.ServeHTTP(w, r)
			return
		}
		if fk == ".." || strings.Contains(fk, "\\") {
			http.NotFound(w, r)
			return
		}
		if !allowResourceAccess(m, st, r, w, fk) {
			return
		}

		internalPath := "/" + fk
		if len(tail) > 0 {
			internalPath = "/" + fk + "/" + strings.Join(tail, "/")
		}
		if rewritten, ok := rewriteResourcePathByAlias(st, fk, tail); ok {
			internalPath = rewritten
		}
		serveResourcePath(w, r, fs, internalPath)
	})
}

func allowResourceAccess(m *Manager, st *store.Store, r *http.Request, w http.ResponseWriter, folderKey string) bool {
	maj, sub, ok := st.LookupFolderInCategories(folderKey)
	if !ok {
		return true
	}
	if model.FolderResourceRequiresViewKey(maj, sub) {
		passHash := model.FolderEncryptedPasswordHash(maj, sub)
		sawKey := false
		for _, key := range []string{
			strings.TrimSpace(r.URL.Query().Get("vg")),
			strings.TrimSpace(r.URL.Query().Get("vp")),
			strings.TrimSpace(r.URL.Query().Get("k")),
		} {
			if key == "" {
				continue
			}
			sawKey = true
			if m.ValidViewGrant(folderKey, key) {
				return true
			}
			if passHash != "" && model.PasswordHashMatches(passHash, key) {
				return true
			}
		}
		if sawKey {
			http.Error(w, "Access denied: invalid view key.", http.StatusForbidden)
			return false
		}
		http.Error(w, "Access denied: missing view key. Use ?k=YOUR_KEY.", http.StatusForbidden)
		return false
	}
	return true
}

func serveResourcePath(w http.ResponseWriter, r *http.Request, fs http.Handler, internalPath string) {
	internalPath = path.Clean("/" + strings.TrimPrefix(internalPath, "/"))
	r2 := r.Clone(r.Context())
	u2 := *r.URL
	u2.Path = internalPath
	r2.URL = &u2
	fs.ServeHTTP(w, r2)
}

func itemDisplayResourceRel(it model.Item) string {
	if it.UseEdited {
		if rel := strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(it.EditedFilename)), "/"); rel != "" {
			return rel
		}
	}
	return strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(it.Filename)), "/")
}

func linkStem(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func itemMatchesResourceAlias(reqNorm string, it model.Item) bool {
	fn := strings.Trim(strings.TrimSpace(filepath.ToSlash(it.Filename)), "/")
	if fn != "" && normalizeResourceAlias(fn) == reqNorm {
		return true
	}
	ln := strings.TrimSpace(it.LinkName)
	if ln == "" {
		return false
	}
	if normalizeResourceAlias(ln) == reqNorm {
		return true
	}
	// 仅对单段别名（无 /）做 stem 匹配，避免 thumb/xxx.jpg 误命中 original/xxx.mp4。
	if !strings.Contains(reqNorm, "/") {
		stem := strings.ToLower(linkStem(ln))
		if stem != "" && stem == reqNorm {
			return true
		}
	}
	return false
}

func resolveGlobalLinkStem(st *store.Store, alias string) (folderKey, rel string, ok bool) {
	aliasNorm := strings.ToLower(strings.TrimSpace(alias))
	if aliasNorm == "" || strings.Contains(aliasNorm, "/") || strings.Contains(aliasNorm, `\`) {
		return "", "", false
	}
	doc, err := st.ReadCategories()
	if err != nil {
		return "", "", false
	}
	for _, maj := range doc.Categories {
		for _, sub := range maj.Subcategories {
			fk := strings.TrimSpace(sub.FolderKey)
			if fk == "" {
				continue
			}
			items, err := st.ReadItems(fk)
			if err != nil {
				continue
			}
			for _, it := range items {
				ln := strings.TrimSpace(it.LinkName)
				if ln == "" {
					continue
				}
				stem := strings.ToLower(linkStem(ln))
				if stem != aliasNorm && normalizeResourceAlias(ln) != aliasNorm {
					continue
				}
				if rel := itemDisplayResourceRel(it); rel != "" {
					return fk, rel, true
				}
			}
		}
	}
	return "", "", false
}

func rewriteResourcePathByAlias(st *store.Store, folderKey string, tail []string) (string, bool) {
	if len(tail) == 0 {
		return "", false
	}
	raw := strings.Trim(strings.Join(tail, "/"), "/")
	if raw == "" || raw == "." || strings.Contains(raw, `\`) || strings.Contains(raw, "..") {
		return "", false
	}
	items, err := st.ReadItems(folderKey)
	if err != nil || len(items) == 0 {
		return "", false
	}
	reqNorm := normalizeResourceAlias(raw)
	// 缩略图路径须优先解析：thumb/{stem}.jpg 的文件名 stem 常与 linkName 相同，
	// 若先走条目别名会误指向 original/*.mp4，浏览器按图片加载会失败并显示黑块。
	for _, it := range items {
		th := strings.Trim(strings.TrimSpace(filepath.ToSlash(it.Thumbnail)), "/")
		if th == "" {
			continue
		}
		eff := st.EffectiveThumbnailRel(folderKey, it.Thumbnail)
		tn := normalizeResourceAlias(th)
		en := normalizeResourceAlias(filepath.ToSlash(strings.TrimSpace(eff)))
		if tn == reqNorm || en == reqNorm {
			serve := strings.Trim(filepath.ToSlash(strings.TrimSpace(eff)), "/")
			if serve == "" {
				serve = th
			}
			return "/" + folderKey + "/" + serve, true
		}
	}
	for _, it := range items {
		if !itemMatchesResourceAlias(reqNorm, it) {
			continue
		}
		if rel := itemDisplayResourceRel(it); rel != "" {
			return "/" + folderKey + "/" + rel, true
		}
	}
	return "", false
}

func resolveFolderFromResourcePath(st *store.Store, segs []string) (folderKey string, tail []string, ok bool) {
	if len(segs) == 0 {
		return "", nil, false
	}
	if _, _, exists := st.LookupFolderInCategories(segs[0]); exists {
		return segs[0], segs[1:], true
	}
	if len(segs) >= 2 {
		majorKey := strings.TrimSpace(strings.ToLower(segs[0]))
		candidate := strings.TrimSpace(segs[1])
		maj, _, exists := st.LookupFolderInCategories(candidate)
		if exists {
			mk := strings.TrimSpace(strings.ToLower(maj.Key))
			if mk != "" && mk == majorKey {
				return candidate, segs[2:], true
			}
		}
	}
	return "", nil, false
}

func normalizeResourceAlias(v string) string {
	s := strings.Trim(strings.TrimSpace(v), "/")
	s = filepath.ToSlash(s)
	return strings.ToLower(s)
}
