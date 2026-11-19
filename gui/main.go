//go:build gui

// Chiquito de la Calzada: interfaz de escritorio, construida con Wails.
package main

import (
	"embed"
	"fmt"
	"math"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"datecalc/datecore"
)

//go:embed all:frontend/dist
var assets embed.FS

// App expone la lógica de fechas al frontend.
type App struct{}

type breakdownView struct {
	Years   int `json:"years"`
	Months  int `json:"months"`
	Days    int `json:"days"`
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
}

// DiffResult es el resultado que consume la interfaz.
type DiffResult struct {
	Human1       string        `json:"human1"`
	Human2       string        `json:"human2"`
	Later        string        `json:"later"`
	Breakdown    breakdownView `json:"breakdown"`
	NaturalDays  int64         `json:"naturalDays"`
	Days24       float64       `json:"days24"`
	Weeks        float64       `json:"weeks"`
	MonthsTotal  float64       `json:"monthsTotal"`
	YearsTotal   float64       `json:"yearsTotal"`
	Hours        float64       `json:"hours"`
	Minutes      float64       `json:"minutes"`
	Seconds      float64       `json:"seconds"`
	Milliseconds float64       `json:"milliseconds"`
	Warning      string        `json:"warning"`
}

// ShiftResult es el resultado de sumar o restar una duración.
type ShiftResult struct {
	Operation   string `json:"operation"`
	Duration    string `json:"duration"`
	StartHuman  string `json:"startHuman"`
	ResultHuman string `json:"resultHuman"`
	ISO         string `json:"iso"`
	Warning     string `json:"warning"`
}

// ZoneInfo describe una zona del selector.
type ZoneInfo struct {
	ID     string `json:"id"`
	City   string `json:"city"`
	Group  string `json:"group"`
	Abbr   string `json:"abbr"`
	Offset string `json:"offset"`
	DST    bool   `json:"dst"`
}

// ZoneState describe la zona activa en este momento.
type ZoneState struct {
	ID     string `json:"id"`
	City   string `json:"city"`
	Abbr   string `json:"abbr"`
	Offset string `json:"offset"`
	DST    bool   `json:"dst"`
	Text   string `json:"text"`
}

type zoneEntry struct {
	id    string
	city  string
	group string
}

// zoneCatalog es una selección de zonas IANA habituales. Se puede ampliar sin
// tocar el frontend: el selector se construye a partir de esta lista.
var zoneCatalog = []zoneEntry{
	{"local", "Hora local", "Sistema"},
	{"UTC", "UTC", "Sistema"},
	{"Europe/Madrid", "Madrid", "Europa"},
	{"Atlantic/Canary", "Canarias", "Europa"},
	{"Europe/London", "Londres", "Europa"},
	{"Europe/Paris", "París", "Europa"},
	{"Europe/Berlin", "Berlín", "Europa"},
	{"Europe/Rome", "Roma", "Europa"},
	{"Europe/Lisbon", "Lisboa", "Europa"},
	{"America/New_York", "Nueva York", "América"},
	{"America/Chicago", "Chicago", "América"},
	{"America/Denver", "Denver", "América"},
	{"America/Los_Angeles", "Los Ángeles", "América"},
	{"America/Mexico_City", "Ciudad de México", "América"},
	{"America/Bogota", "Bogotá", "América"},
	{"America/Lima", "Lima", "América"},
	{"America/Santiago", "Santiago", "América"},
	{"America/Argentina/Buenos_Aires", "Buenos Aires", "América"},
	{"America/Sao_Paulo", "São Paulo", "América"},
	{"Asia/Tokyo", "Tokio", "Asia"},
	{"Asia/Shanghai", "Shanghái", "Asia"},
	{"Asia/Kolkata", "Nueva Delhi", "Asia"},
	{"Asia/Dubai", "Dubái", "Asia"},
	{"Australia/Sydney", "Sídney", "Oceanía"},
	{"Pacific/Auckland", "Auckland", "Oceanía"},
	{"Africa/Casablanca", "Casablanca", "África"},
	{"Africa/Johannesburg", "Johannesburgo", "África"},
}

func resolveLocation(name string) (*time.Location, error) {
	switch name {
	case "", "local":
		return time.Local, nil
	case "UTC":
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("zona horaria desconocida: %q", name)
	}
	return loc, nil
}

func formatOffset(seconds int) string {
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	return fmt.Sprintf("UTC%s%02d:%02d", sign, seconds/3600, (seconds%3600)/60)
}

func zoneState(id, city string, loc *time.Location, at time.Time) ZoneState {
	t := at.In(loc)
	abbr, offset := t.Zone()
	text := abbr + " · " + formatOffset(offset)
	if t.IsDST() {
		text += " · verano"
	}
	return ZoneState{
		ID:     id,
		City:   city,
		Abbr:   abbr,
		Offset: formatOffset(offset),
		DST:    t.IsDST(),
		Text:   text,
	}
}

// Zones devuelve las zonas disponibles con su offset actual.
func (a *App) Zones() []ZoneInfo {
	now := time.Now()
	zones := make([]ZoneInfo, 0, len(zoneCatalog))
	for _, entry := range zoneCatalog {
		loc, err := resolveLocation(entry.id)
		if err != nil {
			continue
		}
		state := zoneState(entry.id, entry.city, loc, now)
		zones = append(zones, ZoneInfo{
			ID:     entry.id,
			City:   entry.city,
			Group:  entry.group,
			Abbr:   state.Abbr,
			Offset: state.Offset,
			DST:    state.DST,
		})
	}
	return zones
}

