package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lumehub/internal/model"
	"lumehub/internal/store"
)

// --- Unified API response ---

type apiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func writeAPI(w http.ResponseWriter, status int, code int, msg string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{Code: code, Message: msg, Data: data})
}

func apiOK(w http.ResponseWriter, data any)        { writeAPI(w, 200, 200, "ok", data) }
func apiCreated(w http.ResponseWriter, data any)    { writeAPI(w, 201, 201, "created", data) }
func apiBadRequest(w http.ResponseWriter, msg string) { writeAPI(w, 400, 400, msg, nil) }
func apiUnauthorized(w http.ResponseWriter)          { writeAPI(w, 401, 401, "invalid or missing api key", nil) }
func apiForbidden(w http.ResponseWriter, msg string) { writeAPI(w, 403, 403, msg, nil) }
func apiNotFound(w http.ResponseWriter)              { writeAPI(w, 404, 404, "not found", nil) }
func apiInternalErr(w http.ResponseWriter, msg string) { writeAPI(w, 500, 500, msg, nil) }

// --- PATCH /api/categories/api-settings ---

type categoriesAPISettingsBody struct {
	Patches []store.CategoryAPISettingsPatch `json:"patches"`
}

func (h *Handler) handleCategoriesAPISettingsPatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, ok := h.requirePermission(w, r, "manage_layout"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<14))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req categoriesAPISettingsBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	results, err := h.store.PatchCategoryAPISettings(req.Patches)
	if err != nil {
		if errors.Is(err, store.ErrCategoryAPISettingsNotFound) {
			http.Error(w, "target not found", http.StatusBadRequest)
			return
		}
		h.log.Printf("api-settings patch: %v", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "newKeys": results})
}

// --- /api/v1/{folderKey}/... ---

func (h *Handler) handlePublicAPI(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(rest, "/")
	folderKey := parts[0]

	apiKey := extractAPIKey(r)
	if apiKey == "" || !h.store.ValidateAPIKey(folderKey, apiKey) {
		apiUnauthorized(w)
		return
	}
	_, enabled := h.store.GetCategoryAPISettings(folderKey)
	if !enabled {
		apiForbidden(w, "api is disabled for this gallery")
		return
	}

	// GET/POST /api/v1/{folderKey}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.publicAPIListItems(w, r, folderKey)
		case http.MethodPost:
			h.publicAPIUploadItem(w, r, folderKey)
		default:
			apiBadRequest(w, "method not allowed")
		}
		return
	}

	// GET/PUT/DELETE /api/v1/{folderKey}/{id}
	if len(parts) == 2 {
		itemID := parts[1]
		switch r.Method {
		case http.MethodGet:
			h.publicAPIGetItem(w, r, folderKey, itemID)
		case http.MethodPut:
			h.publicAPIReplaceItem(w, r, folderKey, itemID)
		case http.MethodDelete:
			h.publicAPIDeleteItem(w, r, folderKey, itemID)
		default:
			apiBadRequest(w, "method not allowed")
		}
		return
	}

	http.NotFound(w, r)
}

func extractAPIKey(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if k := strings.TrimSpace(r.Header.Get("X-API-Key")); k != "" {
		return k
	}
	if k := strings.TrimSpace(r.URL.Query().Get("api_key")); k != "" {
		return k
	}
	return ""
}

// --- Item DTO ---

