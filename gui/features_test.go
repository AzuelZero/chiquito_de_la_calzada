//go:build gui

package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestFactsMethod(t *testing.T) {
	app := &App{}
	got, err := app.Facts("2024-12-31", "UTC")
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	if got.ISOWeek != 1 || got.ISOYear != 2025 {
		t.Errorf("semana ISO = %d/%d, quiero 1/2025", got.ISOWeek, got.ISOYear)
	}
	if got.DayOfYear != 366 || !got.LeapYear || got.DaysLeftInYear != 0 {
		t.Errorf("Facts = %+v", got)
	}
	if got.Weekday != "martes" || got.Quarter != 4 {
		t.Errorf("Facts = %+v", got)
	}
}

func TestAgeMethod(t *testing.T) {
	app := &App{}
	got, err := app.Age("2000-02-29", "2026-02-28", "UTC")
	if err != nil {
		t.Fatalf("Age: %v", err)
	}
	if got.Years != 26 || got.Months != 0 || got.Days != 0 {
		t.Errorf("edad = %+v, quiero 26 años", got)
	}
	if !got.IsBirthday || !got.LeapDayAdjustment || got.Turning != 26 {
		t.Errorf("Age = %+v", got)
	}
	if got.TotalDays != 9496 {
		t.Errorf("TotalDays = %d, quiero 9496", got.TotalDays)
	}
	if !strings.Contains(got.Warning, "29 de febrero") {
		t.Errorf("Warning = %q", got.Warning)
	}

	if _, err := app.Age("2030-01-01", "2026-01-01", "UTC"); err == nil {
		t.Error("una fecha futura debería fallar")
	}
}

func TestWorkdaysMethod(t *testing.T) {
	app := &App{}

	got, err := app.Workdays("2024-01-01T09:00", "2024-01-09T09:00", "UTC", "", nil, false)
	if err != nil {
		t.Fatalf("Workdays: %v", err)
	}
	if got.Workdays != 6 {
		t.Errorf("Workdays sin festivos = %d, quiero 6", got.Workdays)
	}

	got, err = app.Workdays("2024-01-01T09:00", "2024-01-09T09:00", "UTC", "es", nil, false)
	if err != nil {
		t.Fatalf("Workdays ES: %v", err)
	}
	if got.Workdays != 5 {
		t.Errorf("Workdays ES = %d, quiero 5 (Año Nuevo; la Epifanía cae en sábado)", got.Workdays)
	}
	if len(got.Holidays) != 2 {
		t.Errorf("festivos = %+v, quiero 2", got.Holidays)
	}

	got, err = app.Workdays("2024-01-01T09:00", "2024-01-09T09:00", "UTC", "", []string{"2024-01-02"}, false)
	if err != nil {
		t.Fatalf("Workdays manual: %v", err)
	}
	if got.Workdays != 5 {
		t.Errorf("Workdays con festivo manual = %d, quiero 5", got.Workdays)
	}
}

func TestAddWorkdaysMethod(t *testing.T) {
	app := &App{}

	got, err := app.AddWorkdays("2024-01-05T09:00", 1, "UTC", "", nil)
	if err != nil {
		t.Fatalf("AddWorkdays: %v", err)
	}
	if !strings.Contains(got.ResultHuman, "08/01/2024") {
		t.Errorf("ResultHuman = %q, quiero el lunes 8", got.ResultHuman)
	}
	if got.Operation != "sumar" || got.Workdays != 1 {
		t.Errorf("resultado = %+v", got)
	}

	got, err = app.AddWorkdays("2024-01-08T09:00", -1, "UTC", "", nil)
	if err != nil {
		t.Fatalf("AddWorkdays negativo: %v", err)
	}
	if got.Operation != "restar" || !strings.Contains(got.ResultHuman, "05/01/2024") {
		t.Errorf("resultado = %+v", got)
	}
}

