package model

// AccountsDoc 多账号配置（data/accounts.json）。
type AccountsDoc struct {
	Version int `json:"version"`
	// GuestAvatarURL 未登录时导航等使用的头像地址（如 /resource/system/guest.svg）。
	GuestAvatarURL string `json:"guestAvatarUrl,omitempty"`
	// LoggedInFallbackAvatarURL 已登录但用户未设置头像、且无 data/system/avatar 文件时的占位图。
	LoggedInFallbackAvatarURL string    `json:"loggedInFallbackAvatarUrl,omitempty"`
	Accounts                  []Account `json:"accounts"`
}

// Account 单账号；passwordHash 为 UTF-8 密码的 SHA256 十六进制小写。
type Account struct {
	ID           string              `json:"id"`
	Username     string              `json:"username"`
	Email        string              `json:"email"`
	PasswordHash string              `json:"passwordHash"`
	DisplayName  string              `json:"displayName"`
	Avatar       string              `json:"avatar,omitempty"`
	Roles        []string            `json:"roles,omitempty"`
	Permissions  []string            `json:"permissions,omitempty"`
	Passkeys     []PasskeyCredential `json:"passkeys,omitempty"`
}

// PasskeyCredential 存储 WebAuthn 通行证公钥与计数器（服务端验证签名用）。
type PasskeyCredential struct {
	ID         string   `json:"id"`
	Label      string   `json:"label,omitempty"`
	PublicKey  string   `json:"publicKey"`
	Algorithm  int      `json:"algorithm"`
	SignCount  uint32   `json:"signCount"`
	Transports []string `json:"transports,omitempty"`
	CreatedAt  string   `json:"createdAt,omitempty"`
	LastUsedAt string   `json:"lastUsedAt,omitempty"`
}

// AccountPublic 返回给前端的账号信息（不含 passwordHash）。
type AccountPublic struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"displayName"`
	Avatar      string   `json:"avatar,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// ToAccountPublic 从 Account 生成可序列化副本。
func ToAccountPublic(a Account) AccountPublic {
	return AccountPublic{
		ID:          a.ID,
		Username:    a.Username,
		Email:       a.Email,
		DisplayName: a.DisplayName,
		Avatar:      a.Avatar,
		Roles:       append([]string(nil), a.Roles...),
		Permissions: append([]string(nil), a.Permissions...),
	}
}
