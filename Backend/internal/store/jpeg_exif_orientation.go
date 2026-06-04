package store

// JPEG EXIF Orientation：Go 解码不应用方向元数据；浏览器会根据 EXIF 显示竖图，
// 若缩略生成不重排像素，列表会仍为「横构图」。
// 解析与映射参考 disintegration/imageorient（gift 变换）。

import (
	"bytes"
	"encoding/binary"
	"image"
	"io"
	"path/filepath"
	"strings"
)

const (
	jpegMarkerSOI  = uint16(0xffd8)
	jpegMarkerAPP1 = uint16(0xffe1)
	exifHeader     = uint32(0x45786966) // "Exif"
	beTag          = uint16(0x4d4d)     // "MM"
	leTag          = uint16(0x4949)     // "II"
	tagOrientation = uint16(0x0112)
)

func readJPEGEXIFOrientation(jpegPayload []byte) int {
	r := bytes.NewReader(jpegPayload)
	var soi uint16
	if err := binary.Read(r, binary.BigEndian, &soi); err != nil {
		return 0
	}
	if soi != jpegMarkerSOI {
		return 0
	}
	for {
		var marker, size uint16
		if err := binary.Read(r, binary.BigEndian, &marker); err != nil {
			return 0
		}
		if err := binary.Read(r, binary.BigEndian, &size); err != nil {
			return 0
		}
		if marker>>8 != 0xff {
			return 0
		}
		if marker == jpegMarkerAPP1 {
			break
		}
		if size < 2 {
			return 0
		}
		if _, err := io.CopyN(io.Discard, r, int64(size-2)); err != nil {
			return 0
		}
	}

	var hdr uint32
	if err := binary.Read(r, binary.BigEndian, &hdr); err != nil {
		return 0
	}
	if hdr != exifHeader {
		return 0
	}
	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return 0
	}

	var boTag uint16
	if err := binary.Read(r, binary.BigEndian, &boTag); err != nil {
		return 0
	}
	var order binary.ByteOrder
	switch boTag {
	case beTag:
		order = binary.BigEndian
	case leTag:
		order = binary.LittleEndian
	default:
		return 0
	}
	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return 0
	}

	var offset uint32
	if err := binary.Read(r, order, &offset); err != nil {
		return 0
	}
	if offset < 8 {
		return 0
	}
	if _, err := io.CopyN(io.Discard, r, int64(offset-8)); err != nil {
		return 0
	}

	var nTags uint16
	if err := binary.Read(r, order, &nTags); err != nil {
		return 0
	}
	for range int(nTags) {
		var tag uint16
		if err := binary.Read(r, order, &tag); err != nil {
			return 0
		}
		if tag != tagOrientation {
			if _, err := io.CopyN(io.Discard, r, 10); err != nil {
				return 0
			}
			continue
		}
		if _, err := io.CopyN(io.Discard, r, 6); err != nil {
			return 0
		}
		var v uint16
		if err := binary.Read(r, order, &v); err != nil {
			return 0
		}
		if v < 1 || v > 8 {
			return 0
		}
		return int(v)
	}
	return 0
}

func rasterToRGBAUpLeft(src image.Image) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.Set(x, y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

// gift/transformFilter.Draw 映射：与 EXIF 修复表一致。
func applyEXIFOrientationToImage(srcImage image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return srcImage
	}

	src := rasterToRGBAUpLeft(srcImage)
	srcb := src.Bounds()

	dstb := image.Rect(0, 0, srcb.Dx(), srcb.Dy())
	if orientation >= 5 && orientation <= 8 {
		dstb = image.Rect(0, 0, srcb.Dy(), srcb.Dx())
	}
	dst := image.NewRGBA(dstb)

	for sy := srcb.Min.Y; sy < srcb.Max.Y; sy++ {
		for sx := srcb.Min.X; sx < srcb.Max.X; sx++ {
			var dx, dy int
			switch orientation {
			case 2:
				dx = dstb.Min.X + srcb.Max.X - sx - 1
				dy = dstb.Min.Y + sy - srcb.Min.Y
			case 3:
				dx = dstb.Min.X + srcb.Max.X - sx - 1
				dy = dstb.Min.Y + srcb.Max.Y - sy - 1
			case 4:
				dx = dstb.Min.X + sx - srcb.Min.X
				dy = dstb.Min.Y + srcb.Max.Y - sy - 1
			case 5:
				dx = dstb.Min.X + sy - srcb.Min.Y
				dy = dstb.Min.Y + sx - srcb.Min.X
			case 6:
				// EXIF 6 / 8 与部分设备组合时易与「另一旋转方向」混淆；与 8 对调后可消除竖图上下颠倒。
				dx = dstb.Min.X + srcb.Max.Y - sy - 1
				dy = dstb.Min.Y + sx - srcb.Min.X
			case 7:
				dx = dstb.Min.X + srcb.Max.Y - sy - 1
				dy = dstb.Min.Y + srcb.Max.X - sx - 1
			case 8:
				dx = dstb.Min.X + sy - srcb.Min.Y
				dy = dstb.Min.Y + srcb.Max.X - sx - 1
			}
			dst.Set(dx, dy, src.RGBAAt(sx, sy))
		}
	}
	return dst
}

// decodeGalleryRasterForThumbnail 解码栅格并应用 JPEG EXIF 方向摆正像素。
func decodeGalleryRasterForThumbnail(raw []byte, originalFilenameOrRel string) (image.Image, bool) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, false
	}
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(originalFilenameOrRel)))
	if ext == ".jpg" || ext == ".jpeg" {
		if ori := readJPEGEXIFOrientation(raw); ori > 1 {
			img = applyEXIFOrientationToImage(img, ori)
		}
	}
	return img, true
}
