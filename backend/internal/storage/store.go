package storage

import (
	"context"
	"io"
)

// ObjectMeta describes a stored object.
type ObjectMeta struct {
	Key         string
	SizeBytes   int64
	ContentType string
}

// PutRequest describes an object to store.
type PutRequest struct {
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Reader      io.Reader
}

// PutResult is returned after a successful Put.
type PutResult struct {
	ObjectKey string
	SizeBytes int64
}

// ObjectStore abstracts object storage (local FS, S3, R2, etc.).
type ObjectStore interface {
	Put(ctx context.Context, req PutRequest) (PutResult, error)
	GetURL(ctx context.Context, objectKey string) (string, error)
	Delete(ctx context.Context, objectKey string) error
	Stat(ctx context.Context, objectKey string) (*ObjectMeta, error)
}
