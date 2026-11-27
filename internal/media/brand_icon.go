package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// prioritizedBrandIconPaths lists relative paths within a project directory in priority order.
var prioritizedBrandIconPaths = []string{
	"build/appicon.png",
	"build/windows/icon.png",
	"build/linux/icon.png",
	"public/apple-touch-icon.png",
	"public/favicon/apple-touch-icon.png",
	"web/public/icon-512.png",
	"web/public/icon-192.png",
	"web/public/apple-touch-icon.png",
	"frontend/public/apple-touch-icon.png",
	"frontend/public/favicon-256.png",
	"frontend/public/favicon-192.png",
	"frontend/public/favicon-180.png",
	"frontend/assest/favicon/favicon-256.png",
	"frontend/assest/favicon/favicon-192.png",
	"frontend/assest/favicon/apple-touch-icon.png",
	"assets/icon.png",
	"extension/icons/icon128.png",
	"icons/icon128.png",
	"public/favicon/favicon-96x96.png",
	"build/windows/icon.ico",
	"build/icon.ico",
	"frontend/public/favicon.ico",
	"web/public/favicon.ico",
	"public/favicon/favicon.ico",
	"public/favicon.ico",
}

// subAppBrandIconPatterns lists nested app relative paths checked inside immediate child directories
// (e.g. apps/desktop-station/build/appicon.png, noxuscloak-desktop/build/appicon.png).
var subAppBrandIconPatterns = []string{
	"build/appicon.png",
	"build/windows/icon.png",
	"build/windows/icon.ico",
	"build/icon.ico",
	"public/apple-touch-icon.png",
	"public/icon-512.png",
	"public/icon-192.png",
	"icons/icon128.png",
}

var ignoredIconSearchDirs = map[string]struct{}{
	"node_modules": {},
	".git":         {},
	"dist":         {},
	"target":       {},
	"vendor":       {},
	".venv":        {},
	"venv":         {},
	"__pycache__":  {},
	"docs":         {},
	"templates":    {},
	"plugins":      {},
	"data":         {},
	"logs":         {},
	".runtime":     {},
	".agents":      {},
	".gemini":      {},
	".claude":      {},
	".cursor":      {},
}

// HasProjectBrandLogo reports whether localDir contains a valid square brand icon / icon suite.
func HasProjectBrandLogo(localDir string) bool {
	return DiscoverProjectBrandLogo(localDir) != nil
}

// DiscoverProjectBrandLogo searches localDir for a real square application icon (.png or .ico >= 48x48).
// Returns nil if the project does not have a brand icon suite so non-branded projects remain untouched.
func DiscoverProjectBrandLogo(localDir string) image.Image {
	cleaned := strings.TrimSpace(localDir)
	if cleaned == "" {
		return nil
	}

	// 1. Check direct prioritized paths.
	for _, rel := range prioritizedBrandIconPaths {
		full := filepath.Join(cleaned, filepath.FromSlash(rel))
		if img := loadValidBrandIconFile(full); img != nil {
			return img
		}
	}

	// 2. Check 1-2 levels of non-ignored subdirectories (e.g. apps/desktop-station/build/appicon.png).
	entries, err := os.ReadDir(cleaned)
	if err != nil {
		return nil
	}
	for _, e1 := range entries {
		if !e1.IsDir() || shouldSkipIconDir(e1.Name()) {
			continue
		}
		d1 := filepath.Join(cleaned, e1.Name())
		for _, pat := range subAppBrandIconPatterns {
			if img := loadValidBrandIconFile(filepath.Join(d1, filepath.FromSlash(pat))); img != nil {
				return img
			}
		}
		subEntries, err := os.ReadDir(d1)
		if err != nil {
			continue
		}
		for _, e2 := range subEntries {
			if !e2.IsDir() || shouldSkipIconDir(e2.Name()) {
				continue
			}
			d2 := filepath.Join(d1, e2.Name())
			for _, pat := range subAppBrandIconPatterns {
				if img := loadValidBrandIconFile(filepath.Join(d2, filepath.FromSlash(pat))); img != nil {
					return img
				}
			}
		}
	}
	return nil
}

func shouldSkipIconDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	_, skip := ignoredIconSearchDirs[strings.ToLower(name)]
	return skip
}

