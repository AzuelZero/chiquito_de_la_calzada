package datecore

import (
	"testing"
	"time"
)

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := ParseDate(s, time.UTC)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", s, err)
	}
	return d
}

func TestParseDate(t *testing.T) {
	cases := []struct {
		in   string
		want time.Time
	}{
		{"2024-01-31", time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)},
		{"31/01/2024", time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)},
		{"2024-01-31 14:30", time.Date(2024, 1, 31, 14, 30, 0, 0, time.UTC)},
		{"31/01/2024 14:30:05", time.Date(2024, 1, 31, 14, 30, 5, 0, time.UTC)},
		{"2024-01-31T14:30:00Z", time.Date(2024, 1, 31, 14, 30, 0, 0, time.UTC)},
		{"2024-01-31T14:30:00+01:00", time.Date(2024, 1, 31, 13, 30, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got, err := ParseDate(c.in, time.UTC)
		if err != nil {
			t.Errorf("ParseDate(%q): %v", c.in, err)
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("ParseDate(%q) = %v, quiero %v", c.in, got, c.want)
		}
	}
}

func TestParseDateInvalid(t *testing.T) {
	if _, err := ParseDate("31/02/2024", time.UTC); err == nil {
		t.Error("31/02/2024 debería fallar")
	}
	if _, err := ParseDate("", time.UTC); err == nil {
		t.Error("una fecha vacía debería fallar")
	}
}

func TestParseDelta(t *testing.T) {
	d, err := ParseDelta("1a2mes3sem4d5h6min7s")
	if err != nil {
		t.Fatalf("ParseDelta: %v", err)
	}
	want := Delta{Years: 1, Months: 2, Weeks: 3, Days: 4, Hours: 5, Minutes: 6, Seconds: 7}
	if d != want {
		t.Errorf("ParseDelta = %+v, quiero %+v", d, want)
	}

	if d, err = ParseDelta("30"); err != nil || d.Days != 30 {
		t.Errorf("ParseDelta(30) = %+v, %v; quiero 30 días", d, err)
	}
	if d, err = ParseDelta("2 semanas 3 días"); err != nil || d.Weeks != 2 || d.Days != 3 {
		t.Errorf("ParseDelta con espacios = %+v, %v", d, err)
	}
	if d, err = ParseDelta("-2d"); err != nil || d.Days != -2 {
		t.Errorf("ParseDelta(-2d) = %+v, %v", d, err)
	}
	if d, err = ParseDelta("90min"); err != nil || d.Minutes != 90 {
		t.Errorf("ParseDelta(90min) = %+v, %v", d, err)
	}
	if _, err = ParseDelta("3x"); err == nil {
		t.Error("ParseDelta(3x) debería fallar")
	}
}

func TestDiffBreakdown(t *testing.T) {
	a := mustParseDate(t, "2024-01-01 00:00:00")
	b := mustParseDate(t, "2025-03-15 14:30:05")
	want := Breakdown{Years: 1, Months: 2, Days: 14, Hours: 14, Minutes: 30, Seconds: 5}
	if got := DiffBreakdown(a, b); got != want {
		t.Errorf("DiffBreakdown = %+v, quiero %+v", got, want)
	}

	// 29 de febrero + 1 año = 28 de febrero (se ajusta al último día).
	leap := mustParseDate(t, "2024-02-29")
	next := mustParseDate(t, "2025-02-28")
	if got := DiffBreakdown(leap, next); got != (Breakdown{Years: 1}) {
		t.Errorf("2024-02-29 → 2025-02-28 = %+v, quiero 1 año", got)
	}

	// 31 de enero + 1 mes = 29 de febrero de 2024.
	jan := mustParseDate(t, "2024-01-31")
	feb := mustParseDate(t, "2024-02-29")
	if got := DiffBreakdown(jan, feb); got != (Breakdown{Months: 1}) {
		t.Errorf("2024-01-31 → 2024-02-29 = %+v, quiero 1 mes", got)
	}

	if got := DiffBreakdown(a, a); got != (Breakdown{}) {
		t.Errorf("fechas iguales = %+v, quiero cero", got)
	}
}

func TestAddClamped(t *testing.T) {
	cases := []struct {
		name string
		got  time.Time
		want time.Time
	}{
		{
			"31 ene 2024 + 1 mes",
			AddMonthsClamped(time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC), 1),
			time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			"31 ene 2023 + 1 mes",
			AddMonthsClamped(time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC), 1),
			time.Date(2023, 2, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			"29 feb 2024 + 1 año",
			AddYearsClamped(time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), 1),
			time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			"31 mar 2024 - 1 mes",
			AddMonthsClamped(time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC), -1),
			time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			"31 ene 2024 - 2 meses",
			AddMonthsClamped(time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC), -2),
			time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, c := range cases {
		if !c.got.Equal(c.want) {
			t.Errorf("%s: %v, quiero %v", c.name, c.got, c.want)
		}
	}
}

func TestApplyDelta(t *testing.T) {
	start := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	got := ApplyDelta(start, Delta{Months: 1}, -1)
	want := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("restar 1 mes a 2024-03-31 = %v, quiero %v", got, want)
	}

	got = ApplyDelta(start, Delta{Weeks: 2, Days: 3, Hours: 4}, 1)
	want = time.Date(2024, 4, 17, 4, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("sumar 2 semanas 3 días 4 h = %v, quiero %v", got, want)
	}
}

func TestCivilDays(t *testing.T) {
	a := time.Date(2024, 1, 1, 23, 0, 0, 0, time.UTC)
	b := time.Date(2024, 1, 2, 1, 0, 0, 0, time.UTC)
	if got := CivilDays(a, b); got != 1 {
		t.Errorf("CivilDays = %d, quiero 1", got)
	}
	if got := CivilDays(b, a); got != -1 {
		t.Errorf("CivilDays inverso = %d, quiero -1", got)
	}
}

func TestComputeTotals(t *testing.T) {
	totals := ComputeTotals(86400)
	if totals.Days24 != 1 || totals.Hours != 24 || totals.Minutes != 1440 ||
		totals.Seconds != 86400 || totals.Milliseconds != 86400000 {
		t.Errorf("ComputeTotals(86400) = %+v", totals)
	}
}

func TestFormatHelpers(t *testing.T) {
	if got := FormatInt(1234567); got != "1.234.567" {
		t.Errorf("FormatInt = %q", got)
	}
	if got := FormatNumber(1234.5, 2); got != "1.234,5" {
		t.Errorf("FormatNumber = %q", got)
	}
	if got := FormatNumber(0, 2); got != "0" {
		t.Errorf("FormatNumber(0) = %q", got)
	}
}
