package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"lumehub/internal/auth"
	"lumehub/internal/model"
	"lumehub/internal/ordering"
	"lumehub/internal/store"
)

func normalizeItemSortBy(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case "updated_at":
		return "updated_at"
	case "sort", "custom":
		return "sort"
	default:
		return "uploaded_at"
	}
}

func withViewKeyParam(rawURL, viewPass string) string {
	if strings.TrimSpace(viewPass) == "" {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + "k=" + url.QueryEscape(viewPass)
}

func (h *Handler) resolveFolderViewAccess(r *http.Request, folderKey string, maj model.Category, sub model.Subcategory) (accessKey string, ok bool) {
	if !model.FolderRequiresEncryptedPassword(maj, sub) {
		return "", true
	}
	passHash := model.FolderEncryptedPasswordHash(maj, sub)
	if passHash == "" {
		return "", false
	}
	for _, key := range []string{
		strings.TrimSpace(r.URL.Query().Get("vg")),
		strings.TrimSpace(r.URL.Query().Get("vp")),
		strings.TrimSpace(r.URL.Query().Get("k")),
	} {
		if key == "" {
			continue
		}
		if h.auth.ValidViewGrant(folderKey, key) {
			return key, true
		}
		if model.PasswordHashMatches(passHash, key) {
			return key, true
		}
	}
	return "", false
}

func itemShortPath(folderKey string, it model.Item) string {
	rel := strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(it.Filename)), "/")
	if rel == "" {
		rel = strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(it.LinkName)), "/")
	}
	if rel == "" {
		return "/resource/" + folderKey
	}
	return "/resource/" + folderKey + "/" + rel
}

func itemCanonicalResourceRel(it model.Item) string {
	return strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(it.Filename)), "/")
}

func itemEditedResourceRel(it model.Item) string {
	return strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(it.EditedFilename)), "/")
}

func itemDisplayResourceRel(it model.Item) string {
	if it.UseEdited {
		if rel := itemEditedResourceRel(it); rel != "" {
			return rel
		}
	}
	return itemCanonicalResourceRel(it)
}

func itemFormatFromRel(rel string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(rel), "."))
	if ext == "jpeg" {
		return "jpg"
	}
	return ext
}

func itemMediaKindFromFormat(format string) string {
	switch format {
	case "jpg", "jpeg", "png", "webp", "gif", "svg", "bmp", "avif", "ico", "heic":
		return "image"
	case "mp4", "webm", "mov", "mkv", "avi", "m4v", "wmv", "flv":
		return "video"
	case "mp3", "wav", "flac", "aac", "ogg", "m4a", "wma":
		return "audio"
	case "zip", "rar", "7z", "tar", "gz", "bz2", "xz":
		return "archive"
	case "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt", "md", "rtf", "csv":
		return "document"
	default:
		return "other"
	}
}

func itemShortResourceRel(it model.Item) string {
	if ln := strings.TrimSpace(it.LinkName); ln != "" {
		return strings.TrimLeft(filepath.ToSlash(ln), "/")
	}
	return itemCanonicalResourceRel(it)
}

func itemShortURLPath(it model.Item) string {
	ln := strings.TrimSpace(it.LinkName)
	if ln == "" {
		return ""
	}
	stem := strings.TrimSuffix(filepath.Base(ln), filepath.Ext(ln))
	if stem == "" {
		return ""
	}
	return "/resource/" + stem
}

func resourceBasePath(major model.Category, folderKey string) string {
	mk := strings.TrimSpace(strings.ToLower(major.Key))
	if mk == "" {
		return "/resource/" + folderKey
	}
	return "/resource/" + mk + "/" + folderKey
}

type Handler struct {
	store *store.Store
	auth  *auth.Manager
	qr    *qrLoginState
	log   *log.Logger
}

func NewHandler(s *store.Store, am *auth.Manager, lg *log.Logger) *Handler {
	if lg == nil {
		lg = log.Default()
	}
	return &Handler{
		store: s,
		auth:  am,
		qr:    newQRLoginState(),
		log:   lg,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/categories", h.handleCategories)
	mux.HandleFunc("PATCH /api/categories/visibility", h.handleCategoriesVisibilityPatch)
	mux.HandleFunc("PATCH /api/categories/name", h.handleCategoriesNamePatch)
	mux.HandleFunc("PATCH /api/categories/folder-key", h.handleCategoriesFolderKeyPatch)
	mux.HandleFunc("PATCH /api/categories/sub-major", h.handleCategoriesSubMajorPatch)
	mux.HandleFunc("PATCH /api/categories/nav-order", h.handleCategoriesNavOrderPatch)
	mux.HandleFunc("POST /api/categories/major", h.handleCategoriesCreateMajor)
	mux.HandleFunc("POST /api/categories/sub", h.handleCategoriesCreateSub)
	mux.HandleFunc("DELETE /api/categories/major", h.handleCategoriesDeleteMajor)
	mux.HandleFunc("DELETE /api/categories/sub", h.handleCategoriesDeleteSub)
	mux.HandleFunc("/api/category/", h.handleCategoryPath)
	mux.HandleFunc("/api/auth/login", h.handleAuthLogin)
	mux.HandleFunc("/api/auth/logout", h.handleAuthLogout)
	mux.HandleFunc("/api/auth/status", h.handleAuthStatus)
	mux.HandleFunc("POST /api/auth/passkey/register/options", h.handlePasskeyRegisterOptions)
	mux.HandleFunc("POST /api/auth/passkey/register/verify", h.handlePasskeyRegisterVerify)
	mux.HandleFunc("GET /api/auth/passkey/list", h.handlePasskeyList)
	mux.HandleFunc("POST /api/auth/qr/session", h.handleAuthQRSessionCreate)
	mux.HandleFunc("GET /api/auth/qr/session/", h.handleAuthQRSessionPoll)
	mux.HandleFunc("GET /auth/qr/approve", h.handleAuthQRApprovePage)
	mux.HandleFunc("POST /api/auth/qr/passkey/options", h.handleAuthQRPasskeyOptions)
	mux.HandleFunc("POST /api/auth/qr/passkey/verify", h.handleAuthQRPasskeyVerify)
	mux.HandleFunc("GET /api/auth/me", h.handleAuthMeGet)
	mux.HandleFunc("PATCH /api/auth/me", h.handleAuthMePatch)
	mux.HandleFunc("POST /api/auth/me/avatar", h.handleAuthMeAvatarPost)
	mux.HandleFunc("DELETE /api/auth/me/avatar", h.handleAuthMeAvatarDelete)
	mux.HandleFunc("GET /api/auth/accounts", h.handleAuthAccountsList)
	mux.HandleFunc("POST /api/auth/accounts", h.handleAuthAccountsCreate)
	mux.HandleFunc("PATCH /api/auth/accounts/", h.handleAuthAccountsPatch)
	mux.HandleFunc("DELETE /api/auth/accounts/", h.handleAuthAccountsDelete)
	mux.HandleFunc("GET /api/avatar/", h.handleAvatarGet)
	mux.HandleFunc("GET /api/storage", h.handleStorageGet)
	mux.HandleFunc("PATCH /api/storage", h.handleStoragePatch)
	mux.HandleFunc("POST /api/storage/recalculate", h.handleStorageRecalculate)
	mux.HandleFunc("GET /api/trash", h.handleTrashList)
	mux.HandleFunc("DELETE /api/trash", h.handleTrashList)
	mux.HandleFunc("/api/trash/", h.handleTrashPath)
}

func (h *Handler) handleCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	doc, err := h.store.ReadCategories()
	if err != nil {
		h.log.Printf("categories: %v", err)
		http.Error(w, "failed to read categories", http.StatusInternalServerError)
		return
	}
	cats := slices.Clone(doc.Categories)
	ordering.SortCategoriesInPlace(cats)
	for i := range cats {
		subs := slices.Clone(cats[i].Subcategories)
		ordering.SortSubcategoriesInPlace(subs)
		cats[i].Subcategories = subs
	}
	out := *doc
	out.Categories = cats
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(&out)
}

