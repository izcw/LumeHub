package store

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lumehub/internal/model"
)

func nextUploadSort(items []model.Item) int {
	maxSort := 0
	for _, it := range items {
		if it.Sort > maxSort {
			maxSort = it.Sort
		}
	}
	if maxSort <= 0 {
		return 10
	}
	return maxSort + 10
}

func makeUniqueLinkName(seed, ext string, used map[string]struct{}) string {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		seed = "asset"
	}
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		ext = ".bin"
	}
	for i := 1; ; i++ {
		suffix := ""
		if i > 1 {
			suffix = "-" + strconv.Itoa(i)
		}
		name := seed + suffix + ext
		if _, exists := used[strings.ToLower(name)]; exists {
			continue
		}
		return name
	}
}

func randomHex(n int) string {
	if n <= 0 {
		return "000000000000"
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

func newAssetID(now time.Time, used map[string]struct{}) string {
	datePart := now.UTC().Format("20060102")
	for i := 0; i < 100; i++ {
		id := randomHex(6) + "_" + datePart
		if _, exists := used[id]; exists {
			continue
		}
		return id
	}
	return randomHex(8) + "_" + datePart
}

const (
	// 体积不大时跳过重编码（仅当输出路径与 MIME 对齐、且长边未过大时拷贝原字节）。
	thumbNoReencodeMaxBytes int64 = 500 << 10
	// 超过此长边的「小体积」图仍生成缩略，避免高密度小文件产生超大列表图。
	thumbCopyMaxLongEdgePx = 2560

	thumbJpegQualityFine   = 92
	thumbJpegQualityHigh   = 90
	thumbJpegQualityNormal = 88
)

func originalBaseExtLower(originalRel string) string {
	if originalRel == "" {
		return ""
	}
	base := filepath.Base(strings.TrimSpace(filepath.ToSlash(originalRel)))
	return strings.ToLower(filepath.Ext(base))
}

// thumbnailPayloadCopyAllowed 判断是否可将 original 的文件体原样写入 thumb 路径。
func thumbnailPayloadCopyAllowed(originalRel string, payload []byte, longEdgePx int) bool {
	if ThumbRelForOriginal(originalRel) == "" {
		return false
	}
	if int64(len(payload)) > thumbNoReencodeMaxBytes || int64(len(payload)) == 0 {
		return false
	}
	if longEdgePx > thumbCopyMaxLongEdgePx || longEdgePx <= 0 {
		return false
	}
	ext := originalBaseExtLower(originalRel)
	switch ext {
	case ".jpg", ".jpeg", ".png":
		return true
	default:
		return false
	}
}

func thumbnailLongEdgePx(src image.Image) int {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return 0
	}
	if w > h {
		return w
	}
	return h
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// pickThumbnailMaxEdge 按原图分辨率与体量给出缩略长边像素上限（不放大，仅缩小）。
func pickThumbnailMaxEdge(w, h int, rawBytes int64) int {
	if w <= 0 || h <= 0 {
		return 64
	}
	long := w
	if h > long {
		long = h
	}

	var capEdge int
	mb := float64(rawBytes) / (1024 * 1024)
	switch {
	case long >= 5600 || mb >= 16:
		capEdge = 1568
	case long >= 4000 || mb >= 8:
		capEdge = 1344
	case long >= 2800 || mb >= 3:
		capEdge = 1120
	default:
		// 比旧版 640 更清晰，仍为列表友好的体积
		capEdge = 960
	}
	if capEdge > long {
		capEdge = long
	}
	return clampInt(capEdge, 64, 8192)
}

func pickJPEGQuality(longInPx, thumbLongEdgePx int) int {
	if longInPx <= 0 || thumbLongEdgePx <= 0 || thumbLongEdgePx >= longInPx {
		return thumbJpegQualityFine
	}
	r := float64(thumbLongEdgePx) / float64(longInPx)
	switch {
	case r >= 0.72:
		return thumbJpegQualityFine
	case r >= 0.42:
		return thumbJpegQualityHigh
	default:
		return thumbJpegQualityNormal
	}
}

func lerpRGBA16(a0, a1 uint32, t float64) uint32 {
	if t <= 0 {
		return a0
	}
	if t >= 1 {
		return a1
	}
	return uint32(math.Round(float64(a0)*(1-t) + float64(a1)*t))
}

// resizeThumbnailBilinear 双线性缩放，优于旧版邻近取样，弱化列表锯齿。
func resizeThumbnailBilinear(src image.Image, nw, nh int) *image.RGBA {
	b := src.Bounds()
	sw := b.Dx()
	sh := b.Dy()
	if sw <= 0 || sh <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	nw = clampInt(nw, 1, 16384)
	nh = clampInt(nh, 1, 16384)

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	minX := b.Min.X
	minY := b.Min.Y
	maxX := b.Max.X - 1
	maxY := b.Max.Y - 1

	for iy := range nh {
		sy := float64(minY) + (float64(iy)+0.5)*float64(sh)/float64(nh) - 0.5
		y0 := int(math.Floor(sy))
		yFrac := sy - math.Floor(sy)
		if y0 < minY {
			y0 = minY
		}
		y1 := y0 + 1
		if y1 > maxY {
			y1 = maxY
		}

		for ix := range nw {
			sx := float64(minX) + (float64(ix)+0.5)*float64(sw)/float64(nw) - 0.5
			x0 := int(math.Floor(sx))
			xFrac := sx - math.Floor(sx)
			if x0 < minX {
				x0 = minX
			}
			x1 := x0 + 1
			if x1 > maxX {
				x1 = maxX
			}

			r00, g00, b00, a00 := src.At(x0, y0).RGBA()
			r10, g10, b10, a10 := src.At(x1, y0).RGBA()
			r01, g01, b01, a01 := src.At(x0, y1).RGBA()
			r11, g11, b11, a11 := src.At(x1, y1).RGBA()

			r0 := lerpRGBA16(r00, r10, xFrac)
			g0 := lerpRGBA16(g00, g10, xFrac)
			b0 := lerpRGBA16(b00, b10, xFrac)
			a0 := lerpRGBA16(a00, a10, xFrac)
			r1 := lerpRGBA16(r01, r11, xFrac)
			g1 := lerpRGBA16(g01, g11, xFrac)
			b1 := lerpRGBA16(b01, b11, xFrac)
			a1 := lerpRGBA16(a01, a11, xFrac)

			r := lerpRGBA16(r0, r1, yFrac)
			g := lerpRGBA16(g0, g1, yFrac)
			bl := lerpRGBA16(b0, b1, yFrac)
			al := lerpRGBA16(a0, a1, yFrac)

			dst.Set(ix, iy, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(bl >> 8),
				A: uint8(al >> 8),
			})
		}
	}
	return dst
}

func writeRasterThumbnail(dstPath string, src image.Image, maxEdge int, jpegQuality int) error {
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return os.ErrInvalid
	}
	if maxEdge < 64 {
		maxEdge = 64
	}
	if jpegQuality < 75 {
		jpegQuality = 75
	}
	if jpegQuality > 95 {
		jpegQuality = 95
	}
	nw, nh := w, h
	if w >= h {
		if w > maxEdge {
			nw = maxEdge
			nh = int(math.Round(float64(h) * (float64(maxEdge) / float64(w))))
		}
	} else if h > maxEdge {
		nh = maxEdge
		nw = int(math.Round(float64(w) * (float64(maxEdge) / float64(h))))
	}
	nw = clampInt(nw, 1, 16384)
	nh = clampInt(nh, 1, 16384)

	var out *image.RGBA
	if nw == w && nh == h {
		// 尺寸未变仍可走一遍 RGBA，便于后续与白底合成
		out = image.NewRGBA(image.Rect(0, 0, nw, nh))
		for oy := range nh {
			for ox := range nw {
				out.Set(ox, oy, src.At(b.Min.X+ox, b.Min.Y+oy))
			}
		}
	} else {
		out = resizeThumbnailBilinear(src, nw, nh)
	}
	// 转为不透明底图，避免透明 PNG 缩略图编码成 JPEG 后出现黑底。
	rgb := image.NewRGBA(out.Bounds())
	stddraw.Draw(rgb, rgb.Bounds(), image.NewUniform(image.White), image.Point{}, stddraw.Src)
	stddraw.Draw(rgb, rgb.Bounds(), out, image.Point{}, stddraw.Over)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	ext := strings.ToLower(filepath.Ext(dstPath))
	if ext == ".png" {
		return png.Encode(f, rgb)
	}
	return jpeg.Encode(f, rgb, &jpeg.Options{Quality: jpegQuality})
}