type publicAPIItemDTO struct {
	ID         string   `json:"id"`
	UploadedAt string   `json:"uploadedAt,omitempty"`
	UpdatedAt  string   `json:"updatedAt,omitempty"`
	FileSize   int64    `json:"fileSize,omitempty"`
	Original   string   `json:"original"`
	Thumbnail  string   `json:"thumbnail,omitempty"`
	Edited     string   `json:"edited,omitempty"`
	LinkName   string   `json:"linkName,omitempty"`
	Title      string   `json:"title,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Format     string   `json:"format,omitempty"`
	MediaKind  string   `json:"mediaKind,omitempty"`
}

func (h *Handler) itemToPublicDTO(folderKey string, maj model.Category, sub model.Subcategory, it model.Item) publicAPIItemDTO {
	basePath := resourceBasePath(maj, folderKey)
	origRel := itemCanonicalResourceRel(it)
	format := itemFormatFromRel(origRel)
	mediaKind := itemMediaKindFromFormat(format)

	dto := publicAPIItemDTO{
		ID:         it.ID,
		UploadedAt: it.UploadedAt,
		UpdatedAt:  it.UpdatedAt,
		FileSize:   it.FileSize,
		Original:   basePath + "/" + origRel,
		LinkName:   it.LinkName,
		Title:      it.Title,
		Tags:       it.Tags,
		Format:     format,
		MediaKind:  mediaKind,
	}
	if thumbRel := strings.TrimSpace(it.Thumbnail); thumbRel != "" {
		eff := h.store.EffectiveThumbnailRel(folderKey, thumbRel)
		dto.Thumbnail = basePath + "/" + filepath.ToSlash(eff)
	}
	if editedRel := itemEditedResourceRel(it); editedRel != "" {
		dto.Edited = basePath + "/" + editedRel
	}
	return dto
}

// --- Handlers ---

func (h *Handler) publicAPIListItems(w http.ResponseWriter, r *http.Request, folderKey string) {
	maj, sub, ok := h.store.LookupFolderInCategories(folderKey)
	if !ok {
		apiNotFound(w)
		return
	}
	items, err := h.store.ReadItems(folderKey)
	if err != nil {
		h.log.Printf("public api list %s: %v", folderKey, err)
		apiInternalErr(w, "failed to read items")
		return
	}
	dtos := make([]publicAPIItemDTO, 0, len(items))
	for _, it := range items {
		if itemCanonicalResourceRel(it) == "" {
			continue
		}
		dtos = append(dtos, h.itemToPublicDTO(folderKey, maj, sub, it))
	}
	apiOK(w, map[string]any{"items": dtos, "total": len(dtos)})
}

func (h *Handler) publicAPIGetItem(w http.ResponseWriter, r *http.Request, folderKey, itemID string) {
	maj, sub, ok := h.store.LookupFolderInCategories(folderKey)
	if !ok {
		apiNotFound(w)
		return
	}
	items, err := h.store.ReadItems(folderKey)
	if err != nil {
		apiInternalErr(w, "failed to read items")
		return
	}
	for _, it := range items {
		if it.ID == itemID {
			apiOK(w, h.itemToPublicDTO(folderKey, maj, sub, it))
			return
		}
	}
	apiNotFound(w)
}

func (h *Handler) publicAPIUploadItem(w http.ResponseWriter, r *http.Request, folderKey string) {
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		apiBadRequest(w, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		apiBadRequest(w, "missing 'file' field")
		return
	}
	defer file.Close()

	item, err := h.store.SaveUploadedFile(folderKey, header.Filename, file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			apiNotFound(w)
			return
		}
		h.log.Printf("public api upload %s: %v", folderKey, err)
		apiInternalErr(w, "upload failed")
		return
	}

	// Apply optional metadata (title, tags, linkName)
	if title := strings.TrimSpace(r.FormValue("title")); title != "" {
		item, _ = h.store.PatchCategoryItem(folderKey, item.ID, store.PatchCategoryItemInput{
			SetTitle: true, Title: title,
		})
	}
	if rawTags := strings.TrimSpace(r.FormValue("tags")); rawTags != "" {
		var tags []string
		for _, t := range strings.Split(rawTags, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
		if len(tags) > 0 {
			item, _ = h.store.PatchCategoryItem(folderKey, item.ID, store.PatchCategoryItemInput{
				HasTags: true, Tags: tags,
			})
		}
	}
	if linkName := strings.TrimSpace(r.FormValue("linkName")); linkName != "" {
		item, _ = h.store.PatchCategoryItem(folderKey, item.ID, store.PatchCategoryItemInput{
			SetLinkName: true, LinkName: linkName,
		})
	}

	maj, sub, _ := h.store.LookupFolderInCategories(folderKey)
	apiCreated(w, h.itemToPublicDTO(folderKey, maj, sub, item))
}

func (h *Handler) publicAPIReplaceItem(w http.ResponseWriter, r *http.Request, folderKey, itemID string) {
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		apiBadRequest(w, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		apiBadRequest(w, "missing 'file' field")
		return
	}
	defer file.Close()

	patch := store.PatchCategoryItemInput{
		File:            file,
		Filename:        header.Filename,
		ReplaceOriginal: true,
	}
	if title := strings.TrimSpace(r.FormValue("title")); title != "" {
		patch.SetTitle = true
		patch.Title = title
	}
	if rawTags := strings.TrimSpace(r.FormValue("tags")); rawTags != "" {
		for _, t := range strings.Split(rawTags, ",") {
			if t = strings.TrimSpace(t); t != "" {
				patch.Tags = append(patch.Tags, t)
			}
		}
		patch.HasTags = len(patch.Tags) > 0
	}
	if linkName := strings.TrimSpace(r.FormValue("linkName")); linkName != "" {
		patch.SetLinkName = true
		patch.LinkName = linkName
	}

	item, err := h.store.PatchCategoryItem(folderKey, itemID, patch)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			apiNotFound(w)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			apiBadRequest(w, "invalid item id")
			return
		}
		h.log.Printf("public api replace %s/%s: %v", folderKey, itemID, err)
		apiInternalErr(w, "replace failed")
		return
	}
	maj, sub, _ := h.store.LookupFolderInCategories(folderKey)
	apiOK(w, h.itemToPublicDTO(folderKey, maj, sub, item))
}

func (h *Handler) publicAPIDeleteItem(w http.ResponseWriter, r *http.Request, folderKey, itemID string) {
	if err := h.store.DeleteCategoryItem(folderKey, itemID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			apiNotFound(w)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			apiBadRequest(w, "invalid item id")
			return
		}
		h.log.Printf("public api delete %s/%s: %v", folderKey, itemID, err)
		apiInternalErr(w, "delete failed")
		return
	}
	apiOK(w, map[string]bool{"ok": true})
}
