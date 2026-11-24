// Package datecore contiene la lógica de fechas compartida por la CLI y la GUI.
package datecore

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Delta es una duración expresada en unidades de calendario.
type Delta struct {
	Years        int
	Months       int
	Weeks        int
	Days         int
	Hours        int
	Minutes      int
	Seconds      int
	Milliseconds int
}

// Negative indica si alguna unidad es negativa.
func (d Delta) Negative() bool {
	return d.Years < 0 || d.Months < 0 || d.Weeks < 0 ||
		d.Days < 0 || d.Hours < 0 || d.Minutes < 0 || d.Seconds < 0 ||
		d.Milliseconds < 0
}

// Abs devuelve una copia con todas las unidades en valor absoluto.
func (d Delta) Abs() Delta {
	abs := func(n int) int {
		if n < 0 {
			return -n
		}
		return n
	}
	return Delta{
		Years:        abs(d.Years),
		Months:       abs(d.Months),
		Weeks:        abs(d.Weeks),
		Days:         abs(d.Days),
		Hours:        abs(d.Hours),
		Minutes:      abs(d.Minutes),
		Seconds:      abs(d.Seconds),
		Milliseconds: abs(d.Milliseconds),
	}
}

// Negated devuelve la duración con el signo invertido.
func (d Delta) Negated() Delta {
	return Delta{
		Years:        -d.Years,
		Months:       -d.Months,
		Weeks:        -d.Weeks,
		Days:         -d.Days,
		Hours:        -d.Hours,
		Minutes:      -d.Minutes,
		Seconds:      -d.Seconds,
		Milliseconds: -d.Milliseconds,
	}
}

// String describe la duración en español, en valor absoluto.
func (d Delta) String() string {
	d = d.Abs()
	var parts []string
	add := func(n int, one, many string) {
		if n != 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, pick(n, one, many)))
		}
	}
	add(d.Years, "año", "años")
	add(d.Months, "mes", "meses")
	add(d.Weeks, "semana", "semanas")
	add(d.Days, "día", "días")
	add(d.Hours, "hora", "horas")
	add(d.Minutes, "minuto", "minutos")
	add(d.Seconds, "segundo", "segundos")
	add(d.Milliseconds, "milisegundo", "milisegundos")
	return joinParts(parts, "0 días")
}

// Breakdown es la diferencia desglosada en unidades de calendario.
type Breakdown struct {
	Years   int
	Months  int
	Days    int
	Hours   int
	Minutes int
	Seconds int
}

// String describe el desglose en español.
func (b Breakdown) String() string {
	var parts []string
	add := func(n int, one, many string) {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, pick(n, one, many)))
		}
	}
	add(b.Years, "año", "años")
	add(b.Months, "mes", "meses")
	add(b.Days, "día", "días")
	add(b.Hours, "hora", "horas")
	add(b.Minutes, "minuto", "minutos")
	add(b.Seconds, "segundo", "segundos")
	return joinParts(parts, "0 segundos")
}

// Totals contiene la misma diferencia expresada en distintas unidades.
type Totals struct {
	Years        float64
	Months       float64
	Weeks        float64
	Days24       float64
	Hours        float64
	Minutes      float64
	Seconds      float64
	Milliseconds float64
}

// ComputeTotals convierte una duración absoluta en segundos a todas las
// unidades habituales.
func ComputeTotals(absSeconds float64) Totals {
	return Totals{
		Years:        absSeconds / (86400 * 365.2425),
		Months:       absSeconds / (86400 * 30.436875),
		Weeks:        absSeconds / (86400 * 7),
		Days24:       absSeconds / 86400,
		Hours:        absSeconds / 3600,
		Minutes:      absSeconds / 60,
		Seconds:      absSeconds,
		Milliseconds: absSeconds * 1000,
	}
}

var dateLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"2006-1-2 15:04:05",
	"2006-1-2 15:04",
	"2006-1-2",
	"02/01/2006 15:04:05",
	"02/01/2006 15:04",
	"02/01/2006",
	"2/1/2006 15:04:05",
	"2/1/2006 15:04",
	"2/1/2006",
	"02-01-2006",
	"2-1-2006",
	"2006/01/02 15:04:05",
	"2006/01/02 15:04",
	"2006/01/02",
}

