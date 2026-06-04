// vidthumb 为 items.json 中缺少缩略图的视频条目补生成封面（纯 Go，仅 H.264 MP4/MOV 可解码）。
//
// 用法: go run ./cmd/vidthumb [folderKey]
// 省略 folderKey 时处理 data/resource 下全部分类。
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"lumehub/internal/config"
	"lumehub/internal/model"
	"lumehub/internal/store"
)

func main() {
	root := filepath.Join(config.DataDir(), "resource")
	targets := []string{}
	if len(os.Args) > 1 {
		targets = append(targets, os.Args[1])
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			log.Fatal(err)
		}
		for _, e := range entries {
			if e.IsDir() {
				targets = append(targets, e.Name())
			}
		}
	}
	for _, folderKey := range targets {
		if err := processFolder(folderKey, filepath.Join(root, folderKey)); err != nil {
			log.Printf("%s: %v", folderKey, err)
		}
	}
}

func processFolder(folderKey, dir string) error {
	itemsPath := filepath.Join(dir, "items.json")
	raw, err := os.ReadFile(itemsPath)
	if err != nil {
		return err
	}
	var doc model.ItemsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	changed := 0
	for i := range doc.Items {
		it := &doc.Items[i]
		if strings.TrimSpace(it.Thumbnail) != "" {
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(it.Thumbnail))); err == nil {
				continue
			}
		}
		rel := strings.TrimSpace(it.Filename)
		if !strings.HasPrefix(filepath.ToSlash(rel), "original/") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(rel))
		if ext != ".mp4" && ext != ".m4v" && ext != ".mov" {
			continue
		}
		src := filepath.Join(dir, filepath.FromSlash(rel))
		st, err := os.Stat(src)
		if err != nil {
			continue
		}
		gen := store.GenerateGalleryThumbnail(dir, rel, src, nil, st.Size())
		if gen == "" {
			fmt.Printf("skip %s/%s (no thumb generated)\n", folderKey, rel)
			continue
		}
		it.Thumbnail = gen
		changed++
		fmt.Printf("ok %s/%s -> %s\n", folderKey, rel, gen)
	}
	if changed == 0 {
		fmt.Println("no changes", folderKey)
		return nil
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(itemsPath, out, 0o644)
}