// handleCategoryPath:
//   - GET /api/category/{folderKey}
//   - PATCH /api/category/{folderKey}/layout
//   - PATCH /api/category/{folderKey}/item-order
//   - DELETE /api/category/{folderKey}/items/{itemId}
//   - PATCH /api/category/{folderKey}/items/{itemId}
//   - POST /api/category/{folderKey}/items/{itemId}/transfer
//   - POST /api/category/{folderKey}/upload
//   - POST /api/category/{folderKey}/upload/session
//   - GET /api/category/{folderKey}/upload/session/{id}
//   - DELETE /api/category/{folderKey}/upload/session/{id}
//   - PUT /api/category/{folderKey}/upload/session/{id}/chunk/{index}
//   - POST /api/category/{folderKey}/upload/session/{id}/complete
func (h *Handler) handleCategoryPath(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/category/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(rest, "/")
	folderKey := parts[0]

	if len(parts) == 1 && r.Method == http.MethodGet {
		h.getCategory(w, r, folderKey)
		return
	}
	if len(parts) == 2 && parts[1] == "layout" && r.Method == http.MethodPatch {
		h.patchLayout(w, r, folderKey)
		return
	}
	if len(parts) == 2 && parts[1] == "item-order" && r.Method == http.MethodPatch {
		h.patchItemOrder(w, r, folderKey)
		return
	}
	if len(parts) == 3 && parts[1] == "items" {
		itemID := parts[2]
		switch r.Method {
		case http.MethodDelete:
			h.deleteCategoryItem(w, r, folderKey, itemID)
		case http.MethodPatch:
			h.patchCategoryItem(w, r, folderKey, itemID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 4 && parts[1] == "items" && parts[3] == "companion" && r.Method == http.MethodPost {
		h.attachItemCompanion(w, r, folderKey, parts[2])
		return
	}
	if len(parts) == 4 && parts[1] == "items" && parts[3] == "transfer" && r.Method == http.MethodPost {
		h.transferCategoryItem(w, r, folderKey, parts[2])
		return
	}
	if len(parts) == 4 && parts[1] == "items" && parts[3] == "thumbnail" && r.Method == http.MethodPost {
		h.uploadItemThumbnail(w, r, folderKey, parts[2])
		return
	}
	if len(parts) >= 3 && parts[1] == "upload" && parts[2] == "session" {
		h.handleCategoryUploadSession(w, r, parts)
		return
	}
	if len(parts) == 2 && parts[1] == "view-unlock" && r.Method == http.MethodPost {
		h.unlockCategoryView(w, r, folderKey)
		return
	}
	if len(parts) == 2 && parts[1] == "upload" && r.Method == http.MethodPost {
		h.uploadCategoryItem(w, r, folderKey)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) getCategory(w http.ResponseWriter, r *http.Request, folderKey string) {
	maj, sub, ok := h.store.LookupFolderInCategories(folderKey)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if h.auth.Configured() && model.FolderRequiresLogin(maj, sub) && !h.auth.Valid(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "forbidden", "requiresAuth": true})
		return
	}
	viewKey, allowed := h.resolveFolderViewAccess(r, folderKey, maj, sub)
	if !allowed {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "forbidden", "requiresPassword": true})
		return
	}
	cat, err := h.store.GetCategoryByFolderKey(folderKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	items, err := h.store.ReadItems(folderKey)
	if err != nil {
		h.log.Printf("items %s: %v", folderKey, err)
		http.Error(w, "failed to read items", http.StatusInternalServerError)
		return
	}
	itemsSorted := slices.Clone(items)
	mode := normalizeItemSortBy(cat.ItemSortBy)
	ordering.SortItemsByMode(itemsSorted, mode)
	dtos := make([]model.ItemDTO, 0, len(itemsSorted))
	for _, it := range itemsSorted {
		if itemCanonicalResourceRel(it) == "" {
			continue
		}
		dtos = append(dtos, h.itemToDTO(folderKey, maj, sub, it, viewKey))
	}
	out := model.CategoryDetailResponse{
		ID:         cat.ID,
		Name:       cat.Name,
		FolderKey:  cat.FolderKey,
		Layout:     cat.Layout,
		ItemSortBy: mode,
		Items:      dtos,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

type unlockCategoryViewBody struct {
	Password string `json:"password"`
	DeviceID string `json:"deviceId"`
}

func (h *Handler) unlockCategoryView(w http.ResponseWriter, r *http.Request, folderKey string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	maj, sub, ok := h.store.LookupFolderInCategories(folderKey)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if h.auth.Configured() && model.FolderRequiresLogin(maj, sub) && !h.auth.Valid(r) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "forbidden", "requiresAuth": true})
		return
	}
	if !model.FolderRequiresEncryptedPassword(maj, sub) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "folder is not encrypted"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<14))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req unlockCategoryViewBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	passHash := model.FolderEncryptedPasswordHash(maj, sub)
	password := strings.TrimSpace(req.Password)
	actorKey := h.auth.ViewUnlockActorKey(r, req.DeviceID)
	if actorKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "缺少设备标识"})
		return
	}
	if blocked, msg := h.auth.CheckViewPasswordLockout(folderKey, actorKey); blocked {
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": msg})
		return
	}
	if passHash == "" || !model.PasswordHashMatches(passHash, password) {
		if blocked, msg := h.auth.RecordViewPasswordFailure(folderKey, actorKey); blocked {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": msg})
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "查看密码不正确"})
		return
	}
	h.auth.RecordViewPasswordSuccess(folderKey, actorKey)
	grant, expires := h.auth.CreateViewGrant(folderKey)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"grant":     grant,
		"expiresAt": expires.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) patchLayout(w http.ResponseWriter, r *http.Request, folderKey string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req model.PatchLayoutBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Layout.Mode != "masonry" && req.Layout.Mode != "grid" {
		http.Error(w, "layout.mode must be masonry or grid", http.StatusBadRequest)
		return
	}
	col := req.Layout.Columns
	if col != "auto" && col != "1" && col != "2" && col != "3" && col != "4" && col != "5" && col != "6" {
		http.Error(w, "layout.columns invalid", http.StatusBadRequest)
		return
	}
	updated, err := h.store.UpdateCategoryLayout(folderKey, req.Layout)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(updated)
}

