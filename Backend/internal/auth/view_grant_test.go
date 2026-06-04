package auth

import (
	"testing"

	"lumehub/internal/store"
)

func TestInvalidateViewGrantsForFolder(t *testing.T) {
	st := store.New(t.TempDir())
	mgr := New(st, "admin-pass")

	grant, _ := mgr.CreateViewGrant("camera")
	if !mgr.ValidViewGrant("camera", grant) {
		t.Fatal("grant should be valid before invalidation")
	}

	mgr.InvalidateViewGrantsForFolder("camera")
	if mgr.ValidViewGrant("camera", grant) {
		t.Fatal("grant should be invalid after password rotation invalidation")
	}
}
