package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"lumehub/internal/store"
)

type patchStorageBody struct {
	QuotaBytes *int64 `json:"quotaBytes"`
	QuotaGB    *float64 `json:"quotaGb"`
}

func (h *Handler) handleStorageGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requirePermission(w, r, "manage_accounts"); !ok {
		return
	}
	st, err := h.store.StorageStatus()
	if err != nil {
		h.log.Printf("storage status: %v", err)
		http.Error(w, "failed to read storage", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(st)
}

func (h *Handler) handleStoragePatch(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requirePermission(w, r, "manage_accounts"); !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<14))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req patchStorageBody
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	var quota int64
	switch {
	case req.QuotaBytes != nil && *req.QuotaBytes > 0:
		quota = *req.QuotaBytes
	case req.QuotaGB != nil && *req.QuotaGB > 0:
		quota = int64(*req.QuotaGB * (1024 * 1024 * 1024))
	default:
		http.Error(w, "quotaBytes or quotaGb required", http.StatusBadRequest)
		return
	}
	st, err := h.store.PatchStorageQuota(quota)
	if err != nil {
		if errors.Is(err, os.ErrInvalid) {
			http.Error(w, "invalid quota", http.StatusBadRequest)
			return
		}
		h.log.Printf("storage patch: %v", err)
		http.Error(w, "failed to save storage", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(st)
}

func (h *Handler) handleStorageRecalculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.requirePermission(w, r, "manage_accounts"); !ok {
		return
	}
	if removed, err := h.store.CleanupOrphanResourceFolders(); err != nil {
		h.log.Printf("storage orphan cleanup: %v", err)
		http.Error(w, "failed to cleanup orphan folders", http.StatusInternalServerError)
		return
	} else if len(removed) > 0 {
		h.log.Printf("storage orphan cleanup: removed %v", removed)
	}
	if _, err := h.store.RecalculateStorageUsed(); err != nil {
		h.log.Printf("storage recalculate: %v", err)
		http.Error(w, "failed to recalculate", http.StatusInternalServerError)
		return
	}
	st, err := h.store.StorageStatus()
	if err != nil {
		http.Error(w, "failed to read storage", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(st)
}

func storageQuotaHTTPError(w http.ResponseWriter) {
	http.Error(w, "storage quota exceeded", http.StatusInsufficientStorage)
}

func isStorageQuotaErr(err error) bool {
	return errors.Is(err, store.ErrStorageQuotaExceeded)
}