func (h *Handler) patchItemOrder(w http.ResponseWriter, r *http.Request, folderKey string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req model.PatchItemOrderBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := h.store.UpdateCategoryItemOrder(folderKey, req.OrderedItemIDs, req.MasonryPlacement); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"itemSortBy": "sort"})
}

func (h *Handler) deleteCategoryItem(w http.ResponseWriter, r *http.Request, folderKey, itemID string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.store.DeleteCategoryItem(folderKey, itemID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid item id", http.StatusBadRequest)
			return
		}
		h.log.Printf("delete item %s/%s: %v", folderKey, itemID, err)
		http.Error(w, "failed to delete item", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) transferCategoryItem(w http.ResponseWriter, r *http.Request, folderKey, itemID string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	var req model.TransferItemBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	targetFK := strings.TrimSpace(req.TargetFolderKey)
	if targetFK == "" {
		http.Error(w, "targetFolderKey required", http.StatusBadRequest)
		return
	}
	if err := h.store.MoveCategoryItem(folderKey, targetFK, itemID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid transfer", http.StatusBadRequest)
			return
		}
		if errors.Is(err, os.ErrExist) {
			http.Error(w, "target folder has conflicting item", http.StatusConflict)
			return
		}
		h.log.Printf("transfer item %s/%s -> %s: %v", folderKey, itemID, targetFK, err)
		http.Error(w, "failed to transfer item", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) patchCategoryItem(w http.ResponseWriter, r *http.Request, folderKey, itemID string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, _, ok := h.store.LookupFolderInCategories(folderKey); !ok {
		http.NotFound(w, r)
		return
	}

	var patch store.PatchCategoryItemInput
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(512 << 20); err != nil {
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}
		if v := strings.TrimSpace(r.FormValue("linkName")); v != "" {
			patch.SetLinkName = true
			patch.LinkName = v
		}
		if _, ok := r.Form["title"]; ok {
			patch.SetTitle = true
			patch.Title = strings.TrimSpace(r.FormValue("title"))
		}
		if rawTags := strings.TrimSpace(r.FormValue("tags")); rawTags != "" {
			patch.HasTags = true
			for _, part := range strings.Split(rawTags, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					patch.Tags = append(patch.Tags, part)
				}
			}
		} else if _, ok := r.Form["tags"]; ok {
			patch.HasTags = true
			patch.Tags = nil
		}
		saveMode := strings.TrimSpace(strings.ToLower(r.FormValue("saveMode")))
		patch.ReplaceOriginal = saveMode == "replace"
		patch.SaveAsEdited = saveMode != "replace"
		file, fileHeader, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			patch.File = file
			patch.Filename = fileHeader.Filename
		}
	} else {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		var req model.PatchItemBody
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
		}
		if req.RevertEdited {
			patch.RevertEdited = true
		} else {
			if ln := strings.TrimSpace(req.LinkName); ln != "" {
				patch.SetLinkName = true
				patch.LinkName = ln
			}
			patch.SetTitle = true
			patch.Title = req.Title
			patch.HasTags = true
			patch.Tags = req.Tags
		}
	}

	if !patch.RevertEdited && !patch.SetLinkName && !patch.SetTitle && !patch.HasTags && patch.File == nil {
		http.Error(w, "nothing to update", http.StatusBadRequest)
		return
	}

	item, err := h.store.PatchCategoryItem(folderKey, itemID, patch)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if isStorageQuotaErr(err) {
			storageQuotaHTTPError(w)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid patch", http.StatusBadRequest)
			return
		}
		if errors.Is(err, os.ErrExist) {
			http.Error(w, "link name already exists", http.StatusConflict)
			return
		}
		h.log.Printf("patch item %s/%s: %v", folderKey, itemID, err)
		http.Error(w, "failed to update item", http.StatusInternalServerError)
		return
	}
	out := h.itemDTOForUploadedAsset(folderKey, item)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) itemToDTO(folderKey string, maj model.Category, sub model.Subcategory, it model.Item, viewPass string) model.ItemDTO {
	canonicalRel := itemCanonicalResourceRel(it)
	if canonicalRel == "" {
		return model.ItemDTO{ID: it.ID}
	}
	displayRel := itemDisplayResourceRel(it)
	editedRel := itemEditedResourceRel(it)

	buildURL := func(rel string) string {
		if rel == "" {
			return ""
		}
		u := resourceBasePath(maj, folderKey) + "/" + rel
		if model.FolderRequiresEncryptedPassword(maj, sub) {
			u = withViewKeyParam(u, viewPass)
		}
		return u
	}

	displayURL := buildURL(displayRel)
	originalURL := buildURL(canonicalRel)
	editedURL := buildURL(editedRel)
	shortURL := itemShortURLPath(it)
	if shortURL != "" && model.FolderRequiresEncryptedPassword(maj, sub) {
		shortURL = withViewKeyParam(shortURL, viewPass)
	}

	thumbURL := ""
	if t := strings.TrimSpace(it.Thumbnail); t != "" {
		tEff := h.store.EffectiveThumbnailRel(folderKey, t)
		thumbURL = buildURL(strings.TrimLeft(filepath.ToSlash(tEff), "/"))
	}

	rawURL := ""
	liveVideoURL := ""
	isLivePhoto := false
	if rf := strings.TrimSpace(it.RawFilename); rf != "" {
		rawURL = buildURL(strings.TrimLeft(filepath.ToSlash(rf), "/"))
		if store.IsLiveVideoCompanionExt(filepath.Ext(rf)) {
			isLivePhoto = true
			liveVideoURL = rawURL
		}
	}

	var masonryCol *int
	var masonryRow *int
	if it.MasonryCol > 0 && it.MasonryRow > 0 {
		c := it.MasonryCol - 1
		r := it.MasonryRow - 1
		masonryCol = &c
		masonryRow = &r
	}

	format := itemFormatFromRel(displayRel)
	if format == "" {
		format = itemFormatFromRel(canonicalRel)
	}

	fileSize := h.store.EffectiveItemFileSize(folderKey, it)

	return model.ItemDTO{
		ID:           it.ID,
		Sort:         it.Sort,
		MasonryCol:   masonryCol,
		MasonryRow:   masonryRow,
		UploadedAt:   it.UploadedAt,
		UpdatedAt:    it.UpdatedAt,
		FileSize:     fileSize,
		ShortURL:     shortURL,
		LinkName:     strings.TrimSpace(it.LinkName),
		URL:          displayURL,
		OriginalURL:  originalURL,
		EditedURL:    editedURL,
		UseEdited:    it.UseEdited,
		Format:       format,
		MediaKind:    itemMediaKindFromFormat(format),
		ThumbnailURL: thumbURL,
		GroupID:      it.GroupID,
		RawURL:       rawURL,
		IsLivePhoto:  isLivePhoto,
		LiveVideoURL: liveVideoURL,
		Title:        it.Title,
		Tags:         it.Tags,
	}
}

