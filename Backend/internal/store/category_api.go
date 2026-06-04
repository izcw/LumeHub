package store

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

)

// CategoryAPISettingsPatch updates API settings for a subcategory.
type CategoryAPISettingsPatch struct {
	MajorID int  `json:"majorId"`
	SubID   int  `json:"subId"`
	Enabled *bool `json:"enabled,omitempty"`
	// RefreshKey requests a new API key (returns the new plaintext key).
	RefreshKey bool `json:"refreshKey,omitempty"`
}

var ErrCategoryAPISettingsNotFound = errors.New("api settings target not found")

// PatchCategoryAPISettings updates API enabled/key for the given subcategory.
// Returns map of "majorId_subId" -> newPlainKey when a key was generated.
func (s *Store) PatchCategoryAPISettings(patches []CategoryAPISettingsPatch) (map[string]string, error) {
	if len(patches) == 0 {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return nil, err
	}
	results := make(map[string]string)
	for _, p := range patches {
		mi, sj := findSubIndex(doc, p.MajorID, p.SubID)
		if mi < 0 || sj < 0 {
			return nil, ErrCategoryAPISettingsNotFound
		}
		sub := &doc.Categories[mi].Subcategories[sj]
		if p.Enabled != nil {
			sub.APIEnabled = *p.Enabled
			// Auto-generate key on first enable if none exists
			if *p.Enabled && sub.APIKeyHash == "" {
				plainKey := generateAPIKey()
				sub.APIKeyHash = hashAPIKey(plainKey)
				results[fmt.Sprintf("%d_%d", p.MajorID, p.SubID)] = plainKey
			}
		}
		if p.RefreshKey {
			plainKey := generateAPIKey()
			sub.APIKeyHash = hashAPIKey(plainKey)
			results[fmt.Sprintf("%d_%d", p.MajorID, p.SubID)] = plainKey
		}
	}
	if err := s.writeCategoriesUnlocked(doc); err != nil {
		return nil, err
	}
	return results, nil
}

// GetCategoryAPISettings returns the API key hash and enabled state for a folder.
func (s *Store) GetCategoryAPISettings(folderKey string) (hash string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return "", false
	}
	for _, cat := range doc.Categories {
		for _, sub := range cat.Subcategories {
			if sub.FolderKey == folderKey {
				return sub.APIKeyHash, sub.APIEnabled
			}
		}
	}
	return "", false
}

func generateAPIKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(fmt.Sprintf("%d", len(b))))
	}
	return hex.EncodeToString(b)
}

func hashAPIKey(plain string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(plain))))
	return hex.EncodeToString(h[:])
}

// ValidateAPIKey checks if the provided key matches the stored hash for a folder.
func (s *Store) ValidateAPIKey(folderKey, apiKey string) bool {
	hash, enabled := s.GetCategoryAPISettings(folderKey)
	if !enabled || hash == "" {
		return false
	}
	inputHash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(apiKey))))
	return subtle.ConstantTimeCompare([]byte(hash), []byte(hex.EncodeToString(inputHash[:]))) == 1
}