// durationToken reconoce un número seguido de una unidad (ej.: "2mes").
// Las alternativas largas van primero para que "mes" no se confunda con "m".
var durationToken = regexp.MustCompile(`(?i)(\d+)(años?|anos?|a|years?|yrs?|y|meses|mes|months?|mons?|mo|semanas?|sem|weeks?|w|días?|dias?|days?|d|horas?|hours?|hrs?|h|milisegundos?|msegs?|ms|minutos?|minutes?|mins?|m|segundos?|segs?|seconds?|secs?|s)`)

var relativeDate = regexp.MustCompile(`(?i)^(?:en|dentro\s+de)\s+(\d+|un|una|uno)\s+(días?|dias?|semanas?|meses?|años?|anos?|horas?|minutos?|segundos?)$`)
var relativePast = regexp.MustCompile(`(?i)^hace\s+(\d+|un|una|uno)\s+(días?|dias?|semanas?|meses?|años?|anos?|horas?|minutos?|segundos?)$`)
var spanishDate = regexp.MustCompile(`(?i)^(\d{1,2})\s+(?:de\s+)?([a-záéíóúüñ]+)(?:\s+(?:de\s+|del\s+)?(\d{4}))?(?:\s+(?:a\s+las?\s+)?(\d{1,2}):(\d{2})(?::(\d{2}))?)?$`)

var weekdayNames = map[string]time.Weekday{
	"domingo": time.Sunday, "lunes": time.Monday, "martes": time.Tuesday,
	"miercoles": time.Wednesday, "jueves": time.Thursday,
	"viernes": time.Friday, "sabado": time.Saturday,
}

var monthNames = map[string]time.Month{
	"enero": time.January, "ene": time.January,
	"febrero": time.February, "feb": time.February,
	"marzo": time.March, "mar": time.March,
	"abril": time.April, "abr": time.April,
	"mayo": time.May, "may": time.May,
	"junio": time.June, "jun": time.June,
	"julio": time.July, "jul": time.July,
	"agosto": time.August, "ago": time.August,
	"septiembre": time.September, "setiembre": time.September, "sep": time.September, "set": time.September,
	"octubre": time.October, "oct": time.October,
	"noviembre": time.November, "nov": time.November,
	"diciembre": time.December, "dic": time.December,
}

// sinAcentos normaliza una palabra para compararla sin depender de tildes.
func sinAcentos(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case 'á', 'à', 'ä', 'â':
			return 'a'
		case 'é', 'è', 'ë', 'ê':
			return 'e'
		case 'í', 'ì', 'ï', 'î':
			return 'i'
		case 'ó', 'ò', 'ö', 'ô':
			return 'o'
		case 'ú', 'ù', 'ü', 'û':
			return 'u'
		case 'ñ':
			return 'n'
		}
		return r
	}, strings.ToLower(s))
}

func relativeDelta(unit string) Delta {
	switch sinAcentos(strings.TrimSpace(unit)) {
	case "dia", "dias":
		return Delta{Days: 1}
	case "semana", "semanas":
		return Delta{Weeks: 1}
	case "mes", "meses":
		return Delta{Months: 1}
	case "ano", "anos":
		return Delta{Years: 1}
	case "hora", "horas":
		return Delta{Hours: 1}
	case "minuto", "minutos":
		return Delta{Minutes: 1}
	case "segundo", "segundos":
		return Delta{Seconds: 1}
	}
	return Delta{}
}

func nextWeekday(now time.Time, want time.Weekday) time.Time {
	delta := (int(want) - int(now.Weekday()) + 7) % 7
	if delta == 0 {
		delta = 7
	}
	day := now.AddDate(0, 0, delta)
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
}

// numberOf acepta cifras o los artículos "un/una/uno".
func numberOf(s string) (int, bool) {
	if n, err := strconv.Atoi(s); err == nil {
		return n, true
	}
	switch sinAcentos(strings.TrimSpace(s)) {
	case "un", "una", "uno":
		return 1, true
	}
	return 0, false
}

