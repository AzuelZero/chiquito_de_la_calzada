// datecalc: sumador/restador de fechas por línea de comandos.
//
// Calcula la diferencia entre dos fechas (en años, meses, días, horas,
// minutos, segundos y milisegundos) y permite sumar o restar duraciones
// a una fecha.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"datecalc/datecore"
)

const usageText = `Chiquito de la Calzada — calculadora de fechas

Uso:
  chiquito_de_la_calzada_cli                        Modo interactivo
  chiquito_de_la_calzada_cli <fecha1>               Diferencia con hoy
  chiquito_de_la_calzada_cli <fecha1> <fecha2>      Diferencia entre fechas
  chiquito_de_la_calzada_cli diff <fecha1> <fecha2> Igual que arriba
  chiquito_de_la_calzada_cli add <fecha> <duración> Sumar una duración
  chiquito_de_la_calzada_cli sub <fecha> <duración> Restar una duración
  chiquito_de_la_calzada_cli age <nacimiento> [fecha]   Edad y cumpleaños
  chiquito_de_la_calzada_cli workdays <fecha1> <fecha2> Días laborables
  chiquito_de_la_calzada_cli addworkdays <fecha> <días> Sumar días laborables
  chiquito_de_la_calzada_cli facts <fecha>          Ficha de calendario
  chiquito_de_la_calzada_cli convert <fecha> <origen> <destino>
  chiquito_de_la_calzada_cli ics <fecha> [título]   Exportar evento .ics

Formatos de fecha:
  2026-11-18, 18/11/2026, 18-11-2026
  2026-11-18 14:30, 18/11/2026 14:30[:05]
  2026-11-18T14:30:00+01:00 (RFC 3339)
  hoy | ahora | mañana | ayer | anteayer
  en 3 días | hace 2 semanas | el próximo lunes
  18 de noviembre de 2026 | 18 nov 2026 a las 14:30

Duración (para add/sub):
  1a 2mes 3sem 4d 5h 6min 7s 250ms
  (también 1y 2mo 3w 4d 5h 6m 7s)
  Un número suelto se interpreta como días (ej.: 30).
  Ejemplos: "1a2mes", "45min", "2 semanas 3 días", "-1mes".

Opciones:
  -u, --utc              Interpretar las fechas en UTC (por defecto, hora local)
  -z, --tz <zona>        Zona IANA, p. ej. Europe/Madrid o America/Bogota
  -j, --json             Salida en JSON
  -i, --inclusive        Contar ambos extremos en los días naturales
      --country <clave>  Festivos para workdays/addworkdays: es, mx, co, ar
      --holiday <fecha>  Festivo suelto; se puede repetir
      --out <archivo>    Guardar el .ics en un archivo
      --all-day          Crear el .ics como evento de día completo
      --list-zones       Listar zonas IANA disponibles
      --version          Mostrar la versión
  -h, --help             Mostrar esta ayuda
`

// version se rellena en la compilación con -ldflags "-X main.version=...".
var version = "dev"

// options contiene las opciones de línea de comandos.
type options struct {
	utc       bool
	json      bool
	inclusive bool
	help      bool
	version   bool
	listZones bool
	allDay    bool
	zone      string
	country   string
	holidays  []string
	out       string
}

type breakdownJSON struct {
	Anios    int `json:"anios"`
	Meses    int `json:"meses"`
	Dias     int `json:"dias"`
	Horas    int `json:"horas"`
	Minutos  int `json:"minutos"`
	Segundos int `json:"segundos"`
}

type diffJSON struct {
	Fecha1        string        `json:"fecha1"`
	Fecha2        string        `json:"fecha2"`
	Posterior     string        `json:"posterior,omitempty"`
	Iguales       bool          `json:"iguales"`
	Desglose      breakdownJSON `json:"desglose"`
	DiasNaturales int64         `json:"dias_naturales"`
	Dias24h       float64       `json:"dias_24h"`
	Semanas       float64       `json:"semanas"`
	MesesTotales  float64       `json:"meses_totales"`
	AniosTotales  float64       `json:"anios_totales"`
	Horas         float64       `json:"horas"`
	Minutos       float64       `json:"minutos"`
	Segundos      float64       `json:"segundos"`
	Milisegundos  float64       `json:"milisegundos"`
	Aviso         string        `json:"aviso,omitempty"`
}

type shiftJSON struct {
	Operacion string `json:"operacion"`
	Duracion  string `json:"duracion"`
	Inicio    string `json:"inicio"`
	Resultado string `json:"resultado"`
	Aviso     string `json:"aviso,omitempty"`
}

