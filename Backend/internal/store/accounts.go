package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lumehub/internal/model"
)

var (
	// ErrPatchMeCurrentPasswordRequired 修改登录邮箱或密码时未提供当前密码。
	ErrPatchMeCurrentPasswordRequired = errors.New("current password required")
	// ErrPatchMeWrongCurrentPassword 当前密码校验失败。
	ErrPatchMeWrongCurrentPassword = errors.New("wrong current password")
	// ErrPatchMeEmailTaken 邮箱已被其他账号占用。
	ErrPatchMeEmailTaken = errors.New("email taken")
	// ErrPatchMeEmailEmpty 邮箱不能为空。
	ErrPatchMeEmailEmpty = errors.New("email empty")
	// ErrPatchMePasswordTooShort 新密码过短。
	ErrPatchMePasswordTooShort = errors.New("password too short")
	// ErrAdminUpdateDisplayNameEmpty 管理员更新用户时用户名不能为空。
	ErrAdminUpdateDisplayNameEmpty = errors.New("admin update displayName empty")
	// ErrAdminUpdateEmailEmpty 管理员更新用户时邮箱不能为空。
	ErrAdminUpdateEmailEmpty = errors.New("admin update email empty")
	// ErrAdminUpdateEmailTaken 管理员更新用户时邮箱与其他账号冲突。
	ErrAdminUpdateEmailTaken = errors.New("admin update email taken")
	// ErrAdminCreatePasswordEmpty 创建用户时密码不能为空。
	ErrAdminCreatePasswordEmpty = errors.New("admin create password empty")
	// ErrAdminDeleteSelf 不能删除当前登录账号。
	ErrAdminDeleteSelf = errors.New("admin delete self")
)

func (s *Store) accountsPath() string {
	return filepath.Join(s.dataDir, "accounts.json")
}

func (s *Store) ReadAccounts() (*model.AccountsDoc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.accountsPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &model.AccountsDoc{Version: 1, Accounts: nil}, nil
		}
		return nil, err
	}
	var doc model.AccountsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Store) writeAccountsUnlocked(doc *model.AccountsDoc) error {
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.accountsPath(), raw, 0o644)
}

// AuthenticateAccount 按邮箱（忽略大小写）与密码校验，返回账号副本（不含敏感字段由调用方剥离）。
func (s *Store) AuthenticateAccount(email, password string) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.accountsPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	var doc model.AccountsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	u := strings.TrimSpace(email)
	for i := range doc.Accounts {
		loginEmail := strings.TrimSpace(doc.Accounts[i].Email)
		if loginEmail == "" {
			// 兼容旧数据：若未配置 email，退化到 username。
			loginEmail = strings.TrimSpace(doc.Accounts[i].Username)
		}
		if strings.EqualFold(loginEmail, u) {
			if !model.PasswordHashMatches(doc.Accounts[i].PasswordHash, password) {
				return nil, errors.New("invalid password")
			}
			c := doc.Accounts[i]
			return &c, nil
		}
	}
	return nil, os.ErrNotExist
}

func (s *Store) GetAccountByID(id string) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.accountsPath())
	if err != nil {
		return nil, err
	}
	var doc model.AccountsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	for i := range doc.Accounts {
		if doc.Accounts[i].ID == id {
			c := doc.Accounts[i]
			return &c, nil
		}
	}
	return nil, os.ErrNotExist
}