func parseNaturalDate(s string, loc *time.Location) (time.Time, bool, error) {
	now := time.Now().In(loc).Truncate(time.Second)
	low := strings.TrimSpace(strings.ToLower(s))
	norm := sinAcentos(low)
	switch norm {
	case "hoy", "today":
		return now, true, nil
	case "ahora", "now":
		return now, true, nil
	case "ayer", "yesterday":
		return now.AddDate(0, 0, -1), true, nil
	case "manana", "tomorrow":
		return now.AddDate(0, 0, 1), true, nil
	case "anteayer":
		return now.AddDate(0, 0, -2), true, nil
	case "pasado manana":
		return now.AddDate(0, 0, 2), true, nil
	}

	for _, re := range []struct {
		pattern *regexp.Regexp
		sign    int
	}{{relativeDate, 1}, {relativePast, -1}} {
		if m := re.pattern.FindStringSubmatch(low); m != nil {
			n, ok := numberOf(m[1])
			if !ok {
				continue
			}
			d := relativeDelta(m[2])
			d = Delta{
				Days: d.Days * n, Weeks: d.Weeks * n, Months: d.Months * n,
				Years: d.Years * n, Hours: d.Hours * n, Minutes: d.Minutes * n, Seconds: d.Seconds * n,
			}
			return ApplyDelta(now, d, re.sign), true, nil
		}
	}

	// "lunes", "el próximo lunes", "siguiente lunes"…
	weekdayText := norm
	for _, prefix := range []string{"el ", "la ", "los ", "las ", "este ", "esta ", "proximo ", "proxima ", "siguiente "} {
		weekdayText = strings.TrimPrefix(weekdayText, prefix)
	}
	if wd, ok := weekdayNames[weekdayText]; ok {
		return nextWeekday(now, wd), true, nil
	}

	// "18 de noviembre de 2026", "18 nov 2026 a las 14:30"…
	if m := spanishDate.FindStringSubmatch(low); m != nil {
		day, _ := strconv.Atoi(m[1])
		month, ok := monthNames[sinAcentos(m[2])]
		if !ok {
			return time.Time{}, false, nil
		}
		year := now.Year()
		if m[3] != "" {
			year, _ = strconv.Atoi(m[3])
		}
		hour, minute, second := 0, 0, 0
		if m[4] != "" {
			hour, _ = strconv.Atoi(m[4])
			minute, _ = strconv.Atoi(m[5])
			if m[6] != "" {
				second, _ = strconv.Atoi(m[6])
			}
		}
		t := time.Date(year, month, day, hour, minute, second, 0, loc)
		if t.Day() != day || t.Month() != month || t.Year() != year || hour > 23 || minute > 59 || second > 59 {
			return time.Time{}, true, fmt.Errorf("fecha no válida: %q", s)
		}
		return t, true, nil
	}
	return time.Time{}, false, nil
}

// ParseDate interpreta una fecha en varios formatos habituales, incluidos
// atajos en español como "mañana", "en 3 días", "el próximo lunes" o
// "18 de noviembre de 2026".
func ParseDate(s string, loc *time.Location) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("fecha vacía")
	}
	if t, ok, err := parseNaturalDate(s, loc); ok {
		return t, err
	}
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("fecha no válida: %q (prueba 2026-11-18, 18/11/2026, \"mañana\" o \"hoy\")", s)
}

