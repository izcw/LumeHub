package model

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// HashPasswordUTF8 返回密码 UTF-8 字节的 SHA256 十六进制小写（写入 accounts.json）。
func HashPasswordUTF8(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// PasswordHashMatches 比较明文密码与 accounts.json 中的 SHA256 十六进制哈希。
func PasswordHashMatches(storedHex, plaintext string) bool {
	want := strings.TrimSpace(strings.ToLower(storedHex))
	if want == "" {
		return false
	}
	sum := sha256.Sum256([]byte(plaintext))
	got := hex.EncodeToString(sum[:])
	if len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// PasswordHashHexMatches 比较两个 SHA256 十六进制小写串（常量时间比较）。
func PasswordHashHexMatches(storedHex, providedHex string) bool {
	want := strings.TrimSpace(strings.ToLower(storedHex))
	got := strings.TrimSpace(strings.ToLower(providedHex))
	if want == "" || got == "" {
		return false
	}
	if len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