// PatchAccountMe 更新当前用户资料；修改邮箱或密码需校验当前密码。
func (s *Store) PatchAccountMe(id, currentPassword string, displayName, avatar, email, newPassword *string) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readAccountsUnlocked()
	if err != nil {
		return nil, err
	}
	found := -1
	for i := range doc.Accounts {
		if doc.Accounts[i].ID == id {
			found = i
			break
		}
	}
	if found < 0 {
		return nil, os.ErrNotExist
	}

	emailChange := false
	if email != nil {
		ne := strings.TrimSpace(*email)
		if ne == "" {
			return nil, ErrPatchMeEmailEmpty
		}
		currentEmail := strings.TrimSpace(doc.Accounts[found].Email)
		if currentEmail == "" {
			currentEmail = strings.TrimSpace(doc.Accounts[found].Username)
		}
		if !strings.EqualFold(ne, currentEmail) {
			emailChange = true
			for i := range doc.Accounts {
				if i == found {
					continue
				}
				otherEmail := strings.TrimSpace(doc.Accounts[i].Email)
				if otherEmail == "" {
					otherEmail = strings.TrimSpace(doc.Accounts[i].Username)
				}
				if strings.EqualFold(otherEmail, ne) {
					return nil, ErrPatchMeEmailTaken
				}
			}
		}
	}

	newPassTrim := ""
	if newPassword != nil {
		newPassTrim = strings.TrimSpace(*newPassword)
	}
	passwordChange := newPassTrim != ""

	if emailChange || passwordChange {
		cp := strings.TrimSpace(currentPassword)
		if cp == "" {
			return nil, ErrPatchMeCurrentPasswordRequired
		}
		if !model.PasswordHashMatches(doc.Accounts[found].PasswordHash, cp) {
			return nil, ErrPatchMeWrongCurrentPassword
		}
	}
	if passwordChange && len(newPassTrim) < 6 {
		return nil, ErrPatchMePasswordTooShort
	}

	if displayName != nil {
		doc.Accounts[found].DisplayName = strings.TrimSpace(*displayName)
	}
	if avatar != nil {
		doc.Accounts[found].Avatar = strings.TrimSpace(*avatar)
	}
	if email != nil {
		ne := strings.TrimSpace(*email)
		currentEmail := strings.TrimSpace(doc.Accounts[found].Email)
		if currentEmail == "" {
			currentEmail = strings.TrimSpace(doc.Accounts[found].Username)
		}
		if !strings.EqualFold(ne, currentEmail) {
			doc.Accounts[found].Email = ne
		}
	}
	if passwordChange {
		doc.Accounts[found].PasswordHash = model.HashPasswordUTF8(newPassTrim)
	}

	if err := s.writeAccountsUnlocked(doc); err != nil {
		return nil, err
	}
	c := doc.Accounts[found]
	return &c, nil
}

// AdminUpdateAccount 由具备 manage_accounts 的账号更新目标账号资料和权限。
// 修改目标邮箱或密码时须校验操作者当前密码。
func (s *Store) AdminUpdateAccount(
	operatorID, operatorCurrentPassword, targetID, displayName, email string,
	roles, permissions []string,
	newPassword *string,
) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readAccountsUnlocked()
	if err != nil {
		return nil, err
	}
	found := -1
	operatorIdx := -1
	for i := range doc.Accounts {
		if doc.Accounts[i].ID == targetID {
			found = i
		}
		if doc.Accounts[i].ID == operatorID {
			operatorIdx = i
		}
	}
	if found < 0 {
		return nil, os.ErrNotExist
	}
	if operatorIdx < 0 {
		return nil, ErrPatchMeWrongCurrentPassword
	}

	nextDisplayName := strings.TrimSpace(displayName)
	if nextDisplayName == "" {
		return nil, ErrAdminUpdateDisplayNameEmpty
	}
	nextEmail := strings.TrimSpace(email)
	if nextEmail == "" {
		return nil, ErrAdminUpdateEmailEmpty
	}

	currentEmail := strings.TrimSpace(doc.Accounts[found].Email)
	if currentEmail == "" {
		currentEmail = strings.TrimSpace(doc.Accounts[found].Username)
	}
	emailChange := !strings.EqualFold(nextEmail, currentEmail)

	newPassTrim := ""
	if newPassword != nil {
		newPassTrim = strings.TrimSpace(*newPassword)
	}
	passwordChange := newPassTrim != ""

	if emailChange || passwordChange {
		cp := strings.TrimSpace(operatorCurrentPassword)
		if cp == "" {
			return nil, ErrPatchMeCurrentPasswordRequired
		}
		if !model.PasswordHashMatches(doc.Accounts[operatorIdx].PasswordHash, cp) {
			return nil, ErrPatchMeWrongCurrentPassword
		}
	}
	if passwordChange && len(newPassTrim) < 6 {
		return nil, ErrPatchMePasswordTooShort
	}

	for i := range doc.Accounts {
		if i == found {
			continue
		}
		otherEmail := strings.TrimSpace(doc.Accounts[i].Email)
		if otherEmail == "" {
			otherEmail = strings.TrimSpace(doc.Accounts[i].Username)
		}
		if strings.EqualFold(otherEmail, nextEmail) {
			return nil, ErrAdminUpdateEmailTaken
		}
	}
	doc.Accounts[found].DisplayName = nextDisplayName
	if emailChange {
		doc.Accounts[found].Email = nextEmail
	}
	doc.Accounts[found].Roles = append([]string(nil), roles...)
	doc.Accounts[found].Permissions = append([]string(nil), permissions...)
	if passwordChange {
		doc.Accounts[found].PasswordHash = model.HashPasswordUTF8(newPassTrim)
	}
	if err := s.writeAccountsUnlocked(doc); err != nil {
		return nil, err
	}
	c := doc.Accounts[found]
	return &c, nil
}

