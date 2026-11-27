package media

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"t-deck/internal/model"
)

const (
	// Output dimensions (16:9 HD, <2.5x card display width to avoid WebView2 bilinear texel skipping).
	thumbW = 960
	thumbH = 540

	// 2x2 Supersampling render canvas (1920x1080 -> box-filtered 2x2 to 960x540 for true 4x SSAA).
	ssScale = 2
	canvasW = thumbW * ssScale // 1920
	canvasH = thumbH * ssScale // 1080
)

type fontBundle struct {
	once     sync.Once
	bold     *opentype.Font
	semibold *opentype.Font
	regular  *opentype.Font
}

var sysFonts fontBundle

func loadSystemFonts() {
	sysFonts.once.Do(func() {
		sysFonts.bold = parseFontFirst(
			`C:\Windows\Fonts\segoeuib.ttf`,
			`C:\Windows\Fonts\arialbd.ttf`,
			`C:\Windows\Fonts\segoeui.ttf`,
		)
		sysFonts.semibold = parseFontFirst(
			`C:\Windows\Fonts\seguisb.ttf`,
			`C:\Windows\Fonts\segoeuib.ttf`,
			`C:\Windows\Fonts\segoeui.ttf`,
		)
		sysFonts.regular = parseFontFirst(
			`C:\Windows\Fonts\segoeui.ttf`,
		)
	})
}

func parseFontFirst(paths ...string) *opentype.Font {
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if f, err := opentype.Parse(data); err == nil {
			return f
		}
	}
	return nil
}

func newFace(otFont *opentype.Font, size float64) font.Face {
	if otFont == nil {
		return basicfont.Face7x13
	}
	face, err := opentype.NewFace(otFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return basicfont.Face7x13
	}
	return face
}

type accentPalette struct {
	primary    color.RGBA
	softBorder color.RGBA
	badgeBg    color.RGBA
}

var domainPalettes = []accentPalette{
	{primary: color.RGBA{16, 185, 129, 255}, softBorder: color.RGBA{16, 140, 100, 255}, badgeBg: color.RGBA{14, 38, 32, 255}},  // Emerald
	{primary: color.RGBA{245, 158, 11, 255}, softBorder: color.RGBA{195, 125, 14, 255}, badgeBg: color.RGBA{42, 28, 14, 255}},  // Amber
	{primary: color.RGBA{56, 189, 248, 255}, softBorder: color.RGBA{40, 145, 195, 255}, badgeBg: color.RGBA{14, 32, 48, 255}},  // Sky Cyan
	{primary: color.RGBA{168, 85, 247, 255}, softBorder: color.RGBA{132, 68, 196, 255}, badgeBg: color.RGBA{32, 20, 48, 255}},  // Violet
	{primary: color.RGBA{244, 63, 94, 255}, softBorder: color.RGBA{190, 50, 76, 255}, badgeBg: color.RGBA{45, 18, 28, 255}},    // Rose
	{primary: color.RGBA{59, 130, 246, 255}, softBorder: color.RGBA{46, 104, 196, 255}, badgeBg: color.RGBA{18, 28, 52, 255}},  // Blue
}

// DefaultSummaryForProject returns an informative Vietnamese description when a project lacks README summary.
func DefaultSummaryForProject(p model.Project) string {
	if s := strings.TrimSpace(html.UnescapeString(p.Summary)); s != "" {
		return s
	}
	lower := strings.ToLower(p.Slug + " " + p.Name)
	switch {
	case strings.Contains(lower, "botvn"):
		return "Nền tảng tự động hóa tác vụ desktop VNBot tích hợp điều phối quy trình và xử lý kịch bản thông minh."
	case strings.Contains(lower, "t-hub-desktop"):
		return "Ứng dụng T-Hub Desktop độc lập tích hợp Core Backend, plugin runtime và giao diện điều khiển tập trung."
	case strings.Contains(lower, "t-hub-licensing"):
		return "Máy chủ cấp phát và xác thực bản quyền phần mềm sử dụng chữ ký số Ed25519 và SQLite."
	case strings.Contains(lower, "noxus"):
		return "Hệ thống quản lý hồ sơ, định tuyến kết nối và bảo vệ danh tính trình duyệt NoxusCloak."
	case strings.Contains(lower, "t-hub"):
		return "Hệ sinh thái T-Hub Cockpit kết nối Core Backend, Python AI Worker, Web Dashboard và Cloudflare R2 Plugin Registry."
	case strings.Contains(lower, "social-hub"):
		return "Trung tâm quản lý đa tài khoản mạng xã hội, lập lịch đăng bài và đồng bộ luồng nội dung."
	case strings.Contains(lower, "create-only-video"):
		return "Pipeline sản xuất video ngắn tự động bằng AI: sinh kịch bản, lồng tiếng VieNeu/ElevenLabs và dựng Remotion."
	case strings.Contains(lower, "muse2api"):
		return "Cổng chuyển đổi API trung gian tự động đồng bộ phiên làm việc và cung cấp chuẩn REST endpoint cục bộ."
	case strings.Contains(lower, "tech-market"):
		return "Hệ thống thương mại điện tử thiết bị công nghệ, quản lý danh mục sản phẩm, giỏ hàng và đơn hàng."
	default:
		if p.Type == "desktop" {
			return fmt.Sprintf("Ứng dụng Desktop %s chạy trực tiếp trên Windows (%s).", html.UnescapeString(p.Name), nonEmptyOr(p.TechStack, "Native Desktop"))
		}
		return fmt.Sprintf("Ứng dụng Web %s khởi chạy cục bộ không cần VPS (%s).", html.UnescapeString(p.Name), nonEmptyOr(p.TechStack, "Local Web Stack"))
	}
}

