package auth

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var deviceIDPattern = regexp.MustCompile(`^[a-f0-9]{16,128}$`)

type viewPasswordLockoutStore struct {
	inner *loginLockoutStore
}

func newViewPasswordLockoutStore() *viewPasswordLockoutStore {
	return &viewPasswordLockoutStore{inner: newLoginLockoutStore()}
}

func viewPasswordLockoutMessage(wait time.Duration) string {
	return fmt.Sprintf("查看密码失败次数过多，请 %s 后重试", formatLockoutWait(wait))
}

func viewPasswordLockoutKey(folderKey, actorKey string) string {
	folderKey = strings.TrimSpace(folderKey)
	actorKey = strings.TrimSpace(actorKey)
	if folderKey == "" || actorKey == "" {
		return ""
	}
	return folderKey + "\x1f" + actorKey
}

// NormalizeDeviceID 校验客户端 canvas 设备指纹（十六进制，16–128 位）。
func NormalizeDeviceID(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if !deviceIDPattern.MatchString(raw) {
		return ""
	}
	return raw
}

// ViewUnlockActorKey 已登录用用户 ID，未登录用设备 ID。
func (m *Manager) ViewUnlockActorKey(r *http.Request, deviceID string) string {
	if uid, ok := m.SessionUserID(r); ok {
		uid = strings.TrimSpace(uid)
		if uid != "" {
			return "u:" + uid
		}
	}
	if id := NormalizeDeviceID(deviceID); id != "" {
		return "d:" + id
	}
	return ""
}

func (s *viewPasswordLockoutStore) check(folderKey, actorKey string) (blocked bool, msg string) {
	key := viewPasswordLockoutKey(folderKey, actorKey)
	if key == "" {
		return false, ""
	}
	now := time.Now()
	s.inner.mu.Lock()
	defer s.inner.mu.Unlock()
	rec := s.inner.recs[key]
	if rec == nil || rec.LockedUntil.IsZero() || !now.Before(rec.LockedUntil) {
		return false, ""
	}
	return true, viewPasswordLockoutMessage(rec.LockedUntil.Sub(now))
}

func (s *viewPasswordLockoutStore) recordFailure(folderKey, actorKey string) (blocked bool, msg string) {
	key := viewPasswordLockoutKey(folderKey, actorKey)
	if key == "" {
		return false, ""
	}
	now := time.Now()
	s.inner.mu.Lock()
	defer s.inner.mu.Unlock()
	rec := s.inner.recs[key]
	if rec == nil {
		rec = &loginLockoutRec{}
		s.inner.recs[key] = rec
	}
	if !rec.LockedUntil.IsZero() && now.Before(rec.LockedUntil) {
		return true, viewPasswordLockoutMessage(rec.LockedUntil.Sub(now))
	}
	if !rec.LockedUntil.IsZero() && !now.Before(rec.LockedUntil) {
		rec.LockedUntil = time.Time{}
	}
	rec.FailedAttempts++
	if rec.FailedAttempts < loginMaxFailuresBeforeLock {
		return false, ""
	}
	wait := loginLockoutDuration(rec.LockoutLevel)
	rec.LockedUntil = now.Add(wait)
	rec.LockoutLevel++
	rec.FailedAttempts = 0
	return true, viewPasswordLockoutMessage(wait)
}

func (s *viewPasswordLockoutStore) recordSuccess(folderKey, actorKey string) {
	key := viewPasswordLockoutKey(folderKey, actorKey)
	if key == "" {
		return
	}
	s.inner.mu.Lock()
	defer s.inner.mu.Unlock()
	delete(s.inner.recs, key)
}

// CheckViewPasswordLockout 检查该用户/设备在指定相册是否处于查看密码锁定状态。
func (m *Manager) CheckViewPasswordLockout(folderKey, actorKey string) (blocked bool, msg string) {
	return m.viewLockout.check(folderKey, actorKey)
}

// RecordViewPasswordFailure 记录一次查看密码失败。
func (m *Manager) RecordViewPasswordFailure(folderKey, actorKey string) (blocked bool, msg string) {
	return m.viewLockout.recordFailure(folderKey, actorKey)
}

// RecordViewPasswordSuccess 查看密码正确后清除锁定状态。
func (m *Manager) RecordViewPasswordSuccess(folderKey, actorKey string) {
	m.viewLockout.recordSuccess(folderKey, actorKey)
}
