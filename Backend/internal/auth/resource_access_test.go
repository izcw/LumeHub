package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"lumehub/internal/model"
	"lumehub/internal/store"
)

func writePolicyFixture(t *testing.T, dir, folderKey, stem, categoriesJSON string) {
	t.Helper()
	catDir := filepath.Join(dir, "resource", folderKey)
	if err := os.MkdirAll(filepath.Join(catDir, "original"), 0o755); err != nil {
		t.Fatal(err)
	}
	itemFile := stem + ".jpg"
	if err := os.WriteFile(filepath.Join(catDir, "original", itemFile), []byte("jpeg-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	itemsJSON := `{"items":[{"id":"id1","filename":"original/` + itemFile + `","linkName":"` + itemFile + `"}]}`
	if err := os.WriteFile(filepath.Join(catDir, "items.json"), []byte(itemsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "categories.json"), []byte(categoriesJSON), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResourceDirGlobalShortLinkPolicies(t *testing.T) {
	pwd := "1234"
	pwdHash := model.HashPasswordUTF8(pwd)

	subtests := []struct {
		name     string
		subJSON  string
		wantNoK  int
		wantWith int
	}{
		{
			name:     "open",
			subJSON:  `{"id":1,"name":"Open","folderKey":"lab","public":true}`,
			wantNoK:  http.StatusOK,
			wantWith: http.StatusOK,
		},
		{
			name:     "encrypted_public",
			subJSON:  `{"id":1,"name":"EncPub","folderKey":"lab","encrypted":true,"encryptedPasswordHash":"` + pwdHash + `"}`,
			wantNoK:  http.StatusForbidden,
			wantWith: http.StatusOK,
		},
		{
			name:     "private",
			subJSON:  `{"id":1,"name":"Private","folderKey":"lab","public":false}`,
			wantNoK:  http.StatusOK,
			wantWith: http.StatusOK,
		},
		{
			name:     "encrypted_hidden",
			subJSON:  `{"id":1,"name":"EncHid","folderKey":"lab","public":false,"encrypted":true,"encryptedPasswordHash":"` + pwdHash + `"}`,
			wantNoK:  http.StatusForbidden,
			wantWith: http.StatusOK,
		},
	}

	for _, tc := range subtests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			stem := "pic_" + tc.name
			categoriesJSON := `{"version":1,"categories":[{"id":1,"name":"M","public":true,"subcategories":[` + tc.subJSON + `]}]}`
			writePolicyFixture(t, dir, "lab", stem, categoriesJSON)

			st := store.New(dir)
			mgr := New(st, "admin-pass")
			handler := ResourceDir(mgr, st, http.Dir(filepath.Join(dir, "resource")))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+stem, nil))
			if rec.Code != tc.wantNoK {
				t.Fatalf("without key: got %d body=%q", rec.Code, rec.Body.String())
			}

			rec = httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+stem+"?k="+pwd, nil))
			if rec.Code != tc.wantWith {
				t.Fatalf("with key: got %d body=%q", rec.Code, rec.Body.String())
			}
		})
	}
}
