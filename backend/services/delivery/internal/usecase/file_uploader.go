package usecase

import (
	"context"
	"io"
)

// FileUploader abstracts object storage for incident photo uploads.
// Implemented by pkg/storage.Client; defined here to keep the usecase layer
// free of infrastructure imports.
type FileUploader interface {
	Put(ctx context.Context, objectKey, contentType string, r io.Reader, size int64) (string, error)
}
