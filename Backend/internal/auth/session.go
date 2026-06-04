package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"lumehub/internal/model"
	"lumehub/internal/store"
)

const (
	cookieName   = "lumehub_session"
	headerBearer = "Authorization"
	sessionBytes = 32
	defaultTTL   = 7 * 24 * time.Hour
)

type sessionRec struct {
	UserID  string
	Expires time.Time
}

// Manager 进程内会话；优先 accounts.json 多账号，否则可用 LUMEHUB_PASSWORD 单口令兼容。
type Manager struct {
	mu             sync.Mutex
	store          *store.Store
	legacyPassword string
	ttl            time.Duration
	tokens         map[string]sessionRec
	viewGrants     map[string]viewGrantRec
	lockout        *loginLockoutStore
	viewLockout    *viewPasswordLockoutStore
}

func New(st *store.Store, legacyPlainPassword string) *Manager {
	ttl := defaultTTL
	if s := strings.TrimSpace(os.Getenv("LUMEHUB_SESSION_HOURS")); s != "" {
		if h, err := strconv.Atoi(s); err == nil && h > 0 {
			ttl = time.Duration(h) * time.Hour
		}
	}
	return &Manager{
		store:          st,
		legacyPassword: strings.TrimSpace(legacyPlainPassword),
		ttl:            ttl,
		tokens:         make(map[string]sessionRec),
		lockout:        newLoginLockoutStore(),
		viewLockout:    newViewPasswordLockoutStore(),
	}
}

func (m *Manager) Configured() bool {
	doc, err := m.store.ReadAccounts()
	if err == nil && len(doc.Accounts) > 0 {
		return true
	}
	return m.legacyPassword != ""
}

// TryLogin 多账号或兼容单口令（无 accounts 时仅校验密码，会话 userID 为 legacy）。
func (m *Manager) TryLogin(email, password string) (userID string, ok bool) {
	doc, err := m.store.ReadAccounts()
	if err == nil && len(doc.Accounts) > 0 {
		acc, err := m.store.AuthenticateAccount(email, password)
		if err != nil || acc == nil {
			return "", false
		}
		return acc.ID, true
	}
	if m.legacyPassword == "" {
		return "", false
	}
	a := sha256.Sum256([]byte(strings.TrimSpace(password)))
	b := sha256.Sum256([]byte(m.legacyPassword))
	if subtle.ConstantTimeCompare(a[:], b[:]) == 1 {
		return "legacy", true
	}
	return "", false
}

func (m *Manager) Valid(r *http.Request) bool {
	if !m.Configured() {
		return true
	}
	tok := m.tokenFromRequest(r)
	if tok == "" {
		return false
	}
	_, ok := m.validSession(tok)
	return ok
}

func (m *Manager) ValidForResource(r *http.Request) bool {
	if !m.Configured() {
		return true
	}
	if m.Valid(r) {
		return true
	}
	q := strings.TrimSpace(r.URL.Query().Get("access_token"))
	if q == "" {
		return false
	}
	_, ok := m.validSession(q)
	return ok
}

func (m *Manager) validSession(tok string) (userID string, ok bool) {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return "", false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcUnlocked()
	rec, exists := m.tokens[tok]
	if !exists || time.Now().After(rec.Expires) {
		return "", false
	}
	return rec.UserID, true
}

func (m *Manager) SessionUserID(r *http.Request) (string, bool) {
	if !m.Configured() {
		return "", false
	}
	tok := m.tokenFromRequest(r)
	if tok == "" {
		return "", false
	}
	return m.validSession(tok)
}

func (m *Manager) CreateSession(userID string) (token string, expires time.Time) {
	b := make([]byte, sessionBytes)
	if _, err := rand.Read(b); err != nil {
		b = []byte("fallback-insecure-" + time.Now().String())
	}
	token = hex.EncodeToString(b)
	expires = time.Now().Add(m.ttl)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token] = sessionRec{UserID: userID, Expires: expires}
	return token, expires
}

func (m *Manager) Invalidate(token string) {
	if token == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, token)
}

// InvalidateUserSessions 注销指定用户的全部会话（修改登录邮箱或密码后调用）。
func (m *Manager) InvalidateUserSessions(userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for tok, rec := range m.tokens {
		if rec.UserID == userID {
			delete(m.tokens, tok)
		}
	}
}

func (m *Manager) SetSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	sec := os.Getenv("LUMEHUB_COOKIE_SECURE") == "1" || os.Getenv("LUMEHUB_COOKIE_SECURE") == "true"
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sec,
	})
}

func (m *Manager) ClearSessionCookie(w http.ResponseWriter) {
	sec := os.Getenv("LUMEHUB_COOKIE_SECURE") == "1" || os.Getenv("LUMEHUB_COOKIE_SECURE") == "true"
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sec,
	})
}

func (m *Manager) tokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return strings.TrimSpace(c.Value)
	}
	h := strings.TrimSpace(r.Header.Get(headerBearer))
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (m *Manager) TokenFromRequest(r *http.Request) string {
	return m.tokenFromRequest(r)
}

func (m *Manager) gcUnlocked() {
	now := time.Now()
	for k, rec := range m.tokens {
		if now.After(rec.Expires) {
			delete(m.tokens, k)
		}
	}
	m.initViewGrants()
	m.gcViewGrantsUnlocked()
}

// LegacyMeUser 单口令模式下的占位用户信息。
func LegacyMeUser() model.AccountPublic {
	return model.AccountPublic{
		ID:          "legacy",
		Username:    "guest",
		Email:       "",
		DisplayName: "访客",
		Avatar:      "",
		Roles:       nil,
		Permissions: []string{"manage_layout"},
	}
}
