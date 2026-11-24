package datecore

import (
	"strings"
	"testing"
	"time"
)

func FuzzParseDate(f *testing.F) {
	for _, seed := range []string{
		"2026-11-18",
		"18/11/2026 14:30",
		"2026-11-18T14:30:00+01:00",
		"hoy",
		"mañana",
		"en 3 días",
		"el próximo lunes",
		"18 de noviembre de 2026",
		"",
		"31/02/2026",
		"0000-01-01",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		got, err := ParseDate(raw, time.UTC)
		if err != nil {
			return
		}
		want := got.Truncate(time.Second)
		if want.Year() < 1 || want.Year() > 9998 {
			return
		}
		again, err := ParseDate(want.Format("2006-01-02T15:04:05Z07:00"), time.UTC)
		if err != nil {
			t.Fatalf("ida y vuelta de %q (%v): %v", raw, want, err)
		}
		if !again.Equal(want) {
			t.Fatalf("ida y vuelta de %q: %v != %v", raw, again, want)
		}
	})
}

func FuzzParseDelta(f *testing.F) {
	for _, seed := range []string{
		"1 mes",
		"1a2mes3sem4d5h6min7s250ms",
		"-2 días",
		"45min",
		"0",
		"1y 2mo 3w",
		"",
		"999999999999999999999999d",
		"1 mes 2",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		d, err := ParseDelta(raw)
		if err != nil {
			return
		}
		// Aplicar la duración no debe desbordar el calendario ni entrar en pánico.
		base := time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)
		_ = ApplyDelta(base, d.Abs(), 1)
		_ = ApplyDelta(base, d.Abs(), -1)
	})
}

func FuzzResolveLocation(f *testing.F) {
	for _, seed := range []string{"UTC", "Europe/Madrid", "America/Bogota", "local", "", "Nowhere/Invalid"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, name string) {
		loc, err := ResolveLocation(name)
		if err != nil {
			return
		}
		if loc == nil {
			t.Fatalf("ResolveLocation(%q) devolvió nil sin error", name)
		}
		if _, offset := time.Now().In(loc).Zone(); offset < -14*3600 || offset > 14*3600 {
			t.Fatalf("offset fuera de rango para %q: %d", name, offset)
		}
	})
}

func FuzzBuildICS(f *testing.F) {
	for _, seed := range []string{
		"Cita",
		"Línea con\nsalto",
		"Punto y coma; coma, barra \\ y retorno\r",
		strings.Repeat("ñ", 120),
		"",
	} {
		f.Add(seed, seed, seed)
	}
	f.Fuzz(func(t *testing.T, summary, description, location string) {
		start := time.Date(2026, time.November, 18, 14, 30, 0, 0, time.UTC)
		ics := BuildICS([]ICSEvent{{
			Summary:     summary,
			Description: description,
			Location:    location,
			Start:       start,
			End:         start.Add(time.Hour),
		}}, start)

		if !strings.HasPrefix(ics, "BEGIN:VCALENDAR\r\n") || !strings.HasSuffix(ics, "END:VCALENDAR\r\n") {
			t.Fatalf("calendario sin envoltura:\n%q", ics)
		}
		for _, line := range strings.Split(strings.TrimSuffix(ics, "\r\n"), "\r\n") {
			if len(line) > 75 {
				t.Fatalf("línea de %d octetos: %q", len(line), line)
			}
			if strings.ContainsAny(line, "\r\n") {
				t.Fatalf("salto de línea sin plegar: %q", line)
			}
		}
	})
}
