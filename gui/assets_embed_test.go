//go:build gui

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestSoundClipsEmbedded comprueba que todos los sonidos viajan dentro del
// ejecutable y que son MP3 reales, no respuestas HTML del servidor.
func TestSoundClipsEmbedded(t *testing.T) {
	clips := []struct {
		name string
		min  int
	}{
		{"jarl.mp3", 10_000},
		{"ataque.mp3", 8_000},
		{"fuego.mp3", 8_000},
		{"camarero.mp3", 10_000},
		{"tecuen.mp3", 10_000},
		{"epetekan.mp3", 12_000},
	}
	for _, clip := range clips {
		data, err := assets.ReadFile("frontend/dist/assets/" + clip.name)
		if err != nil {
			t.Errorf("falta el audio %s: %v", clip.name, err)
			continue
		}
		if len(data) < clip.min {
			t.Errorf("%s es demasiado pequeño: %d bytes", clip.name, len(data))
			continue
		}
		if !bytes.HasPrefix(data, []byte("ID3")) {
			t.Errorf("%s no parece un MP3 (falta la cabecera ID3)", clip.name)
			continue
		}
		sum := sha256.Sum256(data)
		t.Logf("%s: %d bytes, sha256=%s", clip.name, len(data), hex.EncodeToString(sum[:]))
	}
}
