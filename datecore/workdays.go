package datecore

import "time"

// IsWorkday indica si una fecha es de lunes a viernes y no está en el
// conjunto de festivos (claves "2006-01-02").
func IsWorkday(t time.Time, holidays map[string]bool) bool {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	return !holidays[t.Format("2006-01-02")]
}

// WorkdaysBetween cuenta los días laborables entre dos fechas. Por defecto
// excluye el extremo final, igual que los días naturales; con inclusive se
// cuenta también ese último día. Devuelve siempre un valor no negativo.
func WorkdaysBetween(start, end time.Time, holidays map[string]bool, inclusive bool) int {
	a := dateOnly(start)
	b := dateOnly(end)
	if b.Before(a) {
		a, b = b, a
	}
	count := 0
	for d := a; d.Before(b); d = d.AddDate(0, 0, 1) {
		if IsWorkday(d, holidays) {
			count++
		}
	}
	if inclusive && IsWorkday(b, holidays) {
		count++
	}
	return count
}

// AddWorkdays suma (o resta, con days negativo) días laborables a start,
// conservando la hora de pared. Los días laborables se cuentan desde el día
// siguiente; el propio start no cuenta.
func AddWorkdays(start time.Time, days int, holidays map[string]bool) time.Time {
	cursor := start
	if days > 0 {
		for remaining := days; remaining > 0; {
			cursor = cursor.AddDate(0, 0, 1)
			if IsWorkday(cursor, holidays) {
				remaining--
			}
		}
		return cursor
	}
	for remaining := days; remaining < 0; {
		cursor = cursor.AddDate(0, 0, -1)
		if IsWorkday(cursor, holidays) {
			remaining++
		}
	}
	return cursor
}

// HolidaySetYears amplía un conjunto de festivos con los años que rodean al
// rango indicado.
func HolidaySetYears(country string, start, end time.Time, loc *time.Location) map[string]bool {
	if country == "" {
		return map[string]bool{}
	}
	from, to := start.Year(), end.Year()
	if from > to {
		from, to = to, from
	}
	years := make([]int, 0, to-from+3)
	for y := from - 1; y <= to+1; y++ {
		years = append(years, y)
	}
	return HolidaySet(country, years, loc)
}
