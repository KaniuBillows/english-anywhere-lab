package tts

import (
	"context"
	"io"
)

// SynthesizeRequest holds the parameters for a TTS synthesis call.
type SynthesizeRequest struct {
	Text       string
	Voice      string
	Speed      float32
	Format     string // "wav"
	SampleRate int
}

// SynthesizeResult holds the result of a TTS synthesis call.
type SynthesizeResult struct {
	Audio       io.ReadCloser
	DurationMs  int
	SizeBytes   int64
	ContentType string
}

// TTSProvider generates audio from text.
type TTSProvider interface {
	Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResult, error)
	Close() error
}
