package config

import (
	"os"
	"path/filepath"
)

// Root is the Backend module directory (folder containing go.mod).
func Root() string {
	if v := os.Getenv("LUMEHUB_ROOT"); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func DataDir() string {
	if v := os.Getenv("LUMEHUB_DATA"); v != "" {
		return v
	}
	return filepath.Join(Root(), "data")
}

func WWWDir() string {
	if v := os.Getenv("LUMEHUB_WWW"); v != "" {
		return v
	}
	return filepath.Join(Root(), "www")
}

func ListenAddr() string {
	if v := os.Getenv("LUMEHUB_ADDR"); v != "" {
		return v
	}
	return ":5353"
}

// AuthPassword 设置后启用登录校验；为空则所有目录视为可匿名访问（便于本地开发）。
func AuthPassword() string {
	return os.Getenv("LUMEHUB_PASSWORD")
}