// writeGalleryThumbnail 生成列表缩略图：小体积 JPG/PNG 可原样拷贝；其余按自适应边长与质量编码。
func writeGalleryThumbnail(originalRelPath string, payload []byte, rawSize int64, img image.Image, thumbAbsPath string) error {
	origSlash := filepath.ToSlash(strings.TrimSpace(originalRelPath))
	longPx := thumbnailLongEdgePx(img)
	if thumbnailPayloadCopyAllowed(origSlash, payload, longPx) {
		return os.WriteFile(thumbAbsPath, payload, 0o644)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	maxEdge := pickThumbnailMaxEdge(w, h, rawSize)
	q := pickJPEGQuality(longPx, maxEdge)
	return writeRasterThumbnail(thumbAbsPath, img, maxEdge, q)
}

const maxInMemoryImageForThumb = 40 << 20 // 缩略图解码：避免超大图整文件读入内存

func (s *Store) SaveUploadedFile(folderKey string, originalFilename string, src io.Reader) (model.Item, error) {
	return s.SaveUploadedFileWithRaw(folderKey, originalFilename, src, "", nil)
}

func (s *Store) SaveUploadedFileWithRaw(
	folderKey string,
	originalFilename string,
	src io.Reader,
	rawFilename string,
	raw io.Reader,
) (model.Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return model.Item{}, err
	}
	if _, _, ok := lookupInDoc(doc, folderKey); !ok {
		return model.Item{}, os.ErrNotExist
	}

	items, err := s.readItemsUnlocked(folderKey)
	if err != nil {
		return model.Item{}, err
	}

	resourceDir := filepath.Join(s.dataDir, "resource", folderKey)
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		return model.Item{}, err
	}
	originalDir := filepath.Join(resourceDir, "original")
	if err := os.MkdirAll(originalDir, 0o755); err != nil {
		return model.Item{}, err
	}

	originalBase := filepath.Base(strings.TrimSpace(originalFilename))
	ext := strings.ToLower(filepath.Ext(originalBase))
	if ext == "" {
		ext = ".bin"
	}
	usedIDs := make(map[string]struct{}, len(items))
	usedFilenames := make(map[string]struct{}, len(items))
	usedLinkNames := make(map[string]struct{}, len(items))
	for _, it := range items {
		if it.ID != "" {
			usedIDs[it.ID] = struct{}{}
		}
		if it.Filename != "" {
			usedFilenames[it.Filename] = struct{}{}
		}
		if it.LinkName != "" {
			usedLinkNames[strings.ToLower(it.LinkName)] = struct{}{}
		}
	}

	nowTime := time.Now().UTC()
	id := newAssetID(nowTime, usedIDs)
	baseName := id + ext
	filename := filepath.ToSlash(filepath.Join("original", baseName))
	if _, exists := usedFilenames[filename]; exists {
		for i := 2; ; i++ {
			candidate := filepath.ToSlash(filepath.Join("original", id+"-"+strconv.Itoa(i)+ext))
			if _, exists := usedFilenames[candidate]; exists {
				continue
			}
			filename = candidate
			baseName = strings.TrimPrefix(candidate, "original/")
			break
		}
	}

	dstPath := filepath.Join(resourceDir, filename)
	data, err := io.ReadAll(src)
	if err != nil {
		return model.Item{}, err
	}
	var incoming int64 = int64(len(data))
	if raw != nil {
		rawPeek, err := io.ReadAll(raw)
		if err != nil {
			return model.Item{}, err
		}
		incoming += int64(len(rawPeek))
		raw = bytes.NewReader(rawPeek)
	}
	if err := s.checkStorageQuotaUnlocked(incoming); err != nil {
		return model.Item{}, err
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return model.Item{}, err
	}
	rawStoredPath := ""
	if raw != nil {
		rawExt := strings.ToLower(filepath.Ext(strings.TrimSpace(rawFilename)))
		if rawExt == "" {
			rawExt = ".bin"
		}
		rawStoredPath = filepath.ToSlash(filepath.Join("original", id+"_raw"+rawExt))
		rawData, err := io.ReadAll(raw)
		if err != nil {
			_ = os.Remove(dstPath)
			return model.Item{}, err
		}
		if err := os.WriteFile(filepath.Join(resourceDir, rawStoredPath), rawData, 0o644); err != nil {
			_ = os.Remove(dstPath)
			return model.Item{}, err
		}
	}

	now := nowTime.Format(time.RFC3339)
	tag := strings.TrimPrefix(ext, ".")
	linkName := makeUniqueLinkName(id, ext, usedLinkNames)
	thumbnail := GenerateGalleryThumbnail(resourceDir, filename, dstPath, data, int64(len(data)))
	tags := []string{tag}
	if rawStoredPath != "" && IsLiveVideoCompanionExt(filepath.Ext(rawStoredPath)) {
		tags = append(tags, "live")
	}
	item := model.Item{
		ID:          id,
		Sort:        nextUploadSort(items),
		UploadedAt:  now,
		UpdatedAt:   now,
		Filename:    filename,
		FileSize:    int64(len(data)),
		LinkName:    linkName,
		Thumbnail:   thumbnail,
		RawFilename: rawStoredPath,
		Tags:        tags,
	}
	if rawStoredPath != "" {
		item.GroupID = "grp_" + id
	}
	items = append(items, item)
	if err := s.writeItemsUnlocked(folderKey, items); err != nil {
		_ = os.Remove(dstPath)
		if rawStoredPath != "" {
			_ = os.Remove(filepath.Join(resourceDir, filepath.FromSlash(rawStoredPath)))
		}
		if thumbnail != "" {
			_ = os.Remove(filepath.Join(resourceDir, filepath.FromSlash(thumbnail)))
		}
		return model.Item{}, err
	}
	s.touchStorageUsedAfterMutationUnlocked()
	return item, nil
}

