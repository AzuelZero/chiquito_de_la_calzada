package datecore

import (
	"strings"
	"testing"
	"time"
)

func TestFacts(t *testing.T) {
	f := Facts(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))
	if f.ISOWeek != 1 || f.ISOYear != 2025 {
		t.Errorf("semana ISO = %d/%d, quiero 1/2025", f.ISOWeek, f.ISOYear)
	}
	if f.DayOfYear != 366 || f.DaysInYear != 366 || !f.LeapYear {
		t.Errorf("datos de año = %+v", f)
	}
	if f.Quarter != 4 || f.DaysInMonth != 31 || f.DaysLeftInMonth != 0 || f.DaysLeftInYear != 0 {
		t.Errorf("datos de mes = %+v", f)
	}
	if f.Weekday != "martes" {
		t.Errorf("Weekday = %q", f.Weekday)
	}

	f = Facts(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
	if f.ISOWeek != 52 || f.ISOYear != 2022 {
		t.Errorf("1 ene 2023: semana ISO = %d/%d, quiero 52/2022", f.ISOWeek, f.ISOYear)
	}
}

func TestAgeAt(t *testing.T) {
	birth := time.Date(2000, time.February, 29, 0, 0, 0, 0, time.UTC)

	got, err := AgeAt(birth, time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("AgeAt: %v", err)
	}
	if got.Years != 26 || got.Months != 0 || got.Days != 0 {
		t.Errorf("edad = %d a %d m %d d, quiero 26 a 0 m 0 d", got.Years, got.Months, got.Days)
	}
	if !got.IsBirthday || got.Turning != 26 || got.DaysToNext != 0 {
		t.Errorf("cumpleaños = %+v", got)
	}
	if got.TotalDays != 9496 {
		t.Errorf("TotalDays = %d, quiero 9496", got.TotalDays)
	}
	if !got.LeapDayAdjustment {
		t.Error("esperaba el ajuste de 29 de febrero")
	}

	got, err = AgeAt(time.Date(1990, time.June, 15, 0, 0, 0, 0, time.UTC), time.Date(2026, time.November, 23, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("AgeAt: %v", err)
	}
	if got.Years != 36 || got.Months != 5 || got.Days != 8 {
		t.Errorf("edad = %d a %d m %d d, quiero 36 a 5 m 8 d", got.Years, got.Months, got.Days)
	}
	if got.NextBirthday.Year() != 2027 || got.DaysToNext <= 0 || got.Turning != 37 {
		t.Errorf("próximo cumpleaños = %+v", got)
	}

	if _, err := AgeAt(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Error("una fecha de nacimiento futura debería fallar")
	}
}

func TestWorkdays(t *testing.T) {
	holidays := map[string]bool{"2024-01-08": true}

	start := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC) // lunes
	end := time.Date(2024, 1, 9, 9, 0, 0, 0, time.UTC)   // martes siguiente
	if got := WorkdaysBetween(start, end, holidays, false); got != 5 {
		t.Errorf("WorkdaysBetween = %d, quiero 5 (el 8 es festivo)", got)
	}
	if got := WorkdaysBetween(start, end, nil, false); got != 6 {
		t.Errorf("WorkdaysBetween sin festivos = %d, quiero 6", got)
	}
	if got := WorkdaysBetween(end, start, nil, false); got != 6 {
		t.Errorf("WorkdaysBetween invertido = %d, quiero 6", got)
	}
	if got := WorkdaysBetween(start, start, nil, true); got != 1 {
		t.Errorf("WorkdaysBetween inclusivo del mismo día = %d, quiero 1", got)
	}
	if got := WorkdaysBetween(time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC), nil, false); got != 0 {
		t.Errorf("fin de semana = %d, quiero 0", got)
	}

	friday := time.Date(2024, 1, 5, 9, 0, 0, 0, time.UTC)
	next := AddWorkdays(friday, 1, nil)
	if next.Weekday() != time.Monday || next.Day() != 8 || next.Hour() != 9 {
		t.Errorf("viernes + 1 laborable = %v, quiero lunes 8 a las 9", next)
	}
	next = AddWorkdays(friday, 1, holidays)
	if next.Weekday() != time.Tuesday || next.Day() != 9 {
		t.Errorf("viernes + 1 laborable con festivo = %v, quiero martes 9", next)
	}
	if got := AddWorkdays(friday, -1, nil); got.Weekday() != time.Thursday || got.Day() != 4 {
		t.Errorf("viernes - 1 laborable = %v, quiero jueves 4", got)
	}
}

