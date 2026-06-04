package store

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"github.com/Eyevinn/hi264/pkg/decoder"
	"github.com/Eyevinn/hi264/pkg/frame"
	"github.com/Eyevinn/hi264/pkg/yuv"
	"github.com/Eyevinn/mp4ff/avc"
	"github.com/Eyevinn/mp4ff/mp4"
)

var videoExtSet = map[string]struct{}{
	".mp4":  {},
	".m4v":  {},
	".mov":  {},
}

var videoExtUnsupported = map[string]struct{}{
	".webm": {},
	".mkv":  {},
	".avi":  {},
	".wmv":  {},
	".flv":  {},
}

func isVideoOriginalExt(ext string) bool {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		return false
	}
	_, ok := videoExtSet[ext]
	return ok
}

func isVideoThumbnailDecodableExt(ext string) bool {
	return isVideoOriginalExt(ext)
}

// VideoThumbnailSupported 当前纯 Go 解码器是否可能为该扩展名生成封面（H.264 MP4/MOV）。
func VideoThumbnailSupported(originalRel string) bool {
	ext := originalBaseExtLower(originalRel)
	if _, unsupported := videoExtUnsupported[ext]; unsupported {
		return false
	}
	return isVideoThumbnailDecodableExt(ext)
}

// writeVideoThumbnailFromFile 从 H.264 MP4/MOV 提取首个 IDR 关键帧并写入 JPEG 缩略图。
func writeVideoThumbnailFromFile(videoAbsPath, thumbAbsPath string) error {
	videoAbsPath = strings.TrimSpace(videoAbsPath)
	thumbAbsPath = strings.TrimSpace(thumbAbsPath)
	if videoAbsPath == "" || thumbAbsPath == "" {
		return os.ErrInvalid
	}
	ext := strings.ToLower(filepath.Ext(videoAbsPath))
	if _, unsupported := videoExtUnsupported[ext]; unsupported {
		return errors.New("video format not supported for thumbnail (requires H.264 in MP4/MOV)")
	}
	if !isVideoThumbnailDecodableExt(ext) {
		return errors.New("video extension not supported for thumbnail")
	}
	if err := os.MkdirAll(filepath.Dir(thumbAbsPath), 0o755); err != nil {
		return err
	}

	fr, err := decodeFirstAVCFrameFromMP4(videoAbsPath)
	if err != nil {
		return err
	}

	cs := yuv.BT601
	rng := yuv.LimitedRange
	if fr.ColorDescriptionValid {
		cs = yuv.ColorSpaceFromMatrixCoefficients(fr.MatrixCoefficients)
	}
	if fr.VideoFullRangeFlag {
		rng = yuv.FullRange
	}

	var buf bytes.Buffer
	if err := yuv.WriteJPEGCS(&buf, fr, thumbJpegQualityNormal, cs, rng); err != nil {
		return err
	}
	img, _, err := image.Decode(&buf)
	if err != nil {
		return err
	}
	longPx := fr.Width
	if fr.Height > longPx {
		longPx = fr.Height
	}
	maxEdge := pickThumbnailMaxEdge(fr.Width, fr.Height, int64(buf.Len()))
	return writeRasterThumbnail(
		thumbAbsPath,
		img,
		maxEdge,
		pickJPEGQuality(longPx, maxEdge),
	)
}

// decodeFirstAVCFrameFromMP4 优先解码靠前样本（更接近第一帧），再回退到首个 IDR。
func decodeFirstAVCFrameFromMP4(videoAbsPath string) (*frame.Frame, error) {
	fr, err := decodeFirstSamplesProgressive(videoAbsPath, 8)
	if err == nil {
		return fr, nil
	}
	return decodeFirstIDRFrameFromMP4(videoAbsPath)
}

