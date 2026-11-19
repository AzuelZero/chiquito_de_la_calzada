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

Formatos de fecha:
  2026-11-18, 18/11/2026, 18-11-2026
  2026-11-18 14:30, 18/11/2026 14:30[:05]
  2026-11-18T14:30:00+01:00 (RFC 3339)
  hoy | now | ahora

Duración (para add/sub):
  1a 2mes 3sem 4d 5h 6min 7s   (también 1y 2mo 3w 4d 5h 6m 7s)
  Un número suelto se interpreta como días (ej.: 30).
  Ejemplos: "1a2mes", "45min", "2 semanas 3 días", "-1mes".

Opciones:
  -u, --utc         Interpretar las fechas en UTC (por defecto, hora local)
  -z, --tz <zona>   Zona IANA, p. ej. Europe/Madrid o America/Bogota
  -j, --json        Salida en JSON
  -i, --inclusive   Contar ambos extremos en los días naturales
  -h, --help        Mostrar esta ayuda
`

// options contiene las opciones de línea de comandos.
type options struct {
	utc       bool
	json      bool
	inclusive bool
	help      bool
	zone      string
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
}

type shiftJSON struct {
	Operacion string `json:"operacion"`
	Duracion  string `json:"duracion"`
	Inicio    string `json:"inicio"`
	Resultado string `json:"resultado"`
}

var negativeNumber = regexp.MustCompile(`^-\d`)

// stdin se puede sustituir en los tests.
var stdin io.Reader = os.Stdin

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, w io.Writer) error {
	opts, pos, err := splitArgs(args)
	if err != nil {
		return err
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
			return errors.New("uso: chiquito_de_la_calzada_cli diff <fecha1> <fecha2>")
		}
		return printDiff(w, pos[1], pos[2], opts, loc)
	case "add", "sub":
		if len(pos) != 3 {
			return fmt.Errorf("uso: chiquito_de_la_calzada_cli %s <fecha> <duración>", strings.ToLower(pos[0]))
		}
		return printShift(w, strings.ToLower(pos[0]), pos[1], pos[2], opts, loc)
	case "help":
		fmt.Fprint(w, usageText)
		return nil
	}

	switch len(pos) {
	case 1:
		return printDiff(w, pos[0], "hoy", opts, loc)
	case 2:
		return printDiff(w, pos[0], pos[1], opts, loc)
	default:
		return fmt.Errorf("argumentos no reconocidos: %s (usa --help)", strings.Join(pos, " "))
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
		case a == "-z", a == "--tz", a == "-tz":
			if i+1 >= len(args) {
				return o, nil, fmt.Errorf("falta el valor de %s", a)
			}
			i++
			o.zone = strings.TrimSpace(args[i])
			if o.zone == "" {
				return o, nil, errors.New("zona horaria vacía")
			}
		case strings.HasPrefix(a, "--tz="):
			o.zone = strings.TrimSpace(strings.TrimPrefix(a, "--tz="))
			if o.zone == "" {
				return o, nil, errors.New("zona horaria vacía")
			}
		default:
			if strings.HasPrefix(a, "-") && !negativeNumber.MatchString(a) {
				return o, nil, fmt.Errorf("opción desconocida: %s", a)
			}
			pos = append(pos, a)
		}
	}
	return o, pos, nil
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
	loc, err := time.LoadLocation(o.zone)
	if err != nil {
		return nil, fmt.Errorf("zona horaria no válida: %q (usa un nombre IANA, p. ej. Europe/Madrid)", o.zone)
	}
	return loc, nil
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
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(payload)
	}

	if opts.zone != "" {
		fmt.Fprintf(w, "Zona: %s (%s)\n\n", opts.zone, zoneDescription(d1))
	}
	fmt.Fprintf(w, "Fecha 1: %s\n", humanDate(d1))
	fmt.Fprintf(w, "Fecha 2: %s\n\n", humanDate(d2))
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

	if opts.json {
		payload := shiftJSON{
			Operacion: operation,
			Duracion:  desc,
			Inicio:    start.Format(time.RFC3339),
			Resultado: result.Format(time.RFC3339),
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(payload)
	}

	if opts.zone != "" {
		fmt.Fprintf(w, "Zona: %s (%s)\n\n", opts.zone, zoneDescription(start))
	}
	fmt.Fprintf(w, "Operación: %s %s\n", operation, desc)
	fmt.Fprintf(w, "Inicio:    %s\n", humanDate(start))
	fmt.Fprintf(w, "Resultado: %s\n", humanDate(result))
	fmt.Fprintf(w, "ISO 8601:  %s\n", result.Format(time.RFC3339))
	return nil
}

func interactive(w io.Writer, opts options, loc *time.Location) error {
	fmt.Fprintln(w, "Chiquito de la Calzada · CLI — suma, resta y diferencia de fechas")
	fmt.Fprintf(w, "Zona: %s (%s)\n", zoneTitle(opts), zoneDescription(time.Now().In(loc)))
	fmt.Fprintln(w, "Formatos: 2026-11-18, 18/11/2026, \"2026-11-18 14:30\", RFC 3339 o \"hoy\".")
	fmt.Fprintln(w, "Escribe \"salir\" en la primera fecha para terminar.")
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

		// La pausa evita que la ventana se cierre sola al abrir el .exe
		// con doble clic.
		again, err := ask("\n¿Otra consulta? [s/N]: ")
		if err == io.EOF {
			fmt.Fprintln(w)
			return nil
		}
		if err != nil {
			return fmt.Errorf("no se pudo leer la respuesta: %w", err)
		}
		fmt.Fprintln(w)
		switch strings.ToLower(again) {
		case "s", "si", "sí", "y", "yes":
			continue
		default:
			return nil
		}
	}
}

func humanDate(t time.Time) string {
	return fmt.Sprintf("%s (%s)", t.Format("02/01/2006 15:04:05"), weekdayES(t.Weekday()))
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
	return [...]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}[d]
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