type ageJSON struct {
	Nacimiento       string `json:"nacimiento"`
	Referencia       string `json:"referencia"`
	Anios            int    `json:"anios"`
	Meses            int    `json:"meses"`
	Dias             int    `json:"dias"`
	DiasTotales      int64  `json:"dias_totales"`
	Proximo          string `json:"proximo_cumpleanos"`
	DiasParaProximo  int64  `json:"dias_para_el_proximo"`
	DiaSemanaProximo string `json:"dia_semana_proximo"`
	CumpleAnios      int    `json:"cumple_anios"`
	EsHoy            bool   `json:"es_hoy"`
	Ajuste29Febrero  bool   `json:"ajuste_29_febrero,omitempty"`
	Aviso            string `json:"aviso,omitempty"`
}

type holidayJSON struct {
	Fecha  string `json:"fecha"`
	Nombre string `json:"nombre"`
}

type workdaysJSON struct {
	Fecha1         string        `json:"fecha1"`
	Fecha2         string        `json:"fecha2"`
	DiasLaborables int           `json:"dias_laborables"`
	DiasNaturales  int64         `json:"dias_naturales"`
	Pais           string        `json:"pais,omitempty"`
	Festivos       []holidayJSON `json:"festivos,omitempty"`
}

type workShiftJSON struct {
	Operacion      string        `json:"operacion"`
	DiasLaborables int           `json:"dias_laborables"`
	Inicio         string        `json:"inicio"`
	Resultado      string        `json:"resultado"`
	Festivos       []holidayJSON `json:"festivos,omitempty"`
}

type factsJSON struct {
	Fecha         string `json:"fecha"`
	FechaHumana   string `json:"fecha_humana"`
	DiaSemana     string `json:"dia_semana"`
	SemanaISO     int    `json:"semana_iso"`
	AnioISO       int    `json:"anio_iso"`
	DiaDelAnio    int    `json:"dia_del_anio"`
	DiasDelAnio   int    `json:"dias_del_anio"`
	Trimestre     int    `json:"trimestre"`
	DiasDelMes    int    `json:"dias_del_mes"`
	QuedanDelMes  int    `json:"quedan_del_mes"`
	QuedanDelAnio int    `json:"quedan_del_anio"`
	Bisiesto      bool   `json:"bisiesto"`
}

type convertJSON struct {
	Fecha     string `json:"fecha"`
	Origen    string `json:"origen"`
	Destino   string `json:"destino"`
	FechaISO  string `json:"fecha_iso"`
	Resultado string `json:"resultado"`
	Aviso     string `json:"aviso,omitempty"`
}

// exitError permite distinguir errores de uso (código 2) de errores de datos
// o de ejecución (código 1).
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

func usageError(err error) error {
	return &exitError{code: 2, err: err}
}

func usagef(format string, args ...interface{}) error {
	return usageError(fmt.Errorf(format, args...))
}

var negativeNumber = regexp.MustCompile(`^-\d`)

// stdin se puede sustituir en los tests.
var stdin io.Reader = os.Stdin

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		var ee *exitError
		if errors.As(err, &ee) {
			os.Exit(ee.code)
		}
		os.Exit(1)
	}
}

func run(args []string, w io.Writer) error {
	opts, pos, err := splitArgs(args)
	if err != nil {
		return usageError(err)
	}
	if opts.version {
		fmt.Fprintf(w, "Chiquito de la Calzada %s\n", version)
		return nil
	}
	if opts.listZones {
		printZones(w)
		return nil
	}
	if opts.help {
		fmt.Fprint(w, usageText)
		return nil
	}
	loc, err := resolveCLILocation(opts)
	if err != nil {
		return err
	}
	if len(pos) == 0 {
		return interactive(w, opts, loc)
	}

	switch strings.ToLower(pos[0]) {
	case "diff":
		if len(pos) != 3 {
			return usagef("uso: chiquito_de_la_calzada_cli diff <fecha1> <fecha2>")
		}
		return printDiff(w, pos[1], pos[2], opts, loc)
	case "add", "sub":
		if len(pos) != 3 {
			return usagef("uso: chiquito_de_la_calzada_cli %s <fecha> <duración>", strings.ToLower(pos[0]))
		}
		return printShift(w, strings.ToLower(pos[0]), pos[1], pos[2], opts, loc)
	case "age", "edad":
		if len(pos) < 2 || len(pos) > 3 {
			return usagef("uso: chiquito_de_la_calzada_cli age <nacimiento> [fecha]")
		}
		at := ""
		if len(pos) == 3 {
			at = pos[2]
		}
		return printAge(w, pos[1], at, opts, loc)
	case "workdays", "laborables":
		if len(pos) != 3 {
			return usagef("uso: chiquito_de_la_calzada_cli workdays <fecha1> <fecha2>")
		}
		return printWorkdays(w, pos[1], pos[2], opts, loc)
	case "addworkdays", "sumarlaborables":
		if len(pos) != 3 {
			return usagef("uso: chiquito_de_la_calzada_cli addworkdays <fecha> <días>")
		}
		return printAddWorkdays(w, pos[1], pos[2], opts, loc)
	case "facts", "ficha":
		if len(pos) != 2 {
			return usagef("uso: chiquito_de_la_calzada_cli facts <fecha>")
		}
		return printFacts(w, pos[1], opts, loc)
	case "convert", "convertir":
		if len(pos) != 4 {
			return usagef("uso: chiquito_de_la_calzada_cli convert <fecha> <origen> <destino>")
		}
		return printConvert(w, pos[1], pos[2], pos[3], opts)
	case "ics":
		if len(pos) < 2 {
			return usagef("uso: chiquito_de_la_calzada_cli ics <fecha> [título]")
		}
		title := "Chiquito de la Calzada"
		if len(pos) > 2 {
			title = strings.Join(pos[2:], " ")
		}
		return printICS(w, pos[1], title, opts, loc)
	case "help", "ayuda":
		fmt.Fprint(w, usageText)
		return nil
	}

	switch len(pos) {
	case 1:
		return printDiff(w, pos[0], "hoy", opts, loc)
	case 2:
		return printDiff(w, pos[0], pos[1], opts, loc)
	default:
		return usagef("argumentos no reconocidos: %s (usa --help)", strings.Join(pos, " "))
	}
}