// AdminCreateAccount 创建新账号。
func (s *Store) AdminCreateAccount(
	displayName, email, password string,
	roles, permissions []string,
) (*model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readAccountsUnlocked()
	if err != nil {
		return nil, err
	}
	nextDisplayName := strings.TrimSpace(displayName)
	if nextDisplayName == "" {
		return nil, ErrAdminUpdateDisplayNameEmpty
	}
	nextEmail := strings.TrimSpace(email)
	if nextEmail == "" {
		return nil, ErrAdminUpdateEmailEmpty
	}
	passTrim := strings.TrimSpace(password)
	if passTrim == "" {
		return nil, ErrAdminCreatePasswordEmpty
	}
	if len(passTrim) < 6 {
		return nil, ErrPatchMePasswordTooShort
	}
	for i := range doc.Accounts {
		otherEmail := strings.TrimSpace(doc.Accounts[i].Email)
		if otherEmail == "" {
			otherEmail = strings.TrimSpace(doc.Accounts[i].Username)
		}
		if strings.EqualFold(otherEmail, nextEmail) {
			return nil, ErrAdminUpdateEmailTaken
		}
	}
	id := nextAccountID(doc)
	username := strings.Split(nextEmail, "@")[0]
	if strings.TrimSpace(username) == "" {
		username = "user" + id
	}
	acc := model.Account{
		ID:           id,
		Username:     username,
		Email:        nextEmail,
		PasswordHash: model.HashPasswordUTF8(passTrim),
		DisplayName:  nextDisplayName,
		Roles:        append([]string(nil), roles...),
		Permissions:  append([]string(nil), permissions...),
	}
	doc.Accounts = append(doc.Accounts, acc)
	if err := s.writeAccountsUnlocked(doc); err != nil {
		return nil, err
	}
	c := acc
	return &c, nil
}

// AdminDeleteAccount 删除指定账号（不可删除当前登录账号）。
func (s *Store) AdminDeleteAccount(operatorID, targetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(targetID) == strings.TrimSpace(operatorID) {
		return ErrAdminDeleteSelf
	}
	doc, err := s.readAccountsUnlocked()
	if err != nil {
		return err
	}
	found := -1
	for i := range doc.Accounts {
		if doc.Accounts[i].ID == targetID {
			found = i
			break
		}
	}
	if found < 0 {
		return os.ErrNotExist
	}
	doc.Accounts = append(doc.Accounts[:found], doc.Accounts[found+1:]...)
	return s.writeAccountsUnlocked(doc)
}

func nextAccountID(doc *model.AccountsDoc) string {
	max := -1
	for i := range doc.Accounts {
		if n, err := strconv.Atoi(strings.TrimSpace(doc.Accounts[i].ID)); err == nil && n > max {
			max = n
		}
	}
	return strconv.Itoa(max + 1)
}

func (s *Store) readAccountsUnlocked() (*model.AccountsDoc, error) {
	raw, err := os.ReadFile(s.accountsPath())
	if err != nil {
		return nil, err
	}
	var doc model.AccountsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Store) ListAccountsPublic() ([]model.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.readAccountsUnlocked()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]model.Account, len(doc.Accounts))
	copy(out, doc.Accounts)
	for i := range out {
		out[i].PasswordHash = ""
	}
	return out, nil
}