// SaveUploadedFileFromLocalPath 将已落盘的临时文件迁入资源目录并登记 items（用于分片合并后入库）。
// 成功后会删除 localPath。
func (s *Store) SaveUploadedFileFromLocalPath(folderKey string, originalFilename string, localPath string) (model.Item, error) {
	fi, err := os.Stat(localPath)
	if err != nil {
		return model.Item{}, err
	}
	size := fi.Size()
	if size <= 0 {
		return model.Item{}, os.ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readCategoriesUnlocked()
	if err != nil {
		return model.Item{}, err
	}
	if _, _, ok := lookupInDoc(doc, folderKey); !ok {
		return model.Item{}, os.ErrNotExist
	}
	if err := s.checkStorageQuotaUnlocked(size); err != nil {
		return model.Item{}, err
	}

	items, err := s.readItemsUnlocked(folderKey)
	if err != nil {
		return model.Item{}, err
	}

	resourceDir := filepath.Join(s.dataDir, "resource", folderKey)
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		return model.Item{}, err
	}
	originalDir := filepath.Join(resourceDir, "original")
	if err := os.MkdirAll(originalDir, 0o755); err != nil {
		return model.Item{}, err
	}

	originalBase := filepath.Base(strings.TrimSpace(originalFilename))
	ext := strings.ToLower(filepath.Ext(originalBase))
	if ext == "" {
		ext = ".bin"
	}
	usedIDs := make(map[string]struct{}, len(items))
	usedFilenames := make(map[string]struct{}, len(items))
	usedLinkNames := make(map[string]struct{}, len(items))
	for _, it := range items {
		if it.ID != "" {
			usedIDs[it.ID] = struct{}{}
		}
		if it.Filename != "" {
			usedFilenames[it.Filename] = struct{}{}
		}
		if it.LinkName != "" {
			usedLinkNames[strings.ToLower(it.LinkName)] = struct{}{}
		}
	}

	nowTime := time.Now().UTC()
	id := newAssetID(nowTime, usedIDs)
	baseName := id + ext
	filename := filepath.ToSlash(filepath.Join("original", baseName))
	if _, exists := usedFilenames[filename]; exists {
		for i := 2; ; i++ {
			candidate := filepath.ToSlash(filepath.Join("original", id+"-"+strconv.Itoa(i)+ext))
			if _, exists := usedFilenames[candidate]; exists {
				continue
			}
			filename = candidate
			baseName = strings.TrimPrefix(candidate, "original/")
			break
		}
	}

	dstPath := filepath.Join(resourceDir, filename)
	in, err := os.Open(localPath)
	if err != nil {
		return model.Item{}, err
	}
	defer in.Close()

	out, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return model.Item{}, err
	}
	written, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dstPath)
		return model.Item{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dstPath)
		return model.Item{}, closeErr
	}
	if written != size {
		_ = os.Remove(dstPath)
		return model.Item{}, io.ErrUnexpectedEOF
	}

	var data []byte
	if size <= maxInMemoryImageForThumb {
		data, err = os.ReadFile(dstPath)
		if err != nil {
			_ = os.Remove(dstPath)
			return model.Item{}, err
		}
	}

	now := nowTime.Format(time.RFC3339)
	tag := strings.TrimPrefix(ext, ".")
	linkName := makeUniqueLinkName(id, ext, usedLinkNames)
	thumbnail := GenerateGalleryThumbnail(resourceDir, filename, dstPath, data, written)
	item := model.Item{
		ID:          id,
		Sort:        nextUploadSort(items),
		UploadedAt:  now,
		UpdatedAt:   now,
		Filename:    filename,
		FileSize:    written,
		LinkName:    linkName,
		Thumbnail:   thumbnail,
		RawFilename: "",
		Tags:        []string{tag},
	}
	items = append(items, item)
	if err := s.writeItemsUnlocked(folderKey, items); err != nil {
		_ = os.Remove(dstPath)
		if thumbnail != "" {
			_ = os.Remove(filepath.Join(resourceDir, filepath.FromSlash(thumbnail)))
		}
		return model.Item{}, err
	}

	_ = os.Remove(localPath)
	s.touchStorageUsedAfterMutationUnlocked()
	return item, nil
}

func IsLiveVideoCompanionExt(ext string) bool {
	ext = strings.ToLower(strings.TrimSpace(ext))
	return ext == ".mov" || ext == ".m4v"
}
