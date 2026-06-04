package model

import "strings"

// BoolOrPublicTrue JSON 省略 public 时视为公开。
func BoolOrPublicTrue(p *bool) bool {
	return p == nil || *p
}

// FolderRequiresLogin 浏览画廊 API、列表等内容是否需要登录（私密、加密隐藏；不含静态直链）。
func FolderRequiresLogin(major Category, sub Subcategory) bool {
	if !BoolOrPublicTrue(major.Public) {
		return true
	}
	if !BoolOrPublicTrue(sub.Public) {
		return true
	}
	return false
}

// FolderResourceRequiresViewKey 直接访问 /resource/ 链接是否需要 ?k= 查看密钥（加密公开、加密隐藏）。
func FolderResourceRequiresViewKey(major Category, sub Subcategory) bool {
	return FolderRequiresEncryptedPassword(major, sub)
}

// FolderRequiresEncryptedPassword 加密公开 / 加密隐藏：静态资源与内容需查看密钥。
func FolderRequiresEncryptedPassword(major Category, sub Subcategory) bool {
	if major.Encrypted || sub.Encrypted {
		return true
	}
	// 兼容旧数据：二级已写入密码哈希但未持久化 encrypted 字段
	return strings.TrimSpace(sub.EncryptedPasswordHash) != ""
}

// FolderEncryptedPasswordHash 返回该目录生效的查看密码哈希（子级优先）。
func FolderEncryptedPasswordHash(major Category, sub Subcategory) string {
	if h := subLevelPasswordHash(sub); h != "" {
		return h
	}
	if major.Encrypted {
		return strings.TrimSpace(major.EncryptedPasswordHash)
	}
	return ""
}

func subLevelPasswordHash(sub Subcategory) string {
	h := strings.TrimSpace(sub.EncryptedPasswordHash)
	if h == "" {
		return ""
	}
	if sub.Encrypted {
		return h
	}
	// 兼容旧数据：仅有哈希、缺少 encrypted:true
	return h
}
