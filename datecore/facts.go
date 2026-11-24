package datecore

import (
	"fmt"
	"time"
)

// DateFacts reúne los datos de calendario de una fecha: semana ISO, día del
// año, trimestre, bisiesto y días que quedan de mes y de año.
type DateFacts struct {
	Weekday         string
	ISOWeek         int
	ISOYear         int
	DayOfYear       int
	DaysInYear      int
	Quarter         int
	DaysInMonth     int
	DaysLeftInMonth int
	DaysLeftInYear  int
	LeapYear        bool
}

// Facts calcula los datos de calendario de t.
func Facts(t time.Time) DateFacts {
	isoYear, isoWeek := t.ISOWeek()
	year, month, day := t.Date()
	daysInMonth := DaysInMonth(year, month)
	daysInYear := 365
	if IsLeapYear(year) {
		daysInYear = 366
	}
	dayOfYear := t.YearDay()
	return DateFacts{
		Weekday:         WeekdayES(t.Weekday()),
		ISOWeek:         isoWeek,
		ISOYear:         isoYear,
		DayOfYear:       dayOfYear,
		DaysInYear:      daysInYear,
		Quarter:         (int(month)-1)/3 + 1,
		DaysInMonth:     daysInMonth,
		DaysLeftInMonth: daysInMonth - day,
		DaysLeftInYear:  daysInYear - dayOfYear,
		LeapYear:        IsLeapYear(year),
	}
}

// DaysInMonth devuelve los días que tiene un mes concreto.
func DaysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// IsLeapYear indica si el año es bisiesto en el calendario gregoriano.
func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// WeekdayES devuelve el nombre del día de la semana en español.
func WeekdayES(d time.Weekday) string {
	return [...]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}[d]
}

// MonthES devuelve el nombre del mes en español.
func MonthES(m time.Month) string {
	return [...]string{
		"enero", "febrero", "marzo", "abril", "mayo", "junio",
		"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
	}[m-1]
}

// HumanDate describe una fecha con hora solo si no es medianoche.
func HumanDate(t time.Time) string {
	layout := "02/01/2006"
	if t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 {
		layout += " 15:04:05"
	}
	return t.Format(layout) + " (" + WeekdayES(t.Weekday()) + ")"
}

// LongDate describe una fecha en lenguaje natural: "miércoles, 18 de
// noviembre de 2026".
func LongDate(t time.Time) string {
	return fmt.Sprintf("%s, %d de %s de %d",
		WeekdayES(t.Weekday()), t.Day(), MonthES(t.Month()), t.Year())
}