// splitArgs separa las opciones de los argumentos posicionales. Las opciones
// pueden ir en cualquier posición; los números negativos (ej.: "-1mes") se
// tratan como argumentos.
func splitArgs(args []string) (options, []string, error) {
	var o options
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-u", a == "--utc", a == "-utc":
			o.utc = true
		case a == "-j", a == "--json", a == "-json":
			o.json = true
		case a == "-i", a == "--inclusive", a == "-inclusive":
			o.inclusive = true
		case a == "-h", a == "--help", a == "-help":
			o.help = true
		case a == "--version", a == "-version", a == "-v":
			o.version = true
		case a == "--list-zones", a == "-list-zones":
			o.listZones = true
		case a == "--all-day", a == "-all-day":
			o.allDay = true
		case a == "-z", a == "--tz", a == "-tz":
			value, next, err := optionValue(args, i, a)
			if err != nil {
				return o, nil, err
			}
			i = next
			o.zone = value
		case strings.HasPrefix(a, "--tz="):
			o.zone = strings.TrimSpace(strings.TrimPrefix(a, "--tz="))
			if o.zone == "" {
				return o, nil, errors.New("zona horaria vacía")
			}
		case a == "--country", a == "-country":
			value, next, err := optionValue(args, i, a)
			if err != nil {
				return o, nil, err
			}
			i = next
			o.country = strings.ToLower(value)
		case strings.HasPrefix(a, "--country="):
			o.country = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(a, "--country=")))
		case a == "--holiday", a == "-holiday":
			value, next, err := optionValue(args, i, a)
			if err != nil {
				return o, nil, err
			}
			i = next
			o.holidays = append(o.holidays, value)
		case strings.HasPrefix(a, "--holiday="):
			o.holidays = append(o.holidays, strings.TrimSpace(strings.TrimPrefix(a, "--holiday=")))
		case a == "--out", a == "-out":
			value, next, err := optionValue(args, i, a)
			if err != nil {
				return o, nil, err
			}
			i = next
			o.out = value
		case strings.HasPrefix(a, "--out="):
			o.out = strings.TrimSpace(strings.TrimPrefix(a, "--out="))
		default:
			if strings.HasPrefix(a, "-") && !negativeNumber.MatchString(a) {
				return o, nil, fmt.Errorf("opción desconocida: %s", a)
			}
			pos = append(pos, a)
		}
	}
	return o, pos, nil
}

func optionValue(args []string, i int, name string) (string, int, error) {
	if i+1 >= len(args) {
		return "", i, fmt.Errorf("falta el valor de %s", name)
	}
	value := strings.TrimSpace(args[i+1])
	if value == "" {
		return "", i, fmt.Errorf("valor vacío para %s", name)
	}
	return value, i + 1, nil
}

// resolveCLILocation elige la zona horaria según las opciones.
func resolveCLILocation(o options) (*time.Location, error) {
	if o.utc && o.zone != "" {
		return nil, errors.New("no combines --utc con --tz")
	}
	if o.utc {
		return time.UTC, nil
	}
	if o.zone == "" {
		return time.Local, nil
	}
	loc, err := datecore.ResolveLocation(o.zone)
	if err != nil {
		return nil, fmt.Errorf("zona horaria no válida: %q (usa un nombre IANA, p. ej. Europe/Madrid)", o.zone)
	}
	return loc, nil
}