// RenderProjectThumbnailPNG synthesizes a 2x2 supersampled (1920x1080 -> 960x540) Dark Industrial poster PNG.
func RenderProjectThumbnailPNG(p model.Project) ([]byte, error) {
	return RenderProjectThumbnailPNGWithLogo(p, nil)
}

// RenderProjectThumbnailPNGWithLogo synthesizes a 2x2 supersampled (1920x1080 -> 960x540) poster PNG.
// When brandLogo is non-nil, the project's brand icon is rendered in the left brand header lockup and right hero emblem showcase.
// When brandLogo is nil, the output is 100% identical to the standard schematic poster.
func RenderProjectThumbnailPNGWithLogo(p model.Project, brandLogo image.Image) ([]byte, error) {
	loadSystemFonts()

	hi := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	bgCanvas := color.RGBA{13, 14, 17, 255}    // #0D0E11
	bgSurface := color.RGBA{20, 22, 27, 255}   // #14161B
	bgElevated := color.RGBA{28, 32, 40, 255}  // #1C2028
	borderEdge := color.RGBA{42, 48, 60, 255}  // #2A303C
	textPrimary := color.RGBA{249, 250, 251, 255}
	textBadge := color.RGBA{236, 239, 244, 255}

	fillRect(hi, 0, 0, canvasW, canvasH, bgCanvas)

	domain, palette := classifyDomain(p)

	// Outer poster frame on 1920x1080 canvas (no 1px background grid lines that cause moire).
	fillRect(hi, 36, 36, canvasW-72, canvasH-72, bgSurface)
	strokeRect(hi, 36, 36, canvasW-72, canvasH-72, 4, borderEdge)
	fillRect(hi, 36, 36, canvasW-72, 10, palette.primary)

	// Vertical divider separating poster typography and vector schematic.
	fillRect(hi, 1120, 140, 4, canvasH-240, borderEdge)

	// Use Bold face for ALL text (eyebrow, title, stack pills) so stems are >= 5-10px thick on 1920x1080.
	faceEyebrow := newFace(sysFonts.bold, 50)
	faceBadge := newFace(sysFonts.bold, 46)

	cleanTitle := html.UnescapeString(displayName(p.Name))
	titleSize := 98.0
	titleLineH := 116
	if utf8.RuneCountInString(cleanTitle) > 15 {
		titleSize = 80.0
		titleLineH = 96
	}
	faceTitle := newFace(sysFonts.bold, titleSize)
	defer closeFaces(faceEyebrow, faceBadge, faceTitle)

	// --- Left Poster Column ---
	leftX := 112
	maxLeftW := 968

	eyebrowText := strings.ToUpper(domainDisplayName(domain))
	if brandLogo != nil {
		// Left brand lockup: 120x120 rounded icon frame beside the domain eyebrow & verified brand pill.
		iconBoxX, iconBoxY, iconBoxSize := leftX, 142, 124
		fillRoundedRectAA(hi, iconBoxX, iconBoxY, iconBoxSize, iconBoxSize, 26, bgElevated)
		blitRoundedIconAA(hi, brandLogo, iconBoxX+8, iconBoxY+8, iconBoxSize-16, iconBoxSize-16, 20)
		strokeRoundedRectAA(hi, iconBoxX, iconBoxY, iconBoxSize, iconBoxSize, 26, 4, palette.softBorder)

		drawString(hi, leftX+150, 204, eyebrowText, faceEyebrow, palette.primary)
		drawPillAA(hi, leftX+150, 222, 236, 52, 12, "APP ICON", faceBadge, palette.badgeBg, palette.softBorder, palette.primary)
	} else {
		drawString(hi, leftX, 240, eyebrowText, faceEyebrow, palette.primary)
	}

	// Hero project title (up to 2 lines).
	titleLines := wrapTitleText(cleanTitle, faceTitle, maxLeftW, 2)
	titleY := 378
	if brandLogo != nil {
		titleY = 392
	}
	for _, line := range titleLines {
		drawString(hi, leftX, titleY, line, faceTitle, textPrimary)
		titleY += titleLineH
	}

	// Bold horizontal accent bar below title.
	dividerY := titleY - titleLineH + 54
	if dividerY < 436 {
		dividerY = 436
	}
	fillRoundedRectAA(hi, leftX, dividerY, 124, 10, 5, palette.primary)

	// Large, thick-stemmed stack pills at bottom-left (max 3 pills, height 92px on 1920x1080).
	badges := extractStackBadges(p, 3)
	curX := leftX
	const pillY = 824
	const pillH = 92
	for i, label := range badges {
		textW := font.MeasureString(faceBadge, label).Ceil()
		pillW := textW + 76
		if curX+pillW > 1096 {
			break
		}
		pillBg := bgElevated
		pillBorder := borderEdge
		pillFg := textBadge
		if i == 0 {
			pillBg = palette.badgeBg
			pillBorder = palette.softBorder
			pillFg = palette.primary
		}
		drawPillAA(hi, curX, pillY, pillW, pillH, 16, label, faceBadge, pillBg, pillBorder, pillFg)
		curX += pillW + 22
	}

	// --- Right Column: Pure Vector Schematic (scaled to 1920x1080) ---
	panelX, panelY, panelW, panelH := 1164, 148, 676, 784
	fillRoundedRectAA(hi, panelX, panelY, panelW, panelH, 20, bgCanvas)
	strokeRoundedRectAA(hi, panelX, panelY, panelW, panelH, 20, 4, borderEdge)

	// Window header bar with 3 anti-aliased dots.
	fillRect(hi, panelX+4, panelY+4, panelW-8, 58, bgElevated)
	fillRect(hi, panelX+4, panelY+62, panelW-8, 4, borderEdge)
	fillCircleAA(hi, float64(panelX+36), float64(panelY+33), 10, palette.primary)
	fillCircleAA(hi, float64(panelX+68), float64(panelY+33), 10, borderEdge)
	fillCircleAA(hi, float64(panelX+100), float64(panelY+33), 10, borderEdge)

	drawDomainIllustration(
		hi, domain, p.Slug,
		panelX+36, panelY+96, panelW-72, panelH-132,
		palette, bgSurface, bgElevated, borderEdge,
	)

	if brandLogo != nil {
		// Hero brand emblem showcase centered inside the right preview panel.
		cx := panelX + panelW/2
		cy := panelY + 62 + (panelH-62)/2
		fillCircleAA(hi, float64(cx), float64(cy), 192, palette.badgeBg)
		strokeCircleAA(hi, float64(cx), float64(cy), 192, 4, palette.softBorder)

		cardSize := 288
		cardX := cx - cardSize/2
		cardY := cy - cardSize/2
		fillRoundedRectAA(hi, cardX, cardY, cardSize, cardSize, 52, bgSurface)
		blitRoundedIconAA(hi, brandLogo, cardX+24, cardY+24, cardSize-48, cardSize-48, 38)
		strokeRoundedRectAA(hi, cardX, cardY, cardSize, cardSize, 52, 5, palette.primary)
	}

	// Downsample 2x2 (1920x1080 -> 960x540) with exact 4-pixel box averaging for smooth anti-aliased edges.
	out := downsample2x(hi)

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// downsample2x performs an exact 2x2 area-average reduction from (2W x 2H) to (W x H).
func downsample2x(src *image.RGBA) *image.RGBA {
	sb := src.Bounds()
	dw, dh := sb.Dx()/2, sb.Dy()/2
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	sPix := src.Pix
	dPix := dst.Pix
	sStride := src.Stride
	dStride := dst.Stride

	for y := 0; y < dh; y++ {
		sy0 := (y * 2) * sStride
		sy1 := sy0 + sStride
		dy := y * dStride
		for x := 0; x < dw; x++ {
			sx := x * 8
			i00 := sy0 + sx
			i10 := i00 + 4
			i01 := sy1 + sx
			i11 := i01 + 4

			di := dy + x*4
			dPix[di+0] = uint8((uint16(sPix[i00+0]) + uint16(sPix[i10+0]) + uint16(sPix[i01+0]) + uint16(sPix[i11+0]) + 2) >> 2)
			dPix[di+1] = uint8((uint16(sPix[i00+1]) + uint16(sPix[i10+1]) + uint16(sPix[i01+1]) + uint16(sPix[i11+1]) + 2) >> 2)
			dPix[di+2] = uint8((uint16(sPix[i00+2]) + uint16(sPix[i10+2]) + uint16(sPix[i01+2]) + uint16(sPix[i11+2]) + 2) >> 2)
			dPix[di+3] = 255
		}
	}
	return dst
}

func extractStackBadges(p model.Project, maxCount int) []string {
	if maxCount <= 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	push := func(raw string) {
		if len(out) >= maxCount {
			return
		}
		clean := strings.TrimSpace(strings.TrimPrefix(raw, "#"))
		clean = html.UnescapeString(clean)
		if clean == "" {
			return
		}
		upper := strings.ToUpper(clean)
		if _, dup := seen[upper]; dup {
			return
		}
		seen[upper] = struct{}{}
		out = append(out, truncateRunes(upper, 12))
	}

	replacer := strings.NewReplacer(",", "|", "/", "|", "•", "|", "+", "|", ";", "|")
	for _, tok := range strings.Split(replacer.Replace(p.TechStack), "|") {
		push(tok)
	}
	for _, tag := range p.Tags {
		push(tag)
	}
	if len(out) == 0 {
		switch strings.ToLower(strings.TrimSpace(p.Type)) {
		case "desktop":
			push("DESKTOP")
		case "hybrid":
			push("HYBRID")
		default:
			push("WEB APP")
		}
	}
	return out
}

func classifyDomain(p model.Project) (string, accentPalette) {
	text := strings.ToLower(p.Slug + " " + p.Name + " " + p.Summary + " " + p.TechStack + " " + strings.Join(p.Tags, " "))
	h := fnv.New32a()
	_, _ = h.Write([]byte(p.Slug))
	seedIdx := int(h.Sum32()) % len(domainPalettes)

	switch {
	case strings.Contains(text, "video") || strings.Contains(text, "mvdirector") || strings.Contains(text, "remotion") || strings.Contains(text, "ai-studio") || strings.Contains(text, "station") || strings.Contains(text, "vieneu") || strings.Contains(text, "voice"):
		return "video_ai", domainPalettes[1] // Amber
	case strings.Contains(text, "s3") || strings.Contains(text, "r2") || strings.Contains(text, "storage") || strings.Contains(text, "kho-repo") || strings.Contains(text, "vault"):
		return "storage_cloud", domainPalettes[2] // Sky Cyan
	case strings.Contains(text, "licens") || strings.Contains(text, "noxus") || strings.Contains(text, "cloak") || strings.Contains(text, "security"):
		return "security_license", domainPalettes[4] // Rose
	case strings.Contains(text, "bizmastery") || strings.Contains(text, "market") || strings.Contains(text, "kinh doanh") || strings.Contains(text, "commerce") || strings.Contains(text, "vnpay") || strings.Contains(text, "merchant"):
		return "business_analytics", domainPalettes[0] // Emerald
	case strings.Contains(text, "bot") || strings.Contains(text, "company") || strings.Contains(text, "agent") || strings.Contains(text, "muse2api") || strings.Contains(text, "office"):
		return "automation_bot", domainPalettes[3] // Violet
	default:
		return "cockpit_hub", domainPalettes[(seedIdx+5)%len(domainPalettes)]
	}
}

func domainDisplayName(domain string) string {
	switch domain {
	case "video_ai":
		return "Video & Media AI"
	case "storage_cloud":
		return "Cloud Storage"
	case "security_license":
		return "Security & License"
	case "business_analytics":
		return "Business & Web"
	case "automation_bot":
		return "AI Agent & Bot"
	default:
		return "Platform Hub"
	}
}

func drawDomainIllustration(
	img *image.RGBA,
	domain, slug string,
	x, y, w, h int,
	pal accentPalette,
	surface, elevated, border color.RGBA,
) {
	h32 := fnv.New32a()
	_, _ = h32.Write([]byte(slug))
	seed := int(h32.Sum32())

	switch domain {
	case "video_ai":
		monH := 256
		fillRoundedRectAA(img, x, y, w, monH, 14, surface)
		strokeRoundedRectAA(img, x, y, w, monH, 14, 4, border)
		cx, cy := float64(x+w/2), float64(y+monH/2)
		fillCircleAA(img, cx, cy, 58, pal.badgeBg)
		strokeCircleAA(img, cx, cy, 58, 5, pal.primary)
		drawPlayTriangleAA(img, cx-14, cy-24, 40, 48, pal.primary)

		waveY := y + monH + 28
		fillRoundedRectAA(img, x, waveY, w, 140, 14, surface)
		strokeRoundedRectAA(img, x, waveY, w, 140, 14, 4, border)
		bars := 20
		barW := (w - 48) / bars
		for i := 0; i < bars; i++ {
			bh := 26 + int(math.Abs(math.Sin(float64(i+seed)*0.55))*84)
			bx := x + 24 + i*barW
			by := waveY + 70 - bh/2
			col := pal.primary
			if i%3 == 0 {
				col = border
			}
			fillRoundedRectAA(img, bx, by, max(6, barW-8), bh, 3, col)
		}

		tlY := waveY + 166
		for row := 0; row < 3; row++ {
			ry := tlY + row*64
			fillRoundedRectAA(img, x, ry, w, 48, 10, surface)
			strokeRoundedRectAA(img, x, ry, w, 48, 10, 3, border)
			segX := x + 20 + ((seed + row*57) % 85)
			segW := 200 + ((seed + row*83) % 200)
			if segX+segW > x+w-20 {
				segW = x + w - 20 - segX
			}
			fillRoundedRectAA(img, segX, ry+8, segW, 32, 6, pal.badgeBg)
			strokeRoundedRectAA(img, segX, ry+8, segW, 32, 6, 3, pal.primary)
		}
		phX := x + w/2 + ((seed % 100) - 50)
		fillRect(img, phX, waveY, 5, h-(waveY-y), pal.primary)

	case "storage_cloud":
		for i := 0; i < 2; i++ {
			cw := (w - 24) / 2
			cx := x + i*(cw+24)
			fillRoundedRectAA(img, cx, y, cw, 164, 14, surface)
			strokeRoundedRectAA(img, cx, y, cw, 164, 14, 4, border)
			fillRect(img, cx+22, y+28, cw-44, 12, pal.primary)
			fillRoundedRectAA(img, cx+22, y+68, cw-44, 24, 6, elevated)
			fillRoundedRectAA(img, cx+22, y+68, (cw-44)*(62+i*22)/100, 24, 6, pal.primary)
			fillRect(img, cx+22, y+118, (cw-44)/2, 16, border)
		}

		listY := y + 196
		for row := 0; row < 5; row++ {
			ry := listY + row*88
			fillRoundedRectAA(img, x, ry, w, 68, 10, surface)
			strokeRoundedRectAA(img, x, ry, w, 68, 10, 3, border)
			fillRoundedRectAA(img, x+22, ry+18, 32, 32, 6, pal.badgeBg)
			strokeRoundedRectAA(img, x+22, ry+18, 32, 32, 6, 3, pal.primary)
			barLen := 210 + ((seed + row*61) % 180)
			fillRect(img, x+74, ry+25, barLen, 18, elevated)
			fillRoundedRectAA(img, x+w-112, ry+20, 86, 28, 6, pal.badgeBg)
			strokeRoundedRectAA(img, x+w-112, ry+20, 86, 28, 6, 3, pal.softBorder)
		}

	case "security_license":
		cx, cy := float64(x+w/2), float64(y+188)
		strokeCircleAA(img, cx, cy, 142, 5, border)
		strokeCircleAA(img, cx, cy, 100, 6, pal.primary)
		fillCircleAA(img, cx, cy, 56, pal.badgeBg)
		strokeRoundedRectAA(img, int(cx)-22, int(cy)-18, 44, 36, 8, 5, pal.primary)
		fillCircleAA(img, cx, cy, 8, pal.primary)

		for row := 0; row < 4; row++ {
			ry := y + 374 + row*70
			fillRoundedRectAA(img, x, ry, w, 54, 10, surface)
			strokeRoundedRectAA(img, x, ry, w, 54, 10, 3, border)
			fillCircleAA(img, float64(x+34), float64(ry+27), 9, pal.primary)
			barLen := 240 + ((seed + row*57) % 170)
			fillRect(img, x+64, ry+19, barLen, 16, elevated)
			fillRoundedRectAA(img, x+w-104, ry+15, 76, 24, 6, pal.badgeBg)
		}

	case "business_analytics":
		for i := 0; i < 3; i++ {
			cw := (w - 36) / 3
			cx := x + i*(cw+18)
			fillRoundedRectAA(img, cx, y, cw, 134, 12, surface)
			strokeRoundedRectAA(img, cx, y, cw, 134, 12, 3, border)
			fillRect(img, cx+20, y+24, cw-40, 12, pal.primary)
			fillRect(img, cx+20, y+60, cw-52, 24, elevated)
			fillRect(img, cx+20, y+100, (cw-40)/2, 12, border)
		}

		chartY := y + 162
		chartH := 312
		fillRoundedRectAA(img, x, chartY, w, chartH, 14, surface)
		strokeRoundedRectAA(img, x, chartY, w, chartH, 14, 4, border)
		cols := 8
		colW := (w - 72) / cols
		for i := 0; i < cols; i++ {
			bh := 64 + ((seed + i*59) % 190)
			bx := x + 36 + i*colW
			by := chartY + chartH - 34 - bh
			c := elevated
			if i >= cols-3 {
				c = pal.primary
			}
			fillRoundedRectAA(img, bx, by, colW-14, bh, 5, c)
		}

		for row := 0; row < 2; row++ {
			ry := chartY + chartH + 24 + row*74
			fillRoundedRectAA(img, x, ry, w, 58, 10, surface)
			strokeRoundedRectAA(img, x, ry, w, 58, 10, 3, border)
			fillRoundedRectAA(img, x+22, ry+17, 24, 24, 5, pal.primary)
			fillRect(img, x+68, ry+20, w-128, 18, elevated)
		}

	case "automation_bot":
		nodePositions := [][2]int{
			{x + 28, y + 18},
			{x + w - 226, y + 18},
			{x + w/2 - 99, y + 164},
		}
		fillRect(img, x+126, y+58, w-252, 5, pal.primary)
		fillRect(img, x+w/2-2, y+58, 5, 106, pal.primary)

		for _, pos := range nodePositions {
			fillRoundedRectAA(img, pos[0], pos[1], 198, 86, 12, surface)
			strokeRoundedRectAA(img, pos[0], pos[1], 198, 86, 12, 4, pal.primary)
			fillCircleAA(img, float64(pos[0]+34), float64(pos[1]+43), 11, pal.primary)
			fillRect(img, pos[0]+62, pos[1]+35, 106, 16, elevated)
		}

		termY := y + 286
		termH := h - 286
		fillRoundedRectAA(img, x, termY, w, termH, 14, surface)
		strokeRoundedRectAA(img, x, termY, w, termH, 14, 4, border)
		fillRect(img, x+4, termY+4, w-8, 6, pal.primary)
		for row := 0; row < 5; row++ {
			ry := termY + 40 + row*60
			fillCircleAA(img, float64(x+34), float64(ry+12), 8, pal.primary)
			barW := 230 + ((seed + row*71) % 190)
			fillRect(img, x+62, ry+4, barW, 16, elevated)
			fillRoundedRectAA(img, x+w-104, ry, 74, 24, 6, pal.badgeBg)
			strokeRoundedRectAA(img, x+w-104, ry, 74, 24, 6, 3, pal.softBorder)
		}

	default:
		for row := 0; row < 2; row++ {
			for col := 0; col < 2; col++ {
				cw := (w - 24) / 2
				ch := 192
				cx := x + col*(cw+24)
				cy := y + row*(ch+24)
				fillRoundedRectAA(img, cx, cy, cw, ch, 12, surface)
				strokeRoundedRectAA(img, cx, cy, cw, ch, 12, 4, border)
				fillRect(img, cx+22, cy+24, 62, 10, pal.primary)
				fillRoundedRectAA(img, cx+22, cy+58, cw-44, 68, 6, elevated)
				fillRect(img, cx+22, cy+146, (cw-44)*2/3, 16, border)
			}
		}
		statusY := y + 446
		fillRoundedRectAA(img, x, statusY, w, h-446, 12, surface)
		strokeRoundedRectAA(img, x, statusY, w, h-446, 12, 4, border)
		for i := 0; i < 3; i++ {
			ry := statusY + 34 + i*54
			fillRoundedRectAA(img, x+28, ry, w-56, 30, 6, elevated)
			fillRoundedRectAA(img, x+28, ry, (w-56)*(55+i*18)/100, 30, 6, pal.badgeBg)
			strokeRoundedRectAA(img, x+28, ry, w-56, 30, 6, 3, pal.softBorder)
		}
	}
}

func drawPillAA(img *image.RGBA, x, y, w, h, radius int, text string, face font.Face, bg, border, fg color.RGBA) {
	fillRoundedRectAA(img, x, y, w, h, float64(radius), bg)
	strokeRoundedRectAA(img, x, y, w, h, float64(radius), 3.5, border)

	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	textW := font.MeasureString(face, text).Ceil()
	tx := x + (w-textW)/2
	ty := y + (h+ascent-descent)/2
	drawString(img, tx, ty, text, face, fg)
}

func drawString(img *image.RGBA, x, y int, text string, face font.Face, col color.RGBA) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

func wrapTitleText(text string, face font.Face, maxWidthPx int, maxLines int) []string {
	cleaned := strings.TrimSpace(text)
	if cleaned == "" {
		return nil
	}
	tokens := strings.Fields(cleaned)
	if len(tokens) == 1 && font.MeasureString(face, tokens[0]).Ceil() > maxWidthPx {
		return wrapSlugTokens(tokens[0], face, maxWidthPx, maxLines)
	}

	var lines []string
	cur := tokens[0]
	for i := 1; i < len(tokens); i++ {
		w := tokens[i]
		candidate := cur + " " + w
		if font.MeasureString(face, candidate).Ceil() <= maxWidthPx {
			cur = candidate
			continue
		}
		lines = append(lines, cur)
		cur = w
		if len(lines) == maxLines-1 {
			cur = strings.Join(tokens[i:], " ")
			break
		}
	}
	if len(lines) < maxLines && cur != "" {
		lines = append(lines, truncateToPx(cur, face, maxWidthPx))
	}
	return lines
}

func wrapSlugTokens(slug string, face font.Face, maxWidthPx int, maxLines int) []string {
	sep := "-"
	if !strings.Contains(slug, "-") && strings.Contains(slug, "_") {
		sep = "_"
	}
	parts := strings.Split(slug, sep)
	if len(parts) <= 1 {
		return []string{truncateToPx(slug, face, maxWidthPx)}
	}
	var lines []string
	cur := parts[0]
	for i := 1; i < len(parts); i++ {
		candidate := cur + sep + parts[i]
		if font.MeasureString(face, candidate).Ceil() <= maxWidthPx {
			cur = candidate
			continue
		}
		lines = append(lines, cur+sep)
		cur = parts[i]
		if len(lines) == maxLines-1 {
			cur = strings.Join(parts[i:], sep)
			break
		}
	}
	if len(lines) < maxLines && cur != "" {
		lines = append(lines, truncateToPx(cur, face, maxWidthPx))
	}
	return lines
}

func truncateToPx(s string, face font.Face, maxWidthPx int) string {
	if font.MeasureString(face, s).Ceil() <= maxWidthPx {
		return s
	}
	runes := []rune(s)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := strings.TrimSpace(string(runes)) + "…"
		if font.MeasureString(face, candidate).Ceil() <= maxWidthPx {
			return candidate
		}
	}
	return s
}

