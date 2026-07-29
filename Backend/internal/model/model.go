package model

type Layout struct {
	Mode     string `json:"mode"`    // masonry | grid | card
	Columns  string `json:"columns"` // auto | 1 | 2 | ... | 6
	PageSize int    `json:"pageSize,omitempty"`
}

// Subcategory 二级分类（对应 resource/{folderKey}）
type Subcategory struct {
	ID         int    `json:"id"`
	Sort       int    `json:"sort,omitempty"` // 组内顺序，升序；0 表示未指定（排靠后）
	Name       string `json:"name"`
	FolderKey  string `json:"folderKey"`
	Layout     Layout `json:"layout"`
	ItemSortBy string `json:"itemSortBy,omitempty"` // 资源默认排序: uploaded_at | updated_at | sort
	// Public 为 nil 或 true 表示公开；false 表示私有（需登录后才能在导航与接口中访问）
	Public *bool `json:"public,omitempty"`
	// Encrypted 为 true 时表示该目录需要查看密码。
	Encrypted bool `json:"encrypted,omitempty"`
	// EncryptedPasswordHash 为查看密码的 SHA256 十六进制小写。
	EncryptedPasswordHash string `json:"encryptedPasswordHash,omitempty"`
	// APIEnabled indicates the public API is enabled for this gallery.
	APIEnabled bool   `json:"apiEnabled,omitempty"`
	// APIKeyHash is the SHA256 hex-lowercase hash of the public API key.
	APIKeyHash string `json:"apiKeyHash,omitempty"`
}

// Category 大分类（其下为 Subcategory）
type Category struct {
	ID                    int           `json:"id"`
	Sort                  int           `json:"sort,omitempty"` // 大分类顺序，升序；0 表示未指定（排靠后）
	Name                  string        `json:"name"`
	Key                   string        `json:"key,omitempty"`    // 大分类路径键（如 liulan、tuku）
	Public                *bool         `json:"public,omitempty"` // nil/true 公开；false 私有（整组需登录）
	Encrypted             bool          `json:"encrypted,omitempty"`
	EncryptedPasswordHash string        `json:"encryptedPasswordHash,omitempty"`
	Subcategories         []Subcategory `json:"subcategories"`
}

type CategoriesDoc struct {
	Version       int        `json:"version"`
	HomeFolderKey string     `json:"homeFolderKey,omitempty"`
	Categories    []Category `json:"categories"`
}

type Item struct {
	ID         string `json:"id"`
	Sort       int    `json:"sort,omitempty"` // 自定义排序，升序；0 表示未指定（排靠后，再按 filename）
	MasonryCol int    `json:"masonryCol,omitempty"`
	MasonryRow int    `json:"masonryRow,omitempty"`
	UploadedAt string `json:"uploadedAt,omitempty"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
	Filename   string `json:"filename"`
	// FileSize 为主资源字节数（入库时写入；旧数据可能为 0）。
	FileSize int64 `json:"fileSize,omitempty"`
	// LinkName 是可选的自定义短链文件名（带扩展名），用于输出更友好的资源 URL。
	LinkName string `json:"linkName,omitempty"`
	// Thumbnail 为缩略图文件名（相对 folderKey）：thumb/{与 original 同主干}.jpg（PNG 原图为 .png）；编码与扩展名一致。
	Thumbnail string `json:"thumbnail,omitempty"`
	// GroupID 用于关联一组资源（如 JPG + ARW）。
	GroupID string `json:"groupId,omitempty"`
	// RawFilename 为组内原始文件（可选，如 .ARW），用于下载。
	RawFilename string   `json:"rawFilename,omitempty"`
	Title       string   `json:"title,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	// EditedFilename 为非破坏性编辑产物（如 edited/{id}.jpg），原文件保留在 Filename。
	EditedFilename string `json:"editedFilename,omitempty"`
	// UseEdited 为 true 时列表/默认 URL 使用编辑版，缩略图亦来自编辑版。
	UseEdited bool `json:"useEdited,omitempty"`
}

