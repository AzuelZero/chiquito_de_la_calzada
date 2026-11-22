//go:build gui

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestJarlAudioEmbedded comprueba que el grito viaja dentro del ejecutable.
func TestJarlAudioEmbedded(t *testing.T) {
	data, err := assets.ReadFile("frontend/dist/assets/jarl.mp3")
	if err != nil {
		t.Fatalf("falta el audio del JARL: %v", err)
	}
	if len(data) < 10_000 {
		t.Fatalf("el audio del JARL es demasiado pequeño: %d bytes", len(data))
	}
	sum := sha256.Sum256(data)
	t.Logf("jarl.mp3: %d bytes, sha256=%s", len(data), hex.EncodeToString(sum[:]))
}
