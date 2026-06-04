package model

import "testing"

func boolPtr(v bool) *bool { return &v }

func TestFolderAccessPolicySemantics(t *testing.T) {
	openMajor := Category{Public: boolPtr(true)}
	openSub := Subcategory{}

	encPubMajor := Category{Public: boolPtr(true)}
	encPubSub := Subcategory{Encrypted: true, EncryptedPasswordHash: "abc"}

	privMajor := Category{Public: boolPtr(true)}
	privSub := Subcategory{Public: boolPtr(false)}

	encHidMajor := Category{Public: boolPtr(true)}
	encHidSub := Subcategory{Public: boolPtr(false), Encrypted: true, EncryptedPasswordHash: "abc"}

	cases := []struct {
		name      string
		major     Category
		sub       Subcategory
		wantLogin bool
		wantKey   bool
	}{
		{"open", openMajor, openSub, false, false},
		{"encrypted_public", encPubMajor, encPubSub, false, true},
		{"private", privMajor, privSub, true, false},
		{"encrypted_hidden", encHidMajor, encHidSub, true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FolderRequiresLogin(tc.major, tc.sub); got != tc.wantLogin {
				t.Fatalf("FolderRequiresLogin = %v, want %v", got, tc.wantLogin)
			}
			if got := FolderResourceRequiresViewKey(tc.major, tc.sub); got != tc.wantKey {
				t.Fatalf("FolderResourceRequiresViewKey = %v, want %v", got, tc.wantKey)
			}
		})
	}
}

func TestFolderEncryptedPasswordHashSubPriority(t *testing.T) {
	major := Category{
		Encrypted:             true,
		EncryptedPasswordHash: "major-hash",
	}
	sub := Subcategory{
		Encrypted:             true,
		EncryptedPasswordHash: "sub-hash",
	}
	if got := FolderEncryptedPasswordHash(major, sub); got != "sub-hash" {
		t.Fatalf("got %q want sub-hash", got)
	}
}

func TestFolderEncryptedPasswordHashLegacySubHash(t *testing.T) {
	major := Category{Public: boolPtr(true)}
	sub := Subcategory{EncryptedPasswordHash: "legacy-hash"}
	if !FolderResourceRequiresViewKey(major, sub) {
		t.Fatal("legacy sub hash should require view key")
	}
	if got := FolderEncryptedPasswordHash(major, sub); got != "legacy-hash" {
		t.Fatalf("got %q want legacy-hash", got)
	}
}

func TestFolderEncryptedPasswordHashMajorNotInheritedWhenOpen(t *testing.T) {
	major := Category{
		Public:                boolPtr(true),
		EncryptedPasswordHash: "orphan-major-hash",
	}
	sub := Subcategory{Public: boolPtr(true)}
	if FolderResourceRequiresViewKey(major, sub) {
		t.Fatal("open sub should not inherit orphan major hash without encrypted flag")
	}
}
