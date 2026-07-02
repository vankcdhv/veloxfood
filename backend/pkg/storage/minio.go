// Package storage provides an S3-compatible (MinIO) object storage client
// for storing user-uploaded files (shipper documents, menu images, etc.).
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"project/pkg/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client wraps a MinIO client + the target bucket.
type Client struct {
	mc         *minio.Client
	bucket     string
	publicBase string // "<scheme>://<endpoint>/<bucket>" — prefix for object URLs
	private    bool   // private buckets have no anon-read policy; Put returns the key
}

// NewClient builds a MinIO client, ensures the bucket exists, and makes it
// anonymous-read so uploaded images (avatars, menu photos, incident photos) are
// directly fetchable by the browser via the URL returned from Put.
func NewClient(cfg config.MinIOConfig) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: init client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := mc.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("minio: bucket check: %w", err)
	}
	if !exists {
		if err := mc.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio: make bucket: %w", err)
		}
	}

	// Allow anonymous GET on objects so returned URLs render without signing.
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, cfg.Bucket)
	if err := mc.SetBucketPolicy(ctx, cfg.Bucket, policy); err != nil {
		return nil, fmt.Errorf("minio: set public-read policy: %w", err)
	}

	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	return &Client{
		mc:         mc,
		bucket:     cfg.Bucket,
		publicBase: fmt.Sprintf("%s://%s/%s", scheme, cfg.Endpoint, cfg.Bucket),
	}, nil
}

// NewPrivateClient builds a MinIO client against cfg.PrivateBucket for
// sensitive uploads (KYC documents). The bucket keeps the default private
// policy — objects are only reachable via PresignedURL — and Put returns the
// object key (not a URL) so callers persist keys and sign on read.
func NewPrivateClient(cfg config.MinIOConfig) (*Client, error) {
	if cfg.PrivateBucket == "" {
		return nil, fmt.Errorf("minio: private_bucket is not configured")
	}
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: init client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := mc.BucketExists(ctx, cfg.PrivateBucket)
	if err != nil {
		return nil, fmt.Errorf("minio: bucket check: %w", err)
	}
	if !exists {
		if err := mc.MakeBucket(ctx, cfg.PrivateBucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio: make bucket: %w", err)
		}
	}

	return &Client{mc: mc, bucket: cfg.PrivateBucket, private: true}, nil
}

// Put uploads an object. Public clients return the object's public URL;
// private clients return the object key for later presigning.
func (c *Client) Put(ctx context.Context, objectKey, contentType string, r io.Reader, size int64) (string, error) {
	_, err := c.mc.PutObject(ctx, c.bucket, objectKey, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("minio: put object: %w", err)
	}
	if c.private {
		return objectKey, nil
	}
	return fmt.Sprintf("%s/%s", c.publicBase, objectKey), nil
}

// PresignedURL returns a temporary GET URL for an object key.
func (c *Client) PresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, c.bucket, objectKey, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("minio: presign: %w", err)
	}
	return u.String(), nil
}