func TestConvertMethod(t *testing.T) {
	app := &App{}

	got, err := app.Convert("2024-03-31T02:30", "Europe/Madrid", "UTC")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got.ToISO != "2024-03-31T01:30:00Z" {
		t.Errorf("ToISO = %q, quiero 2024-03-31T01:30:00Z", got.ToISO)
	}
	if !strings.Contains(got.Warning, "no existe") {
		t.Errorf("Warning = %q", got.Warning)
	}

	input, err := app.ConvertInput("2024-03-31T12:00", "Europe/Madrid", "America/Bogota")
	if err != nil {
		t.Fatalf("ConvertInput: %v", err)
	}
	if input != "2024-03-31T05:00:00" {
		t.Errorf("ConvertInput = %q, quiero 2024-03-31T05:00:00", input)
	}
}

func TestEventICSMethod(t *testing.T) {
	app := &App{}

	got, err := app.EventICS("2026-11-18T14:30:00Z", "Europe/Madrid", "Cita", "Prueba", false)
	if err != nil {
		t.Fatalf("EventICS: %v", err)
	}
	if got.Filename != "chiquito-2026-11-18.ics" {
		t.Errorf("Filename = %q", got.Filename)
	}
	if !strings.Contains(got.Content, "SUMMARY:Cita") || !strings.Contains(got.Content, "DTSTART:20261118T143000Z") {
		t.Errorf("ICS inesperado:\n%s", got.Content)
	}

	allDay, err := app.EventICS("2026-11-18T14:30:00Z", "Europe/Madrid", "Cita", "", true)
	if err != nil {
		t.Fatalf("EventICS all-day: %v", err)
	}
	if !strings.Contains(allDay.Content, "DTSTART;VALUE=DATE:20261118") {
		t.Errorf("ICS de día completo:\n%s", allDay.Content)
	}
}

func TestCountriesMethod(t *testing.T) {
	app := &App{}
	got := app.Countries()
	if len(got) != 5 {
		t.Fatalf("Countries = %d, quiero 5", len(got))
	}
	if got[0].ID != "" {
		t.Errorf("la primera opción debería ser sin festivos: %+v", got[0])
	}
}

func TestFrontendIDsExist(t *testing.T) {
	js, err := os.ReadFile("frontend/dist/app.js")
	if err != nil {
		t.Fatalf("leer app.js: %v", err)
	}
	html, err := os.ReadFile("frontend/dist/index.html")
	if err != nil {
		t.Fatalf("leer index.html: %v", err)
	}
	idPattern := regexp.MustCompile(`\$\("([A-Za-z0-9_-]+)"\)`)
	seen := map[string]bool{}
	for _, match := range idPattern.FindAllStringSubmatch(string(js), -1) {
		id := match[1]
		if seen[id] {
			continue
		}
		seen[id] = true
		if !strings.Contains(string(html), `id="`+id+`"`) {
			t.Errorf("app.js usa #%s, pero no existe en index.html", id)
		}
	}
}

func TestFrontendAPIMethodsExist(t *testing.T) {
	js, err := os.ReadFile("frontend/dist/app.js")
	if err != nil {
		t.Fatalf("leer app.js: %v", err)
	}
	goFile, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("leer main.go: %v", err)
	}
	methodPattern := regexp.MustCompile(`api\(\)\.([A-Za-z0-9_]+)\(`)
	declPattern := regexp.MustCompile(`func \(a \*App\) ([A-Za-z0-9_]+)\(`)
	declared := map[string]bool{}
	for _, match := range declPattern.FindAllStringSubmatch(string(goFile), -1) {
		declared[match[1]] = true
	}
	seen := map[string]bool{}
	for _, match := range methodPattern.FindAllStringSubmatch(string(js), -1) {
		name := match[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		if !declared[name] {
			t.Errorf("app.js llama a api().%s, pero no existe en gui/main.go", name)
		}
	}
}
