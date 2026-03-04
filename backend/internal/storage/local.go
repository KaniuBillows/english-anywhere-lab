package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// LocalStore stores objects on the local filesystem.
type LocalStore struct {
	rootDir string
	baseURL string
}

// NewLocalStore creates a new LocalStore. rootDir is the directory where files
// are stored, baseURL is the URL prefix used to serve them (e.g. "/static/files").
func NewLocalStore(rootDir, baseURL string) (*LocalStore, error) {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve root dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("create root dir: %w", err)
	}
	return &LocalStore{
		rootDir: abs,
		baseURL: strings.TrimRight(baseURL, "/"),
	}, nil
}

func (s *LocalStore) resolvePath(objectKey string) (string, error) {
	cleaned := path.Clean("/" + objectKey)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("invalid object key: %s", objectKey)
	}
	cleaned = strings.TrimPrefix(cleaned, "/")
	full := filepath.Join(s.rootDir, filepath.FromSlash(cleaned))
	if !strings.HasPrefix(full, s.rootDir) {
		return "", fmt.Errorf("path traversal detected: %s", objectKey)
	}
	return full, nil
}

func (s *LocalStore) Put(_ context.Context, req PutRequest) (PutResult, error) {
	fpath, err := s.resolvePath(req.ObjectKey)
	if err != nil {
		return PutResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(fpath), 0o755); err != nil {
		return PutResult{}, fmt.Errorf("create dirs: %w", err)
	}
	f, err := os.Create(fpath)
	if err != nil {
		return PutResult{}, fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, req.Reader)
	if err != nil {
		os.Remove(fpath)
		return PutResult{}, fmt.Errorf("write file: %w", err)
	}
	return PutResult{ObjectKey: req.ObjectKey, SizeBytes: n}, nil
}

func (s *LocalStore) GetURL(_ context.Context, objectKey string) (string, error) {
	cleaned := path.Clean("/" + objectKey)
	cleaned = strings.TrimPrefix(cleaned, "/")
	return s.baseURL + "/" + cleaned, nil
}

func (s *LocalStore) Delete(_ context.Context, objectKey string) error {
	fpath, err := s.resolvePath(objectKey)
	if err != nil {
		return err
	}
	if err := os.Remove(fpath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

func (s *LocalStore) Stat(_ context.Context, objectKey string) (*ObjectMeta, error) {
	fpath, err := s.resolvePath(objectKey)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(fpath)
	if err != nil {
		return nil, err
	}
	return &ObjectMeta{
		Key:       objectKey,
		SizeBytes: info.Size(),
	}, nil
}