func (h *Handler) itemDTOForUploadedAsset(folderKey string, item model.Item) model.ItemDTO {
	maj, sub, ok := h.store.LookupFolderInCategories(folderKey)
	if !ok {
		canonicalRel := itemCanonicalResourceRel(item)
		displayRel := itemDisplayResourceRel(item)
		format := itemFormatFromRel(displayRel)
		return model.ItemDTO{
			ID:          item.ID,
			Sort:        item.Sort,
			UploadedAt:  item.UploadedAt,
			UpdatedAt:   item.UpdatedAt,
			FileSize:    h.store.EffectiveItemFileSize(folderKey, item),
			URL:         itemShortPath(folderKey, item),
			LinkName:    strings.TrimSpace(item.LinkName),
			OriginalURL: "/resource/" + folderKey + "/" + canonicalRel,
			UseEdited:   item.UseEdited,
			Format:      format,
			MediaKind:   itemMediaKindFromFormat(format),
			Title:       item.Title,
			Tags:        item.Tags,
		}
	}
	return h.itemToDTO(folderKey, maj, sub, item, "")
}

func (h *Handler) uploadCategoryItem(w http.ResponseWriter, r *http.Request, folderKey string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, _, ok := h.store.LookupFolderInCategories(folderKey); !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	var rawFile io.ReadCloser
	var rawHeaderFilename string
	if rf, rh, rawErr := r.FormFile("rawFile"); rawErr == nil {
		rawFile = rf
		rawHeaderFilename = rh.Filename
		defer rawFile.Close()
	}
	item, err := h.store.SaveUploadedFileWithRaw(folderKey, fileHeader.Filename, file, rawHeaderFilename, rawFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if isStorageQuotaErr(err) {
			storageQuotaHTTPError(w)
			return
		}
		h.log.Printf("upload item %s: %v", folderKey, err)
		http.Error(w, "failed to upload file", http.StatusInternalServerError)
		return
	}
	out := h.itemDTOForUploadedAsset(folderKey, item)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) uploadItemThumbnail(w http.ResponseWriter, r *http.Request, folderKey, itemID string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, _, ok := h.store.LookupFolderInCategories(folderKey); !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	poster, header, err := r.FormFile("poster")
	if err != nil {
		http.Error(w, "poster is required", http.StatusBadRequest)
		return
	}
	defer poster.Close()
	size := int64(0)
	if header != nil {
		size = header.Size
	}
	item, err := h.store.SetItemThumbnailFromReader(folderKey, itemID, poster, size)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if isStorageQuotaErr(err) {
			storageQuotaHTTPError(w)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid poster or not a video item", http.StatusBadRequest)
			return
		}
		h.log.Printf("upload thumbnail %s/%s: %v", folderKey, itemID, err)
		http.Error(w, "failed to save thumbnail", http.StatusInternalServerError)
		return
	}
	out := h.itemDTOForUploadedAsset(folderKey, item)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) attachItemCompanion(w http.ResponseWriter, r *http.Request, folderKey, itemID string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, _, ok := h.store.LookupFolderInCategories(folderKey); !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	raw, rawHeader, err := r.FormFile("rawFile")
	if err != nil {
		http.Error(w, "rawFile is required", http.StatusBadRequest)
		return
	}
	defer raw.Close()
	item, err := h.store.AttachItemCompanionRaw(folderKey, itemID, rawHeader.Filename, raw)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if isStorageQuotaErr(err) {
			storageQuotaHTTPError(w)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid companion file", http.StatusBadRequest)
			return
		}
		h.log.Printf("attach companion %s/%s: %v", folderKey, itemID, err)
		http.Error(w, "failed to attach companion", http.StatusInternalServerError)
		return
	}
	out := h.itemDTOForUploadedAsset(folderKey, item)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) handleCategoryUploadSession(w http.ResponseWriter, r *http.Request, parts []string) {
	folderKey := parts[0]
	switch {
	case len(parts) == 3 && r.Method == http.MethodPost:
		h.uploadSessionCreate(w, r, folderKey)
	case len(parts) == 4 && r.Method == http.MethodGet:
		h.uploadSessionStatus(w, r, folderKey, parts[3])
	case len(parts) == 4 && r.Method == http.MethodDelete:
		h.uploadSessionDelete(w, r, folderKey, parts[3])
	case len(parts) == 5 && parts[4] == "complete" && r.Method == http.MethodPost:
		h.uploadSessionComplete(w, r, folderKey, parts[3])
	case len(parts) == 6 && parts[4] == "chunk" && r.Method == http.MethodPut:
		idx, err := strconv.Atoi(parts[5])
		if err != nil || idx < 0 {
			http.Error(w, "bad chunk index", http.StatusBadRequest)
			return
		}
		h.uploadSessionChunkPut(w, r, folderKey, parts[3], idx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type uploadSessionCreateBody struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Sha256   string `json:"sha256"`
}

func (h *Handler) uploadSessionCreate(w http.ResponseWriter, r *http.Request, folderKey string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, _, ok := h.store.LookupFolderInCategories(folderKey); !ok {
		http.NotFound(w, r)
		return
	}
	var body uploadSessionCreateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	fn := strings.TrimSpace(body.Filename)
	if fn == "" || body.Size <= 0 {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	id, chunk, err := h.store.CreateUploadSession(folderKey, fn, body.Size, body.Sha256)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if isStorageQuotaErr(err) {
			storageQuotaHTTPError(w)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid size or hash", http.StatusBadRequest)
			return
		}
		h.log.Printf("upload session create %s: %v", folderKey, err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"uploadId":  id,
		"chunkSize": chunk,
	})
}

func (h *Handler) uploadSessionStatus(w http.ResponseWriter, r *http.Request, folderKey, sessionID string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	st, err := h.store.UploadSessionGetStatus(sessionID, folderKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		h.log.Printf("upload session status %s: %v", sessionID, err)
		http.Error(w, "failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(st)
}

func (h *Handler) uploadSessionDelete(w http.ResponseWriter, r *http.Request, folderKey, sessionID string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.store.DeleteUploadSession(sessionID, folderKey); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		h.log.Printf("upload session delete %s: %v", sessionID, err)
		http.Error(w, "failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) uploadSessionChunkPut(w http.ResponseWriter, r *http.Request, folderKey, sessionID string, index int) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	const maxChunk = (16 << 20) + (64 << 10)
	body := http.MaxBytesReader(w, r.Body, maxChunk)
	if err := h.store.WriteUploadChunk(sessionID, folderKey, index, body); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, store.ErrUploadBadChunk) {
			http.Error(w, "invalid chunk", http.StatusBadRequest)
			return
		}
		h.log.Printf("upload chunk %s %d: %v", sessionID, index, err)
		http.Error(w, "failed to save chunk", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type uploadSessionCompleteBody struct {
	Sha256 string `json:"sha256"`
}

func (h *Handler) uploadSessionComplete(w http.ResponseWriter, r *http.Request, folderKey, sessionID string) {
	if h.auth.Configured() && !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var opt uploadSessionCompleteBody
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&opt)
	}
	item, sum, err := h.store.CompleteUploadSession(sessionID, folderKey, strings.TrimSpace(opt.Sha256))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if isStorageQuotaErr(err) {
			storageQuotaHTTPError(w)
			return
		}
		if errors.Is(err, store.ErrUploadBadChunk) || errors.Is(err, store.ErrUploadHashMismatch) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Printf("upload complete %s: %v", sessionID, err)
		http.Error(w, "failed to finalize upload", http.StatusInternalServerError)
		return
	}
	dto := h.itemDTOForUploadedAsset(folderKey, item)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(struct {
		model.ItemDTO
		Sha256 string `json:"sha256"`
	}{ItemDTO: dto, Sha256: sum})
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !h.auth.Configured() {
		_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": true, "authConfigured": false})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<14))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req loginBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	doc, _ := h.store.ReadAccounts()
	useAccounts := doc != nil && len(doc.Accounts) > 0
	if useAccounts && strings.TrimSpace(req.Email) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false, "error": "请输入邮箱"})
		return
	}
	emailKey := auth.NormalizeLoginEmail(req.Email)
	if useAccounts {
		if blocked, msg := h.auth.CheckLoginLockout(emailKey); blocked {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false, "error": msg})
			return
		}
	}
	userID, ok := h.auth.TryLogin(req.Email, req.Password)
	if !ok {
		if useAccounts {
			if blocked, msg := h.auth.RecordLoginFailure(emailKey); blocked {
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false, "error": msg})
				return
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false, "error": "邮箱或密码错误"})
		return
	}
	if useAccounts {
		h.auth.RecordLoginSuccess(emailKey)
	}
	token, exp := h.auth.CreateSession(userID)
	h.auth.SetSessionCookie(w, token, exp)

	var pub model.AccountPublic
	if userID == "legacy" {
		pub = auth.LegacyMeUser()
	} else if acc, err := h.store.GetAccountByID(userID); err == nil {
		pub = model.ToAccountPublic(*acc)
		h.fillDefaultAvatar(&pub, acc)
	} else {
		pub = model.AccountPublic{ID: userID, Username: userID, Email: userID, DisplayName: userID}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"authenticated":  true,
		"authConfigured": true,
		"token":          token,
		"expiresAt":      exp.UTC().Format(time.RFC3339),
		"user":           pub,
	})
}

