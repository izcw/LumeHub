package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"lumehub/internal/model"
)

func (h *Handler) handleTrashList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleTrashListGet(w, r)
	case http.MethodDelete:
		h.handleTrashClearAll(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleTrashListGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requirePermission(w, r, "manage_accounts"); !ok {
		return
	}
	entries, err := h.store.ListTrashItems()
	if err != nil {
		h.log.Printf("trash list: %v", err)
		http.Error(w, "failed to read trash", http.StatusInternalServerError)
		return
	}
	out := make([]model.TrashItemDTO, 0, len(entries))
	viewGrants := make(map[string]string)
	for _, entry := range entries {
		maj, sub, ok := h.store.LookupFolderInCategories(entry.FolderKey)
		if !ok {
			maj = model.Category{Name: entry.MajorName}
			sub = model.Subcategory{Name: entry.SubName, FolderKey: entry.FolderKey}
		}
		viewKey := ""
		if model.FolderRequiresEncryptedPassword(maj, sub) {
			if g, ok := viewGrants[entry.FolderKey]; ok {
				viewKey = g
			} else {
				grant, _ := h.auth.CreateViewGrant(entry.FolderKey)
				viewGrants[entry.FolderKey] = grant
				viewKey = grant
			}
		}
		dto := h.itemToDTO(entry.FolderKey, maj, sub, entry.Item, viewKey)
		out = append(out, model.TrashItemDTO{
			ItemDTO:         dto,
			FolderKey:       entry.FolderKey,
			MajorName:       entry.MajorName,
			SubName:         entry.SubName,
			DeletedAt:       entry.DeletedAt,
			ExpiresAt:       trashExpiresAtRFC3339(entry.DeletedAt),
			CategoryMissing: !ok,
		})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) handleTrashClearAll(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requirePermission(w, r, "manage_accounts"); !ok {
		return
	}
	n, err := h.store.ClearAllTrash()
	if err != nil {
		h.log.Printf("clear all trash: %v", err)
		http.Error(w, "failed to clear trash", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]int{"deleted": n})
}

func trashExpiresAtRFC3339(deletedAt string) string {
	deletedAt = strings.TrimSpace(deletedAt)
	if deletedAt == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, deletedAt)
	if err != nil {
		return ""
	}
	return t.Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
}

// handleTrashPath:
//   - DELETE /api/trash/{folderKey}
//   - DELETE /api/trash/{folderKey}/items/{itemId}
//   - POST   /api/trash/{folderKey}/items/{itemId}/restore
//   - POST   /api/trash/{folderKey}/restore
func (h *Handler) handleTrashPath(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requirePermission(w, r, "manage_accounts"); !ok {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/trash/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		if r.Method == http.MethodDelete {
			h.handleTrashClearAll(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(rest, "/")
	if len(parts) == 1 {
		if r.Method == http.MethodDelete {
			h.handleTrashClearFolder(w, r, parts[0])
			return
		}
		http.NotFound(w, r)
		return
	}
	if len(parts) == 2 && parts[1] == "restore" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleTrashRestoreFolder(w, r, parts[0])
		return
	}
	if len(parts) == 4 && parts[1] == "items" && parts[3] == "restore" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleTrashRestore(w, r, parts[0], parts[2])
		return
	}
	if len(parts) != 3 || parts[1] != "items" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	folderKey := parts[0]
	itemID := parts[2]
	if err := h.store.PermanentDeleteTrashItem(folderKey, itemID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid item id", http.StatusBadRequest)
			return
		}
		h.log.Printf("permanent delete trash %s/%s: %v", folderKey, itemID, err)
		http.Error(w, "failed to delete item", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleTrashRestore(w http.ResponseWriter, r *http.Request, folderKey, itemID string) {
	categoryRecreated, err := h.store.RestoreTrashItem(folderKey, itemID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, os.ErrExist) {
			http.Error(w, "item already exists in category", http.StatusConflict)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid item id", http.StatusBadRequest)
			return
		}
		h.log.Printf("restore trash %s/%s: %v", folderKey, itemID, err)
		http.Error(w, "failed to restore item", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"categoryRecreated": categoryRecreated,
	})
}

func (h *Handler) handleTrashRestoreFolder(w http.ResponseWriter, r *http.Request, folderKey string) {
	restored, categoryRecreated, err := h.store.RestoreTrashFolder(folderKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid folder key", http.StatusBadRequest)
			return
		}
		h.log.Printf("restore trash folder %s: %v", folderKey, err)
		http.Error(w, "failed to restore folder", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"restored":          restored,
		"categoryRecreated": categoryRecreated,
	})
}

func (h *Handler) handleTrashClearFolder(w http.ResponseWriter, r *http.Request, folderKey string) {
	n, err := h.store.ClearTrashFolder(folderKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid folder key", http.StatusBadRequest)
			return
		}
		h.log.Printf("clear trash folder %s: %v", folderKey, err)
		http.Error(w, "failed to clear folder trash", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]int{"deleted": n})
}
