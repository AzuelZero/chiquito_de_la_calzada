//go:build gui

// Chiquito de la Calzada: interfaz de escritorio, construida con Wails.
package main

import (
	"context"
	"embed"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"datecalc/datecore"
)

//go:embed all:frontend/dist
var assets embed.FS

// version se rellena en la compilación con -ldflags "-X main.version=...".
var version = "dev"

// App expone la lógica de fechas al frontend.
type App struct {
	ctx context.Context
}

// OnStartup guarda el contexto que necesitan los diálogos nativos.
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
}

type breakdownView struct {
	Years        int `json:"years"`
	Months       int `json:"months"`
	Days         int `json:"days"`
	Hours        int `json:"hours"`
	Minutes      int `json:"minutes"`
	Seconds      int `json:"seconds"`
	Milliseconds int `json:"milliseconds"`
}

// DiffResult es el resultado que consume la interfaz.
type DiffResult struct {
	Human1       string        `json:"human1"`
	Human2       string        `json:"human2"`
	ISO1         string        `json:"iso1"`
	ISO2         string        `json:"iso2"`
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

// FactsResult es la ficha de calendario de una fecha.
type FactsResult struct {
	Human           string `json:"human"`
	Long            string `json:"long"`
	ISO             string `json:"iso"`
	Weekday         string `json:"weekday"`
	ISOWeek         int    `json:"isoWeek"`
	ISOYear         int    `json:"isoYear"`
	DayOfYear       int    `json:"dayOfYear"`
	DaysInYear      int    `json:"daysInYear"`
	Quarter         int    `json:"quarter"`
	DaysInMonth     int    `json:"daysInMonth"`
	DaysLeftInMonth int    `json:"daysLeftInMonth"`
	DaysLeftInYear  int    `json:"daysLeftInYear"`
	LeapYear        bool   `json:"leapYear"`
}

// AgeView es la edad exacta y el próximo cumpleaños.
type AgeView struct {
	Years             int    `json:"years"`
	Months            int    `json:"months"`
	Days              int    `json:"days"`
	TotalDays         int64  `json:"totalDays"`
	NextBirthday      string `json:"nextBirthday"`
	NextBirthdayISO   string `json:"nextBirthdayISO"`
	DaysToNext        int64  `json:"daysToNext"`
	NextWeekday       string `json:"nextWeekday"`
	Turning           int    `json:"turning"`
	IsBirthday        bool   `json:"isBirthday"`
	LeapDayAdjustment bool   `json:"leapDayAdjustment"`
	Human             string `json:"human"`
	Warning           string `json:"warning"`
}

// HolidayView es un festivo mostrado en el resultado.
type HolidayView struct {
	Date string `json:"date"`
	Name string `json:"name"`
}

// WorkdaysResult es la diferencia en días laborables.
type WorkdaysResult struct {
	StartHuman  string        `json:"startHuman"`
	EndHuman    string        `json:"endHuman"`
	Workdays    int           `json:"workdays"`
	NaturalDays int64         `json:"naturalDays"`
	Holidays    []HolidayView `json:"holidays"`
	Country     string        `json:"country"`
}

// WorkShiftResult es el resultado de sumar o restar días laborables.
type WorkShiftResult struct {
	Operation   string        `json:"operation"`
	Workdays    int           `json:"workdays"`
	StartHuman  string        `json:"startHuman"`
	ResultHuman string        `json:"resultHuman"`
	ISO         string        `json:"iso"`
	Holidays    []HolidayView `json:"holidays"`
}

// ConvertResult es una hora de pared convertida entre dos zonas.
type ConvertResult struct {
	FromHuman string `json:"fromHuman"`
	ToHuman   string `json:"toHuman"`
	FromISO   string `json:"fromISO"`
	ToISO     string `json:"toISO"`
	FromZone  string `json:"fromZone"`
	ToZone    string `json:"toZone"`
	Warning   string `json:"warning"`
}

// CountryView describe un pack de festivos.
type CountryView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Note string `json:"note"`
}

// ICSResult es el calendario generado y un nombre de archivo sugerido.
type ICSResult struct {
	Content  string `json:"content"`
	Filename string `json:"filename"`
}

