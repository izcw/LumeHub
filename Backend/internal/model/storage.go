package model

// StorageDoc 实例存储配额与用量（data/storage.json）。
type StorageDoc struct {
	Version int `json:"version"`
	// QuotaBytes 允许占用的总字节数，默认 30 GiB。
	QuotaBytes int64 `json:"quotaBytes"`
	// UsedBytes 缓存的已用字节数（由扫描 resource/ 与 upload_sessions/ 得到）。
	UsedBytes int64 `json:"usedBytes"`
	// CalculatedAt 最近一次统计时间（RFC3339 UTC）。
	CalculatedAt string `json:"calculatedAt,omitempty"`
}

// StorageStatus 返回给前端的存储概况。
type StorageStatus struct {
	QuotaBytes     int64   `json:"quotaBytes"`
	UsedBytes      int64   `json:"usedBytes"`
	AvailableBytes int64   `json:"availableBytes"`
	UsedPercent    float64 `json:"usedPercent"`
	CalculatedAt   string  `json:"calculatedAt,omitempty"`
}