// Zone devuelve el estado actual de una zona concreta.
func (a *App) Zone(id string) (ZoneState, error) {
	loc, err := resolveLocation(id)
	if err != nil {
		return ZoneState{}, err
	}
	city := id
	for _, entry := range zoneCatalog {
		if entry.id == id {
			city = entry.city
			break
		}
	}
	return zoneState(id, city, loc, time.Now()), nil
}

// Diff calcula la diferencia entre dos fechas en la zona indicada.
func (a *App) Diff(date1, date2, zone string, inclusive bool) (DiffResult, error) {
	loc, err := resolveLocation(zone)
	if err != nil {
		return DiffResult{}, err
	}
	d1, err := datecore.ParseDate(date1, loc)
	if err != nil {
		return DiffResult{}, err
	}
	d2, err := datecore.ParseDate(date2, loc)
	if err != nil {
		return DiffResult{}, err
	}

	bd := datecore.DiffBreakdown(d1, d2)

	sec1 := float64(d1.Unix()) + float64(d1.Nanosecond())/1e9
	sec2 := float64(d2.Unix()) + float64(d2.Nanosecond())/1e9
	absSec := math.Abs(sec2 - sec1)
	totals := datecore.ComputeTotals(absSec)

	natural := datecore.CivilDays(d1, d2)
	if natural < 0 {
		natural = -natural
	}
	if inclusive {
		natural++
	}

	later := ""
	if d1.Before(d2) {
		later = "date2"
	} else if d2.Before(d1) {
		later = "date1"
	}

	warnings := make([]string, 0, 2)
	if w := datecore.DSTWarning("Fecha 1", date1, d1, loc); w != "" {
		warnings = append(warnings, w)
	}
	if w := datecore.DSTWarning("Fecha 2", date2, d2, loc); w != "" {
		warnings = append(warnings, w)
	}

	return DiffResult{
		Human1: humanDate(d1),
		Human2: humanDate(d2),
		Later:  later,
		Breakdown: breakdownView{
			Years:   bd.Years,
			Months:  bd.Months,
			Days:    bd.Days,
			Hours:   bd.Hours,
			Minutes: bd.Minutes,
			Seconds: bd.Seconds,
		},
		NaturalDays:  natural,
		Days24:       datecore.Round(totals.Days24, 4),
		Weeks:        datecore.Round(totals.Weeks, 4),
		MonthsTotal:  datecore.Round(totals.Months, 4),
		YearsTotal:   datecore.Round(totals.Years, 4),
		Hours:        datecore.Round(totals.Hours, 4),
		Minutes:      datecore.Round(totals.Minutes, 4),
		Seconds:      datecore.Round(totals.Seconds, 4),
		Milliseconds: datecore.Round(totals.Milliseconds, 2),
		Warning:      strings.Join(warnings, "\n"),
	}, nil
}

// Shift suma (operation="add") o resta (operation="sub") una duración.
func (a *App) Shift(date, duration, operation, zone string) (ShiftResult, error) {
	loc, err := resolveLocation(zone)
	if err != nil {
		return ShiftResult{}, err
	}
	start, err := datecore.ParseDate(date, loc)
	if err != nil {
		return ShiftResult{}, err
	}
	d, err := datecore.ParseDelta(duration)
	if err != nil {
		return ShiftResult{}, err
	}

	sign := 1
	switch operation {
	case "sub":
		sign = -1
	case "add":
		sign = 1
	default:
		return ShiftResult{}, fmt.Errorf("operación no válida: %q", operation)
	}
	if d.Negative() {
		sign = -sign
		d = d.Abs()
	}

	result := datecore.ApplyDelta(start, d, sign)
	op := "sumar"
	if sign < 0 {
		op = "restar"
	}

	return ShiftResult{
		Operation:   op,
		Duration:    d.String(),
		StartHuman:  humanDate(start),
		ResultHuman: humanDate(result),
		ISO:         result.Format(time.RFC3339),
		Warning:     datecore.DSTWarning("Fecha base", date, start, loc),
	}, nil
}

// Now devuelve la fecha y hora actual en el formato de los inputs.
func (a *App) Now(zone string) (string, error) {
	loc, err := resolveLocation(zone)
	if err != nil {
		return "", err
	}
	return time.Now().In(loc).Format("2006-01-02T15:04"), nil
}

// humanDate formatea una fecha para mostrarla en la interfaz.
func humanDate(t time.Time) string {
	layout := "02/01/2006"
	if t.Hour() != 0 || t.Minute() != 0 {
		layout += " 15:04"
	}
	return t.Format(layout) + " (" + weekdayES(t.Weekday()) + ")"
}

func weekdayES(d time.Weekday) string {
	return [...]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}[d]
}

func main() {
	app := &App{}
	err := wails.Run(&options.App{
		Title:            "Chiquito de la Calzada — calculadora de fechas",
		Width:            1000,
		Height:           720,
		MinWidth:         720,
		MinHeight:        560,
		BackgroundColour: &options.RGBA{R: 47, G: 7, B: 52, A: 1},
		AssetServer:      &assetserver.Options{Assets: assets},
		Bind:             []interface{}{app},
	})
	if err != nil {
		panic(err)
	}
}
