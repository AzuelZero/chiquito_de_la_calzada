package datecore

import (
	"fmt"
	"strings"
	"time"
)

// Holiday es un festivo con nombre.
type Holiday struct {
	Date time.Time
	Name string
}

// Country describe un conjunto de festivos nacionales.
type Country struct {
	ID   string
	Name string
	Note string
}

var countries = []Country{
	{ID: "es", Name: "España", Note: "Festivos nacionales; no incluye autonomicos ni locales."},
	{ID: "mx", Name: "México", Note: "Festivos oficiales federales."},
	{ID: "co", Name: "Colombia", Note: "Festivos nacionales con traslado al lunes (Ley Emiliani)."},
	{ID: "ar", Name: "Argentina", Note: "Festivos nacionales; no incluye provinciales."},
}

// Countries devuelve los packs de festivos disponibles.
func Countries() []Country {
	out := make([]Country, len(countries))
	copy(out, countries)
	return out
}

// Holidays devuelve los festivos nacionales de un país y un año. Si el país
// no existe, devuelve una lista vacía.
func Holidays(country string, year int, loc *time.Location) []Holiday {
	if loc == nil {
		loc = time.UTC
	}
	d := func(month time.Month, day int) time.Time {
		return time.Date(year, month, day, 0, 0, 0, 0, loc)
	}
	easter := easterSunday(year, loc)
	offset := func(days int) time.Time { return easter.AddDate(0, 0, days) }

	switch strings.ToLower(strings.TrimSpace(country)) {
	case "es":
		return []Holiday{
			{d(time.January, 1), "Año Nuevo"},
			{d(time.January, 6), "Epifanía del Señor"},
			{offset(-2), "Viernes Santo"},
			{d(time.May, 1), "Fiesta del Trabajo"},
			{d(time.August, 15), "Asunción de la Virgen"},
			{d(time.October, 12), "Fiesta Nacional de España"},
			{d(time.November, 1), "Todos los Santos"},
			{d(time.December, 6), "Día de la Constitución"},
			{d(time.December, 8), "Inmaculada Concepción"},
			{d(time.December, 25), "Navidad"},
		}
	case "mx":
		return []Holiday{
			{d(time.January, 1), "Año Nuevo"},
			{nthWeekday(year, time.February, time.Monday, 1, loc), "Día de la Constitución"},
			{nthWeekday(year, time.March, time.Monday, 3, loc), "Natalicio de Benito Juárez"},
			{d(time.May, 1), "Día del Trabajo"},
			{d(time.September, 16), "Día de la Independencia"},
			{nthWeekday(year, time.November, time.Monday, 3, loc), "Revolución Mexicana"},
			{d(time.December, 25), "Navidad"},
		}
	case "co":
		return []Holiday{
			{d(time.January, 1), "Año Nuevo"},
			{moveToMonday(d(time.January, 6), loc), "Epifanía"},
			{moveToMonday(d(time.March, 19), loc), "San José"},
			{offset(-3), "Jueves Santo"},
			{offset(-2), "Viernes Santo"},
			{d(time.May, 1), "Día del Trabajo"},
			{moveToMonday(offset(39), loc), "Ascensión"},
			{moveToMonday(offset(60), loc), "Corpus Christi"},
			{moveToMonday(offset(68), loc), "Sagrado Corazón"},
			{moveToMonday(d(time.June, 29), loc), "San Pedro y San Pablo"},
			{d(time.July, 20), "Día de la Independencia"},
			{d(time.August, 7), "Batalla de Boyacá"},
			{moveToMonday(d(time.August, 15), loc), "Asunción"},
			{moveToMonday(d(time.October, 12), loc), "Día de la Raza"},
			{moveToMonday(d(time.November, 1), loc), "Todos los Santos"},
			{moveToMonday(d(time.November, 11), loc), "Independencia de Cartagena"},
			{d(time.December, 8), "Inmaculada Concepción"},
			{d(time.December, 25), "Navidad"},
		}
	case "ar":
		return []Holiday{
			{d(time.January, 1), "Año Nuevo"},
			{offset(-48), "Lunes de Carnaval"},
			{offset(-47), "Martes de Carnaval"},
			{d(time.March, 24), "Día de la Memoria"},
			{d(time.April, 2), "Día del Veterano de Malvinas"},
			{offset(-2), "Viernes Santo"},
			{d(time.May, 1), "Día del Trabajo"},
			{d(time.May, 25), "Revolución de Mayo"},
			{d(time.June, 20), "Día de la Bandera"},
			{d(time.July, 9), "Día de la Independencia"},
			{moveToMonday(d(time.August, 17), loc), "Paso a la Inmortalidad de San Martín"},
			{moveToMonday(d(time.October, 12), loc), "Día del Respeto a la Diversidad Cultural"},
			{moveToMonday(d(time.November, 20), loc), "Día de la Soberanía Nacional"},
			{d(time.December, 8), "Inmaculada Concepción"},
			{d(time.December, 25), "Navidad"},
		}
	}
	return nil
}

// HolidaySet devuelve un conjunto de claves "2006-01-02" para varios años.
func HolidaySet(country string, years []int, loc *time.Location) map[string]bool {
	set := map[string]bool{}
	for _, year := range years {
		for _, h := range Holidays(country, year, loc) {
			set[h.Date.Format("2006-01-02")] = true
		}
	}
	return set
}

// ParseHolidayDates convierte una lista de fechas sueltas en un conjunto de
// claves "2006-01-02".
func ParseHolidayDates(dates []string, loc *time.Location) (map[string]bool, error) {
	set := map[string]bool{}
	for _, raw := range dates {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		t, err := ParseDate(raw, loc)
		if err != nil {
			return nil, fmt.Errorf("festivo %q: %w", raw, err)
		}
		set[t.Format("2006-01-02")] = true
	}
	return set, nil
}

func nthWeekday(year int, month time.Month, weekday time.Weekday, n int, loc *time.Location) time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	offset := (int(weekday) - int(first.Weekday()) + 7) % 7
	return first.AddDate(0, 0, offset+(n-1)*7)
}

func moveToMonday(t time.Time, loc *time.Location) time.Time {
	offset := (8 - int(t.Weekday())) % 7
	return t.AddDate(0, 0, offset).In(loc)
}

func easterSunday(year int, loc *time.Location) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
}