type ItemsDoc struct {
	Items []Item `json:"items"`
}

// TrashEntry 回收站条目（软删除后保留原资源文件）。
type TrashEntry struct {
	FolderKey string `json:"folderKey"`
	MajorName string `json:"majorName"`
	SubName   string `json:"subName"`
	MajorKey  string `json:"majorKey,omitempty"`
	MajorID   int    `json:"majorId,omitempty"`
	SubID     int    `json:"subId,omitempty"`
	SubSort   int    `json:"subSort,omitempty"`
	Layout    Layout `json:"layout,omitempty"`
	ItemSortBy string `json:"itemSortBy,omitempty"`
	Public    *bool  `json:"public,omitempty"`
	Encrypted bool   `json:"encrypted,omitempty"`
	EncryptedPasswordHash string `json:"encryptedPasswordHash,omitempty"`
	DeletedAt string `json:"deletedAt"`
	Item      Item   `json:"item"`
}

type TrashDoc struct {
	Items []TrashEntry `json:"items"`
}

type TrashItemDTO struct {
	ItemDTO
	FolderKey string `json:"folderKey"`
	MajorName string `json:"majorName"`
	SubName   string `json:"subName"`
	DeletedAt string `json:"deletedAt"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	// CategoryMissing 为 true 表示原画廊/导航已从 categories.json 移除，恢复时会自动重建。
	CategoryMissing bool `json:"categoryMissing,omitempty"`
}

type ItemDTO struct {
	ID           string   `json:"id"`
	Sort         int      `json:"sort,omitempty"`
	MasonryCol   *int     `json:"masonryCol,omitempty"`
	MasonryRow   *int     `json:"masonryRow,omitempty"`
	UploadedAt   string   `json:"uploadedAt,omitempty"`
	UpdatedAt    string   `json:"updatedAt,omitempty"`
	FileSize     int64    `json:"fileSize,omitempty"`
	ShortURL     string   `json:"shortUrl,omitempty"` // 基于 linkName 的较短 URL（与 url 指向同一原图时可与 url 相同）
	LinkName     string   `json:"linkName,omitempty"`
	URL          string   `json:"url"` // 基于 filename 的规范路径（含 original/…）
	ThumbnailURL string   `json:"thumbnailUrl,omitempty"`
	GroupID      string   `json:"groupId,omitempty"`
	RawURL       string   `json:"rawUrl,omitempty"`
	Title        string   `json:"title,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	OriginalURL  string   `json:"originalUrl,omitempty"`
	EditedURL    string   `json:"editedUrl,omitempty"`
	UseEdited    bool     `json:"useEdited,omitempty"`
	Format       string   `json:"format,omitempty"`
	MediaKind    string   `json:"mediaKind,omitempty"`
	// IsLivePhoto 表示 Apple 式实况图（静图 + MOV 伴生视频）。
	IsLivePhoto bool `json:"isLivePhoto,omitempty"`
	// LiveVideoURL 为实况图伴生视频的访问 URL（与 rawUrl 在实况场景下相同）。
	LiveVideoURL string `json:"liveVideoUrl,omitempty"`
}

type CategoryDetailResponse struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	FolderKey  string    `json:"folderKey"`
	Layout     Layout    `json:"layout"`
	ItemSortBy string    `json:"itemSortBy,omitempty"`
	Items      []ItemDTO `json:"items"`
}

type PatchLayoutBody struct {
	Layout Layout `json:"layout"`
}

type PatchItemOrderBody struct {
	OrderedItemIDs   []string                   `json:"orderedItemIds"`
	MasonryPlacement map[string]MasonryPosition `json:"masonryPlacement,omitempty"`
}

type PatchItemBody struct {
	LinkName     string   `json:"linkName,omitempty"`
	Title        string   `json:"title,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	RevertEdited bool     `json:"revertEdited,omitempty"`
}

type TransferItemBody struct {
	TargetFolderKey string `json:"targetFolderKey"`
}

type MasonryPosition struct {
	Col int `json:"col"`
	Row int `json:"row"`
}