func printZones(w io.Writer) {
	fmt.Fprintln(w, "Zonas IANA disponibles:")
	group := ""
	for _, z := range datecore.ZoneCatalog() {
		if z.Group != group {
			group = z.Group
			fmt.Fprintf(w, "\n%s:\n", group)
		}
		fmt.Fprintf(w, "  %-32s %s\n", z.ID, z.City)
	}
}

func printDiff(w io.Writer, s1, s2 string, opts options, loc *time.Location) error {
	d1, err := datecore.ParseDate(s1, loc)
	if err != nil {
		return err
	}
	d2, err := datecore.ParseDate(s2, loc)
	if err != nil {
		return err
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
	if opts.inclusive {
		natural++
	}

	later := ""
	if d1.Before(d2) {
		later = "fecha2"
	} else if d2.Before(d1) {
		later = "fecha1"
	}

	warnings := make([]string, 0, 2)
	if w := datecore.DSTWarning("Fecha 1", s1, d1, loc); w != "" {
		warnings = append(warnings, w)
	}
	if w := datecore.DSTWarning("Fecha 2", s2, d2, loc); w != "" {
		warnings = append(warnings, w)
	}
	warning := strings.Join(warnings, "\n")

	if opts.json {
		payload := diffJSON{
			Fecha1:    d1.Format(time.RFC3339),
			Fecha2:    d2.Format(time.RFC3339),
			Posterior: later,
			Iguales:   later == "",
			Desglose: breakdownJSON{
				Anios:    bd.Years,
				Meses:    bd.Months,
				Dias:     bd.Days,
				Horas:    bd.Hours,
				Minutos:  bd.Minutes,
				Segundos: bd.Seconds,
			},
			DiasNaturales: natural,
			Dias24h:       datecore.Round(totals.Days24, 4),
			Semanas:       datecore.Round(totals.Weeks, 4),
			MesesTotales:  datecore.Round(totals.Months, 4),
			AniosTotales:  datecore.Round(totals.Years, 4),
			Horas:         datecore.Round(totals.Hours, 4),
			Minutos:       datecore.Round(totals.Minutes, 4),
			Segundos:      datecore.Round(totals.Seconds, 4),
			Milisegundos:  datecore.Round(totals.Milliseconds, 2),
			Aviso:         warning,
		}
		return writeJSON(w, payload)
	}

	if opts.zone != "" {
		fmt.Fprintf(w, "Zona: %s (%s)\n\n", opts.zone, zoneDescription(d1))
	}
	fmt.Fprintf(w, "Fecha 1: %s\n", humanDate(d1))
	fmt.Fprintf(w, "Fecha 2: %s\n\n", humanDate(d2))
	if warning != "" {
		for _, line := range strings.Split(warning, "\n") {
			fmt.Fprintf(w, "Aviso: %s\n", line)
		}
		fmt.Fprintln(w)
	}
	switch later {
	case "":
		fmt.Fprintln(w, "Las dos fechas son iguales.")
	case "fecha2":
		fmt.Fprintln(w, "La fecha 2 es posterior a la fecha 1.")
	default:
		fmt.Fprintln(w, "La fecha 1 es posterior a la fecha 2.")
	}

	older, newer := d1, d2
	if newer.Before(older) {
		older, newer = newer, older
	}

	fmt.Fprintf(w, "Diferencia: %s\n", bd.String())
	if opts.inclusive {
		fmt.Fprintf(w, "Días naturales (ambos extremos incluidos): %s (%s → %s)\n",
			datecore.FormatInt(natural), older.Format("02/01/2006"), newer.Format("02/01/2006"))
	} else {
		fmt.Fprintf(w, "Días naturales: %s (%s → %s)\n",
			datecore.FormatInt(natural), older.Format("02/01/2006"), newer.Format("02/01/2006"))
	}

	rows := []struct {
		label string
		value string
	}{
		{"Años (365,2425 días)", datecore.FormatNumber(totals.Years, 2)},
		{"Meses (30,436875 días)", datecore.FormatNumber(totals.Months, 2)},
		{"Semanas", datecore.FormatNumber(totals.Weeks, 2)},
		{"Días (24 h)", datecore.FormatNumber(totals.Days24, 2)},
		{"Horas", datecore.FormatNumber(totals.Hours, 2)},
		{"Minutos", datecore.FormatNumber(totals.Minutes, 2)},
		{"Segundos", datecore.FormatNumber(totals.Seconds, 2)},
		{"Milisegundos", datecore.FormatNumber(totals.Milliseconds, 0)},
	}
	fmt.Fprintln(w, "\nTotales:")
	for _, row := range rows {
		fmt.Fprintf(w, "  %s %s\n", padRight(row.label, 24), padLeft(row.value, 18))
	}
	return nil
}

func printShift(w io.Writer, cmd, dateStr, durStr string, opts options, loc *time.Location) error {
	start, err := datecore.ParseDate(dateStr, loc)
	if err != nil {
		return err
	}
	d, err := datecore.ParseDelta(durStr)
	if err != nil {
		return err
	}

	sign := 1
	if cmd == "sub" {
		sign = -1
	}
	if d.Negative() {
		sign = -sign
		d = d.Abs()
	}
	operation := "sumar"
	if sign < 0 {
		operation = "restar"
	}
	result := datecore.ApplyDelta(start, d, sign)
	desc := d.String()
	warning := datecore.DSTWarning("Fecha base", dateStr, start, loc)

	if opts.json {
		payload := shiftJSON{
			Operacion: operation,
			Duracion:  desc,
			Inicio:    start.Format(time.RFC3339),
			Resultado: result.Format(time.RFC3339),
			Aviso:     warning,
		}
		return writeJSON(w, payload)
	}

	if opts.zone != "" {
		fmt.Fprintf(w, "Zona: %s (%s)\n\n", opts.zone, zoneDescription(start))
	}
	fmt.Fprintf(w, "Operación: %s %s\n", operation, desc)
	fmt.Fprintf(w, "Inicio:    %s\n", humanDate(start))
	fmt.Fprintf(w, "Resultado: %s\n", humanDate(result))
	fmt.Fprintf(w, "ISO 8601:  %s\n", result.Format(time.RFC3339Nano))
	if warning != "" {
		fmt.Fprintf(w, "Aviso:     %s\n", warning)
	}
	return nil
}

func printAge(w io.Writer, birthStr, atStr string, opts options, loc *time.Location) error {
	birth, err := datecore.ParseDate(birthStr, loc)
	if err != nil {
		return err
	}
	if strings.TrimSpace(atStr) == "" {
		atStr = "hoy"
	}
	at, err := datecore.ParseDate(atStr, loc)
	if err != nil {
		return err
	}
	age, err := datecore.AgeAt(birth, at)
	if err != nil {
		return err
	}

	warning := datecore.DSTWarning("Nacimiento", birthStr, birth, loc)
	if w := datecore.DSTWarning("Referencia", atStr, at, loc); w != "" {
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

	if opts.json {
		return writeJSON(w, ageJSON{
			Nacimiento:       birth.Format(time.RFC3339),
			Referencia:       at.Format(time.RFC3339),
			Anios:            age.Years,
			Meses:            age.Months,
			Dias:             age.Days,
			DiasTotales:      age.TotalDays,
			Proximo:          age.NextBirthday.Format("2006-01-02"),
			DiasParaProximo:  age.DaysToNext,
			DiaSemanaProximo: age.NextWeekday,
			CumpleAnios:      age.Turning,
			EsHoy:            age.IsBirthday,
			Ajuste29Febrero:  age.LeapDayAdjustment,
			Aviso:            warning,
		})
	}

	fmt.Fprintf(w, "Nacimiento: %s\n", humanDate(birth))
	fmt.Fprintf(w, "Referencia: %s\n\n", humanDate(at))
	weeks := age.TotalDays / 7
	rest := age.TotalDays % 7
	restWord := "días"
	if rest == 1 {
		restWord = "día"
	}
	fmt.Fprintf(w, "Edad: %d años, %d meses y %d días\n", age.Years, age.Months, age.Days)
	fmt.Fprintf(w, "Días de vida: %s (%s semanas y %s %s)\n",
		datecore.FormatInt(age.TotalDays), datecore.FormatInt(weeks), datecore.FormatInt(rest), restWord)
	if age.IsBirthday {
		fmt.Fprintf(w, "¡Hoy cumple %d años!\n", age.Turning)
	} else {
		fmt.Fprintf(w, "Próximo cumpleaños: %s (%s), dentro de %s días. Cumplirá %d.\n",
			age.NextBirthday.Format("02/01/2006"), age.NextWeekday,
			datecore.FormatInt(age.DaysToNext), age.Turning)
	}
	if warning != "" {
		fmt.Fprintf(w, "Aviso: %s\n", warning)
	}
	return nil
}

func printWorkdays(w io.Writer, startStr, endStr string, opts options, loc *time.Location) error {
	start, err := datecore.ParseDate(startStr, loc)
	if err != nil {
		return err
	}
	end, err := datecore.ParseDate(endStr, loc)
	if err != nil {
		return err
	}
	holidays, err := cliHolidaySet(opts, start, end, loc)
	if err != nil {
		return err
	}
	list := cliHolidayList(opts, start, end, loc)
	workdays := datecore.WorkdaysBetween(start, end, holidays, opts.inclusive)
	natural := datecore.CivilDays(start, end)
	if natural < 0 {
		natural = -natural
	}
	if opts.inclusive {
		natural++
	}

	if opts.json {
		return writeJSON(w, workdaysJSON{
			Fecha1:         start.Format(time.RFC3339),
			Fecha2:         end.Format(time.RFC3339),
			DiasLaborables: workdays,
			DiasNaturales:  natural,
			Pais:           opts.country,
			Festivos:       holidayJSONs(list),
		})
	}

	if opts.zone != "" {
		fmt.Fprintf(w, "Zona: %s (%s)\n\n", opts.zone, zoneDescription(start))
	}
	fmt.Fprintf(w, "Fecha 1: %s\n", humanDate(start))
	fmt.Fprintf(w, "Fecha 2: %s\n\n", humanDate(end))
	fmt.Fprintf(w, "Días laborables: %s\n", datecore.FormatInt(int64(workdays)))
	if opts.inclusive {
		fmt.Fprintf(w, "Días naturales (ambos extremos incluidos): %s\n", datecore.FormatInt(natural))
	} else {
		fmt.Fprintf(w, "Días naturales: %s\n", datecore.FormatInt(natural))
	}
	if len(list) > 0 {
		fmt.Fprintln(w, "\nFestivos en el rango:")
		for _, h := range list {
			fmt.Fprintf(w, "  %s  %s\n", h.Date.Format("02/01/2006"), h.Name)
		}
	}
	return nil
}

func printAddWorkdays(w io.Writer, dateStr, daysStr string, opts options, loc *time.Location) error {
	start, err := datecore.ParseDate(dateStr, loc)
	if err != nil {
		return err
	}
	days, err := strconv.Atoi(strings.TrimSpace(daysStr))
	if err != nil {
		return fmt.Errorf("número de días no válido: %q", daysStr)
	}
	holidays, err := cliHolidaySet(opts, start, start, loc)
	if err != nil {
		return err
	}
	result := datecore.AddWorkdays(start, days, holidays)
	list := cliHolidayList(opts, start, result, loc)
	operation := "sumar"
	if days < 0 {
		operation = "restar"
	}
	abs := days
	if abs < 0 {
		abs = -abs
	}

	if opts.json {
		return writeJSON(w, workShiftJSON{
			Operacion:      operation,
			DiasLaborables: abs,
			Inicio:         start.Format(time.RFC3339),
			Resultado:      result.Format(time.RFC3339),
			Festivos:       holidayJSONs(list),
		})
	}

	if opts.zone != "" {
		fmt.Fprintf(w, "Zona: %s (%s)\n\n", opts.zone, zoneDescription(start))
	}
	fmt.Fprintf(w, "Operación: %s %s días laborables\n", operation, datecore.FormatInt(int64(abs)))
	fmt.Fprintf(w, "Inicio:    %s\n", humanDate(start))
	fmt.Fprintf(w, "Resultado: %s\n", humanDate(result))
	fmt.Fprintf(w, "ISO 8601:  %s\n", result.Format(time.RFC3339Nano))
	if len(list) > 0 {
		fmt.Fprintln(w, "\nFestivos en el rango:")
		for _, h := range list {
			fmt.Fprintf(w, "  %s  %s\n", h.Date.Format("02/01/2006"), h.Name)
		}
	}
	return nil
}

func printFacts(w io.Writer, dateStr string, opts options, loc *time.Location) error {
	t, err := datecore.ParseDate(dateStr, loc)
	if err != nil {
		return err
	}
	f := datecore.Facts(t)
	if opts.json {
		return writeJSON(w, factsJSON{
			Fecha:         t.Format(time.RFC3339),
			FechaHumana:   datecore.LongDate(t),
			DiaSemana:     f.Weekday,
			SemanaISO:     f.ISOWeek,
			AnioISO:       f.ISOYear,
			DiaDelAnio:    f.DayOfYear,
			DiasDelAnio:   f.DaysInYear,
			Trimestre:     f.Quarter,
			DiasDelMes:    f.DaysInMonth,
			QuedanDelMes:  f.DaysLeftInMonth,
			QuedanDelAnio: f.DaysLeftInYear,
			Bisiesto:      f.LeapYear,
		})
	}

	fmt.Fprintf(w, "Fecha: %s\n", humanDate(t))
	fmt.Fprintf(w, "En largo: %s\n\n", datecore.LongDate(t))
	fmt.Fprintf(w, "Día de la semana: %s\n", f.Weekday)
	fmt.Fprintf(w, "Semana ISO:       %d (año ISO %d)\n", f.ISOWeek, f.ISOYear)
	fmt.Fprintf(w, "Día del año:      %d de %d\n", f.DayOfYear, f.DaysInYear)
	fmt.Fprintf(w, "Trimestre:        T%d\n", f.Quarter)
	fmt.Fprintf(w, "Días del mes:     %d\n", f.DaysInMonth)
	fmt.Fprintf(w, "Quedan del mes:   %d\n", f.DaysLeftInMonth)
	fmt.Fprintf(w, "Quedan del año:   %d\n", f.DaysLeftInYear)
	fmt.Fprintf(w, "Año bisiesto:     %s\n", yesNo(f.LeapYear))
	return nil
}

func printConvert(w io.Writer, dateStr, fromZone, toZone string, opts options) error {
	from, err := datecore.ResolveLocation(fromZone)
	if err != nil {
		return fmt.Errorf("zona de origen no válida: %q", fromZone)
	}
	to, err := datecore.ResolveLocation(toZone)
	if err != nil {
		return fmt.Errorf("zona de destino no válida: %q", toZone)
	}
	t, err := datecore.ParseDate(dateStr, from)
	if err != nil {
		return err
	}
	converted := t.In(to)
	warning := datecore.DSTWarning("Hora de origen", dateStr, t, from)

	if opts.json {
		return writeJSON(w, convertJSON{
			Fecha:     t.Format(time.RFC3339),
			Origen:    from.String(),
			Destino:   to.String(),
			FechaISO:  converted.Format(time.RFC3339),
			Resultado: humanDate(converted),
			Aviso:     warning,
		})
	}

	fmt.Fprintf(w, "Origen:    %s (%s)\n", humanDate(t), zoneDescription(t))
	fmt.Fprintf(w, "Destino:   %s (%s)\n", humanDate(converted), zoneDescription(converted))
	fmt.Fprintf(w, "ISO 8601:  %s → %s\n", t.Format(time.RFC3339), converted.Format(time.RFC3339))
	if warning != "" {
		fmt.Fprintf(w, "Aviso:     %s\n", warning)
	}
	return nil
}

func printICS(w io.Writer, dateStr, title string, opts options, loc *time.Location) error {
	t, err := datecore.ParseDate(dateStr, loc)
	if err != nil {
		return err
	}
	allDay := opts.allDay || !strings.Contains(dateStr, ":")
	start := t
	end := t.Add(time.Hour)
	if allDay {
		start = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		end = time.Time{}
	}
	content := datecore.EventICS(title, "", "", start, end, allDay)

	if opts.out != "" {
		if err := os.WriteFile(opts.out, []byte(content), 0o644); err != nil {
			return fmt.Errorf("no se pudo escribir %s: %w", opts.out, err)
		}
		fmt.Fprintf(w, "Evento guardado en %s\n", opts.out)
		return nil
	}
	if opts.json {
		return writeJSON(w, map[string]string{"ics": content})
	}
	fmt.Fprint(w, content)
	return nil
}

func interactive(w io.Writer, opts options, loc *time.Location) error {
	fmt.Fprintln(w, "Chiquito de la Calzada · CLI — suma, resta y diferencia de fechas")
	fmt.Fprintf(w, "Zona: %s (%s)\n", zoneTitle(opts), zoneDescription(time.Now().In(loc)))
	fmt.Fprintln(w, "Formatos: 2026-11-18, 18/11/2026, \"2026-11-18 14:30\", RFC 3339,")
	fmt.Fprintln(w, "          \"hoy\", \"mañana\", \"en 3 días\", \"el próximo lunes\".")
	fmt.Fprintln(w, "Escribe \"add\" o \"sub\" para sumar/restar, o \"salir\" para terminar.")
	fmt.Fprintln(w)

	sc := bufio.NewScanner(stdin)
	ask := func(label string) (string, error) {
		fmt.Fprint(w, label)
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return "", err
			}
			return "", io.EOF
		}
		return strings.TrimSpace(sc.Text()), nil
	}

	for {
		s1, err := ask("Fecha 1 (Enter = hoy): ")
		if err == io.EOF {
			fmt.Fprintln(w)
			return nil
		}
		if err != nil {
			return fmt.Errorf("no se pudo leer la fecha 1: %w", err)
		}
		switch strings.ToLower(s1) {
		case "salir", "exit", "quit", "q":
			fmt.Fprintln(w, "¡Hasta luego!")
			return nil
		case "ayuda", "help", "?":
			fmt.Fprint(w, usageText)
			continue
		case "add", "sumar", "sub", "restar":
			mode := "add"
			if strings.HasPrefix(strings.ToLower(s1), "sub") || strings.HasPrefix(strings.ToLower(s1), "restar") {
				mode = "sub"
			}
			base, err := ask("Fecha base: ")
			if err == io.EOF {
				fmt.Fprintln(w)
				return nil
			}
			if err != nil {
				return fmt.Errorf("no se pudo leer la fecha base: %w", err)
			}
			dur, err := ask("Duración: ")
			if err == io.EOF {
				fmt.Fprintln(w)
				return nil
			}
			if err != nil {
				return fmt.Errorf("no se pudo leer la duración: %w", err)
			}
			fmt.Fprintln(w)
			if base == "" || dur == "" {
				fmt.Fprintln(w, "error: la fecha base y la duración son obligatorias.")
				fmt.Fprintln(w)
			} else if err := printShift(w, mode, base, dur, opts, loc); err != nil {
				fmt.Fprintf(w, "error: %v\n\n", err)
			}
			if !askAgain(w, ask) {
				return nil
			}
			continue
		}
		if s1 == "" {
			s1 = "hoy"
		}

		s2, err := ask("Fecha 2 (Enter = hoy): ")
		if err == io.EOF {
			fmt.Fprintln(w)
			return nil
		}
		if err != nil {
			return fmt.Errorf("no se pudo leer la fecha 2: %w", err)
		}
		if s2 == "" {
			s2 = "hoy"
		}
		fmt.Fprintln(w)

		if err := printDiff(w, s1, s2, opts, loc); err != nil {
			fmt.Fprintf(w, "error: %v\n\n", err)
			continue
		}

		if !askAgain(w, ask) {
			return nil
		}
	}
}

