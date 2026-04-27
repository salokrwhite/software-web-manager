package storage

import (
	"context"
	"errors"

	"software-web-manager/backend/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func New(ctx context.Context, cfg config.Config) (Driver, error) {
	switch cfg.StorageDriver {
	case "local":
		return &LocalStorage{RootPath: cfg.LocalStoragePath, PublicURL: cfg.LocalPublicBaseURL}, nil
	case "s3":
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(cfg.S3Region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
			awsconfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if cfg.S3Endpoint == "" {
					return aws.Endpoint{}, errors.New("missing S3 endpoint")
				}
				return aws.Endpoint{URL: cfg.S3Endpoint, HostnameImmutable: true}, nil
			})),
		)
		if err != nil {
			return nil, err
		}
		client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.UsePathStyle = cfg.S3UsePathStyle
		})
		return &S3Storage{Client: client, Bucket: cfg.S3Bucket, PublicBaseURL: cfg.S3PublicBaseURL}, nil
	default:
		return nil, errors.New("unknown storage driver")
	}
}

