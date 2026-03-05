package tts

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bennyshi/english-anywhere-lab/internal/storage"
)

// TTSConfig holds TTS service configuration.
type TTSConfig struct {
	Voice        string
	Speed        float32
	Format       string
	SampleRate   int
	MaxTextChars int
}

// Service orchestrates TTS generation, storage, and dedup.
type Service struct {
	provider     TTSProvider
	store        storage.ObjectStore
	voice        string
	speed        float32
	format       string
	sampleRate   int
	maxTextChars int
}

// NewService creates a new TTS service.
func NewService(provider TTSProvider, store storage.ObjectStore, cfg TTSConfig) *Service {
	voice := cfg.Voice
	if voice == "" {
		voice = "en_default_female"
	}
	format := cfg.Format
	if format == "" {
		format = "wav"
	}
	speed := cfg.Speed
	if speed == 0 {
		speed = 1.0
	}
	sampleRate := cfg.SampleRate
	if sampleRate == 0 {
		sampleRate = 22050
	}
	maxTextChars := cfg.MaxTextChars
	if maxTextChars == 0 {
		maxTextChars = 280
	}
	return &Service{
		provider:     provider,
		store:        store,
		voice:        voice,
		speed:        speed,
		format:       format,
		sampleRate:   sampleRate,
		maxTextChars: maxTextChars,
	}
}

// SynthesizeAndStore generates audio for the given text, stores it, and returns
// the URL. If audio for this text already exists (dedup), it returns the
// existing URL without re-synthesizing.
func (s *Service) SynthesizeAndStore(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", errors.New("text is empty")
	}
	if len(text) > s.maxTextChars {
		return "", fmt.Errorf("text exceeds max length (%d > %d)", len(text), s.maxTextChars)
	}

	key := ObjectKey(text, s.voice, s.speed, s.format, s.sampleRate)

	// Dedup check: if object already exists, return its URL
	_, err := s.store.Stat(ctx, key)
	if err == nil {
		return s.store.GetURL(ctx, key)
	}
	if !errors.Is(err, os.ErrNotExist) && !os.IsNotExist(err) {
		// For non-local stores, Stat may return a different error for "not found".
		// We treat any Stat error as "not found" and proceed with synthesis.
	}

	result, err := s.provider.Synthesize(ctx, SynthesizeRequest{
		Text:       text,
		Voice:      s.voice,
		Speed:      s.speed,
		Format:     s.format,
		SampleRate: s.sampleRate,
	})
	if err != nil {
		return "", fmt.Errorf("synthesize: %w", err)
	}
	defer result.Audio.Close()

	_, err = s.store.Put(ctx, storage.PutRequest{
		ObjectKey:   key,
		ContentType: result.ContentType,
		SizeBytes:   result.SizeBytes,
		Reader:      result.Audio,
	})
	if err != nil {
		return "", fmt.Errorf("store audio: %w", err)
	}

	return s.store.GetURL(ctx, key)
}
