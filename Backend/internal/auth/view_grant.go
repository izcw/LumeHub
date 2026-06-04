package auth

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

const viewGrantTTL = 24 * time.Hour

type viewGrantRec struct {
	FolderKey string
	Expires   time.Time
}

func (m *Manager) initViewGrants() {
	if m.viewGrants == nil {
		m.viewGrants = make(map[string]viewGrantRec)
	}
}

// CreateViewGrant 为已通过查看密码校验的相册签发短期访问令牌（不含明文密码）。
func (m *Manager) CreateViewGrant(folderKey string) (token string, expires time.Time) {
	folderKey = strings.TrimSpace(folderKey)
	b := make([]byte, sessionBytes)
	if _, err := rand.Read(b); err != nil {
		b = []byte("fallback-view-grant-" + time.Now().String())
	}
	token = hex.EncodeToString(b)
	expires = time.Now().Add(viewGrantTTL)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initViewGrants()
	m.viewGrants[token] = viewGrantRec{FolderKey: folderKey, Expires: expires}
	m.gcViewGrantsUnlocked()
	return token, expires
}

// ValidViewGrant 校验相册查看令牌是否有效且匹配目录。
func (m *Manager) ValidViewGrant(folderKey, token string) bool {
	folderKey = strings.TrimSpace(folderKey)
	token = strings.TrimSpace(token)
	if folderKey == "" || token == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initViewGrants()
	m.gcViewGrantsUnlocked()
	rec, ok := m.viewGrants[token]
	if !ok || time.Now().After(rec.Expires) {
		return false
	}
	return rec.FolderKey == folderKey
}

func (m *Manager) gcViewGrantsUnlocked() {
	now := time.Now()
	for k, rec := range m.viewGrants {
		if now.After(rec.Expires) {
			delete(m.viewGrants, k)
		}
	}
}

// InvalidateViewGrantsForFolder 目录查看密码变更后作废该目录已签发的短期令牌。
func (m *Manager) InvalidateViewGrantsForFolder(folderKey string) {
	folderKey = strings.TrimSpace(folderKey)
	if folderKey == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initViewGrants()
	for k, rec := range m.viewGrants {
		if rec.FolderKey == folderKey {
			delete(m.viewGrants, k)
		}
	}
}
