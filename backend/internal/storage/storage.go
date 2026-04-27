package storage

import (
	"context"
	"io"
	"time"
)

type Driver interface {
	Save(ctx context.Context, reader io.Reader, size int64, key string, contentType string) (string, error)
	GetDownloadURL(ctx context.Context, storagePath string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, storagePath string) error
}

