package datecore

import (
	"errors"
	"time"
)

// AgeResult describe la edad exacta y el próximo cumpleaños.
type AgeResult struct {
	Years             int
	Months            int
	Days              int
	TotalDays         int64
	NextBirthday      time.Time
	DaysToNext        int64
	NextWeekday       string
	Turning           int
	IsBirthday        bool
	LeapDayAdjustment bool
}

// AgeAt calcula la edad de alguien nacido en birth en el momento at.
func AgeAt(birth, at time.Time) (AgeResult, error) {
	if birth.After(at) {
		return AgeResult{}, errors.New("la fecha de nacimiento es posterior a la fecha de referencia")
	}

	b := dateOnly(birth)
	a := dateOnly(at)
	bd := DiffBreakdown(b, a)

	next := birthdayInYear(b, a.Year())
	if next.Before(a) {
		next = birthdayInYear(b, a.Year()+1)
	}

	adjusted := b.Month() == time.February && b.Day() == 29 && !IsLeapYear(next.Year())

	return AgeResult{
		Years:             bd.Years,
		Months:            bd.Months,
		Days:              bd.Days,
		TotalDays:         int64(CivilDays(b, a)),
		NextBirthday:      next,
		DaysToNext:        int64(CivilDays(a, next)),
		NextWeekday:       WeekdayES(next.Weekday()),
		Turning:           next.Year() - b.Year(),
		IsBirthday:        next.Equal(a),
		LeapDayAdjustment: adjusted,
	}, nil
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// birthdayInYear construye el cumpleaños en un año concreto. Si la persona
// nació un 29 de febrero y el año no es bisiesto, se usa el 28 de febrero.
func birthdayInYear(birth time.Time, year int) time.Time {
	day := birth.Day()
	if birth.Month() == time.February && day == 29 && !IsLeapYear(year) {
		day = 28
	}
	return time.Date(year, birth.Month(), day, 0, 0, 0, 0, birth.Location())
}
