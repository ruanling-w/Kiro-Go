package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	chatAttachmentMaxFiles     = 4
	chatAttachmentMaxBytes     = 10 << 20
	chatAttachmentRequestMax   = chatAttachmentMaxFiles*chatAttachmentMaxBytes + 1<<20
	chatAttachmentMaxPixels    = 40_000_000
	chatAttachmentMaxDimension = 16_384
)

var errInvalidChatImage = errors.New("invalid chat image")

type chatAssetStore struct{ root string }

type validatedChatImage struct {
	MIMEType string
	Width    int
	Height   int
	Size     int64
}

func newChatAssetStore(root string) (*chatAssetStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("chat asset root is empty")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	if err = os.Chmod(root, 0o700); err != nil {
		return nil, err
	}
	return &chatAssetStore{root: root}, nil
}

func (s *chatAssetStore) store(conversationID string, file io.Reader) (string, validatedChatImage, error) {
	conversationDir := filepath.Join(s.root, conversationID)
	if err := os.MkdirAll(conversationDir, 0o700); err != nil {
		return "", validatedChatImage{}, err
	}
	if err := os.Chmod(conversationDir, 0o700); err != nil {
		return "", validatedChatImage{}, err
	}
	id := uuid.NewString()
	temp, err := os.OpenFile(filepath.Join(conversationDir, "."+id+".upload"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", validatedChatImage{}, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	written, copyErr := io.Copy(temp, io.LimitReader(file, chatAttachmentMaxBytes+1))
	if syncErr := temp.Sync(); copyErr == nil {
		copyErr = syncErr
	}
	if closeErr := temp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return "", validatedChatImage{}, copyErr
	}
	if written == 0 || written > chatAttachmentMaxBytes {
		return "", validatedChatImage{}, errInvalidChatImage
	}
	validated, err := validateStoredChatImage(tempPath, written)
	if err != nil {
		return "", validatedChatImage{}, err
	}
	storageKey := filepath.Join(conversationID, id)
	finalPath, err := s.path(storageKey)
	if err != nil {
		return "", validatedChatImage{}, err
	}
	if err = os.Rename(tempPath, finalPath); err != nil {
		return "", validatedChatImage{}, err
	}
	return filepath.ToSlash(storageKey), validated, nil
}

func (s *chatAssetStore) removeConversation(conversationID string) error {
	path := filepath.Join(s.root, conversationID)
	rel, err := filepath.Rel(s.root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errInvalidChatImage
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errInvalidChatImage
	}
	return os.RemoveAll(path)
}

func (s *chatAssetStore) path(storageKey string) (string, error) {
	if filepath.IsAbs(storageKey) {
		return "", errInvalidChatImage
	}
	clean := filepath.Clean(filepath.FromSlash(storageKey))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errInvalidChatImage
	}
	path := filepath.Join(s.root, clean)
	rel, err := filepath.Rel(s.root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errInvalidChatImage
	}
	info, err := os.Lstat(filepath.Dir(path))
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return "", errInvalidChatImage
	}
	return path, nil
}

type chatAssetReconcileResult struct {
	RemovedOrphans int
	RemovedUploads int
	RemovedExpired int
}

func (s *chatAssetStore) reconcile(referenced map[string]bool, expired []string, staleBefore time.Time) (chatAssetReconcileResult, error) {
	result := chatAssetReconcileResult{}
	for _, storageKey := range expired {
		path, err := s.path(storageKey)
		if err == nil && os.Remove(path) == nil {
			result.RemovedExpired++
		}
	}
	err := filepath.WalkDir(s.root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == s.root || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		storageKey := filepath.ToSlash(rel)
		if strings.HasSuffix(entry.Name(), ".upload") {
			info, infoErr := entry.Info()
			if infoErr == nil && info.ModTime().Before(staleBefore) && os.Remove(path) == nil {
				result.RemovedUploads++
			}
			return nil
		}
		if !referenced[storageKey] && os.Remove(path) == nil {
			result.RemovedOrphans++
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	_ = filepath.WalkDir(s.root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr == nil && entry.IsDir() && path != s.root {
			_ = os.Remove(path)
		}
		return nil
	})
	return result, nil
}

func validateStoredChatImage(path string, size int64) (validatedChatImage, error) {
	file, err := os.Open(path)
	if err != nil {
		return validatedChatImage{}, err
	}
	defer file.Close()
	header := make([]byte, 30)
	n, err := io.ReadFull(file, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return validatedChatImage{}, errInvalidChatImage
	}
	header = header[:n]
	mimeType := http.DetectContentType(header)
	var width, height int
	switch mimeType {
	case "image/png", "image/jpeg":
		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return validatedChatImage{}, err
		}
		cfg, _, decodeErr := image.DecodeConfig(file)
		if decodeErr != nil {
			return validatedChatImage{}, errInvalidChatImage
		}
		width, height = cfg.Width, cfg.Height
	case "image/webp":
		width, height, err = webPDimensions(header)
		if err != nil {
			return validatedChatImage{}, errInvalidChatImage
		}
	default:
		return validatedChatImage{}, errInvalidChatImage
	}
	if width <= 0 || height <= 0 || width > chatAttachmentMaxDimension || height > chatAttachmentMaxDimension || int64(width)*int64(height) > chatAttachmentMaxPixels {
		return validatedChatImage{}, errInvalidChatImage
	}
	return validatedChatImage{MIMEType: mimeType, Width: width, Height: height, Size: size}, nil
}

func webPDimensions(b []byte) (int, int, error) {
	if len(b) < 30 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return 0, 0, errInvalidChatImage
	}
	switch string(b[12:16]) {
	case "VP8X":
		return 1 + int(b[24]) + int(b[25])<<8 + int(b[26])<<16, 1 + int(b[27]) + int(b[28])<<8 + int(b[29])<<16, nil
	case "VP8 ":
		if b[23] != 0x9d || b[24] != 0x01 || b[25] != 0x2a {
			return 0, 0, errInvalidChatImage
		}
		return int(binary.LittleEndian.Uint16(b[26:28]) & 0x3fff), int(binary.LittleEndian.Uint16(b[28:30]) & 0x3fff), nil
	case "VP8L":
		if b[20] != 0x2f {
			return 0, 0, errInvalidChatImage
		}
		bits := binary.LittleEndian.Uint32(b[21:25])
		return int(bits&0x3fff) + 1, int((bits>>14)&0x3fff) + 1, nil
	default:
		return 0, 0, fmt.Errorf("%w: unsupported WebP chunk", errInvalidChatImage)
	}
}