func truncateRunes(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:maxRunes-1])) + "…"
}

func displayName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "T-Deck Project"
	}
	return trimmed
}

func nonEmptyOr(val, fallback string) string {
	if strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}

func fillRect(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	rect := image.Rect(x, y, x+w, y+h)
	draw.Draw(img, rect, &image.Uniform{C: col}, image.Point{}, draw.Src)
}

func strokeRect(img *image.RGBA, x, y, w, h, thick int, col color.RGBA) {
	fillRect(img, x, y, w, thick, col)
	fillRect(img, x, y+h-thick, w, thick, col)
	fillRect(img, x, y, thick, h, col)
	fillRect(img, x+w-thick, y, thick, h, col)
}

func blendPixel(img *image.RGBA, x, y int, col color.RGBA, alpha float64) {
	if alpha <= 0 {
		return
	}
	if !image.Pt(x, y).In(img.Bounds()) {
		return
	}
	if alpha >= 1 {
		img.SetRGBA(x, y, col)
		return
	}
	dst := img.RGBAAt(x, y)
	inv := 1.0 - alpha
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(math.Round(float64(col.R)*alpha + float64(dst.R)*inv)),
		G: uint8(math.Round(float64(col.G)*alpha + float64(dst.G)*inv)),
		B: uint8(math.Round(float64(col.B)*alpha + float64(dst.B)*inv)),
		A: 255,
	})
}

