//go:build gui

package main

import (
	"strings"
	"testing"
	"time"
)

func TestDiffNaturalDaysAndTotals(t *testing.T) {
	app := &App{}

	got, err := app.Diff("2024-01-01T00:00", "2024-01-31T00:00", "UTC", false)
	if err != nil {
		t.Fatalf("Diff devolvió error: %v", err)
	}

	if got.NaturalDays != 30 {
		t.Errorf("NaturalDays = %d, esperaba 30", got.NaturalDays)
	}
	if got.Days24 != 30 {
		t.Errorf("Days24 = %v, esperaba 30", got.Days24)
	}
	if got.Later != "date2" {
		t.Errorf("Later = %q, esperaba date2", got.Later)
	}
	if got.Breakdown.Days != 30 {
		t.Errorf("Breakdown.Days = %d, esperaba 30", got.Breakdown.Days)
	}
	if got.Hours != 720 {
		t.Errorf("Hours = %v, esperaba 720", got.Hours)
	}
	if !strings.Contains(got.Human1, "01/01/2024") {
		t.Errorf("Human1 = %q, esperaba contener 01/01/2024", got.Human1)
	}
}

func TestDiffInclusiveAndOrder(t *testing.T) {
	app := &App{}

	got, err := app.Diff("2024-05-10T08:00", "2024-05-01T08:00", "UTC", true)
	if err != nil {
		t.Fatalf("Diff devolvió error: %v", err)
	}

	if got.NaturalDays != 10 {
		t.Errorf("NaturalDays = %d, esperaba 10 (9 + extremo)", got.NaturalDays)
	}
	if got.Later != "date1" {
		t.Errorf("Later = %q, esperaba date1", got.Later)
	}
	if got.Breakdown.Days != 9 {
		t.Errorf("Breakdown.Days = %d, esperaba 9", got.Breakdown.Days)
	}
}

func TestDiffSameDate(t *testing.T) {
	app := &App{}

	got, err := app.Diff("2024-05-10T08:30", "2024-05-10T08:30", "UTC", false)
	if err != nil {
		t.Fatalf("Diff devolvió error: %v", err)
	}

	if got.Later != "" {
		t.Errorf("Later = %q, esperaba cadena vacía", got.Later)
	}
	if got.NaturalDays != 0 || got.Seconds != 0 {
		t.Errorf("esperaba diferencia cero, obtuve %d días / %v s", got.NaturalDays, got.Seconds)
	}
}

func TestDiffDSTSpringForward(t *testing.T) {
	app := &App{}

	// En Madrid, el 31/03/2024 se pasa de las 02:00 CET a las 03:00 CEST:
	// entre el 30 y el 1 hay 47 horas reales, pero 2 días naturales.
	got, err := app.Diff("2024-03-30T12:00", "2024-04-01T12:00", "Europe/Madrid", false)
	if err != nil {
		t.Fatalf("Diff devolvió error: %v", err)
	}

	if got.NaturalDays != 2 {
		t.Errorf("NaturalDays = %d, esperaba 2", got.NaturalDays)
	}
	if got.Breakdown.Days != 2 || got.Breakdown.Hours != 0 {
		t.Errorf("desglose = %+v, esperaba 2 días y 0 horas", got.Breakdown)
	}
	if got.Hours != 47 {
		t.Errorf("Hours = %v, esperaba 47", got.Hours)
	}
	if got.Seconds != 169200 {
		t.Errorf("Seconds = %v, esperaba 169200", got.Seconds)
	}
	if got.Days24 != 1.9583 {
		t.Errorf("Days24 = %v, esperaba 1.9583", got.Days24)
	}
}

