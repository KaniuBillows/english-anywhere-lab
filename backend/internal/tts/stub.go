package tts

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
)

const stubDurationSec = 1 // 1 second of silence

// StubProvider generates minimal valid WAV silence for testing.
type StubProvider struct{}

// NewStubProvider creates a new StubProvider.
func NewStubProvider() *StubProvider {
	return &StubProvider{}
}

func (s *StubProvider) Synthesize(_ context.Context, req SynthesizeRequest) (*SynthesizeResult, error) {
	sampleRate := req.SampleRate
	if sampleRate == 0 {
		sampleRate = 22050
	}

	// 16-bit mono PCM silence
	numSamples := sampleRate * stubDurationSec
	dataSize := numSamples * 2 // 2 bytes per sample (16-bit)
	fileSize := 44 + dataSize  // WAV header + data

	buf := &bytes.Buffer{}
	buf.Grow(fileSize)

	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(fileSize-8))
	buf.WriteString("WAVE")

	// fmt subchunk
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))  // subchunk size
	binary.Write(buf, binary.LittleEndian, uint16(1))   // PCM format
	binary.Write(buf, binary.LittleEndian, uint16(1))   // mono
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate*2)) // byte rate
	binary.Write(buf, binary.LittleEndian, uint16(2))   // block align
	binary.Write(buf, binary.LittleEndian, uint16(16))  // bits per sample

	// data subchunk
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(dataSize))

	// Silence: zero bytes
	buf.Write(make([]byte, dataSize))

	data := buf.Bytes()
	return &SynthesizeResult{
		Audio:       io.NopCloser(bytes.NewReader(data)),
		DurationMs:  stubDurationSec * 1000,
		SizeBytes:   int64(len(data)),
		ContentType: "audio/wav",
	}, nil
}

func (s *StubProvider) Close() error {
	return nil
}