func askAgain(w io.Writer, ask func(string) (string, error)) bool {
	// La pausa evita que la ventana se cierre sola al abrir el .exe
	// con doble clic.
	again, err := ask("\n¿Otra consulta? [s/N]: ")
	if err == io.EOF {
		fmt.Fprintln(w)
		return false
	}
	if err != nil {
		fmt.Fprintf(w, "error: %v\n", err)
		return false
	}
	fmt.Fprintln(w)
	switch strings.ToLower(again) {
	case "s", "si", "sí", "y", "yes":
		return true
	default:
		return false
	}
}

func cliHolidaySet(opts options, start, end time.Time, loc *time.Location) (map[string]bool, error) {
	set := datecore.HolidaySetYears(opts.country, start, end, loc)
	extra, err := datecore.ParseHolidayDates(opts.holidays, loc)
	if err != nil {
		return nil, err
	}
	for k := range extra {
		set[k] = true
	}
	return set, nil
}

func cliHolidayList(opts options, start, end time.Time, loc *time.Location) []datecore.Holiday {
	if end.Before(start) {
		start, end = end, start
	}
	from := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	to := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)
	seen := map[string]bool{}
	var out []datecore.Holiday
	for year := from.Year() - 1; year <= to.Year()+1; year++ {
		for _, h := range datecore.Holidays(opts.country, year, loc) {
			day := time.Date(h.Date.Year(), h.Date.Month(), h.Date.Day(), 0, 0, 0, 0, loc)
			if day.Before(from) || day.After(to) {
				continue
			}
			key := day.Format("2006-01-02")
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, datecore.Holiday{Date: day, Name: h.Name})
		}
	}
	for _, raw := range opts.holidays {
		t, err := datecore.ParseDate(strings.TrimSpace(raw), loc)
		if err != nil {
			continue
		}
		key := t.Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, datecore.Holiday{Date: t, Name: "Festivo manual"})
	}
	sortHolidays(out)
	return out
}