func TestDiffDSTFallBack(t *testing.T) {
	app := &App{}

	// En Madrid, el 27/10/2024 se repite de las 03:00 a las 02:00 CET:
	// entre el 26 y el 28 hay 49 horas reales y 2 días naturales.
	got, err := app.Diff("2024-10-26T12:00", "2024-10-28T12:00", "Europe/Madrid", false)
	if err != nil {
		t.Fatalf("Diff devolvió error: %v", err)
	}

	if got.NaturalDays != 2 {
		t.Errorf("NaturalDays = %d, esperaba 2", got.NaturalDays)
	}
	if got.Hours != 49 {
		t.Errorf("Hours = %v, esperaba 49", got.Hours)
	}
	if got.Days24 != 2.0417 {
		t.Errorf("Days24 = %v, esperaba 2.0417", got.Days24)
	}
}

func TestShiftMonthsClampToLastDay(t *testing.T) {
	app := &App{}

	got, err := app.Shift("2024-01-31T10:00", "1 mes", "add", "UTC")
	if err != nil {
		t.Fatalf("Shift devolvió error: %v", err)
	}

	if got.Operation != "sumar" {
		t.Errorf("Operation = %q, esperaba sumar", got.Operation)
	}
	if got.ISO != "2024-02-29T10:00:00Z" {
		t.Errorf("ISO = %q, esperaba 2024-02-29T10:00:00Z", got.ISO)
	}
	if !strings.Contains(got.ResultHuman, "29/02/2024 10:00") {
		t.Errorf("ResultHuman = %q", got.ResultHuman)
	}
}

func TestShiftSubAndNegativeDuration(t *testing.T) {
	app := &App{}

	got, err := app.Shift("2024-03-31T10:00", "1 mes", "sub", "UTC")
	if err != nil {
		t.Fatalf("Shift devolvió error: %v", err)
	}
	if got.ISO != "2024-02-29T10:00:00Z" {
		t.Errorf("ISO = %q, esperaba 2024-02-29T10:00:00Z", got.ISO)
	}
	if got.Operation != "restar" {
		t.Errorf("Operation = %q, esperaba restar", got.Operation)
	}

	got, err = app.Shift("2024-03-10T10:00", "-2d", "add", "UTC")
	if err != nil {
		t.Fatalf("Shift devolvió error: %v", err)
	}
	if got.ISO != "2024-03-08T10:00:00Z" {
		t.Errorf("ISO = %q, esperaba 2024-03-08T10:00:00Z", got.ISO)
	}
	if got.Operation != "restar" {
		t.Errorf("Operation = %q, esperaba restar al invertir el signo", got.Operation)
	}
}

func TestShiftCalendarDayAcrossDST(t *testing.T) {
	app := &App{}

	// Un día natural conserva la hora de pared aunque ese día tenga 23 horas.
	got, err := app.Shift("2024-03-30T12:00", "1d", "add", "Europe/Madrid")
	if err != nil {
		t.Fatalf("Shift devolvió error: %v", err)
	}
	if got.ISO != "2024-03-31T12:00:00+02:00" {
		t.Errorf("ISO = %q, esperaba 2024-03-31T12:00:00+02:00", got.ISO)
	}

	// 24 horas absolutas sí atraviesan el cambio: 12:00 → 13:00.
	got, err = app.Shift("2024-03-30T12:00", "24h", "add", "Europe/Madrid")
	if err != nil {
		t.Fatalf("Shift devolvió error: %v", err)
	}
	if got.ISO != "2024-03-31T13:00:00+02:00" {
		t.Errorf("ISO = %q, esperaba 2024-03-31T13:00:00+02:00", got.ISO)
	}
}

func TestShiftRejectsInvalidOperation(t *testing.T) {
	app := &App{}

	if _, err := app.Shift("2024-03-10T10:00", "1d", "multiplicar", "UTC"); err == nil {
		t.Fatal("esperaba error con una operación no válida")
	}
}

func TestDSTWarningForNonexistentTime(t *testing.T) {
	app := &App{}

	got, err := app.Diff("2024-03-31T02:30", "2024-03-31T12:00", "Europe/Madrid", false)
	if err != nil {
		t.Fatalf("Diff devolvió error: %v", err)
	}
	if !strings.Contains(got.Warning, "no existe") || !strings.Contains(got.Warning, "03:30") {
		t.Errorf("Warning = %q, esperaba aviso de hora inexistente", got.Warning)
	}
	if !strings.Contains(got.Human1, "03:30") {
		t.Errorf("Human1 = %q, esperaba la hora normalizada 03:30", got.Human1)
	}

	ok, err := app.Diff("2024-03-30T12:00", "2024-04-01T12:00", "Europe/Madrid", false)
	if err != nil {
		t.Fatalf("Diff devolvió error: %v", err)
	}
	if ok.Warning != "" {
		t.Errorf("Warning = %q, esperaba sin aviso", ok.Warning)
	}
}

