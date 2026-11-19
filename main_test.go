package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSplitArgs(t *testing.T) {
	opts, pos, err := splitArgs([]string{"--utc", "2024-01-01", "-1mes", "--json"})
	if err != nil {
		t.Fatalf("splitArgs: %v", err)
	}
	if !opts.utc || !opts.json {
		t.Errorf("opciones = %+v, quiero utc y json", opts)
	}
	if len(pos) != 2 || pos[0] != "2024-01-01" || pos[1] != "-1mes" {
		t.Errorf("posicionales = %v", pos)
	}

	if _, _, err := splitArgs([]string{"--nope"}); err == nil {
		t.Error("una opción desconocida debería fallar")
	}
}

func TestSplitArgsZone(t *testing.T) {
	opts, pos, err := splitArgs([]string{"--tz", "Europe/Madrid", "2024-01-01"})
	if err != nil {
		t.Fatalf("splitArgs --tz: %v", err)
	}
	if opts.zone != "Europe/Madrid" || len(pos) != 1 {
		t.Errorf("opciones = %+v, pos = %v", opts, pos)
	}

	opts, _, err = splitArgs([]string{"--tz=America/Bogota"})
	if err != nil || opts.zone != "America/Bogota" {
		t.Errorf("--tz=: opciones = %+v, err = %v", opts, err)
	}

	if _, _, err := splitArgs([]string{"--tz"}); err == nil {
		t.Error("--tz sin valor debería fallar")
	}
	if _, _, err := splitArgs([]string{"--tz", "  "}); err == nil {
		t.Error("--tz vacío debería fallar")
	}
}

func TestResolveCLILocation(t *testing.T) {
	loc, err := resolveCLILocation(options{zone: "Europe/Madrid"})
	if err != nil || loc.String() != "Europe/Madrid" {
		t.Errorf("loc = %v, err = %v", loc, err)
	}

	if _, err := resolveCLILocation(options{utc: true, zone: "Europe/Madrid"}); err == nil {
		t.Error("--utc y --tz juntos deberían fallar")
	}

	if _, err := resolveCLILocation(options{zone: "Marte/Olympus"}); err == nil {
		t.Error("una zona desconocida debería fallar")
	}

	loc, err = resolveCLILocation(options{zone: "America/Bogota"})
	if err != nil || loc.String() != "America/Bogota" {
		t.Errorf("Bogotá: loc = %v, err = %v", loc, err)
	}
}

func TestDiffTZOutput(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}

	var buf bytes.Buffer
	if err := printDiff(&buf, "2024-03-30T12:00", "2024-04-01T12:00", options{zone: "Europe/Madrid"}, loc); err != nil {
		t.Fatalf("printDiff --tz: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Zona: Europe/Madrid (CET, UTC+01:00)",
		"Días naturales: 2",
		"47",
		"1,96",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}
}

func TestDiffOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := printDiff(&buf, "2024-01-01", "2024-01-02", options{}, time.UTC); err != nil {
		t.Fatalf("printDiff: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"1 día",
		"Días naturales: 1",
		"La fecha 2 es posterior a la fecha 1.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}

	buf.Reset()
	if err := printDiff(&buf, "2024-01-01", "2024-01-02", options{inclusive: true}, time.UTC); err != nil {
		t.Fatalf("printDiff inclusive: %v", err)
	}
	if !strings.Contains(buf.String(), "ambos extremos incluidos): 2") {
		t.Errorf("la salida inclusive no cuenta 2 días:\n%s", buf.String())
	}
}

func TestDiffJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := printDiff(&buf, "2024-01-01", "2024-01-02", options{json: true}, time.UTC); err != nil {
		t.Fatalf("printDiff JSON: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		`"dias_naturales": 1`,
		`"dias": 1`,
		`"posterior": "fecha2"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("el JSON no contiene %q:\n%s", want, out)
		}
	}
}

func TestShiftOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := printShift(&buf, "add", "2024-01-31", "1mes", options{}, time.UTC); err != nil {
		t.Fatalf("printShift: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Operación: sumar 1 mes",
		"29/02/2024",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, out)
		}
	}

	buf.Reset()
	if err := printShift(&buf, "add", "2024-01-31", "-1mes", options{}, time.UTC); err != nil {
		t.Fatalf("printShift negativo: %v", err)
	}
	if !strings.Contains(buf.String(), "Operación: restar 1 mes") {
		t.Errorf("un delta negativo debería restar:\n%s", buf.String())
	}
}

func TestInteractiveLoop(t *testing.T) {
	oldStdin := stdin
	defer func() { stdin = oldStdin }()

	// Dos consultas seguidas y luego salir con "n".
	stdin = strings.NewReader("2024-01-01\n2024-01-02\ns\n2024-01-03\n2024-01-04\nn\n")

	var buf bytes.Buffer
	if err := run(nil, &buf); err != nil {
		t.Fatalf("run interactivo: %v", err)
	}
	out := buf.String()
	if got := strings.Count(out, "Días naturales:"); got != 2 {
		t.Errorf("esperaba 2 consultas, hay %d:\n%s", got, out)
	}
	if !strings.Contains(out, "¿Otra consulta?") {
		t.Errorf("falta la pausa final:\n%s", out)
	}
}

func TestInteractiveSalir(t *testing.T) {
	oldStdin := stdin
	defer func() { stdin = oldStdin }()

	stdin = strings.NewReader("salir\n")

	var buf bytes.Buffer
	if err := run(nil, &buf); err != nil {
		t.Fatalf("run interactivo: %v", err)
	}
	if !strings.Contains(buf.String(), "¡Hasta luego!") {
		t.Errorf("no se despidió:\n%s", buf.String())
	}
}