func decodeFirstSamplesProgressive(videoAbsPath string, maxTry int) (*frame.Frame, error) {
	f, err := os.Open(videoAbsPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	mp4File, err := mp4.DecodeFile(f, mp4.WithDecodeMode(mp4.DecModeLazyMdat))
	if err != nil {
		return nil, err
	}
	if mp4File.Moov == nil {
		return nil, errors.New("no moov box in video")
	}
	var videoTrack *mp4.TrakBox
	for _, trak := range mp4File.Moov.Traks {
		if trak.Mdia != nil && trak.Mdia.Hdlr != nil && trak.Mdia.Hdlr.HandlerType == "vide" {
			videoTrack = trak
			break
		}
	}
	if videoTrack == nil {
		return nil, errors.New("no video track")
	}
	stbl := videoTrack.Mdia.Minf.Stbl
	if stbl.Stsd == nil || stbl.Stsd.AvcX == nil || stbl.Stsd.AvcX.AvcC == nil {
		return nil, errors.New("not an H.264/AVC track")
	}
	spsNALUs := stbl.Stsd.AvcX.AvcC.SPSnalus
	ppsNALUs := stbl.Stsd.AvcX.AvcC.PPSnalus
	dec := decoder.New()
	if videoTrack.GetNrSamples() <= 0 {
		return nil, errors.New("no video samples")
	}
	nr := videoTrack.GetNrSamples()
	if maxTry <= 0 {
		maxTry = 1
	}
	if uint32(maxTry) > nr {
		maxTry = int(nr)
	}
	for sampleNr := uint32(1); sampleNr <= uint32(maxTry); sampleNr++ {
		ranges, err := videoTrack.GetRangesForSampleInterval(sampleNr, sampleNr)
		if err != nil {
			continue
		}
		var sampleData []byte
		for _, dr := range ranges {
			data, err := mp4File.Mdat.ReadData(int64(dr.Offset), int64(dr.Size), f)
			if err != nil {
				sampleData = nil
				break
			}
			sampleData = append(sampleData, data...)
		}
		if len(sampleData) == 0 {
			continue
		}
		fr, err := decodeAVCSample(sampleData, spsNALUs, ppsNALUs, dec)
		if err == nil {
			return fr, nil
		}
	}
	return nil, errors.New("no decodable frame in leading samples")
}

func decodeFirstIDRFrameFromMP4(videoAbsPath string) (*frame.Frame, error) {
	f, err := os.Open(videoAbsPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	mp4File, err := mp4.DecodeFile(f, mp4.WithDecodeMode(mp4.DecModeLazyMdat))
	if err != nil {
		return nil, err
	}
	if mp4File.Moov == nil {
		return nil, errors.New("no moov box in video")
	}

	var videoTrack *mp4.TrakBox
	for _, trak := range mp4File.Moov.Traks {
		if trak.Mdia != nil && trak.Mdia.Hdlr != nil && trak.Mdia.Hdlr.HandlerType == "vide" {
			videoTrack = trak
			break
		}
	}
	if videoTrack == nil {
		return nil, errors.New("no video track")
	}

	stbl := videoTrack.Mdia.Minf.Stbl
	if stbl.Stsd == nil || stbl.Stsd.AvcX == nil || stbl.Stsd.AvcX.AvcC == nil {
		return nil, errors.New("not an H.264/AVC track")
	}
	spsNALUs := stbl.Stsd.AvcX.AvcC.SPSnalus
	ppsNALUs := stbl.Stsd.AvcX.AvcC.PPSnalus

	dec := decoder.New()
	if videoTrack.GetNrSamples() > 0 {
		return decodeFirstIDRProgressive(mp4File, videoTrack, f, spsNALUs, ppsNALUs, dec)
	}
	if len(mp4File.Segments) > 0 {
		return decodeFirstIDRFragmented(mp4File, videoTrack, f, spsNALUs, ppsNALUs, dec)
	}
	return nil, errors.New("no video samples in file")
}

func decodeFirstIDRProgressive(
	mp4File *mp4.File,
	videoTrack *mp4.TrakBox,
	f *os.File,
	spsNALUs, ppsNALUs [][]byte,
	dec *decoder.Decoder,
) (*frame.Frame, error) {
	nrSamples := videoTrack.GetNrSamples()
	stss := videoTrack.Mdia.Minf.Stbl.Stss
	for sampleNr := uint32(1); sampleNr <= nrSamples; sampleNr++ {
		if stss != nil && !stss.IsSyncSample(sampleNr) {
			continue
		}
		ranges, err := videoTrack.GetRangesForSampleInterval(sampleNr, sampleNr)
		if err != nil {
			return nil, err
		}
		var sampleData []byte
		for _, dr := range ranges {
			data, err := mp4File.Mdat.ReadData(int64(dr.Offset), int64(dr.Size), f)
			if err != nil {
				return nil, err
			}
			sampleData = append(sampleData, data...)
		}
		fr, err := decodeAVCSample(sampleData, spsNALUs, ppsNALUs, dec)
		if err == nil {
			return fr, nil
		}
	}
	return nil, errors.New("no decodable IDR frame")
}

func decodeFirstIDRFragmented(
	mp4File *mp4.File,
	videoTrack *mp4.TrakBox,
	f *os.File,
	spsNALUs, ppsNALUs [][]byte,
	dec *decoder.Decoder,
) (*frame.Frame, error) {
	trackID := videoTrack.Tkhd.TrackID
	var trex *mp4.TrexBox
	if mp4File.Moov.Mvex != nil {
		trex, _ = mp4File.Moov.Mvex.GetTrex(trackID)
	}
	for _, seg := range mp4File.Segments {
		for _, frag := range seg.Fragments {
			for _, traf := range frag.Moof.Trafs {
				if traf.Tfhd.TrackID != trackID {
					continue
				}
				for _, trun := range traf.Truns {
					trun.AddSampleDefaultValues(traf.Tfhd, trex)
					baseOffset := frag.Moof.StartPos
					if traf.Tfhd.HasBaseDataOffset() {
						baseOffset = traf.Tfhd.BaseDataOffset
					}
					if trun.HasDataOffset() {
						baseOffset = uint64(int64(baseOffset) + int64(trun.DataOffset))
					}
					sampleOffset := uint64(0)
					for i := uint32(0); i < trun.SampleCount(); i++ {
						sample := trun.Samples[i]
						if !sample.IsSync() {
							sampleOffset += uint64(sample.Size)
							continue
						}
						data := make([]byte, sample.Size)
						if _, err := f.ReadAt(data, int64(baseOffset+sampleOffset)); err != nil {
							return nil, err
						}
						fr, err := decodeAVCSample(data, spsNALUs, ppsNALUs, dec)
						if err == nil {
							return fr, nil
						}
						sampleOffset += uint64(sample.Size)
					}
				}
			}
		}
	}
	return nil, errors.New("no decodable IDR frame in fragmented mp4")
}

func decodeAVCSample(sampleData []byte, spsNALUs, ppsNALUs [][]byte, dec *decoder.Decoder) (*frame.Frame, error) {
	sampleNALUs, err := avc.GetNalusFromSample(sampleData)
	if err != nil {
		return nil, err
	}
	nalus := make([][]byte, 0, len(spsNALUs)+len(ppsNALUs)+len(sampleNALUs))
	nalus = append(nalus, spsNALUs...)
	nalus = append(nalus, ppsNALUs...)
	nalus = append(nalus, sampleNALUs...)
	return dec.DecodeNALUs(nalus)
}

func verifyThumbFile(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Size() <= 0 {
		_ = os.Remove(path)
		return errors.New("empty video thumbnail")
	}
	return nil
}

// GenerateGalleryThumbnail 为 original/ 资源生成 thumb/ 缩略图（图片解码；视频为 H.264 首帧/关键帧，HEVC 等需客户端上传封面）。
func GenerateGalleryThumbnail(
	resourceDir string,
	originalRel string,
	sourceAbsPath string,
	payload []byte,
	payloadSize int64,
) string {
	originalRel = strings.TrimSpace(filepath.ToSlash(originalRel))
	if originalRel == "" {
		return ""
	}
	thumbRel := ThumbRelForOriginal(originalRel)
	if thumbRel == "" {
		return ""
	}
	thumbPath := filepath.Join(resourceDir, filepath.FromSlash(thumbRel))

	if len(payload) > 0 {
		if img, ok := decodeGalleryRasterForThumbnail(payload, originalRel); ok {
			if err := writeGalleryThumbnail(originalRel, payload, payloadSize, img, thumbPath); err == nil {
				return thumbRel
			}
			return ""
		}
	}

	if isVideoOriginalExt(originalBaseExtLower(originalRel)) {
		src := strings.TrimSpace(sourceAbsPath)
		if src == "" {
			return ""
		}
		if err := writeVideoThumbnailFromFile(src, thumbPath); err == nil {
			if err := verifyThumbFile(thumbPath); err == nil {
				return thumbRel
			}
			_ = os.Remove(thumbPath)
		}
	}
	return ""
}
