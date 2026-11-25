package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"testing"

	"t-deck/internal/model"
)

func TestRenderProjectThumbnailPNGAllDomains(t *testing.T) {
	cases := []model.Project{
		{
			ID:        "p1",
			Slug:      "ai-studio",
			Name:      "AI Studio — Video Engine",
			Type:      "desktop",
			TechStack: "React, Remotion, Neural TTS",
			Tags:      []string{"video", "ai"},
		},
		{
			ID:        "p2",
			Slug:      "s3-desk",
			Name:      "S3 Desk",
			Type:      "desktop",
			TechStack: "Go, Wails, React",
			Tags:      []string{"r2", "storage"},
		},
		{
			ID:        "p3",
			Slug:      "t-hub-licensing-server",
			Name:      "T-Hub Licensing Server",
			Type:      "web",
			TechStack: "Go, Ed25519, SQLite",
			Tags:      []string{"security", "license"},
		},
		{
			ID:        "p4",
			Slug:      "admin-one-vue-tailwind",
			Name:      "Admin One &mdash; Free Vue 3 Dashboard",
			Type:      "web",
			TechStack: "Vue 3, Tailwind CSS, Vite",
			Summary:   "Admin One &mdash; Free Vue 3.x Tailwind 4.x Admin Dashboard",
			Tags:      []string{"web", "vue", "market"},
		},
		{
			ID:        "p5",
			Slug:      "ai-company-studio-office-integration",
			Name:      "ai-company-studio-office-integration",
			Type:      "web",
			TechStack: "Node.js / React / Vite / TypeScript",
			Tags:      []string{"bot", "agent"},
		},
		{
			ID:   "p6",
			Slug: "generic-hub",
			Name: "Tổng hợp dự án Tiếng Việt Có Dấu",
			Type: "hybrid",
		},
	}

	for _, tc := range cases {
		raw, err := RenderProjectThumbnailPNG(tc)
		if err != nil {
			t.Fatalf("RenderProjectThumbnailPNG(%s) failed: %v", tc.Slug, err)
		}
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("png.Decode(%s) failed: %v", tc.Slug, err)
		}
		if got := img.Bounds(); got.Dx() != thumbW || got.Dy() != thumbH {
			t.Fatalf("%s expected %dx%d, got %dx%d", tc.Slug, thumbW, thumbH, got.Dx(), got.Dy())
		}
		// Verify non-flat output (pixels differ across background and artwork).
		c0 := img.At(10, 10)
		diff := 0
		for x := 40; x < thumbW-40; x += 25 {
			for y := 60; y < thumbH-60; y += 25 {
				if img.At(x, y) != c0 {
					diff++
				}
			}
		}
		if diff < 20 {
			t.Fatalf("%s expected rich pixel variance, only found %d non-bg samples", tc.Slug, diff)
		}
	}
}

func TestExtractStackBadges(t *testing.T) {
	p1 := model.Project{
		Type:      "web",
		TechStack: "React, react, Node.js / Tailwind CSS + VeryLongFrameworkNameHere",
		Tags:      []string{"vite", "typescript"},
	}
	got := extractStackBadges(p1, 3)
	want := []string{"REACT", "NODE.JS", "TAILWIND CSS"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	p2 := model.Project{
		Type: "desktop",
		Tags: []string{"#go", "wails"},
	}
	got2 := extractStackBadges(p2, 3)
	want2 := []string{"GO", "WAILS"}
	if !reflect.DeepEqual(got2, want2) {
		t.Fatalf("expected %v, got %v", want2, got2)
	}

	p3 := model.Project{Type: "desktop"}
	got3 := extractStackBadges(p3, 3)
	want3 := []string{"DESKTOP"}
	if !reflect.DeepEqual(got3, want3) {
		t.Fatalf("expected %v, got %v", want3, got3)
	}
}

func TestAntiAliasedPrimitives(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 120))
	bg := color.RGBA{10, 10, 10, 255}
	fg := color.RGBA{250, 250, 250, 255}
	fillRect(img, 0, 0, 120, 120, bg)

	fillCircleAA(img, 40, 40, 20, fg)
	if center := img.RGBAAt(40, 40); center != fg {
		t.Fatalf("expected circle center %v, got %v", fg, center)
	}

	// Scan for intermediate anti-aliased edge pixel strictly between bg and fg.
	hasIntermediate := false
	for y := 18; y <= 62; y++ {
		for x := 18; x <= 62; x++ {
			c := img.RGBAAt(x, y)
			if c.R > bg.R && c.R < fg.R {
				hasIntermediate = true
				break
			}
		}
		if hasIntermediate {
			break
		}
	}
	if !hasIntermediate {
		t.Fatalf("expected anti-aliased intermediate edge pixels on fillCircleAA")
	}

	drawPlayTriangleAA(img, 80, 30, 26, 32, fg)
	if triCenter := img.RGBAAt(88, 46); triCenter != fg {
		t.Fatalf("expected play triangle interior %v, got %v", fg, triCenter)
	}
}