func (h *Handler) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tok := h.auth.TokenFromRequest(r)
	h.auth.Invalidate(tok)
	h.auth.ClearSessionCookie(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
}

func (h *Handler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	cfg := h.auth.Configured()
	if !cfg {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"authenticated":  false,
			"authConfigured": false,
			"useAccounts":    false,
		})
		return
	}
	doc, err := h.store.ReadAccounts()
	if err != nil || doc == nil {
		doc = &model.AccountsDoc{}
	}
	useAccounts := len(doc.Accounts) > 0
	ok := h.auth.Valid(r)
	out := map[string]any{
		"authenticated":  ok,
		"authConfigured": true,
		"useAccounts":    useAccounts,
	}
	if g := strings.TrimSpace(doc.GuestAvatarURL); g != "" {
		out["guestAvatarUrl"] = g
	}
	if f := strings.TrimSpace(doc.LoggedInFallbackAvatarURL); f != "" {
		out["loggedInFallbackAvatarUrl"] = f
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) handleAuthMeGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !h.auth.Configured() {
		_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false, "authConfigured": false, "user": nil})
		return
	}
	uid, ok := h.auth.SessionUserID(r)
	if !ok {
		_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false, "authConfigured": true, "user": nil})
		return
	}
	var pub model.AccountPublic
	if uid == "legacy" {
		pub = auth.LegacyMeUser()
	} else if acc, err := h.store.GetAccountByID(uid); err == nil {
		pub = model.ToAccountPublic(*acc)
		h.fillDefaultAvatar(&pub, acc)
	} else {
		pub = model.AccountPublic{ID: uid, Username: uid, Email: uid, DisplayName: uid}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": true, "authConfigured": true, "user": pub})
}

type patchMeBody struct {
	DisplayName     *string `json:"displayName"`
	Avatar          *string `json:"avatar"`
	Email           *string `json:"email"`
	NewPassword     *string `json:"newPassword"`
	CurrentPassword string  `json:"currentPassword"`
}

