package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalStorage struct {
	RootPath  string
	PublicURL string
}

func (s *LocalStorage) Save(ctx context.Context, reader io.Reader, size int64, key string, contentType string) (string, error) {
	_ = ctx
	path := filepath.Join(s.RootPath, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(key), nil
}

func (s *LocalStorage) GetDownloadURL(ctx context.Context, storagePath string, expiry time.Duration) (string, error) {
	_ = ctx
	_ = expiry
	base := strings.TrimRight(s.PublicURL, "/")
	return fmt.Sprintf("%s/%s", base, strings.TrimLeft(storagePath, "/")), nil
}

func (s *LocalStorage) Delete(ctx context.Context, storagePath string) error {
	_ = ctx
	if strings.TrimSpace(storagePath) == "" {
		return nil
	}
	path := filepath.Join(s.RootPath, filepath.FromSlash(storagePath))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