func fillCircleAA(img *image.RGBA, cx, cy, r float64, col color.RGBA) {
	minX := int(math.Floor(cx - r - 1))
	maxX := int(math.Ceil(cx + r + 1))
	minY := int(math.Floor(cy - r - 1))
	maxY := int(math.Ceil(cy + r + 1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dist := math.Hypot(float64(x)-cx+0.5, float64(y)-cy+0.5)
			cov := clamp01(r + 0.5 - dist)
			if cov > 0 {
				blendPixel(img, x, y, col, cov)
			}
		}
	}
}

func strokeCircleAA(img *image.RGBA, cx, cy, r, thickness float64, col color.RGBA) {
	half := thickness * 0.5
	minX := int(math.Floor(cx - r - half - 1))
	maxX := int(math.Ceil(cx + r + half + 1))
	minY := int(math.Floor(cy - r - half - 1))
	maxY := int(math.Ceil(cy + r + half + 1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dist := math.Hypot(float64(x)-cx+0.5, float64(y)-cy+0.5)
			cov := clamp01(half + 0.5 - math.Abs(dist-r))
			if cov > 0 {
				blendPixel(img, x, y, col, cov)
			}
		}
	}
}

func roundedRectSDF(px, py, cx, cy, halfW, halfH, r float64) float64 {
	qx := math.Abs(px-cx) - (halfW - r)
	qy := math.Abs(py-cy) - (halfH - r)
	ox := math.Max(qx, 0)
	oy := math.Max(qy, 0)
	return math.Hypot(ox, oy) + math.Min(math.Max(qx, qy), 0) - r
}