func sortHolidays(list []datecore.Holiday) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].Date.Before(list[j-1].Date); j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

func holidayJSONs(list []datecore.Holiday) []holidayJSON {
	out := make([]holidayJSON, 0, len(list))
	for _, h := range list {
		out = append(out, holidayJSON{Fecha: h.Date.Format("02/01/2006"), Nombre: h.Name})
	}
	return out
}

func writeJSON(w io.Writer, payload interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(payload)
}

func humanDate(t time.Time) string {
	layout := "02/01/2006 15:04:05"
	if t.Nanosecond() != 0 {
		layout += ".000"
	}
	return fmt.Sprintf("%s (%s)", t.Format(layout), weekdayES(t.Weekday()))
}

func zoneTitle(o options) string {
	switch {
	case o.utc:
		return "UTC"
	case o.zone != "":
		return o.zone
	default:
		return "local"
	}
}

// zoneDescription muestra la abreviatura y el offset de la zona en esa fecha.
func zoneDescription(t time.Time) string {
	abbr, offset := t.Zone()
	if abbr == "" {
		abbr = "UTC"
	}
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return fmt.Sprintf("%s, UTC%s%02d:%02d", abbr, sign, offset/3600, (offset%3600)/60)
}

func weekdayES(d time.Weekday) string {
	return datecore.WeekdayES(d)
}

func yesNo(v bool) string {
	if v {
		return "sí"
	}
	return "no"
}

func padRight(s string, width int) string {
	if n := utf8.RuneCountInString(s); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

func padLeft(s string, width int) string {
	if n := utf8.RuneCountInString(s); n < width {
		return strings.Repeat(" ", width-n) + s
	}
	return s
}
