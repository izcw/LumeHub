package auth

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	loginMaxFailuresBeforeLock = 5
	loginBaseLockoutDuration   = 10 * time.Minute
	loginLockoutMultiplier     = 6
)

type loginLockoutRec struct {
	FailedAttempts int
	LockoutLevel   int
	LockedUntil    time.Time
}

type loginLockoutStore struct {
	mu   sync.Mutex
	recs map[string]*loginLockoutRec
}

func newLoginLockoutStore() *loginLockoutStore {
	return &loginLockoutStore{recs: make(map[string]*loginLockoutRec)}
}

// NormalizeLoginEmail 统一登录邮箱键（忽略大小写与首尾空格）。
func NormalizeLoginEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func loginLockoutDuration(level int) time.Duration {
	if level < 0 {
		level = 0
	}
	mult := 1
	for i := 0; i < level; i++ {
		mult *= loginLockoutMultiplier
	}
	return time.Duration(mult) * loginBaseLockoutDuration
}

func formatLockoutWait(d time.Duration) string {
	if d < time.Minute {
		return "1 分钟"
	}
	mins := int(math.Ceil(d.Minutes()))
	if mins < 60 {
		return fmt.Sprintf("%d 分钟", mins)
	}
	hours := mins / 60
	rem := mins % 60
	if rem == 0 {
		return fmt.Sprintf("%d 小时", hours)
	}
	return fmt.Sprintf("%d 小时 %d 分钟", hours, rem)
}

func loginLockoutMessage(wait time.Duration) string {
	return fmt.Sprintf("登录失败次数过多，请 %s 后重试", formatLockoutWait(wait))
}

func (s *loginLockoutStore) check(emailKey string) (blocked bool, msg string) {
	if emailKey == "" {
		return false, ""
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.recs[emailKey]
	if rec == nil || rec.LockedUntil.IsZero() || !now.Before(rec.LockedUntil) {
		return false, ""
	}
	return true, loginLockoutMessage(rec.LockedUntil.Sub(now))
}

func (s *loginLockoutStore) recordFailure(emailKey string) (blocked bool, msg string) {
	if emailKey == "" {
		return false, ""
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.recs[emailKey]
	if rec == nil {
		rec = &loginLockoutRec{}
		s.recs[emailKey] = rec
	}
	if !rec.LockedUntil.IsZero() && now.Before(rec.LockedUntil) {
		return true, loginLockoutMessage(rec.LockedUntil.Sub(now))
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
	return true, loginLockoutMessage(wait)
}

func (s *loginLockoutStore) recordSuccess(emailKey string) {
	if emailKey == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.recs, emailKey)
}

func (s *loginLockoutStore) clear(emailKey string) {
	if emailKey == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.recs, emailKey)
}

// CheckLoginLockout 检查邮箱是否处于登录锁定状态。
func (m *Manager) CheckLoginLockout(email string) (blocked bool, msg string) {
	return m.lockout.check(NormalizeLoginEmail(email))
}

// RecordLoginFailure 记录一次登录失败；连续失败 5 次后按阶梯时长锁定。
func (m *Manager) RecordLoginFailure(email string) (blocked bool, msg string) {
	return m.lockout.recordFailure(NormalizeLoginEmail(email))
}

// RecordLoginSuccess 登录成功后清除该邮箱的失败计数与锁定状态。
func (m *Manager) RecordLoginSuccess(email string) {
	m.lockout.recordSuccess(NormalizeLoginEmail(email))
}

// ClearLoginLockout 清除指定邮箱的登录锁定（如管理员重置密码后）。
func (m *Manager) ClearLoginLockout(email string) {
	m.lockout.clear(NormalizeLoginEmail(email))
}
