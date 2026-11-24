package datecore

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ZoneEntry describe una zona horaria del selector.
type ZoneEntry struct {
	ID    string
	City  string
	Group string
}

var zoneGroups = map[string]string{
	"Europe":     "Europa",
	"America":    "América",
	"Asia":       "Asia",
	"Africa":     "África",
	"Australia":  "Oceanía",
	"Pacific":    "Oceanía",
	"Atlantic":   "Atlántico",
	"Indian":     "Índico",
	"Arctic":     "Ártico",
	"Antarctica": "Antártida",
}

var zoneGroupOrder = []string{
	"Europa", "América", "Asia", "África", "Oceanía",
	"Atlántico", "Índico", "Ártico", "Antártida", "Otros",
}

// zoneCityOverrides traduce las ciudades más habituales. El resto se muestra
// con el nombre del identificador IANA.
var zoneCityOverrides = map[string]string{
	"Atlantic/Canary":                "Canarias",
	"Europe/Athens":                  "Atenas",
	"Europe/Berlin":                  "Berlín",
	"Europe/Brussels":                "Bruselas",
	"Europe/Bucharest":               "Bucarest",
	"Europe/Budapest":                "Budapest",
	"Europe/Copenhagen":              "Copenhague",
	"Europe/Dublin":                  "Dublín",
	"Europe/Helsinki":                "Helsinki",
	"Europe/Kyiv":                    "Kiev",
	"Europe/Lisbon":                  "Lisboa",
	"Europe/London":                  "Londres",
	"Europe/Madrid":                  "Madrid",
	"Europe/Moscow":                  "Moscú",
	"Europe/Oslo":                    "Oslo",
	"Europe/Paris":                   "París",
	"Europe/Prague":                  "Praga",
	"Europe/Rome":                    "Roma",
	"Europe/Stockholm":               "Estocolmo",
	"Europe/Vienna":                  "Viena",
	"Europe/Warsaw":                  "Varsovia",
	"Europe/Zurich":                  "Zúrich",
	"America/Argentina/Buenos_Aires": "Buenos Aires",
	"America/Asuncion":               "Asunción",
	"America/Bogota":                 "Bogotá",
	"America/Caracas":                "Caracas",
	"America/Chicago":                "Chicago",
	"America/Costa_Rica":             "San José",
	"America/Denver":                 "Denver",
	"America/El_Salvador":            "San Salvador",
	"America/Guatemala":              "Guatemala",
	"America/Guayaquil":              "Guayaquil",
	"America/Havana":                 "La Habana",
	"America/La_Paz":                 "La Paz",
	"America/Lima":                   "Lima",
	"America/Los_Angeles":            "Los Ángeles",
	"America/Managua":                "Managua",
	"America/Mexico_City":            "Ciudad de México",
	"America/Montevideo":             "Montevideo",
	"America/New_York":               "Nueva York",
	"America/Panama":                 "Panamá",
	"America/Puerto_Rico":            "San Juan",
	"America/Santiago":               "Santiago",
	"America/Santo_Domingo":          "Santo Domingo",
	"America/Sao_Paulo":              "São Paulo",
	"America/Tegucigalpa":            "Tegucigalpa",
	"Asia/Dubai":                     "Dubái",
	"Asia/Kathmandu":                 "Katmandú",
	"Asia/Kolkata":                   "Kolkata",
	"Asia/Shanghai":                  "Shanghái",
	"Asia/Tokyo":                     "Tokio",
	"Australia/Sydney":               "Sídney",
	"Africa/Casablanca":              "Casablanca",
	"Africa/Johannesburg":            "Johannesburgo",
}

func zoneGroup(id string) string {
	prefix, _, _ := strings.Cut(id, "/")
	if group, ok := zoneGroups[prefix]; ok {
		return group
	}
	return "Otros"
}

func zoneCity(id string) string {
	if city, ok := zoneCityOverrides[id]; ok {
		return city
	}
	_, city, ok := strings.Cut(id, "/")
	if !ok {
		city = id
	}
	return strings.ReplaceAll(city, "_", " ")
}

var zoneCatalog = buildZoneCatalog()

func buildZoneCatalog() []ZoneEntry {
	entries := []ZoneEntry{
		{ID: "local", City: "Hora local", Group: "Sistema"},
		{ID: "UTC", City: "UTC", Group: "Sistema"},
	}
	for _, id := range canonicalZoneIDs {
		entries = append(entries, ZoneEntry{ID: id, City: zoneCity(id), Group: zoneGroup(id)})
	}
	groupRank := map[string]int{}
	for i, group := range zoneGroupOrder {
		groupRank[group] = i
	}
	sort.SliceStable(entries[2:], func(i, j int) bool {
		a, b := entries[2+i], entries[2+j]
		if groupRank[a.Group] != groupRank[b.Group] {
			return groupRank[a.Group] < groupRank[b.Group]
		}
		if a.City != b.City {
			return a.City < b.City
		}
		return a.ID < b.ID
	})
	return entries
}

// ZoneCatalog devuelve todas las zonas disponibles. La primera es la hora
// local del sistema y la segunda, UTC.
func ZoneCatalog() []ZoneEntry {
	out := make([]ZoneEntry, len(zoneCatalog))
	copy(out, zoneCatalog)
	return out
}

// ResolveLocation resuelve "local", "UTC" o un identificador IANA.
func ResolveLocation(name string) (*time.Location, error) {
	switch name {
	case "", "local":
		return time.Local, nil
	case "UTC":
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("zona horaria desconocida: %q", name)
	}
	return loc, nil
}