func fillRoundedRectAA(img *image.RGBA, x, y, w, h int, r float64, col color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	cx := float64(x) + float64(w)*0.5
	cy := float64(y) + float64(h)*0.5
	halfW := float64(w) * 0.5
	halfH := float64(h) * 0.5
	r = math.Min(r, math.Min(halfW, halfH))
	for py := y - 1; py <= y+h; py++ {
		for px := x - 1; px <= x+w; px++ {
			d := roundedRectSDF(float64(px)+0.5, float64(py)+0.5, cx, cy, halfW, halfH, r)
			cov := clamp01(0.5 - d)
			if cov > 0 {
				blendPixel(img, px, py, col, cov)
			}
		}
	}
}

func strokeRoundedRectAA(img *image.RGBA, x, y, w, h int, r, thickness float64, col color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	cx := float64(x) + float64(w)*0.5
	cy := float64(y) + float64(h)*0.5
	halfW := float64(w) * 0.5
	halfH := float64(h) * 0.5
	r = math.Min(r, math.Min(halfW, halfH))
	halfT := thickness * 0.5
	pad := int(math.Ceil(halfT)) + 1
	for py := y - pad; py <= y+h+pad; py++ {
		for px := x - pad; px <= x+w+pad; px++ {
			d := roundedRectSDF(float64(px)+0.5, float64(py)+0.5, cx, cy, halfW-halfT, halfH-halfT, math.Max(0, r-halfT))
			cov := clamp01(halfT + 0.5 - math.Abs(d))
			if cov > 0 {
				blendPixel(img, px, py, col, cov)
			}
		}
	}
}

