package media

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"t-deck/internal/model"
)

const maxThumbnailBytes = 10 * 1024 * 1024 // 10 MB

var pngMagic = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// ThumbnailRepo defines the persistence operations required by ThumbnailService.
type ThumbnailRepo interface {
	GetProjectModel(id string) (model.Project, bool, error)
	UpdateThumbnailTimestamp(projectID string, ts int64) error
}

// ThumbnailService handles clipboard PNG validation, storage, and HTTP serving.
type ThumbnailService struct {
	repo          ThumbnailRepo
	thumbnailsDir string
}

// NewThumbnailService creates a ThumbnailService storing images in thumbnailsDir.
func NewThumbnailService(repo ThumbnailRepo, thumbnailsDir string) *ThumbnailService {
	return &ThumbnailService{
		repo:          repo,
		thumbnailsDir: thumbnailsDir,
	}
}

// SaveClipboardPNG decodes base64 PNG data, verifies its PNG header, saves it, and updates the DB timestamp.
func (s *ThumbnailService) SaveClipboardPNG(projectID string, base64PNG string) (string, error) {
	if s.repo == nil {
		return "", errors.New("repository not initialized")
	}
	proj, found, err := s.repo.GetProjectModel(projectID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("project %q not found", projectID)
	}

	payload := strings.TrimSpace(base64PNG)
	if idx := strings.Index(payload, ","); idx != -1 && strings.HasPrefix(payload[:idx], "data:") {
		payload = payload[idx+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("invalid base64 image data: %w", err)
	}
	if len(raw) == 0 || len(raw) > maxThumbnailBytes {
		return "", errors.New("thumbnail size must be between 1 byte and 10 MB")
	}
	if len(raw) < len(pngMagic) || !bytes.Equal(raw[:len(pngMagic)], pngMagic) {
		return "", errors.New("clipboard image must be PNG format")
	}

	return s.persistThumbnail(proj, raw)
}

// ServeHTTP handles /thumbnails/<project_id>.png requests with path-traversal protection.
func (s *ThumbnailService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.thumbnailsDir == "" {
		http.NotFound(w, r)
		return
	}
	rel := strings.TrimPrefix(r.URL.Path, "/thumbnails/")
	name := filepath.Base(rel)
	if name == "." || name == ".." || name != rel || !strings.HasSuffix(strings.ToLower(name), ".png") {
		http.NotFound(w, r)
		return
	}
	fullPath := filepath.Join(s.thumbnailsDir, name)
	if _, err := os.Stat(fullPath); err != nil {
		http.NotFound(w, r)
		return
	}
	if strings.TrimSpace(r.URL.Query().Get("v")) != "" {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	http.ServeFile(w, r, fullPath)
}

// GenerateProjectThumbnail synthesizes a 1280x720 thumbnail PNG from project metadata and saves it locally.
func (s *ThumbnailService) GenerateProjectThumbnail(projectID string) (string, error) {
	if s.repo == nil {
		return "", errors.New("repository not initialized")
	}
	proj, found, err := s.repo.GetProjectModel(projectID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("project %q not found", projectID)
	}

	raw, err := RenderProjectThumbnailPNG(proj)
	if err != nil {
		return "", fmt.Errorf("render thumbnail: %w", err)
	}
	return s.persistThumbnail(proj, raw)
}

func (s *ThumbnailService) persistThumbnail(proj model.Project, raw []byte) (string, error) {
	if err := os.MkdirAll(s.thumbnailsDir, 0o755); err != nil {
		return "", fmt.Errorf("create thumbnails dir: %w", err)
	}

	thumbPath := filepath.Join(s.thumbnailsDir, proj.ID+".png")
	if err := os.WriteFile(thumbPath, raw, 0o644); err != nil {
		return "", fmt.Errorf("write thumbnail: %w", err)
	}

	ts := time.Now().Unix()
	if ts <= proj.ThumbnailUpdatedAt {
		ts = proj.ThumbnailUpdatedAt + 1
	}
	if err := s.repo.UpdateThumbnailTimestamp(proj.ID, ts); err != nil {
		return "", err
	}
	if marker, ok := s.repo.(interface{ MarkThumbnailExists(string, bool) }); ok {
		marker.MarkThumbnailExists(proj.ID, true)
	}
	return fmt.Sprintf("/thumbnails/%s.png?v=%d", proj.ID, ts), nil
}
