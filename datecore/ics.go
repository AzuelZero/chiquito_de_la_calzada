package datecore

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"
)

// ICSEvent es un evento exportable a un calendario.
type ICSEvent struct {
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	AllDay      bool
}

// BuildICS genera un calendario iCalendar (RFC 5545) con los eventos dados.
// Las fechas se escriben en UTC; los eventos de día completo usan VALUE=DATE.
func BuildICS(events []ICSEvent, stamp time.Time) string {
	var b strings.Builder
	line := func(s string) {
		b.WriteString(foldICSLine(s))
		b.WriteString("\r\n")
	}
	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//Chiquito de la Calzada//Calculadora de fechas//ES")
	line("CALSCALE:GREGORIAN")
	line("METHOD:PUBLISH")
	line("X-WR-CALNAME:Chiquito de la Calzada")

	for _, ev := range events {
		start := ev.Start
		end := ev.End
		if end.IsZero() || !end.After(start) {
			end = start.Add(time.Hour)
		}
		line("BEGIN:VEVENT")
		line("UID:" + icsUID(ev, stamp))
		line("DTSTAMP:" + stamp.UTC().Format("20060102T150405Z"))
		if ev.AllDay {
			line("DTSTART;VALUE=DATE:" + start.Format("20060102"))
			line("DTEND;VALUE=DATE:" + end.AddDate(0, 0, 1).Format("20060102"))
		} else {
			line("DTSTART:" + start.UTC().Format("20060102T150405Z"))
			line("DTEND:" + end.UTC().Format("20060102T150405Z"))
		}
		line("SUMMARY:" + escapeICSText(ev.Summary))
		if ev.Description != "" {
			line("DESCRIPTION:" + escapeICSText(ev.Description))
		}
		if ev.Location != "" {
			line("LOCATION:" + escapeICSText(ev.Location))
		}
		line("END:VEVENT")
	}

	line("END:VCALENDAR")
	return b.String()
}

// EventICS genera un calendario con un único evento.
func EventICS(summary, description, location string, start, end time.Time, allDay bool) string {
	return BuildICS([]ICSEvent{{
		Summary:     summary,
		Description: description,
		Location:    location,
		Start:       start,
		End:         end,
		AllDay:      allDay,
	}}, time.Now())
}

func icsUID(ev ICSEvent, stamp time.Time) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s|%s|%d", ev.Summary, ev.Start.UTC().Format(time.RFC3339), stamp.UnixNano())
	return fmt.Sprintf("%x@chiquito-de-la-calzada", h.Sum64())
}

func escapeICSText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `;`, `\;`)
	s = strings.ReplaceAll(s, `,`, `\,`)
	s = strings.ReplaceAll(s, "\r\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\n`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// foldICSLine pliega las líneas a 75 octetos, como exige RFC 5545, sin partir
// caracteres UTF-8 por la mitad.
func foldICSLine(s string) string {
	const limit = 75
	var b strings.Builder
	lineBytes := 0
	for _, r := range s {
		size := len(string(r))
		if lineBytes+size > limit {
			b.WriteString("\r\n ")
			lineBytes = 1
		}
		b.WriteRune(r)
		lineBytes += size
	}
	return b.String()
}
