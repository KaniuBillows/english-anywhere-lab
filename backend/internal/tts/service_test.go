package tts_test

import (
	"context"
	"strings"
	"testing"

	"github.com/bennyshi/english-anywhere-lab/internal/storage"
	"github.com/bennyshi/english-anywhere-lab/internal/tts"
)

func TestService_SynthesizeAndStore(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewLocalStore(tmpDir, "/static/files")
	if err != nil {
		t.Fatalf("new local store: %v", err)
	}

	svc := tts.NewService(tts.NewStubProvider(), store, tts.TTSConfig{
		Voice:        "en_default_female",
		Speed:        1.0,
		Format:       "wav",
		SampleRate:   22050,
		MaxTextChars: 280,
	})

	ctx := context.Background()

	// First synthesis
	url, err := svc.SynthesizeAndStore(ctx, "hello world")
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if !strings.HasPrefix(url, "/static/files/tts/en/") {
		t.Fatalf("unexpected url: %s", url)
	}
	if !strings.HasSuffix(url, ".wav") {
		t.Fatalf("url should end with .wav: %s", url)
	}

	// Second synthesis (dedup) — should return same URL
	url2, err := svc.SynthesizeAndStore(ctx, "hello world")
	if err != nil {
		t.Fatalf("synthesize dedup: %v", err)
	}
	if url != url2 {
		t.Fatalf("dedup failed: url1=%s url2=%s", url, url2)
	}
}

func TestService_SynthesizeAndStore_Dedup(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewLocalStore(tmpDir, "/static/files")
	if err != nil {
		t.Fatalf("new local store: %v", err)
	}

	callCount := 0
	provider := &countingProvider{inner: tts.NewStubProvider(), count: &callCount}

	svc := tts.NewService(provider, store, tts.TTSConfig{
		Voice:        "en_default_female",
		Speed:        1.0,
		Format:       "wav",
		SampleRate:   22050,
		MaxTextChars: 280,
	})

	ctx := context.Background()

	_, err = svc.SynthesizeAndStore(ctx, "test dedup")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}

	_, err = svc.SynthesizeAndStore(ctx, "test dedup")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("dedup should prevent second Synthesize call, got %d", callCount)
	}
}

func TestService_EmptyText(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewLocalStore(tmpDir, "/static/files")
	if err != nil {
		t.Fatalf("new local store: %v", err)
	}

	svc := tts.NewService(tts.NewStubProvider(), store, tts.TTSConfig{MaxTextChars: 280})

	_, err = svc.SynthesizeAndStore(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestService_TextTooLong(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewLocalStore(tmpDir, "/static/files")
	if err != nil {
		t.Fatalf("new local store: %v", err)
	}

	svc := tts.NewService(tts.NewStubProvider(), store, tts.TTSConfig{MaxTextChars: 10})

	_, err = svc.SynthesizeAndStore(context.Background(), "this text is way too long")
	if err == nil {
		t.Fatal("expected error for text too long")
	}
}

// countingProvider wraps a TTSProvider and counts Synthesize calls.
type countingProvider struct {
	inner tts.TTSProvider
	count *int
}

func (c *countingProvider) Synthesize(ctx context.Context, req tts.SynthesizeRequest) (*tts.SynthesizeResult, error) {
	*c.count++
	return c.inner.Synthesize(ctx, req)
}

func (c *countingProvider) Close() error {
	return c.inner.Close()
}