var canonicalInput = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}`)

// DSTWarning avisa cuando una hora local no existe por el salto al horario de
// verano o cuando existe dos veces al retrasar el reloj en otoño. Go normaliza
// el primer caso (p. ej. 02:30 → 03:30) y elige una de las dos ocurrencias en
// el segundo, así que lo hacemos explícito.
func DSTWarning(label, input string, parsed time.Time, loc *time.Location) string {
	if !canonicalInput.MatchString(input) {
		return ""
	}
	want := input[:16]
	got := parsed.Format("2006-01-02T15:04")
	if want != got {
		return fmt.Sprintf(
			"%s: la hora %s no existe en %s por el cambio al horario de verano; se interpreta como %s.",
			label, strings.Replace(want, "T", " ", 1), loc, strings.Replace(got, "T", " ", 1),
		)
	}
	if ambiguousLocalTime(want, parsed, loc) {
		abbr, _ := parsed.Zone()
		return fmt.Sprintf(
			"%s: la hora %s existe dos veces en %s por el cambio al horario de invierno; se interpreta como %s.",
			label, strings.Replace(want, "T", " ", 1), loc, abbr,
		)
	}
	return ""
}

// ambiguousLocalTime indica si una hora de pared se repite por un cambio de
// hora. Se apoya en los offsets reales de la zona alrededor del instante.
func ambiguousLocalTime(want string, parsed time.Time, loc *time.Location) bool {
	naive, err := time.Parse("2006-01-02T15:04", want)
	if err != nil {
		return false
	}
	offsets := map[int]bool{}
	for _, t := range []time.Time{parsed.Add(-26 * time.Hour), parsed, parsed.Add(26 * time.Hour)} {
		_, offset := t.In(loc).Zone()
		offsets[offset] = true
	}
	matches := 0
	for offset := range offsets {
		candidate := naive.Add(-time.Duration(offset) * time.Second)
		if candidate.In(loc).Format("2006-01-02T15:04") == want {
			matches++
		}
	}
	return matches > 1
}

// ParseDelta interpreta una duración como "1a2mes3d 4h5min6s".
func ParseDelta(s string) (Delta, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return Delta{}, errors.New("duración vacía")
	}
	neg := false
	if strings.HasPrefix(raw, "-") {
		neg = true
		raw = strings.TrimSpace(raw[1:])
	} else if strings.HasPrefix(raw, "+") {
		raw = strings.TrimSpace(raw[1:])
	}
	if raw == "" {
		return Delta{}, fmt.Errorf("duración no válida: %q", s)
	}

	// Un número suelto se interpreta como días.
	if n, err := strconv.Atoi(raw); err == nil {
		d := Delta{Days: n}
		if neg {
			d = d.Negated()
		}
		return d, nil
	}

	compact := strings.NewReplacer(" ", "", "\t", "", "\n", "", "\r", "").Replace(raw)
	var d Delta
	i := 0
	for i < len(compact) {
		loc := durationToken.FindStringSubmatchIndex(compact[i:])
		if loc == nil || loc[0] != 0 {
			return Delta{}, fmt.Errorf("duración no válida cerca de %q (usa por ejemplo 1a2mes3d4h5min6s)", compact[i:])
		}
		n, err := strconv.Atoi(compact[i+loc[2] : i+loc[3]])
		if err != nil {
			return Delta{}, fmt.Errorf("número no válido en %q", compact[i:])
		}
		unit := strings.ToLower(compact[i+loc[4] : i+loc[5]])
		i += loc[1]

		switch unit {
		case "a", "año", "años", "ano", "anos", "y", "year", "years", "yr", "yrs":
			d.Years += n
		case "mes", "meses", "mo", "month", "months", "mon", "mons":
			d.Months += n
		case "sem", "semana", "semanas", "w", "week", "weeks":
			d.Weeks += n
		case "d", "día", "días", "dia", "dias", "day", "days":
			d.Days += n
		case "h", "hora", "horas", "hour", "hours", "hr", "hrs":
			d.Hours += n
		case "ms", "mseg", "msegs", "milisegundo", "milisegundos":
			d.Milliseconds += n
		case "m", "min", "minuto", "minutos", "minute", "minutes", "mins":
			d.Minutes += n
		case "s", "seg", "segs", "segundo", "segundos", "sec", "secs", "second", "seconds":
			d.Seconds += n
		default:
			return Delta{}, fmt.Errorf("unidad de duración desconocida: %q", unit)
		}
	}
	if neg {
		d = d.Negated()
	}
	return d, nil
}

func daysInMonth(year int, month time.Month, loc *time.Location) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
}

// AddYearsClamped suma años ajustando el día al último del mes si hace falta.
func AddYearsClamped(t time.Time, n int) time.Time {
	y := t.Year() + n
	m := t.Month()
	d := t.Day()
	if last := daysInMonth(y, m, t.Location()); d > last {
		d = last
	}
	return time.Date(y, m, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// AddMonthsClamped suma meses ajustando el día al último del mes si hace falta.
func AddMonthsClamped(t time.Time, n int) time.Time {
	total := t.Year()*12 + int(t.Month()) - 1 + n
	y := total / 12
	m := time.Month(total%12 + 1)
	if m < 1 {
		m += 12
		y--
	}
	d := t.Day()
	if last := daysInMonth(y, m, t.Location()); d > last {
		d = last
	}
	return time.Date(y, m, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// AddDays suma días manteniendo la hora local.
func AddDays(t time.Time, n int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()+n,
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// ApplyDelta suma (sign=1) o resta (sign=-1) una duración a una fecha.
func ApplyDelta(t time.Time, d Delta, sign int) time.Time {
	if sign < 0 {
		d = d.Negated()
	}
	t = AddYearsClamped(t, d.Years)
	t = AddMonthsClamped(t, d.Months)
	t = AddDays(t, d.Weeks*7+d.Days)
	dur := time.Duration(d.Hours)*time.Hour +
		time.Duration(d.Minutes)*time.Minute +
		time.Duration(d.Seconds)*time.Second +
		time.Duration(d.Milliseconds)*time.Millisecond
	return t.Add(dur)
}

// DiffBreakdown desglosa la diferencia entre dos fechas, de la más antigua a
// la más reciente, en años, meses, días, horas, minutos y segundos.
func DiffBreakdown(a, b time.Time) Breakdown {
	if b.Before(a) {
		a, b = b, a
	}

	years := b.Year() - a.Year()
	for years > 0 && AddYearsClamped(a, years).After(b) {
		years--
	}
	cur := AddYearsClamped(a, years)

	months := 0
	for {
		next := AddMonthsClamped(cur, months+1)
		if next.After(b) {
			break
		}
		months++
	}
	cur = AddMonthsClamped(cur, months)

	days := 0
	for {
		next := AddDays(cur, days+1)
		if next.After(b) {
			break
		}
		days++
	}
	cur = AddDays(cur, days)

	rem := b.Sub(cur)
	if rem < 0 {
		rem = 0
	}
	hours := int(rem / time.Hour)
	rem -= time.Duration(hours) * time.Hour
	minutes := int(rem / time.Minute)
	rem -= time.Duration(minutes) * time.Minute
	seconds := int(rem / time.Second)

	return Breakdown{
		Years:   years,
		Months:  months,
		Days:    days,
		Hours:   hours,
		Minutes: minutes,
		Seconds: seconds,
	}
}

// CivilDays devuelve la diferencia en días naturales, ignorando la hora.
// Usa UTC para evitar problemas de horario de verano.
func CivilDays(a, b time.Time) int64 {
	da := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	db := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return (db.Unix() - da.Unix()) / 86400
}

func joinParts(parts []string, empty string) string {
	switch len(parts) {
	case 0:
		return empty
	case 1:
		return parts[0]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " y " + parts[len(parts)-1]
	}
}

func pick(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// Round redondea v a la cantidad de decimales indicada.
func Round(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}

// FormatNumber formatea con separador de miles y coma decimal, sin ceros
// decimales sobrantes.
func FormatNumber(v float64, decimals int) string {
	if math.Abs(v) < 1e-9 {
		v = 0
	}
	if math.Abs(v-math.Round(v)) < 1e-9 {
		return FormatInt(int64(math.Round(v)))
	}
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	intPart, frac, hasFrac := strings.Cut(s, ".")
	intPart = groupThousands(intPart)
	if hasFrac && frac != "" {
		return intPart + "," + frac
	}
	return intPart
}

// FormatInt formatea un entero con separador de miles.
func FormatInt(n int64) string {
	return groupThousands(strconv.FormatInt(n, 10))
}

func groupThousands(s string) string {
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteByte(s[i])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