func drawPlayTriangleAA(img *image.RGBA, x, y, w, h float64, col color.RGBA) {
	ax, ay := x, y
	bx, by := x+w, y+h*0.5
	cx, cy := x, y+h

	minX := int(math.Floor(x)) - 1
	maxX := int(math.Ceil(x+w)) + 1
	minY := int(math.Floor(y)) - 1
	maxY := int(math.Ceil(y+h)) + 1

	edgeDist := func(px, py, x1, y1, x2, y2 float64) float64 {
		dx, dy := x2-x1, y2-y1
		l := math.Hypot(dx, dy)
		if l == 0 {
			return -1
		}
		return ((px-x1)*dy - (py-y1)*dx) / l
	}

	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			sx, sy := float64(px)+0.5, float64(py)+0.5
			d1 := -edgeDist(sx, sy, ax, ay, bx, by)
			d2 := -edgeDist(sx, sy, bx, by, cx, cy)
			d3 := -edgeDist(sx, sy, cx, cy, ax, ay)
			dMin := math.Min(d1, math.Min(d2, d3))
			cov := clamp01(dMin + 0.5)
			if cov > 0 {
				blendPixel(img, px, py, col, cov)
			}
		}
	}
}

func clamp01(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	return v
}

func closeFaces(faces ...font.Face) {
	for _, f := range faces {
		if c, ok := f.(interface{ Close() error }); ok && f != basicfont.Face7x13 {
			_ = c.Close()
		}
	}
}