func loadValidBrandIconFile(path string) image.Image {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() < 128 || info.Size() > 8*1024*1024 {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var img image.Image
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".png"):
		img, err = png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil
		}
	case strings.HasSuffix(lower, ".ico"):
		img, err = decodeLargestICOFrame(data)
		if err != nil {
			return nil
		}
	default:
		return nil
	}

	if !isValidSquareBrandIcon(img) {
		return nil
	}
	return img
}

func isValidSquareBrandIcon(img image.Image) bool {
	if img == nil {
		return false
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 48 || h < 48 || w > 2048 || h > 2048 {
		return false
	}
	ratio := float64(w) / float64(h)
	if ratio < 0.78 || ratio > 1.28 {
		return false
	}
	return true
}

// decodeLargestICOFrame parses a Windows .ico container and decodes its largest frame (>= 48x48).
func decodeLargestICOFrame(data []byte) (image.Image, error) {
	if len(data) < 6 {
		return nil, errors.New("ico too short")
	}
	reserved := binary.LittleEndian.Uint16(data[0:2])
	icoType := binary.LittleEndian.Uint16(data[2:4])
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if reserved != 0 || icoType != 1 || count <= 0 || len(data) < 6+count*16 {
		return nil, errors.New("invalid ico header")
	}

	bestSize := 0
	var bestImg image.Image

	for i := 0; i < count; i++ {
		entry := data[6+i*16 : 6+(i+1)*16]
		w := int(entry[0])
		if w == 0 {
			w = 256
		}
		h := int(entry[1])
		if h == 0 {
			h = 256
		}
		bytesInRes := int(binary.LittleEndian.Uint32(entry[8:12]))
		offset := int(binary.LittleEndian.Uint32(entry[12:16]))
		if bytesInRes <= 0 || offset < 0 || offset+bytesInRes > len(data) {
			continue
		}
		frame := data[offset : offset+bytesInRes]

		var decoded image.Image
		if len(frame) >= len(pngMagic) && bytes.Equal(frame[:len(pngMagic)], pngMagic) {
			img, err := png.Decode(bytes.NewReader(frame))
			if err == nil {
				decoded = img
			}
		} else if img, err := decodeICOBMPDIB(frame, w, h); err == nil {
			decoded = img
		}
		if decoded == nil {
			continue
		}
		score := decoded.Bounds().Dx() * decoded.Bounds().Dy()
		if score > bestSize {
			bestSize = score
			bestImg = decoded
		}
	}
	if bestImg == nil {
		return nil, errors.New("no decodable ico frame")
	}
	return bestImg, nil
}

// decodeICOBMPDIB decodes a 32-bit BGRA BITMAPINFOHEADER DIB frame inside an .ico file.
func decodeICOBMPDIB(frame []byte, fallbackW, fallbackH int) (image.Image, error) {
	if len(frame) < 40 {
		return nil, errors.New("dib header too short")
	}
	headerSize := int(binary.LittleEndian.Uint32(frame[0:4]))
	if headerSize < 40 || len(frame) < headerSize {
		return nil, errors.New("invalid dib header size")
	}
	w := int(int32(binary.LittleEndian.Uint32(frame[4:8])))
	hTotal := int(int32(binary.LittleEndian.Uint32(frame[8:12])))
	bpp := int(binary.LittleEndian.Uint16(frame[14:16]))
	compression := binary.LittleEndian.Uint32(frame[16:20])
	if w <= 0 {
		w = fallbackW
	}
	h := hTotal / 2
	if h <= 0 {
		h = fallbackH
	}
	if bpp != 32 || compression != 0 || w <= 0 || h <= 0 {
		return nil, errors.New("unsupported ico dib format")
	}
	need := headerSize + w*h*4
	if len(frame) < need {
		return nil, errors.New("dib pixel data truncated")
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	pix := frame[headerSize:]
	for y := 0; y < h; y++ {
		srcY := h - 1 - y
		rowOff := srcY * w * 4
		for x := 0; x < w; x++ {
			p := rowOff + x*4
			out.SetNRGBA(x, y, color.NRGBA{
				R: pix[p+2],
				G: pix[p+1],
				B: pix[p+0],
				A: pix[p+3],
			})
		}
	}
	return out, nil
}

// EncodeBrandIconPNG resizes src to size x size and encodes it as a standalone PNG.
func EncodeBrandIconPNG(src image.Image, size int) ([]byte, error) {
	if src == nil {
		return nil, errors.New("nil brand icon")
	}
	resized := ResizeIconNRGBA(src, size, size)
	var buf bytes.Buffer
	if err := png.Encode(&buf, resized); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ResizeIconNRGBA performs high-quality area-averaged / bilinear resampling into targetW x targetH.
func ResizeIconNRGBA(src image.Image, targetW, targetH int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, targetW, targetH))
	if src == nil || targetW <= 0 || targetH <= 0 {
		return dst
	}
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 {
		return dst
	}

	scaleX := float64(sw) / float64(targetW)
	scaleY := float64(sh) / float64(targetH)

	for dy := 0; dy < targetH; dy++ {
		sy0 := float64(dy) * scaleY
		sy1 := sy0 + scaleY
		iy0 := int(math.Floor(sy0))
		iy1 := int(math.Ceil(sy1))
		if iy1 <= iy0 {
			iy1 = iy0 + 1
		}

		for dx := 0; dx < targetW; dx++ {
			sx0 := float64(dx) * scaleX
			sx1 := sx0 + scaleX
			ix0 := int(math.Floor(sx0))
			ix1 := int(math.Ceil(sx1))
			if ix1 <= ix0 {
				ix1 = ix0 + 1
			}

			var sumR, sumG, sumB, sumA, totalW float64
			for sy := iy0; sy < iy1; sy++ {
				cy := clampInt(sy, 0, sh-1)
				wy := math.Min(sy1, float64(sy+1)) - math.Max(sy0, float64(sy))
				if wy <= 0 {
					wy = 1
				}
				for sx := ix0; sx < ix1; sx++ {
					cx := clampInt(sx, 0, sw-1)
					wx := math.Min(sx1, float64(sx+1)) - math.Max(sx0, float64(sx))
					if wx <= 0 {
						wx = 1
					}
					w := wx * wy
					r, g, b, a := src.At(sb.Min.X+cx, sb.Min.Y+cy).RGBA()
					af := float64(a) / 65535.0
					// Premultiplied accumulation for clean alpha edges
					sumR += (float64(r) / 257.0) * w
					sumG += (float64(g) / 257.0) * w
					sumB += (float64(b) / 257.0) * w
					sumA += af * 255.0 * w
					totalW += w
				}
			}

			if totalW > 0 {
				pa := sumA / totalW
				if pa > 0.5 {
					invA := 255.0 / pa
					dst.SetNRGBA(dx, dy, color.NRGBA{
						R: uint8(clampFloat(sumR/totalW*invA, 0, 255)),
						G: uint8(clampFloat(sumG/totalW*invA, 0, 255)),
						B: uint8(clampFloat(sumB/totalW*invA, 0, 255)),
						A: uint8(clampFloat(pa, 0, 255)),
					})
				}
			}
		}
	}
	return dst
}

// blitRoundedIconAA composites a resized NRGBA icon onto dst at (x, y, w, h) with rounded-rect anti-aliased clipping.
func blitRoundedIconAA(dst *image.RGBA, icon image.Image, x, y, w, h int, radius float64) {
	if icon == nil || w <= 0 || h <= 0 {
		return
	}
	resized := ResizeIconNRGBA(icon, w, h)
	cx := float64(x) + float64(w)*0.5
	cy := float64(y) + float64(h)*0.5
	halfW := float64(w) * 0.5
	halfH := float64(h) * 0.5
	r := math.Min(radius, math.Min(halfW, halfH))

	for iy := 0; iy < h; iy++ {
		py := y + iy
		for ix := 0; ix < w; ix++ {
			px := x + ix
			d := roundedRectSDF(float64(px)+0.5, float64(py)+0.5, cx, cy, halfW, halfH, r)
			cov := clamp01(0.5 - d)
			if cov <= 0 {
				continue
			}
			c := resized.NRGBAAt(ix, iy)
			alpha := (float64(c.A) / 255.0) * cov
			if alpha > 0 {
				blendPixel(dst, px, py, color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}, alpha)
			}
		}
	}
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

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

