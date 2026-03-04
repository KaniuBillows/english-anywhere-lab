package tts

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

var multiSpace = regexp.MustCompile(`\s+`)

// ObjectKey returns a deterministic storage key for a TTS synthesis result.
// Format: tts/en/{voice}/{format}/{sha256}.{ext}
func ObjectKey(text, voice string, speed float32, format string, sampleRate int) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = multiSpace.ReplaceAllString(normalized, " ")

	hashInput := fmt.Sprintf("%s|%s|%.2f|%s|%d", normalized, voice, speed, format, sampleRate)
	hash := sha256.Sum256([]byte(hashInput))

	ext := format
	if ext == "" {
		ext = "wav"
	}
	if voice == "" {
		voice = "default"
	}

	return fmt.Sprintf("tts/en/%s/%s/%x.%s", voice, format, hash, ext)
}