func resolveLocation(name string) (*time.Location, error) {
	return datecore.ResolveLocation(name)
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

func zoneCity(id string) string {
	for _, entry := range datecore.ZoneCatalog() {
		if entry.ID == id {
			return entry.City
		}
	}
	return id
}

// Version devuelve la versión con la que se compiló la aplicación.
func (a *App) Version() string {
	return version
}

// Zones devuelve las zonas disponibles con su offset actual.
func (a *App) Zones() []ZoneInfo {
	now := time.Now()
	catalog := datecore.ZoneCatalog()
	zones := make([]ZoneInfo, 0, len(catalog))
	for _, entry := range catalog {
		loc, err := resolveLocation(entry.ID)
		if err != nil {
			continue
		}
		state := zoneState(entry.ID, entry.City, loc, now)
		zones = append(zones, ZoneInfo{
			ID:     entry.ID,
			City:   entry.City,
			Group:  entry.Group,
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
	return zoneState(id, zoneCity(id), loc, time.Now()), nil
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
		ISO1:   d1.Format(time.RFC3339),
		ISO2:   d2.Format(time.RFC3339),
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

// Facts devuelve la ficha de calendario de una fecha.
func (a *App) Facts(date, zone string) (FactsResult, error) {
	loc, err := resolveLocation(zone)
	if err != nil {
		return FactsResult{}, err
	}
	t, err := datecore.ParseDate(date, loc)
	if err != nil {
		return FactsResult{}, err
	}
	f := datecore.Facts(t)
	return FactsResult{
		Human:           humanDate(t),
		Long:            datecore.LongDate(t),
		ISO:             t.Format(time.RFC3339),
		Weekday:         f.Weekday,
		ISOWeek:         f.ISOWeek,
		ISOYear:         f.ISOYear,
		DayOfYear:       f.DayOfYear,
		DaysInYear:      f.DaysInYear,
		Quarter:         f.Quarter,
		DaysInMonth:     f.DaysInMonth,
		DaysLeftInMonth: f.DaysLeftInMonth,
		DaysLeftInYear:  f.DaysLeftInYear,
		LeapYear:        f.LeapYear,
	}, nil
}

// Age calcula la edad y el próximo cumpleaños. Si at queda vacío, usa hoy.
func (a *App) Age(birth, at, zone string) (AgeView, error) {
	loc, err := resolveLocation(zone)
	if err != nil {
		return AgeView{}, err
	}
	b, err := datecore.ParseDate(birth, loc)
	if err != nil {
		return AgeView{}, err
	}
	if strings.TrimSpace(at) == "" {
		at = "hoy"
	}
	ref, err := datecore.ParseDate(at, loc)
	if err != nil {
		return AgeView{}, err
	}
	age, err := datecore.AgeAt(b, ref)
	if err != nil {
		return AgeView{}, err
	}
	warning := datecore.DSTWarning("Nacimiento", birth, b, loc)
	if w := datecore.DSTWarning("Fecha de referencia", at, ref, loc); w != "" {
		if warning != "" {
			warning += "\n"
		}
		warning += w
	}
	if age.LeapDayAdjustment {
		if warning != "" {
			warning += "\n"
		}
		warning += fmt.Sprintf("El 29 de febrero no existe en %d; el cumpleaños se cuenta el 28 de febrero.", age.NextBirthday.Year())
	}
	return AgeView{
		Years:             age.Years,
		Months:            age.Months,
		Days:              age.Days,
		TotalDays:         age.TotalDays,
		NextBirthday:      datecore.LongDate(age.NextBirthday),
		NextBirthdayISO:   age.NextBirthday.Format(time.RFC3339),
		DaysToNext:        age.DaysToNext,
		NextWeekday:       age.NextWeekday,
		Turning:           age.Turning,
		IsBirthday:        age.IsBirthday,
		LeapDayAdjustment: age.LeapDayAdjustment,
		Human:             fmt.Sprintf("%d años, %d meses y %d días", age.Years, age.Months, age.Days),
		Warning:           warning,
	}, nil
}

// Workdays cuenta los días laborables entre dos fechas.
func (a *App) Workdays(start, end, zone, country string, manual []string, inclusive bool) (WorkdaysResult, error) {
	loc, err := resolveLocation(zone)
	if err != nil {
		return WorkdaysResult{}, err
	}
	s, err := datecore.ParseDate(start, loc)
	if err != nil {
		return WorkdaysResult{}, err
	}
	e, err := datecore.ParseDate(end, loc)
	if err != nil {
		return WorkdaysResult{}, err
	}
	holidays, err := holidaySet(country, manual, s, e, loc)
	if err != nil {
		return WorkdaysResult{}, err
	}
	natural := datecore.CivilDays(s, e)
	if natural < 0 {
		natural = -natural
	}
	if inclusive {
		natural++
	}
	return WorkdaysResult{
		StartHuman:  humanDate(s),
		EndHuman:    humanDate(e),
		Workdays:    datecore.WorkdaysBetween(s, e, holidays, inclusive),
		NaturalDays: natural,
		Holidays:    holidayList(country, manual, s, e, loc),
		Country:     country,
	}, nil
}

// AddWorkdays suma o resta días laborables a una fecha.
func (a *App) AddWorkdays(date string, days int, zone, country string, manual []string) (WorkShiftResult, error) {
	loc, err := resolveLocation(zone)
	if err != nil {
		return WorkShiftResult{}, err
	}
	start, err := datecore.ParseDate(date, loc)
	if err != nil {
		return WorkShiftResult{}, err
	}
	holidays, err := holidaySet(country, manual, start, start, loc)
	if err != nil {
		return WorkShiftResult{}, err
	}
	result := datecore.AddWorkdays(start, days, holidays)
	op := "sumar"
	if days < 0 {
		op = "restar"
	}
	abs := days
	if abs < 0 {
		abs = -abs
	}
	return WorkShiftResult{
		Operation:   op,
		Workdays:    abs,
		StartHuman:  humanDate(start),
		ResultHuman: humanDate(result),
		ISO:         result.Format(time.RFC3339),
		Holidays:    holidayList(country, manual, start, result, loc),
	}, nil
}

// Convert interpreta una hora de pared en fromZone y la muestra en toZone.
func (a *App) Convert(date, fromZone, toZone string) (ConvertResult, error) {
	from, err := resolveLocation(fromZone)
	if err != nil {
		return ConvertResult{}, err
	}
	to, err := resolveLocation(toZone)
	if err != nil {
		return ConvertResult{}, err
	}
	t, err := datecore.ParseDate(date, from)
	if err != nil {
		return ConvertResult{}, err
	}
	converted := t.In(to)
	warning := datecore.DSTWarning("Hora de origen", date, t, from)
	return ConvertResult{
		FromHuman: humanDate(t),
		ToHuman:   humanDate(converted),
		FromISO:   t.Format(time.RFC3339),
		ToISO:     converted.Format(time.RFC3339),
		FromZone:  zoneState(fromZone, zoneCity(fromZone), from, t).Text,
		ToZone:    zoneState(toZone, zoneCity(toZone), to, converted).Text,
		Warning:   warning,
	}, nil
}

// ConvertInput convierte un valor datetime-local a otra zona y devuelve el
// mismo formato, para poder reescribir los campos al cambiar de zona.
func (a *App) ConvertInput(value, fromZone, toZone string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	from, err := resolveLocation(fromZone)
	if err != nil {
		return "", err
	}
	to, err := resolveLocation(toZone)
	if err != nil {
		return "", err
	}
	t, err := datecore.ParseDate(value, from)
	if err != nil {
		return "", err
	}
	return t.In(to).Format("2006-01-02T15:04:05"), nil
}

// Countries devuelve los packs de festivos disponibles.
func (a *App) Countries() []CountryView {
	out := []CountryView{{ID: "", Name: "Sin festivos", Note: "Solo fines de semana."}}
	for _, c := range datecore.Countries() {
		out = append(out, CountryView{ID: c.ID, Name: c.Name, Note: c.Note})
	}
	return out
}

// EventICS genera un evento de calendario para una fecha.
func (a *App) EventICS(startISO, zone, title, description string, allDay bool) (ICSResult, error) {
	loc, err := resolveLocation(zone)
	if err != nil {
		return ICSResult{}, err
	}
	start, err := time.Parse(time.RFC3339, startISO)
	if err != nil {
		return ICSResult{}, fmt.Errorf("fecha ISO no válida: %w", err)
	}
	start = start.In(loc)
	if title == "" {
		title = "Chiquito de la Calzada"
	}
	filename := "chiquito-" + start.Format("2006-01-02") + ".ics"
	if allDay {
		day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
		return ICSResult{
			Content:  datecore.EventICS(title, description, "", day, time.Time{}, true),
			Filename: filename,
		}, nil
	}
	return ICSResult{
		Content:  datecore.EventICS(title, description, "", start, start.Add(time.Hour), false),
		Filename: filename,
	}, nil
}

// SaveICS abre un diálogo nativo y guarda el calendario generado.
func (a *App) SaveICS(content, filename string) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Guardar evento de calendario",
		DefaultFilename: filename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Calendario iCalendar (*.ics)", Pattern: "*.ics"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	if filepath.Ext(path) == "" {
		path += ".ics"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("no se pudo guardar %s: %w", path, err)
	}
	return path, nil
}

// Now devuelve la fecha y hora actual en el formato de los inputs.
func (a *App) Now(zone string) (string, error) {
	loc, err := resolveLocation(zone)
	if err != nil {
		return "", err
	}
	return time.Now().In(loc).Format("2006-01-02T15:04"), nil
}

func holidaySet(country string, manual []string, start, end time.Time, loc *time.Location) (map[string]bool, error) {
	set := datecore.HolidaySetYears(country, start, end, loc)
	extra, err := datecore.ParseHolidayDates(manual, loc)
	if err != nil {
		return nil, err
	}
	for k := range extra {
		set[k] = true
	}
	return set, nil
}

func holidayList(country string, manual []string, start, end time.Time, loc *time.Location) []HolidayView {
	if end.Before(start) {
		start, end = end, start
	}
	from := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	to := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)

	seen := map[string]bool{}
	out := make([]HolidayView, 0)
	for year := from.Year() - 1; year <= to.Year()+1; year++ {
		for _, h := range datecore.Holidays(country, year, loc) {
			day := time.Date(h.Date.Year(), h.Date.Month(), h.Date.Day(), 0, 0, 0, 0, loc)
			if day.Before(from) || day.After(to) {
				continue
			}
			key := day.Format("2006-01-02")
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, HolidayView{Date: day.Format("02/01/2006"), Name: h.Name})
		}
	}
	for _, raw := range manual {
		t, err := datecore.ParseDate(strings.TrimSpace(raw), loc)
		if err != nil {
			continue
		}
		key := t.Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, HolidayView{Date: t.Format("02/01/2006"), Name: "Festivo manual"})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Date[6:]+out[i].Date[3:5]+out[i].Date[:2] <
			out[j].Date[6:]+out[j].Date[3:5]+out[j].Date[:2]
	})
	return out
}