func TestDSTWarningForAmbiguousTime(t *testing.T) {
	app := &App{}

	// En Madrid, el 27/10/2024 a las 03:00 se retrasa el reloj a las 02:00:
	// las 02:30 existen dos veces. Go elige una de las dos ocurrencias.
	got, err := app.Diff("2024-10-27T02:30", "2024-10-27T12:00", "Europe/Madrid", false)
	if err != nil {
		t.Fatalf("Diff devolvió error: %v", err)
	}
	if !strings.Contains(got.Warning, "dos veces") || !strings.Contains(got.Warning, "se interpreta como") {
		t.Errorf("Warning = %q, esperaba aviso de hora ambigua", got.Warning)
	}
	if !strings.Contains(got.Human1, "02:30") {
		t.Errorf("Human1 = %q, esperaba la hora elegida 02:30", got.Human1)
	}

	ok, err := app.Diff("2024-10-27T05:00", "2024-10-27T12:00", "Europe/Madrid", false)
	if err != nil {
		t.Fatalf("Diff devolvió error: %v", err)
	}
	if ok.Warning != "" {
		t.Errorf("Warning = %q, esperaba sin aviso fuera de la hora repetida", ok.Warning)
	}
}

func TestZoneStateDSTLabels(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}

	winter := zoneState("Europe/Madrid", "Madrid", loc, time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC))
	if winter.Abbr != "CET" || winter.DST || winter.Offset != "UTC+01:00" {
		t.Errorf("invierno = %+v, esperaba CET / UTC+01:00 / sin DST", winter)
	}

	summer := zoneState("Europe/Madrid", "Madrid", loc, time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))
	if summer.Abbr != "CEST" || !summer.DST || summer.Offset != "UTC+02:00" {
		t.Errorf("verano = %+v, esperaba CEST / UTC+02:00 / DST", summer)
	}
}

func TestZonesCatalogAndZone(t *testing.T) {
	app := &App{}

	zones := app.Zones()
	if len(zones) < 20 {
		t.Fatalf("Zones devolvió %d zonas, esperaba al menos 20", len(zones))
	}

	found := map[string]bool{}
	for _, z := range zones {
		found[z.ID] = true
		if z.City == "" || z.Group == "" {
			t.Errorf("zona incompleta: %+v", z)
		}
	}
	for _, id := range []string{"local", "UTC", "Europe/Madrid", "America/Bogota", "Asia/Tokyo"} {
		if !found[id] {
			t.Errorf("falta la zona %q", id)
		}
	}

	state, err := app.Zone("Europe/Madrid")
	if err != nil {
		t.Fatalf("Zone devolvió error: %v", err)
	}
	if state.City != "Madrid" || state.ID != "Europe/Madrid" || state.Text == "" {
		t.Errorf("Zone = %+v", state)
	}

	if _, err := app.Zone("Marte/Olympus"); err == nil {
		t.Fatal("esperaba error con una zona desconocida")
	}
}

func TestNowZones(t *testing.T) {
	app := &App{}

	now, err := app.Now("UTC")
	if err != nil {
		t.Fatalf("Now(UTC): %v", err)
	}
	if len(now) != 16 || now[4] != '-' || now[10] != 'T' {
		t.Errorf("Now(UTC) = %q, esperaba formato 2006-01-02T15:04", now)
	}

	if _, err := app.Now("Europe/Madrid"); err != nil {
		t.Errorf("Now(Europe/Madrid): %v", err)
	}
	if _, err := app.Now("Marte/Olympus"); err == nil {
		t.Fatal("esperaba error con una zona desconocida")
	}
}