func (h *Handler) handleAuthMePatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !h.auth.Configured() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	uid, ok := h.auth.SessionUserID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if uid == "legacy" {
		http.Error(w, "legacy session cannot edit profile", http.StatusForbidden)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<14))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req patchMeBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	before, _ := h.store.GetAccountByID(uid)
	updated, err := h.store.PatchAccountMe(uid, strings.TrimSpace(req.CurrentPassword), req.DisplayName, req.Avatar, req.Email, req.NewPassword)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		switch {
		case errors.Is(err, store.ErrPatchMeCurrentPasswordRequired):
			http.Error(w, "修改登录邮箱或密码需要填写当前密码", http.StatusBadRequest)
		case errors.Is(err, store.ErrPatchMeWrongCurrentPassword):
			http.Error(w, "当前密码不正确", http.StatusUnauthorized)
		case errors.Is(err, store.ErrPatchMeEmailTaken):
			http.Error(w, "该邮箱已被占用", http.StatusConflict)
		case errors.Is(err, store.ErrPatchMeEmailEmpty):
			http.Error(w, "邮箱不能为空", http.StatusBadRequest)
		case errors.Is(err, store.ErrPatchMePasswordTooShort):
			http.Error(w, "新密码至少 6 位", http.StatusBadRequest)
		default:
			http.Error(w, "failed to save", http.StatusInternalServerError)
		}
		return
	}
	emailChanged := false
	if req.Email != nil && before != nil {
		prevEmail := strings.TrimSpace(before.Email)
		if prevEmail == "" {
			prevEmail = strings.TrimSpace(before.Username)
		}
		emailChanged = !strings.EqualFold(strings.TrimSpace(*req.Email), prevEmail)
	}
	passwordChanged := req.NewPassword != nil && strings.TrimSpace(*req.NewPassword) != ""
	if emailChanged || passwordChanged {
		h.auth.InvalidateUserSessions(uid)
	}
	if passwordChanged {
		if before != nil {
			h.auth.ClearLoginLockout(accountLoginEmail(before))
		}
		h.auth.ClearLoginLockout(accountLoginEmail(updated))
	}
	pub := model.ToAccountPublic(*updated)
	h.fillDefaultAvatar(&pub, updated)
	_ = json.NewEncoder(w).Encode(pub)
}

func (h *Handler) handleAuthMeAvatarPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !h.auth.Configured() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	uid, ok := h.auth.SessionUserID(r)
	if !ok || uid == "legacy" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := r.ParseMultipartForm(avatarMaxBytes + (1 << 20)); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	size := int64(0)
	if header != nil {
		size = header.Size
	}
	updated, err := h.store.SaveAccountAvatar(uid, file, header.Filename, size)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		switch {
		case errors.Is(err, store.ErrAvatarUnsupportedType):
			http.Error(w, "仅支持 PNG、JPG、WEBP 格式头像", http.StatusBadRequest)
		case errors.Is(err, store.ErrAvatarTooLarge):
			http.Error(w, "头像不能超过 10MB", http.StatusBadRequest)
		default:
			h.log.Printf("avatar upload %s: %v", uid, err)
			http.Error(w, "failed to save avatar", http.StatusInternalServerError)
		}
		return
	}
	pub := model.ToAccountPublic(*updated)
	h.fillDefaultAvatar(&pub, updated)
	_ = json.NewEncoder(w).Encode(pub)
}

func (h *Handler) handleAuthMeAvatarDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !h.auth.Configured() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	uid, ok := h.auth.SessionUserID(r)
	if !ok || uid == "legacy" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	updated, err := h.store.ClearAccountAvatar(uid)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		h.log.Printf("avatar delete %s: %v", uid, err)
		http.Error(w, "failed to remove avatar", http.StatusInternalServerError)
		return
	}
	pub := model.ToAccountPublic(*updated)
	h.fillDefaultAvatar(&pub, updated)
	_ = json.NewEncoder(w).Encode(pub)
}

const avatarMaxBytes = 10 << 20

func accountLoginEmail(acc *model.Account) string {
	if acc == nil {
		return ""
	}
	email := strings.TrimSpace(acc.Email)
	if email == "" {
		email = strings.TrimSpace(acc.Username)
	}
	return email
}

func (h *Handler) fillDefaultAvatar(pub *model.AccountPublic, acc *model.Account) {
	if strings.TrimSpace(pub.Avatar) != "" {
		return
	}
	if _, ok := h.store.AccountAvatarFilePath(*acc); ok {
		pub.Avatar = "/api/avatar/" + acc.ID
	}
}

func (h *Handler) requirePermission(w http.ResponseWriter, r *http.Request, perm string) (*model.Account, bool) {
	if !h.auth.Configured() || !h.auth.Valid(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	uid, ok := h.auth.SessionUserID(r)
	if !ok || uid == "legacy" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	acc, err := h.store.GetAccountByID(uid)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	if !slices.Contains(acc.Permissions, perm) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return acc, true
}

func (h *Handler) handleAvatarGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.auth.Configured() || !h.auth.ValidForResource(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/avatar/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	acc, err := h.store.GetAccountByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p, ok := h.store.AccountAvatarFilePath(*acc)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch strings.ToLower(filepath.Ext(p)) {
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	http.ServeFile(w, r, p)
}

type patchAccountBody struct {
	DisplayName     string   `json:"displayName"`
	Email           string   `json:"email"`
	Roles           []string `json:"roles"`
	Permissions     []string `json:"permissions"`
	CurrentPassword string   `json:"currentPassword"`
	NewPassword     *string  `json:"newPassword"`
}

type createAccountBody struct {
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

func (h *Handler) handleAuthAccountsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_accounts"); !ok {
		return
	}
	list, err := h.store.ListAccountsPublic()
	if err != nil {
		http.Error(w, "failed to list", http.StatusInternalServerError)
		return
	}
	out := make([]model.AccountPublic, 0, len(list))
	for i := range list {
		pub := model.ToAccountPublic(list[i])
		a := list[i]
		h.fillDefaultAvatar(&pub, &a)
		out = append(out, pub)
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) handleAuthAccountsPatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	operator, ok := h.requirePermission(w, r, "manage_accounts")
	if !ok {
		return
	}
	targetID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/auth/accounts/"), "/")
	if targetID == "" {
		http.NotFound(w, r)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<14))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req patchAccountBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	before, _ := h.store.GetAccountByID(targetID)
	updated, err := h.store.AdminUpdateAccount(
		operator.ID,
		strings.TrimSpace(req.CurrentPassword),
		targetID,
		req.DisplayName,
		req.Email,
		req.Roles,
		req.Permissions,
		req.NewPassword,
	)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		switch {
		case errors.Is(err, store.ErrAdminUpdateDisplayNameEmpty):
			http.Error(w, "用户名不能为空", http.StatusBadRequest)
		case errors.Is(err, store.ErrAdminUpdateEmailEmpty):
			http.Error(w, "邮箱不能为空", http.StatusBadRequest)
		case errors.Is(err, store.ErrAdminUpdateEmailTaken):
			http.Error(w, "邮箱已被占用", http.StatusBadRequest)
		case errors.Is(err, store.ErrPatchMeCurrentPasswordRequired):
			http.Error(w, "修改登录邮箱或密码需要填写当前密码", http.StatusBadRequest)
		case errors.Is(err, store.ErrPatchMeWrongCurrentPassword):
			http.Error(w, "当前密码不正确", http.StatusUnauthorized)
		case errors.Is(err, store.ErrPatchMePasswordTooShort):
			http.Error(w, "新密码至少 6 位", http.StatusBadRequest)
		default:
			http.Error(w, "failed to save", http.StatusInternalServerError)
		}
		return
	}

	emailChanged := false
	if before != nil {
		prevEmail := strings.TrimSpace(before.Email)
		if prevEmail == "" {
			prevEmail = strings.TrimSpace(before.Username)
		}
		emailChanged = !strings.EqualFold(strings.TrimSpace(req.Email), prevEmail)
	}
	passwordChanged := req.NewPassword != nil && strings.TrimSpace(*req.NewPassword) != ""
	if emailChanged || passwordChanged {
		h.auth.InvalidateUserSessions(targetID)
	}
	if passwordChanged {
		if before != nil {
			h.auth.ClearLoginLockout(accountLoginEmail(before))
		}
		h.auth.ClearLoginLockout(accountLoginEmail(updated))
	}

	pub := model.ToAccountPublic(*updated)
	h.fillDefaultAvatar(&pub, updated)
	_ = json.NewEncoder(w).Encode(pub)
}

