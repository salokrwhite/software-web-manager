package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Storage struct {
	Client         *s3.Client
	PresignClient  *s3.PresignClient
	Bucket         string
	PublicBaseURL  string
}

func (s *S3Storage) Save(ctx context.Context, reader io.Reader, size int64, key string, contentType string) (string, error) {
	uploader := manager.NewUploader(s.Client)
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPrivate,
	})
	if err != nil {
		return "", err
	}
	_ = size
	return key, nil
}

func (s *S3Storage) GetDownloadURL(ctx context.Context, storagePath string, expiry time.Duration) (string, error) {
	if s.PublicBaseURL != "" {
		base := strings.TrimRight(s.PublicBaseURL, "/")
		return fmt.Sprintf("%s/%s", base, strings.TrimLeft(storagePath, "/")), nil
	}
	presigner := s.PresignClient
	if presigner == nil {
		presigner = s3.NewPresignClient(s.Client)
	}
	resp, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(storagePath),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", err
	}
	return resp.URL, nil
}

func (s *S3Storage) Delete(ctx context.Context, storagePath string) error {
	if strings.TrimSpace(storagePath) == "" {
		return nil
	}
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(storagePath),
	})
	return err
}