// humanDate formatea una fecha para mostrarla en la interfaz.
func humanDate(t time.Time) string {
	layout := "02/01/2006"
	if t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0 {
		layout += " 15:04:05"
		if t.Nanosecond() != 0 {
			layout += ".000"
		}
	}
	return t.Format(layout) + " (" + weekdayES(t.Weekday()) + ")"
}

func weekdayES(d time.Weekday) string {
	return datecore.WeekdayES(d)
}

func main() {
	// La interfaz es en español: fuerza también el idioma de los controles
	// nativos de WebView2. Sin esto, datetime-local muestra mm/dd/yyyy y AM/PM
	// en sistemas cuyo locale de Chromium es en-US, aunque Windows esté en español.
	args := os.Getenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS")
	if !strings.Contains(args, "--lang=") {
		if args != "" {
			args += " "
		}
		os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", args+"--lang=es")
	}

	app := &App{}
	err := wails.Run(&options.App{
		Title:            "Chiquito de la Calzada — calculadora de fechas",
		Width:            1000,
		Height:           760,
		MinWidth:         720,
		MinHeight:        600,
		BackgroundColour: &options.RGBA{R: 47, G: 7, B: 52, A: 1},
		AssetServer:      &assetserver.Options{Assets: assets},
		Bind:             []interface{}{app},
	})
	if err != nil {
		panic(err)
	}
}
