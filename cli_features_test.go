package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func runText(t *testing.T, args ...string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := run(args, &buf); err != nil {
		t.Fatalf("run(%v): %v", args, err)
	}
	return buf.String()
}

func TestVersionFlag(t *testing.T) {
	out := runText(t, "--version")
	if !strings.Contains(out, "Chiquito de la Calzada") {
		t.Errorf("salida = %q", out)
	}
}

func TestListZonesFlag(t *testing.T) {
	out := runText(t, "--list-zones")
	for _, want := range []string{"Europe/Madrid", "America/Bogota", "Asia/Tokyo", "Europa:"} {
		if !strings.Contains(out, want) {
			t.Errorf("la lista no contiene %q", want)
		}
	}
}

func TestAgeCommand(t *testing.T) {
	out := runText(t, "--utc", "age", "2000-02-29", "2026-02-28")
	for _, want := range []string{"Edad: 26 años, 0 meses y 0 días", "¡Hoy cumple 26 años!", "29 de febrero"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
}

func TestFactsCommand(t *testing.T) {
	out := runText(t, "--utc", "facts", "2024-12-31")
	for _, want := range []string{"martes", "Semana ISO:", "1 (año ISO 2025)", "366 de 366", "T4", "Año bisiesto:     sí"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
}

func TestWorkdaysCommand(t *testing.T) {
	out := runText(t, "--utc", "--country", "es", "workdays", "2024-01-01", "2024-01-09")
	for _, want := range []string{"Días laborables: 5", "Días naturales: 8", "01/01/2024  Año Nuevo", "06/01/2024  Epifanía"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
}

func TestAddWorkdaysCommand(t *testing.T) {
	out := runText(t, "--utc", "addworkdays", "2024-01-05", "1")
	for _, want := range []string{"sumar 1 días laborables", "08/01/2024"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
}

func TestConvertCommand(t *testing.T) {
	out := runText(t, "convert", "2024-03-31T02:30", "Europe/Madrid", "UTC")
	for _, want := range []string{"2024-03-31T01:30:00Z", "no existe", "Aviso:"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
}

func TestICSCommand(t *testing.T) {
	out := runText(t, "--utc", "ics", "2026-11-18", "Cita importante")
	for _, want := range []string{"BEGIN:VCALENDAR", "SUMMARY:Cita importante", "DTSTART;VALUE=DATE:20261118"} {
		if !strings.Contains(out, want) {
			t.Errorf("el ICS no contiene %q:\n%s", want, out)
		}
	}
}

func TestDSTWarningInDiff(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	var buf bytes.Buffer
	if err := printDiff(&buf, "2024-03-31T02:30", "2024-03-31T12:00", options{zone: "Europe/Madrid"}, loc); err != nil {
		t.Fatalf("printDiff: %v", err)
	}
	if !strings.Contains(buf.String(), "Aviso:") || !strings.Contains(buf.String(), "no existe") {
		t.Errorf("falta el aviso DST:\n%s", buf.String())
	}

	buf.Reset()
	if err := printDiff(&buf, "2024-03-31T02:30", "2024-03-31T12:00", options{json: true, zone: "Europe/Madrid"}, loc); err != nil {
		t.Fatalf("printDiff JSON: %v", err)
	}
	if !strings.Contains(buf.String(), `"aviso"`) {
		t.Errorf("el JSON no incluye el aviso:\n%s", buf.String())
	}
}

func TestUsageExitCode(t *testing.T) {
	err := run([]string{"--nope"}, &bytes.Buffer{})
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != 2 {
		t.Fatalf("err = %v, quiero un error de uso con código 2", err)
	}
}

func TestMillisecondsInShift(t *testing.T) {
	var buf bytes.Buffer
	if err := printShift(&buf, "add", "2024-01-01T00:00:00", "250ms", options{}, time.UTC); err != nil {
		t.Fatalf("printShift: %v", err)
	}
	if !strings.Contains(buf.String(), "250 milisegundos") {
		t.Errorf("la salida no menciona los milisegundos:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "00:00:00.25") {
		t.Errorf("la salida no aplica los milisegundos:\n%s", buf.String())
	}
}