func (h *Handler) handleAuthAccountsCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_accounts"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<14))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req createAccountBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	created, err := h.store.AdminCreateAccount(
		req.DisplayName,
		req.Email,
		req.Password,
		req.Roles,
		req.Permissions,
	)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrAdminUpdateDisplayNameEmpty):
			http.Error(w, "用户名不能为空", http.StatusBadRequest)
		case errors.Is(err, store.ErrAdminUpdateEmailEmpty):
			http.Error(w, "邮箱不能为空", http.StatusBadRequest)
		case errors.Is(err, store.ErrAdminUpdateEmailTaken):
			http.Error(w, "邮箱已被占用", http.StatusBadRequest)
		case errors.Is(err, store.ErrAdminCreatePasswordEmpty):
			http.Error(w, "密码不能为空", http.StatusBadRequest)
		case errors.Is(err, store.ErrPatchMePasswordTooShort):
			http.Error(w, "新密码至少 6 位", http.StatusBadRequest)
		default:
			http.Error(w, "failed to create", http.StatusInternalServerError)
		}
		return
	}
	pub := model.ToAccountPublic(*created)
	h.fillDefaultAvatar(&pub, created)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(pub)
}

func (h *Handler) handleAuthAccountsDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	operator, ok := h.requirePermission(w, r, "manage_accounts")
	if !ok {
		return
	}
	targetID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/auth/accounts/"), "/")
	if targetID == "" {
		http.NotFound(w, r)
		return
	}
	if err := h.store.AdminDeleteAccount(operator.ID, targetID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, store.ErrAdminDeleteSelf) {
			http.Error(w, "不能删除当前登录账号", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}
	h.auth.InvalidateUserSessions(targetID)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

type categoriesVisibilityBody struct {
	Patches []store.CategoryVisibilityPatch `json:"patches"`
}

type categoriesNameBody struct {
	Patches []store.CategoryNamePatch `json:"patches"`
}

type categoriesFolderKeyBody struct {
	Patches []store.CategoryFolderKeyPatch `json:"patches"`
}

type categoriesSubMajorBody struct {
	Patches []store.CategorySubMajorPatch `json:"patches"`
}

type categoriesNavOrderBody struct {
	PrimaryMajorIds []int                 `json:"primaryMajorIds,omitempty"`
	SubOrders       []store.SubOrderPatch `json:"subOrders,omitempty"`
}

func folderKeysForVisibilityPasswordPatch(doc *model.CategoriesDoc, p store.CategoryVisibilityPatch) []string {
	switch p.Scope {
	case "sub":
		if p.SubID == nil {
			return nil
		}
		for i := range doc.Categories {
			if doc.Categories[i].ID != p.MajorID {
				continue
			}
			for j := range doc.Categories[i].Subcategories {
				sub := doc.Categories[i].Subcategories[j]
				if sub.ID != *p.SubID {
					continue
				}
				fk := strings.TrimSpace(sub.FolderKey)
				if fk == "" {
					return nil
				}
				return []string{fk}
			}
			return nil
		}
	case "major":
		for i := range doc.Categories {
			if doc.Categories[i].ID != p.MajorID {
				continue
			}
			var keys []string
			for j := range doc.Categories[i].Subcategories {
				fk := strings.TrimSpace(doc.Categories[i].Subcategories[j].FolderKey)
				if fk != "" {
					keys = append(keys, fk)
				}
			}
			return keys
		}
	}
	return nil
}

func (h *Handler) handleCategoriesVisibilityPatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_layout"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req categoriesVisibilityBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := h.store.PatchCategoriesVisibility(req.Patches); err != nil {
		if errors.Is(err, store.ErrCategoryVisibilityNotFound) {
			http.Error(w, "目标分类不存在", http.StatusBadRequest)
			return
		}
		if errors.Is(err, store.ErrCategoryEncryptedPasswordRequired) {
			http.Error(w, "加密目录必须设置查看密码", http.StatusBadRequest)
			return
		}
		if errors.Is(err, store.ErrCategoryEncryptedPasswordTooShort) {
			http.Error(w, "查看密码至少 4 位", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	doc, err := h.store.ReadCategories()
	if err != nil {
		http.Error(w, "saved but failed to read back", http.StatusInternalServerError)
		return
	}
	for _, p := range req.Patches {
		if p.EncryptedPassword == nil {
			continue
		}
		for _, fk := range folderKeysForVisibilityPasswordPatch(doc, p) {
			h.auth.InvalidateViewGrantsForFolder(fk)
		}
	}
	cats := slices.Clone(doc.Categories)
	ordering.SortCategoriesInPlace(cats)
	for i := range cats {
		subs := slices.Clone(cats[i].Subcategories)
		ordering.SortSubcategoriesInPlace(subs)
		cats[i].Subcategories = subs
	}
	out := *doc
	out.Categories = cats
	_ = json.NewEncoder(w).Encode(&out)
}

func (h *Handler) handleCategoriesNamePatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_layout"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req categoriesNameBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := h.store.PatchCategoryNames(req.Patches); err != nil {
		if errors.Is(err, store.ErrCategoryNameTargetNotFound) {
			http.Error(w, "目标分类不存在", http.StatusBadRequest)
			return
		}
		if errors.Is(err, store.ErrCategoryNameEmpty) {
			http.Error(w, "名称不能为空", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	doc, err := h.store.ReadCategories()
	if err != nil {
		http.Error(w, "saved but failed to read back", http.StatusInternalServerError)
		return
	}
	h.writeSortedCategoriesJSON(w, doc)
}

func (h *Handler) handleCategoriesFolderKeyPatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_layout"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req categoriesFolderKeyBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := h.store.PatchCategoryFolderKeys(req.Patches); err != nil {
		switch {
		case errors.Is(err, store.ErrCategoryFolderKeyTargetNotFound):
			http.Error(w, "目标分类不存在", http.StatusBadRequest)
			return
		case errors.Is(err, store.ErrCategoryFolderKeyEmpty):
			http.Error(w, "目录键不能为空", http.StatusBadRequest)
			return
		case errors.Is(err, store.ErrCategoryFolderKeyInvalid):
			http.Error(w, "目录键仅支持英文和下划线", http.StatusBadRequest)
			return
		case errors.Is(err, store.ErrCategoryFolderKeyTaken):
			http.Error(w, "目录键已存在", http.StatusBadRequest)
			return
		default:
			http.Error(w, "failed to save", http.StatusInternalServerError)
			return
		}
	}
	doc, err := h.store.ReadCategories()
	if err != nil {
		http.Error(w, "saved but failed to read back", http.StatusInternalServerError)
		return
	}
	h.writeSortedCategoriesJSON(w, doc)
}

func (h *Handler) handleCategoriesSubMajorPatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_layout"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req categoriesSubMajorBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := h.store.PatchCategorySubMajors(req.Patches); err != nil {
		switch {
		case errors.Is(err, store.ErrCategorySubMajorInvalid):
			http.Error(w, "无效的所属导航", http.StatusBadRequest)
			return
		case errors.Is(err, store.ErrCategorySubMajorTargetNotFound):
			http.Error(w, "目标分类不存在", http.StatusBadRequest)
			return
		default:
			http.Error(w, "failed to save", http.StatusInternalServerError)
			return
		}
	}
	doc, err := h.store.ReadCategories()
	if err != nil {
		http.Error(w, "saved but failed to read back", http.StatusInternalServerError)
		return
	}
	h.writeSortedCategoriesJSON(w, doc)
}

