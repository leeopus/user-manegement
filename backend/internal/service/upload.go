package service

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user-system/backend/internal/config"
	apperrors "github.com/user-system/backend/pkg/errors"
)

type UploadService interface {
	UploadAvatar(userID uint, file io.ReadSeeker, filename string, size int64) (url string, err error)
}

type uploadService struct {
	cfg config.UploadConfig
}

func NewUploadService(cfg config.UploadConfig) UploadService {
	return &uploadService{cfg: cfg}
}

func (s *uploadService) UploadAvatar(userID uint, file io.ReadSeeker, filename string, size int64) (string, error) {
	if size > s.cfg.MaxFileSize {
		return "", apperrors.ErrFileTooLarge.WithDetails(map[string]interface{}{
			"max_bytes": s.cfg.MaxFileSize,
		})
	}

	// Read first 512 bytes to detect actual content type
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", apperrors.ErrFileInvalid
	}

	contentType := http.DetectContentType(buf[:n])
	if !s.isAllowedType(contentType) {
		return "", apperrors.ErrFileInvalidType.WithDetails(map[string]interface{}{
			"allowed": s.cfg.AllowedTypes,
		})
	}

	// Seek back to start for full file copy
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", apperrors.ErrInternalServer
	}

	ext := extensionFromType(contentType)
	safeFilename := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixMilli(), ext)

	dir := s.cfg.AvatarDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", apperrors.ErrInternalServer
	}

	destPath := filepath.Join(dir, safeFilename)
	// Path traversal check
	absDest, _ := filepath.Abs(destPath)
	absDir, _ := filepath.Abs(dir)
	if !strings.HasPrefix(absDest, absDir) {
		return "", apperrors.ErrInternalServer
	}

	f, err := os.Create(destPath)
	if err != nil {
		return "", apperrors.ErrInternalServer
	}
	defer f.Close()

	if _, err := io.Copy(f, file); err != nil {
		os.Remove(destPath)
		return "", apperrors.ErrInternalServer
	}

	return "/uploads/avatars/" + safeFilename, nil
}

func (s *uploadService) isAllowedType(contentType string) bool {
	allowed := strings.Split(s.cfg.AllowedTypes, ",")
	for _, a := range allowed {
		a = strings.TrimSpace(a)
		if a == contentType {
			return true
		}
	}
	return false
}

func extensionFromType(contentType string) string {
	ext, _ := mime.ExtensionsByType(contentType)
	if len(ext) > 0 {
		return ext[0]
	}
	return ".bin"
}
