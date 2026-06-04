package auth

import (
	"os"
	"path/filepath"
	"testing"

	"lumehub/internal/store"
)

func TestRewriteResourcePathByAlias_thumbnailNotMappedToOriginal(t *testing.T) {
	dir := t.TempDir()
	jilu := filepath.Join(dir, "resource", "jilu")
	if err := os.MkdirAll(filepath.Join(jilu, "thumb"), 0o755); err != nil {
		t.Fatal(err)
	}
	itemsJSON := `{"items":[{"id":"ff6c59dcb698_20260529","filename":"original/ff6c59dcb698_20260529.mp4","linkName":"ff6c59dcb698_20260529.mp4","thumbnail":"thumb/ff6c59dcb698_20260529.jpg"}]}`
	if err := os.WriteFile(filepath.Join(jilu, "items.json"), []byte(itemsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	st := store.New(dir)
	got, ok := rewriteResourcePathByAlias(st, "jilu", []string{"thumb", "ff6c59dcb698_20260529.jpg"})
	if !ok {
		t.Fatal("expected rewrite")
	}
	want := "/jilu/thumb/ff6c59dcb698_20260529.jpg"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