func (h *Handler) handleCategoriesNavOrderPatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_layout"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req categoriesNavOrderBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := h.store.PatchCategoriesNavOrder(req.PrimaryMajorIds, req.SubOrders); err != nil {
		if errors.Is(err, store.ErrCategoryNavOrderInvalid) {
			http.Error(w, "invalid nav order", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	doc, err := h.store.ReadCategories()
	if err != nil {
		http.Error(w, "saved but failed to read back", http.StatusInternalServerError)
		return
	}
	h.writeSortedCategoriesJSON(w, doc)
}

type createCategoryMajorBody struct {
	MajorName string `json:"majorName"`
	SubName   string `json:"subName"`
	FolderKey string `json:"folderKey"`
	Public    bool   `json:"public"`
}

type createCategorySubBody struct {
	MajorID   int    `json:"majorId"`
	Name      string `json:"name"`
	FolderKey string `json:"folderKey"`
	Public    bool   `json:"public"`
}

func (h *Handler) writeSortedCategoriesJSON(w http.ResponseWriter, doc *model.CategoriesDoc) {
	cats := slices.Clone(doc.Categories)
	ordering.SortCategoriesInPlace(cats)
	for i := range cats {
		subs := slices.Clone(cats[i].Subcategories)
		ordering.SortSubcategoriesInPlace(subs)
		cats[i].Subcategories = subs
	}
	out := *doc
	out.Categories = cats
	_ = json.NewEncoder(w).Encode(&out)
}

func (h *Handler) handleCategoriesCreateMajor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_layout"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req createCategoryMajorBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	doc, err := h.store.CreateCategoryMajor(req.MajorName, req.SubName, req.FolderKey, req.Public)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrFolderKeyTaken):
			http.Error(w, "folderKey 已存在", http.StatusBadRequest)
		case errors.Is(err, store.ErrFolderKeyInvalid):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			if strings.Contains(err.Error(), "name required") {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			h.log.Printf("create major category: %v", err)
			http.Error(w, "failed to save", http.StatusInternalServerError)
		}
		return
	}
	h.writeSortedCategoriesJSON(w, doc)
}

func (h *Handler) handleCategoriesCreateSub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_layout"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req createCategorySubBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	doc, err := h.store.CreateCategorySub(req.MajorID, req.Name, req.FolderKey, req.Public)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrMajorNotFound):
			http.Error(w, "大分类不存在", http.StatusBadRequest)
		case errors.Is(err, store.ErrFolderKeyTaken):
			http.Error(w, "folderKey 已存在", http.StatusBadRequest)
		case errors.Is(err, store.ErrFolderKeyInvalid):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			if strings.Contains(err.Error(), "name required") || strings.Contains(err.Error(), "folderKey invalid") {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			h.log.Printf("create sub category: %v", err)
			http.Error(w, "failed to save", http.StatusInternalServerError)
		}
		return
	}
	h.writeSortedCategoriesJSON(w, doc)
}

type deleteCategoryMajorBody struct {
	MajorID int `json:"majorId"`
}

type deleteCategorySubBody struct {
	MajorID int `json:"majorId"`
	SubID   int `json:"subId"`
}

func (h *Handler) writeCategoriesDeleteResponse(w http.ResponseWriter, doc *model.CategoriesDoc, trashedItems int) {
	cats := slices.Clone(doc.Categories)
	ordering.SortCategoriesInPlace(cats)
	for i := range cats {
		subs := slices.Clone(cats[i].Subcategories)
		ordering.SortSubcategoriesInPlace(subs)
		cats[i].Subcategories = subs
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"version":      doc.Version,
		"categories":   cats,
		"trashedItems": trashedItems,
	})
}

func (h *Handler) handleCategoriesDeleteMajor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_layout"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req deleteCategoryMajorBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	doc, trashed, err := h.store.DeleteCategoryMajor(req.MajorID)
	if err != nil {
		if errors.Is(err, store.ErrCategoryDeleteTargetNotFound) {
			http.Error(w, "目标分类不存在", http.StatusBadRequest)
			return
		}
		h.log.Printf("delete major category: %v", err)
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}
	h.writeCategoriesDeleteResponse(w, doc, trashed)
}

func (h *Handler) handleCategoriesDeleteSub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_layout"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req deleteCategorySubBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	doc, trashed, err := h.store.DeleteCategorySub(req.MajorID, req.SubID)
	if err != nil {
		if errors.Is(err, store.ErrCategoryDeleteTargetNotFound) {
			http.Error(w, "目标分类不存在", http.StatusBadRequest)
			return
		}
		h.log.Printf("delete sub category: %v", err)
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}
	h.writeCategoriesDeleteResponse(w, doc, trashed)
}
