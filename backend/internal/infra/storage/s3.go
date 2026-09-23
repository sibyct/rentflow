// Package storage adapts domain.FileStorage to an S3-compatible object
// store (MinIO locally; S3, Tigris or R2 in production) using minio-go.
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"propertymanagement/internal/domain"
)

type S3Config struct {
	// Endpoint is how this process reaches the store (http://minio:9000).
	Endpoint string
	// PublicEndpoint is how a browser reaches it; presigned URLs are
	// signed against this host. Defaults to Endpoint.
	PublicEndpoint string
	Bucket         string
	AccessKey      string
	SecretKey      string
	Region         string
}

type S3 struct {
	bucket string
	// internal does the server-side reads/writes; public only ever signs
	// URLs for the browser (signing is offline, so it needs no network
	// route from this process to PublicEndpoint).
	internal *minio.Client
	public   *minio.Client
}

var _ domain.FileStorage = (*S3)(nil)

func newClient(endpoint string, cfg S3Config) (*minio.Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid S3 endpoint %q (want a full URL like http://minio:9000)", endpoint)
	}
	return minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: u.Scheme == "https",
		// Region set explicitly so no bucket-location lookup (a network
		// call) is needed to sign or use the client.
		Region: cfg.Region,
	})
}

func NewS3(cfg S3Config) (*S3, error) {
	internal, err := newClient(cfg.Endpoint, cfg)
	if err != nil {
		return nil, err
	}
	public := internal
	if cfg.PublicEndpoint != "" && cfg.PublicEndpoint != cfg.Endpoint {
		if public, err = newClient(cfg.PublicEndpoint, cfg); err != nil {
			return nil, err
		}
	}
	return &S3{bucket: cfg.Bucket, internal: internal, public: public}, nil
}

// EnsureBucket creates the bucket if it doesn't exist — a convenience
// for local MinIO; in production the bucket is provisioned out of band.
func (s *S3) EnsureBucket(ctx context.Context) error {
	exists, err := s.internal.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %q: %w", s.bucket, err)
	}
	if exists {
		return nil
	}
	if err := s.internal.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket %q: %w", s.bucket, err)
	}
	return nil
}

func (s *S3) PresignUpload(ctx context.Context, key, contentType string, maxBytes int64, ttl time.Duration) (string, map[string]string, error) {
	policy := minio.NewPostPolicy()
	if err := policy.SetBucket(s.bucket); err != nil {
		return "", nil, fmt.Errorf("presign upload: %w", err)
	}
	if err := policy.SetKey(key); err != nil {
		return "", nil, fmt.Errorf("presign upload: %w", err)
	}
	if err := policy.SetExpires(time.Now().UTC().Add(ttl)); err != nil {
		return "", nil, fmt.Errorf("presign upload: %w", err)
	}
	if err := policy.SetContentType(contentType); err != nil {
		return "", nil, fmt.Errorf("presign upload: %w", err)
	}
	if err := policy.SetContentLengthRange(1, maxBytes); err != nil {
		return "", nil, fmt.Errorf("presign upload: %w", err)
	}
	u, fields, err := s.public.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return "", nil, fmt.Errorf("presign upload: %w", err)
	}
	return u.String(), fields, nil
}

func (s *S3) PresignDownload(ctx context.Context, key, filename string, ttl time.Duration) (string, error) {
	params := url.Values{}
	params.Set("response-content-disposition", fmt.Sprintf("inline; filename=%q", filename))
	u, err := s.public.PresignedGetObject(ctx, s.bucket, key, ttl, params)
	if err != nil {
		return "", fmt.Errorf("presign download: %w", err)
	}
	return u.String(), nil
}

func (s *S3) Put(ctx context.Context, key, contentType string, data []byte) error {
	_, err := s.internal.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

func (s *S3) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.internal.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}
	defer func() { _ = obj.Close() }()
	data, err := io.ReadAll(obj)
	if err != nil {
		if isNoSuchKey(err) {
			return nil, fmt.Errorf("get object %q: %w", key, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("read object %q: %w", key, err)
	}
	return data, nil
}

func (s *S3) Stat(ctx context.Context, key string) (int64, string, error) {
	info, err := s.internal.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNoSuchKey(err) {
			return 0, "", fmt.Errorf("stat object %q: %w", key, domain.ErrNotFound)
		}
		return 0, "", fmt.Errorf("stat object %q: %w", key, err)
	}
	return info.Size, info.ContentType, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	if err := s.internal.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object %q: %w", key, err)
	}
	return nil
}

func isNoSuchKey(err error) bool {
	var resp minio.ErrorResponse
	return errors.As(err, &resp) && (resp.Code == "NoSuchKey" || resp.Code == "NotFound")
}

// Noop stands in when no object storage is configured: every operation
// answers domain.ErrUnavailable, which the API maps to 503, so the rest
// of the app keeps working without it.
type Noop struct{}

var _ domain.FileStorage = Noop{}

func (Noop) PresignUpload(context.Context, string, string, int64, time.Duration) (string, map[string]string, error) {
	return "", nil, domain.ErrUnavailable
}
func (Noop) PresignDownload(context.Context, string, string, time.Duration) (string, error) {
	return "", domain.ErrUnavailable
}
func (Noop) Put(context.Context, string, string, []byte) error   { return domain.ErrUnavailable }
func (Noop) Get(context.Context, string) ([]byte, error)         { return nil, domain.ErrUnavailable }
func (Noop) Stat(context.Context, string) (int64, string, error) { return 0, "", domain.ErrUnavailable }
func (Noop) Delete(context.Context, string) error                { return domain.ErrUnavailable }
