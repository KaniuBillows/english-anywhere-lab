package storage_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/bennyshi/english-anywhere-lab/internal/storage"
)

func TestLocalStore_Lifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewLocalStore(tmpDir, "/static/files")
	if err != nil {
		t.Fatalf("new local store: %v", err)
	}

	ctx := context.Background()
	key := "tts/en/default/wav/abc123.wav"
	content := "fake wav content"

	// Put
	result, err := store.Put(ctx, storage.PutRequest{
		ObjectKey:   key,
		ContentType: "audio/wav",
		SizeBytes:   int64(len(content)),
		Reader:      strings.NewReader(content),
	})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if result.ObjectKey != key {
		t.Fatalf("expected key %s, got %s", key, result.ObjectKey)
	}
	if result.SizeBytes != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), result.SizeBytes)
	}

	// Stat
	meta, err := store.Stat(ctx, key)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if meta.Key != key {
		t.Fatalf("stat key: expected %s, got %s", key, meta.Key)
	}
	if meta.SizeBytes != int64(len(content)) {
		t.Fatalf("stat size: expected %d, got %d", len(content), meta.SizeBytes)
	}

	// GetURL
	url, err := store.GetURL(ctx, key)
	if err != nil {
		t.Fatalf("get url: %v", err)
	}
	expected := "/static/files/" + key
	if url != expected {
		t.Fatalf("expected url %s, got %s", expected, url)
	}

	// Delete
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Stat after delete should fail
	_, err = store.Stat(ctx, key)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestLocalStore_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewLocalStore(tmpDir, "/static/files")
	if err != nil {
		t.Fatalf("new local store: %v", err)
	}

	ctx := context.Background()

	// Attempt path traversal — should be sanitized and stored safely inside rootDir
	_, err = store.Put(ctx, storage.PutRequest{
		ObjectKey:   "../../../etc/passwd",
		ContentType: "text/plain",
		Reader:      strings.NewReader("safe"),
	})
	if err != nil {
		t.Fatalf("put should succeed (path sanitized): %v", err)
	}

	// The file should be stored inside tmpDir, not at /etc/passwd
	_, err = store.Stat(ctx, "etc/passwd")
	if err != nil {
		t.Fatal("expected file to be inside rootDir as etc/passwd")
	}
}

func TestLocalStore_DeleteNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewLocalStore(tmpDir, "/static/files")
	if err != nil {
		t.Fatalf("new local store: %v", err)
	}

	// Delete non-existent file should not error
	err = store.Delete(context.Background(), "does/not/exist.wav")
	if err != nil {
		t.Fatalf("expected no error for deleting non-existent file, got: %v", err)
	}
}

func TestLocalStore_StatNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewLocalStore(tmpDir, "/static/files")
	if err != nil {
		t.Fatalf("new local store: %v", err)
	}

	_, err = store.Stat(context.Background(), "does/not/exist.wav")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected os.ErrNotExist, got: %v", err)
	}
}