func TestEasterAndHolidays(t *testing.T) {
	cases := []struct {
		year  int
		month time.Month
		day   int
	}{
		{2024, time.March, 31},
		{2025, time.April, 20},
		{2026, time.April, 5},
		{2027, time.March, 28},
	}
	for _, c := range cases {
		got := easterSunday(c.year, time.UTC)
		if got.Month() != c.month || got.Day() != c.day {
			t.Errorf("Pascua %d = %v, quiero %d/%d", c.year, got, c.day, c.month)
		}
	}

	es := Holidays("es", 2024, time.UTC)
	if len(es) != 10 {
		t.Fatalf("festivos ES = %d, quiero 10", len(es))
	}
	foundGoodFriday := false
	for _, h := range es {
		if h.Name == "Viernes Santo" {
			foundGoodFriday = true
			if h.Date.Month() != time.March || h.Date.Day() != 29 {
				t.Errorf("Viernes Santo 2024 = %v", h.Date)
			}
		}
	}
	if !foundGoodFriday {
		t.Error("falta el Viernes Santo")
	}

	for _, country := range []string{"es", "mx", "co", "ar"} {
		if got := HolidaySet(country, []int{2024, 2025}, time.UTC); len(got) == 0 {
			t.Errorf("%s no generó festivos", country)
		}
	}
}

func TestNaturalDates(t *testing.T) {
	got, err := ParseDate("18 de noviembre de 2026", time.UTC)
	if err != nil {
		t.Fatalf("ParseDate: %v", err)
	}
	want := time.Date(2026, time.November, 18, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("fecha en español = %v, quiero %v", got, want)
	}

	got, err = ParseDate("18 nov 2026 a las 14:30", time.UTC)
	if err != nil {
		t.Fatalf("ParseDate con hora: %v", err)
	}
	want = time.Date(2026, time.November, 18, 14, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("fecha con hora = %v, quiero %v", got, want)
	}

	got, err = ParseDate("en 3 días", time.UTC)
	if err != nil {
		t.Fatalf("ParseDate en 3 días: %v", err)
	}
	diff := CivilDays(time.Now().UTC(), got)
	if diff < 2 || diff > 3 {
		t.Errorf("en 3 días = %v (diferencia %d)", got, diff)
	}

	got, err = ParseDate("hace una semana", time.UTC)
	if err != nil {
		t.Fatalf("ParseDate hace una semana: %v", err)
	}
	diff = CivilDays(got, time.Now().UTC())
	if diff < 6 || diff > 7 {
		t.Errorf("hace una semana = %v (diferencia %d)", got, diff)
	}

	got, err = ParseDate("próximo lunes", time.UTC)
	if err != nil {
		t.Fatalf("ParseDate próximo lunes: %v", err)
	}
	if got.Weekday() != time.Monday {
		t.Errorf("próximo lunes = %v (%s)", got, got.Weekday())
	}
}

func TestBuildICS(t *testing.T) {
	start := time.Date(2026, time.November, 18, 14, 30, 0, 0, time.UTC)
	ics := EventICS("Cita; importante, con acentos: ñ", "Línea 1\nLínea 2", "Madrid", start, start.Add(time.Hour), false)
	for _, want := range []string{
		"BEGIN:VCALENDAR",
		"BEGIN:VEVENT",
		"DTSTART:20261118T143000Z",
		"DTEND:20261118T153000Z",
		"SUMMARY:Cita\\; importante\\, con acentos: ñ",
		"DESCRIPTION:Línea 1\\nLínea 2",
		"END:VCALENDAR",
	} {
		if !strings.Contains(ics, want) {
			t.Errorf("el ICS no contiene %q:\n%s", want, ics)
		}
	}
	if strings.Contains(ics, "\n") && !strings.Contains(ics, "\r\n") {
		t.Error("el ICS debe usar CRLF")
	}

	allDay := EventICS("Día completo", "", "", start, time.Time{}, true)
	if !strings.Contains(allDay, "DTSTART;VALUE=DATE:20261118") {
		t.Errorf("evento de día completo:\n%s", allDay)
	}
	if !strings.Contains(allDay, "DTEND;VALUE=DATE:20261119") {
		t.Errorf("fin de día completo:\n%s", allDay)
	}
}
