"use strict";

if ("scrollRestoration" in history) {
  history.scrollRestoration = "manual";
}

const $ = (id) => document.getElementById(id);
const nf = new Intl.NumberFormat("es-ES", { maximumFractionDigits: 2 });
const nf0 = new Intl.NumberFormat("es-ES", { maximumFractionDigits: 0 });

const STORAGE_KEY = "chiquito.estado.v2";
const HISTORY_KEY = "chiquito.historial.v1";
const FAVORITES_KEY = "chiquito.favoritos.v1";

let zone = "local";
let zoneText = "";
let zoneMode = "wall";
let soundEnabled = true;
let ready = false;
let hasDiff = false;
let hasShift = false;
let lastOperation = "";
let lastDiff = null;
let lastShift = null;
let lastAge = null;
let lastWork = null;
let lastWorkShift = null;
let lastConvert = null;
let lastFacts = null;
let currentTab = "diff";
let totalsExpanded = false;
let lastPrimary = [];
let diffTimer = 0;

let zones = [];
const zonesById = new Map();
const zonesByCity = new Map();
let historyEntries = [];
let favorites = [];

const tabNames = ["diff", "shift", "age", "work", "convert", "facts"];

function api() {
  return window.go.main.App;
}

function pad(n) {
  return String(n).padStart(2, "0");
}

function toInputValue(date) {
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
  );
}

// Suma días de calendario sin pasar por la zona local del navegador: así el
// ejemplo inicial de 30 días no cambia de hora si el sistema está en DST.
function shiftInputDays(value, days) {
  const [datePart, timePart = "00:00:00"] = value.split("T");
  const [year, month, day] = datePart.split("-").map(Number);
  const [hour, minute, second = 0] = timePart.split(":").map(Number);
  const date = new Date(Date.UTC(year, month - 1, day + days, hour, minute, second));
  return (
    `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}` +
    `T${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}:${pad(date.getUTCSeconds())}`
  );
}

function cleanError(err) {
  if (typeof err === "string") return err;
  if (err && typeof err.message === "string") return err.message;
  try {
    return JSON.stringify(err);
  } catch {
    return String(err);
  }
}

function showError(id, message) {
  const el = $(id);
  if (!el) return;
  el.textContent = message;
  el.hidden = false;
}

function clearError(id) {
  const el = $(id);
  if (el) el.hidden = true;
}

function reveal(el) {
  el.hidden = false;
  el.classList.remove("reveal");
  void el.offsetWidth;
  el.classList.add("reveal");
}

// Lleva el resultado a la vista solo cuando el usuario acaba de pedirlo y la
// tarjeta no está ya visible (por ejemplo, a 720×600).
function revealResult(el) {
  const rect = el.getBoundingClientRect();
  if (rect.top >= 0 && rect.top < window.innerHeight - 80) return;
  const reduce = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  el.scrollIntoView({ block: "start", behavior: reduce ? "auto" : "smooth" });
}

function capitalize(text) {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

function splitHuman(human) {
  const match = human.match(/^(.*?) \(([^)]+)\)$/);
  if (!match) {
    return [human, ""];
  }
  return [match[1], capitalize(match[2])];
}

function setCopyStatus(message, delay = 2400) {
  const el = $("copy-status");
  el.textContent = message;
  clearTimeout(el._timer);
  el._timer = setTimeout(() => {
    el.textContent = "";
  }, delay);
}

function announce(id, text) {
  const el = $(id);
  if (el) el.textContent = text;
}

// --------------------------------------------------------------- persistencia

function readJSON(key, fallback) {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return fallback;
    const value = JSON.parse(raw);
    return value ?? fallback;
  } catch {
    return fallback;
  }
}

function writeJSON(key, value) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Sin almacenamiento disponible: la app sigue funcionando en memoria.
  }
}

function collectState() {
  return {
    zone,
    zoneMode,
    sound: soundEnabled,
    tab: currentTab,
    inclusive: $("inclusive").checked,
    date1: $("date1").value,
    date2: $("date2").value,
    shiftDate: $("shift-date").value,
    shiftDuration: $("shift-duration").value,
    ageBirth: $("age-birth").value,
    ageAt: $("age-at").value,
    workStart: $("work-start").value,
    workEnd: $("work-end").value,
    workBase: $("work-base").value,
    workDays: $("work-days").value,
    workCountry: $("work-country").value,
    workHolidays: $("work-holidays").value,
    workInclusive: $("work-inclusive").checked,
    convertDate: $("convert-date").value,
    convertFrom: $("convert-from").value,
    convertTo: $("convert-to").value,
    factsDate: $("facts-date").value,
  };
}

function saveState() {
  if (!ready) return;
  writeJSON(STORAGE_KEY, collectState());
}

function restoreState() {
  const s = readJSON(STORAGE_KEY, {});
  if (typeof s.zone === "string" && (s.zone === "local" || s.zone === "UTC" || zonesById.has(s.zone))) {
    zone = s.zone;
  }
  if (s.zoneMode === "instant" || s.zoneMode === "wall") zoneMode = s.zoneMode;
  if (typeof s.sound === "boolean") soundEnabled = s.sound;
  const values = {
    date1: s.date1,
    date2: s.date2,
    "shift-date": s.shiftDate,
    "shift-duration": s.shiftDuration,
    "age-birth": s.ageBirth,
    "age-at": s.ageAt,
    "work-start": s.workStart,
    "work-end": s.workEnd,
    "work-base": s.workBase,
    "work-days": s.workDays,
    "work-country": s.workCountry,
    "work-holidays": s.workHolidays,
    "convert-date": s.convertDate,
    "convert-from": s.convertFrom,
    "convert-to": s.convertTo,
    "facts-date": s.factsDate,
  };
  for (const [id, value] of Object.entries(values)) {
    if (typeof value === "string" && $(id)) $(id).value = value;
  }
  $("inclusive").checked = Boolean(s.inclusive);
  $("work-inclusive").checked = Boolean(s.workInclusive);
  for (const name of tabNames) {
    if (s.tab === name) currentTab = name;
  }
}

// ------------------------------------------------------------------ historial

function loadLists() {
  historyEntries = readJSON(HISTORY_KEY, []);
  favorites = readJSON(FAVORITES_KEY, []);
  if (!Array.isArray(historyEntries)) historyEntries = [];
  if (!Array.isArray(favorites)) favorites = [];
}

function entryLabel(entry) {
  return entry.label || "Operación";
}

function entryMeta(entry) {
  const parts = [];
  if (entry.zone) parts.push(entry.zone === "local" ? "hora local" : entry.zone);
  if (entry.at) {
    try {
      parts.push(new Date(entry.at).toLocaleString("es-ES", { dateStyle: "short", timeStyle: "short" }));
    } catch {
      // Sin fecha legible.
    }
  }
  return parts.join(" · ");
}

function pushHistory(entry) {
  const copy = { ...entry, at: Date.now() };
  const key = `${copy.kind}|${copy.label}|${JSON.stringify(copy.data || {})}`;
  historyEntries = historyEntries.filter((item) => `${item.kind}|${item.label}|${JSON.stringify(item.data || {})}` !== key);
  historyEntries.unshift(copy);
  historyEntries = historyEntries.slice(0, 12);
  writeJSON(HISTORY_KEY, historyEntries);
  renderHistory();
}

function renderHistory() {
  renderList($("history-list"), historyEntries, false, "Todavía no hay operaciones.");
  renderList($("favorites-list"), favorites, true, "Guarda una diferencia con el botón «Guardar favorito».");
}

function renderList(container, list, isFavorite, emptyText) {
  container.replaceChildren();
  if (!list.length) {
    const empty = document.createElement("p");
    empty.className = "history-empty";
    empty.textContent = emptyText;
    container.append(empty);
    return;
  }
  list.forEach((entry, index) => {
    const item = document.createElement("div");
    item.className = "history-item";

    const text = document.createElement("div");
    text.className = "history-item-text";
    const title = document.createElement("strong");
    title.textContent = isFavorite ? entry.name || entryLabel(entry) : entryLabel(entry);
    const meta = document.createElement("span");
    meta.textContent = entryMeta(entry);
    text.append(title, meta);

    const actions = document.createElement("div");
    actions.className = "history-item-actions";
    const use = document.createElement("button");
    use.type = "button";
    use.className = "copy";
    use.textContent = "Usar";
    use.dataset.action = "use";
    use.dataset.index = String(index);
    use.dataset.favorite = isFavorite ? "1" : "0";
    actions.append(use);

    if (isFavorite) {
      const remove = document.createElement("button");
      remove.type = "button";
      remove.className = "copy";
      remove.textContent = "Quitar";
      remove.dataset.action = "remove";
      remove.dataset.index = String(index);
      actions.append(remove);
    }

    item.append(text, actions);
    container.append(item);
  });
}

function useEntry(entry) {
  $("history-dialog").close();
  switch (entry.kind) {
    case "diff":
      $("date1").value = entry.data.date1 || "";
      $("date2").value = entry.data.date2 || "";
      $("inclusive").checked = Boolean(entry.data.inclusive);
      if (entry.zone) applyZoneSelection(entry.zone, false);
      activateTab("diff", false, false);
      updateZone().then(() => calcDiff(false));
      break;
    case "shift":
      $("shift-date").value = entry.data.date || "";
      $("shift-duration").value = entry.data.duration || "";
      if (entry.zone) applyZoneSelection(entry.zone, false);
      activateTab("shift", false, false);
      updateZone().then(() => doShift(entry.data.operation || "add", false));
      break;
    case "age":
      $("age-birth").value = entry.data.birth || "";
      $("age-at").value = entry.data.at || $("age-at").value;
      if (entry.zone) applyZoneSelection(entry.zone, false);
      activateTab("age", false, false);
      updateZone().then(() => calcAge(false));
      break;
    case "work":
      $("work-start").value = entry.data.start || "";
      $("work-end").value = entry.data.end || "";
      $("work-country").value = entry.data.country || "";
      $("work-holidays").value = entry.data.holidays || "";
      $("work-inclusive").checked = Boolean(entry.data.inclusive);
      if (entry.zone) applyZoneSelection(entry.zone, false);
      activateTab("work", false, false);
      updateZone().then(() => calcWork(false));
      break;
    case "workadd":
      $("work-base").value = entry.data.base || "";
      $("work-days").value = entry.data.days || "10";
      $("work-country").value = entry.data.country || "";
      $("work-holidays").value = entry.data.holidays || "";
      if (entry.zone) applyZoneSelection(entry.zone, false);
      activateTab("work", false, false);
      updateZone().then(() => doWorkShift(entry.data.operation || "add", false));
      break;
    case "convert":
      $("convert-date").value = entry.data.date || "";
      $("convert-from").value = entry.data.from || "local";
      $("convert-to").value = entry.data.to || "UTC";
      activateTab("convert", false, false);
      runConvert(false);
      break;
    case "facts":
      $("facts-date").value = entry.data.date || "";
      if (entry.zone) applyZoneSelection(entry.zone, false);
      activateTab("facts", false, false);
      updateZone().then(() => runFacts(false));
      break;
    default:
      break;
  }
  saveState();
}

function wireHistory() {
  $("history-open").addEventListener("click", () => {
    renderHistory();
    $("history-dialog").showModal();
  });

  $("history-dialog").addEventListener("click", (event) => {
    const button = event.target.closest("button[data-action]");
    if (!button) return;
    const index = Number(button.dataset.index);
    const isFavorite = button.dataset.favorite === "1";
    const list = isFavorite ? favorites : historyEntries;
    const entry = list[index];
    if (!entry) return;
    if (button.dataset.action === "use") {
      useEntry(entry);
    } else if (button.dataset.action === "remove") {
      favorites.splice(index, 1);
      writeJSON(FAVORITES_KEY, favorites);
      renderHistory();
    }
  });
}

function saveFavorite() {
  if (!lastDiff) {
    setCopyStatus("Calcula una diferencia antes de guardar un favorito.");
    return;
  }
  const suggested = `Del ${lastDiff.human1.split(" (")[0]} al ${lastDiff.human2.split(" (")[0]}`;
  const name = window.prompt("Nombre del favorito:", suggested);
  if (name === null) return;
  const clean = name.trim() || suggested;
  favorites.unshift({
    name: clean,
    kind: "diff",
    label: suggested,
    zone,
    data: {
      date1: $("date1").value,
      date2: $("date2").value,
      inclusive: $("inclusive").checked,
    },
    at: Date.now(),
  });
  favorites = favorites.slice(0, 30);
  writeJSON(FAVORITES_KEY, favorites);
  renderHistory();
  setCopyStatus(`Favorito «${clean}» guardado.`);
}

// --------------------------------------------------------------------- zonas

function normalizeText(text) {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .trim()
    .toLowerCase();
}

function zoneDisplay(id) {
  if (id === "local") return "Hora local";
  if (id === "UTC") return "UTC";
  const z = zonesById.get(id);
  return z ? `${z.city} — ${z.id}` : id;
}

function resolveZoneText(text) {
  const raw = text.trim();
  if (!raw) return null;
  if (zonesById.has(raw) || raw === "local" || raw === "UTC") return raw;
  const afterDash = raw.includes("—") ? raw.split("—").pop().trim() : raw;
  if (zonesById.has(afterDash) || afterDash === "local" || afterDash === "UTC") return afterDash;
  const norm = normalizeText(afterDash);
  const byCity = zonesByCity.get(norm);
  if (byCity) return byCity;
  for (const z of zones) {
    if (normalizeText(z.city) === norm || normalizeText(z.id) === norm) return z.id;
  }
  for (const z of zones) {
    if (normalizeText(z.city).startsWith(norm) || normalizeText(z.id).startsWith(norm)) return z.id;
  }
  for (const z of zones) {
    if (normalizeText(z.id).includes(norm) || normalizeText(z.city).includes(norm)) return z.id;
  }
  return null;
}

function showZoneError() {
  const input = $("zone-select");
  input.setAttribute("aria-invalid", "true");
  $("zone-chip").textContent = "Zona no válida";
  $("zone-chip").classList.add("is-invalid");
  const error = $("zone-error");
  error.textContent = "Zona no válida. Prueba con «Madrid», «Europe/Madrid», «Bogotá» o «UTC».";
  error.hidden = false;
}

function clearZoneError() {
  $("zone-select").removeAttribute("aria-invalid");
  const error = $("zone-error");
  error.hidden = true;
  error.textContent = "";
  $("zone-chip").classList.remove("is-invalid");
}

async function loadZones() {
  zones = await api().Zones();
  zonesById.clear();
  zonesByCity.clear();
  for (const item of zones) {
    zonesById.set(item.id, item);
    zonesByCity.set(normalizeText(item.city), item.id);
  }

  const list = $("zone-list");
  list.replaceChildren();
  for (const item of zones) {
    const option = document.createElement("option");
    option.value = zoneDisplay(item.id);
    option.label = `${item.group} · ${item.id}`;
    list.append(option);
  }

  fillZoneSelect($("convert-from"));
  fillZoneSelect($("convert-to"));
}

function fillZoneSelect(select) {
  select.replaceChildren();
  const groups = new Map();
  for (const item of zones) {
    let group = groups.get(item.group);
    if (!group) {
      group = document.createElement("optgroup");
      group.label = item.group;
      groups.set(item.group, group);
      select.append(group);
    }
    const option = document.createElement("option");
    option.value = item.id;
    option.textContent = item.id === "local" ? "Hora local" : item.city;
    option.title = `${item.id}${item.offset ? " · " + item.offset : ""}`;
    group.append(option);
  }
}

async function updateZone() {
  const state = await api().Zone(zone);
  zoneText = state.id === "UTC" ? `UTC · ${state.offset}` : `${state.city} · ${state.text}`;
  $("zone-select").value = zoneDisplay(zone);
  $("zone-chip").textContent = state.text;
  $("zone-chip").title =
    state.id === "local"
      ? `Hora local · ${state.text}`
      : `${state.id} · ${state.text}${state.dst ? " · horario de verano" : ""}`;
  $("zone-chip").classList.toggle("is-dst", state.dst);
  clearZoneError();
  $("footer-zone").textContent = zoneText;
  $("result-zone").textContent = hasDiff ? state.text : "";
  const note = $("panel-zone");
  if (note) note.textContent = state.text;
  if (!$("convert-from").value) {
    $("convert-from").value = zone;
  }
}

function applyZoneSelection(id, convertInputs) {
  zone = id;
  $("zone-select").value = zoneDisplay(id);
}

async function convertInputsToZone(oldZone, newZone) {
  const ids = ["date1", "date2", "shift-date", "work-start", "work-end", "work-base", "facts-date"];
  for (const id of ids) {
    const input = $(id);
    if (!input || !input.value) continue;
    try {
      const converted = await api().ConvertInput(input.value, oldZone, newZone);
      if (converted) input.value = converted;
    } catch {
      // Si una fecha no se puede convertir, se deja como estaba.
    }
  }
}

async function changeZone(id) {
  if (!id || id === zone) {
    $("zone-select").value = zoneDisplay(zone);
    return;
  }
  const old = zone;
  if (zoneMode === "instant") {
    await convertInputsToZone(old, id);
  }
  zone = id;
  try {
    await updateZone();
  } catch (err) {
    zone = old;
    $("zone-select").value = zoneDisplay(old);
    showZoneError();
    showError(currentTab === "shift" ? "shift-error" : "diff-error", cleanError(err));
    return;
  }
  saveState();
  if (hasDiff) calcDiff(false);
  if (hasShift && lastOperation) doShift(lastOperation, false);
  if (currentTab === "age" && $("age-birth").value) calcAge(false);
  if (currentTab === "work" && $("work-start").value && $("work-end").value) calcWork(false);
  if (currentTab === "facts" && $("facts-date").value) runFacts(false);
}

// ---------------------------------------------------------------------- diff

function hideDiffResult() {
  hasDiff = false;
  lastDiff = null;
  $("diff-result").hidden = true;
  $("totals-panel").hidden = true;
  $("diff-announce").textContent = "";
}

function buildBreakdown(breakdown) {
  const entries = [
    [breakdown.years, "año", "años"],
    [breakdown.months, "mes", "meses"],
    [breakdown.days, "día", "días"],
    [breakdown.hours, "hora", "horas"],
    [breakdown.minutes, "minuto", "minutos"],
    [breakdown.seconds, "segundo", "segundos"],
  ].filter(([value]) => value > 0);

  if (entries.length === 0) {
    entries.push([0, "segundo", "segundos"]);
  }

  const wrap = $("breakdown");
  wrap.replaceChildren();

  for (const [value, one, many] of entries) {
    const chip = document.createElement("div");
    chip.className = "chip";

    const strong = document.createElement("strong");
    strong.textContent = String(value);

    const label = document.createElement("span");
    label.textContent = value === 1 ? one : many;

    chip.append(strong, label);
    wrap.append(chip);
  }
}

const primaryUnits = (days24) => {
  if (days24 >= 730) return ["Años", "Meses", "Semanas", "Días de 24 h"];
  if (days24 >= 61) return ["Meses", "Semanas", "Días de 24 h", "Horas"];
  if (days24 >= 14) return ["Semanas", "Días de 24 h", "Horas", "Minutos"];
  if (days24 >= 2) return ["Días de 24 h", "Horas", "Minutos", "Segundos"];
  return ["Horas", "Minutos", "Segundos", "Milisegundos"];
};

function renderTotals(result) {
  const rows = [
    ["Años", result.yearsTotal, false, "365,2425 días por año"],
    ["Meses", result.monthsTotal, false, "30,436875 días por mes"],
    ["Semanas", result.weeks, false, "Siete días por semana"],
    ["Días de 24 h", result.days24, false, "Tiempo real transcurrido"],
    ["Horas", result.hours, false, ""],
    ["Minutos", result.minutes, false, ""],
    ["Segundos", result.seconds, false, ""],
    ["Milisegundos", result.milliseconds, true, ""],
  ];

  const wrap = $("totals");
  wrap.replaceChildren();

  lastPrimary = primaryUnits(Number(result.days24));

  for (const [label, value, integer, hint] of rows) {
    const row = document.createElement("div");
    row.className = "total";
    row.dataset.label = label;

    const labelEl = document.createElement("div");
    labelEl.className = "total-label";
    labelEl.textContent = label;
    if (hint) labelEl.title = hint;

    const valueEl = document.createElement("div");
    valueEl.className = "total-value";
    const n = Number(value);
    valueEl.textContent = Number.isFinite(n) ? (integer ? nf0 : nf).format(n) : "—";

    row.append(labelEl, valueEl);
    wrap.append(row);
  }
  applyTotalsVisibility();
}

function applyTotalsVisibility() {
  const wrap = $("totals");
  const toggle = $("totals-toggle");
  let anyHidden = false;
  for (const row of wrap.children) {
    const visible = totalsExpanded || lastPrimary.includes(row.dataset.label);
    row.hidden = !visible;
    if (!visible) anyHidden = true;
  }
  toggle.hidden = !anyHidden;
  toggle.textContent = totalsExpanded ? "Ver solo las principales" : "Ver todas las unidades";
  toggle.setAttribute("aria-expanded", String(totalsExpanded));
}

function renderWarning(id, message) {
  const el = $(id);
  if (!el) return;
  el.textContent = message || "";
  el.hidden = !message;
  el.classList.toggle("is-dst", /horario de (verano|invierno)/.test(message || ""));
}

function renderDiff(result) {
  lastDiff = result;
  const daysText = nf.format(result.naturalDays);
  $("natural-days").textContent = daysText;
  $("natural-days").classList.toggle("is-long", daysText.replace(/\D/g, "").length > 6);
  $("natural-unit").textContent = result.naturalDays === 1 ? "día" : "días";

  const [date1, meta1] = splitHuman(result.human1);
  const [date2, meta2] = splitHuman(result.human2);
  $("span-date1").textContent = date1;
  $("span-meta1").textContent = meta1;
  $("span-date2").textContent = date2;
  $("span-meta2").textContent = meta2;

  let relation = "Las dos fechas son iguales";
  if (result.later === "date2") {
    relation = "La fecha 2 es posterior a la fecha 1";
  } else if (result.later === "date1") {
    relation = "La fecha 1 es posterior a la fecha 2";
  }
  $("relation-text").textContent = relation;

  buildBreakdown(result.breakdown);
  renderTotals(result);
  renderWarning("diff-warning", result.warning);
  $("result-zone").textContent = zoneText ? `Zona: ${$("zone-chip").textContent}` : "";
  $("diff-announce").textContent =
    `${result.naturalDays} ${result.naturalDays === 1 ? "día natural" : "días naturales"}. ` +
    `${relation}.`;

  hasDiff = true;
  reveal($("diff-result"));
  reveal($("totals-panel"));
}

function buildDiffText(result) {
  const parts = [];
  parts.push(
    `${nf.format(result.naturalDays)} ${result.naturalDays === 1 ? "día natural" : "días naturales"}.`
  );
  if (result.later === "date2") {
    parts.push("La fecha 2 es posterior a la fecha 1.");
  } else if (result.later === "date1") {
    parts.push("La fecha 1 es posterior a la fecha 2.");
  } else {
    parts.push("Las dos fechas son iguales.");
  }
  parts.push(`Desde ${result.human1}.`);
  parts.push(`Hasta ${result.human2}.`);
  const breakdown = [];
  for (const [key, one, many] of [
    ["years", "año", "años"],
    ["months", "mes", "meses"],
    ["days", "día", "días"],
    ["hours", "hora", "horas"],
    ["minutes", "minuto", "minutos"],
    ["seconds", "segundo", "segundos"],
  ]) {
    const value = result.breakdown[key];
    if (value > 0) {
      breakdown.push(`${value} ${value === 1 ? one : many}`);
    }
  }
  if (breakdown.length) {
    parts.push(`Desglose: ${breakdown.join(", ")}.`);
  }
  if (zoneText) {
    parts.push(`Zona: ${zoneText}.`);
  }
  return parts.join("\n");
}

async function copyText(text) {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      // WebView2 puede denegar el portapapeles; probamos el método clásico.
    }
  }
  return fallbackCopy(text);
}

function fallbackCopy(text) {
  return new Promise((resolve, reject) => {
    const area = document.createElement("textarea");
    area.value = text;
    area.setAttribute("readonly", "");
    area.style.position = "fixed";
    area.style.top = "-1000px";
    area.style.opacity = "0";
    document.body.append(area);
    area.select();
    try {
      if (document.execCommand("copy")) {
        resolve();
      } else {
        reject(new Error("No se pudo copiar"));
      }
    } catch (err) {
      reject(err);
    } finally {
      area.remove();
    }
  });
}

async function copyWithFeedback(button, text, okText = "Copiado") {
  const label = button.textContent;
  try {
    await copyText(text);
    button.textContent = okText;
    button.classList.add("is-copied");
    setCopyStatus("Resultado copiado al portapapeles.");
    chiquito("camarero");
  } catch {
    button.textContent = "No se pudo copiar";
    setCopyStatus("No se pudo copiar el resultado.");
  } finally {
    setTimeout(() => {
      button.textContent = label;
      button.classList.remove("is-copied");
    }, 1800);
  }
}

async function saveICS(ics, button) {
  const label = button.textContent;
  try {
    const path = await api().SaveICS(ics.content, ics.filename);
    if (path) {
      button.textContent = "Guardado";
      setCopyStatus(`Evento guardado en ${path}`);
      chiquito("camarero");
    } else {
      setCopyStatus("Guardado cancelado.");
    }
  } catch (err) {
    button.textContent = "No se pudo guardar";
    setCopyStatus(cleanError(err));
  } finally {
    setTimeout(() => {
      button.textContent = label;
    }, 2000);
  }
}

async function exportDiff() {
  if (!lastDiff) return;
  const allDay = /T00:00(:00)?$/.test($("date2").value);
  try {
    const ics = await api().EventICS(
      lastDiff.iso2,
      zone,
      `Fecha: ${lastDiff.human2.split(" (")[0]}`,
      buildDiffText(lastDiff),
      allDay
    );
    await saveICS(ics, $("export-diff"));
  } catch (err) {
    showError("diff-error", cleanError(err));
  }
}

async function calcDiff(record = false) {
  if (!ready) return;

  clearError("diff-error");
  const date1 = $("date1").value;
  const date2 = $("date2").value;

  if (!date1 || !date2) {
    hideDiffResult();
    showError("diff-error", "Introduce las dos fechas para calcular la diferencia.");
    return;
  }

  const button = $("calc");
  button.disabled = true;
  button.textContent = "Calculando…";

  try {
    const result = await api().Diff(date1, date2, zone, $("inclusive").checked);
    renderDiff(result);
    if (record) {
      chiquito("grito");
      revealResult($("diff-result"));
      pushHistory({
        kind: "diff",
        label: `${nf.format(result.naturalDays)} días · ${result.human1.split(" (")[0]} → ${result.human2.split(" (")[0]}`,
        zone,
        data: { date1, date2, inclusive: $("inclusive").checked },
      });
    }
  } catch (err) {
    showError("diff-error", cleanError(err));
  } finally {
    button.disabled = false;
    button.textContent = "Calcular diferencia";
    saveState();
  }
}

function scheduleDiff(delay = 300) {
  clearTimeout(diffTimer);
  diffTimer = setTimeout(() => {
    if (!ready) return;
    if ($("date1").value && $("date2").value) {
      calcDiff(false);
    } else {
      hideDiffResult();
      clearError("diff-error");
    }
  }, delay);
}

async function setNow(input, errorId) {
  try {
    input.value = await api().Now(zone);
    clearError(errorId);
    if (input === $("date1") || input === $("date2")) scheduleDiff(0);
    if (input === $("facts-date")) runFacts(false);
    saveState();
  } catch (err) {
    showError(errorId, cleanError(err));
  }
}

// ---------------------------------------------------------------- sumar/restar

function hideShiftResult() {
  hasShift = false;
  lastShift = null;
  $("shift-result").hidden = true;
  $("shift-empty").hidden = false;
  $("shift-announce").textContent = "";
}

function renderShift(result) {
  lastShift = result;
  $("shift-op").textContent = `${capitalize(result.operation)} ${result.duration}`;
  $("shift-value").textContent = result.resultHuman;
  $("shift-sub").textContent = `Desde ${result.startHuman} · ${result.iso}`;
  renderWarning("shift-warning", result.warning);
  $("shift-announce").textContent = `${capitalize(result.operation)} ${result.duration}: ${result.resultHuman}.`;

  hasShift = true;
  $("shift-empty").hidden = true;
  reveal($("shift-result"));
}

function buildShiftText(result) {
  const parts = [
    `${capitalize(result.operation)} ${result.duration}: ${result.resultHuman}.`,
    `Desde ${result.startHuman}.`,
    `ISO 8601: ${result.iso}.`,
  ];
  if (zoneText) parts.push(`Zona: ${zoneText}.`);
  return parts.join("\n");
}

async function doShift(operation, record = false) {
  if (!ready) return;

  clearError("shift-error");
  const date = $("shift-date").value;
  const duration = $("shift-duration").value.trim();

  if (!date || !duration) {
    hideShiftResult();
    showError("shift-error", "Introduce una fecha base y una duración.");
    return;
  }

  const button = $(operation === "sub" ? "sub" : "add");
  const other = $(operation === "sub" ? "add" : "sub");
  const label = button.textContent;

  button.disabled = true;
  other.disabled = true;
  button.textContent = operation === "sub" ? "Restando…" : "Sumando…";

  try {
    const result = await api().Shift(date, duration, operation, zone);
    lastOperation = operation;
    renderShift(result);
    if (record) {
      chiquito("ataque");
      revealResult($("shift-result"));
      pushHistory({
        kind: "shift",
        label: `${capitalize(result.operation)} ${result.duration} · ${result.resultHuman.split(" (")[0]}`,
        zone,
        data: { date, duration, operation },
      });
    }
  } catch (err) {
    showError("shift-error", cleanError(err));
  } finally {
    button.disabled = false;
    other.disabled = false;
    button.textContent = label;
    saveState();
  }
}

async function exportShift() {
  if (!lastShift) return;
  const allDay = /T00:00(:00)?$/.test($("shift-date").value);
  try {
    const ics = await api().EventICS(
      lastShift.iso,
      zone,
      lastShift.resultHuman.split(" (")[0],
      buildShiftText(lastShift),
      allDay
    );
    await saveICS(ics, $("export-shift"));
  } catch (err) {
    showError("shift-error", cleanError(err));
  }
}

// ------------------------------------------------------------------------ edad

function lifeDaysBreakdown(totalDays) {
  const days = Number(totalDays);
  const weeks = Math.floor(days / 7);
  const rest = days % 7;
  return `${nf0.format(weeks)} semanas y ${rest} ${rest === 1 ? "día" : "días"}`;
}

function renderAge(result) {
  lastAge = result;
  const totalDays = Number(result.totalDays);
  const weeks = Math.floor(totalDays / 7);
  const rest = totalDays % 7;
  const until = $("age-at").value;
  $("age-big").textContent = `${nf0.format(totalDays)} días`;
  $("age-reference").textContent = until ? `hasta ${formatDateShort(until)}` : "hasta hoy";
  const grid = $("age-grid");
  grid.replaceChildren();
  const entries = [
    { label: "Edad exacta", value: result.human },
    { label: "Semanas vividas", value: `${nf0.format(weeks)} semanas`, sub: `${rest} ${rest === 1 ? "día" : "días"} más` },
    {
      label: "Próximo cumpleaños",
      value: result.nextBirthday,
      sub: `${capitalize(result.nextWeekday)} · cumplirá ${result.turning}`,
    },
    {
      label: result.isBirthday ? "Hoy" : "Faltan",
      value: result.isBirthday ? "¡Felicidades!" : `${nf0.format(result.daysToNext)} días`,
    },
  ];
  for (const entry of entries) {
    const cell = document.createElement("div");
    cell.className = "feature-cell";
    const l = document.createElement("span");
    l.className = "feature-cell-label";
    l.textContent = entry.label;
    const v = document.createElement("strong");
    v.className = "feature-cell-value";
    v.textContent = entry.value;
    cell.append(l, v);
    if (entry.sub) {
      const sub = document.createElement("span");
      sub.className = "feature-cell-sub";
      sub.textContent = entry.sub;
      cell.append(sub);
    }
    grid.append(cell);
  }
  renderWarning("age-warning", result.warning);
  announce(
    "age-announce",
    `${nf0.format(totalDays)} días de vida. ` +
      `${result.years} años, ${result.months} meses y ${result.days} días. ` +
      (result.isBirthday
        ? `Hoy cumple ${result.turning} años.`
        : `Próximo cumpleaños: ${result.nextBirthday}, dentro de ${result.daysToNext} días.`)
  );
  $("age-empty").hidden = true;
  reveal($("age-result"));
}

function formatDateShort(value) {
  const [year, month, day] = value.split("-");
  if (!year || !month || !day) return value;
  return `${Number(day)}/${Number(month)}/${year}`;
}

async function calcAge(record = false) {
  if (!ready) return;
  clearError("age-error");
  const birth = $("age-birth").value;
  if (!birth) {
    $("age-result").hidden = true;
    $("age-empty").hidden = false;
    showError("age-error", "Introduce la fecha de nacimiento.");
    return;
  }
  const at = $("age-at").value;
  if (!at) {
    showError("age-error", "Elige hasta qué fecha contar los días de vida o pulsa «Hoy».");
    return;
  }

  const button = $("age-calc");
  button.disabled = true;
  button.textContent = "Calculando…";
  try {
    const result = await api().Age(birth, at, zone);
    renderAge(result);
    if (record) {
      chiquito("fuego");
      revealResult($("age-result"));
      pushHistory({
        kind: "age",
        label: `${nf0.format(result.totalDays)} días de vida · ${result.human}`,
        zone,
        data: { birth, at },
      });
    }
  } catch (err) {
    showError("age-error", cleanError(err));
  } finally {
    button.disabled = false;
    button.textContent = "Calcular días de vida";
    saveState();
  }
}

async function setAgeToday() {
  try {
    const now = await api().Now(zone);
    $("age-at").value = now.slice(0, 10);
    clearError("age-error");
    if ($("age-birth").value) calcAge(false);
    saveState();
  } catch (err) {
    showError("age-error", cleanError(err));
  }
}

function buildAgeText(result) {
  const parts = [`Días de vida: ${nf0.format(result.totalDays)} (${lifeDaysBreakdown(result.totalDays)}).`];
  parts.push(`Edad exacta: ${result.human}.`);
  parts.push(
    result.isBirthday
      ? `Hoy cumple ${result.turning} años.`
      : `Próximo cumpleaños: ${result.nextBirthday} (${result.nextWeekday}), dentro de ${result.daysToNext} días; cumplirá ${result.turning}.`
  );
  if (result.warning) parts.push(result.warning);
  if (zoneText) parts.push(`Zona: ${zoneText}.`);
  return parts.join("\n");
}

// ----------------------------------------------------------------- laborables

function holidayManual() {
  return $("work-holidays")
    .value.split(/[,;\n]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function countryName(id) {
  const option = $("work-country").querySelector(`option[value="${id}"]`);
  return option ? option.textContent : id;
}

async function loadCountries() {
  const countries = await api().Countries();
  const select = $("work-country");
  select.replaceChildren();
  for (const country of countries) {
    const option = document.createElement("option");
    option.value = country.id;
    option.textContent = country.name;
    option.title = country.note || "";
    select.append(option);
  }
}

function renderHolidays(list) {
  const wrap = $("work-holiday-list");
  wrap.replaceChildren();
  if (!list || !list.length) {
    const empty = document.createElement("p");
    empty.className = "history-empty";
    empty.textContent = "Sin festivos en el rango.";
    wrap.append(empty);
    return;
  }
  for (const holiday of list) {
    const row = document.createElement("div");
    row.className = "holiday-item";
    const date = document.createElement("span");
    date.textContent = holiday.date;
    const name = document.createElement("span");
    name.textContent = holiday.name;
    row.append(date, name);
    wrap.append(row);
  }
}

function renderWork(result) {
  lastWork = result;
  $("work-big").textContent = `${nf0.format(result.workdays)} ${result.workdays === 1 ? "día laborable" : "días laborables"}`;
  $("work-range").textContent = `${result.startHuman.split(" (")[0]} → ${result.endHuman.split(" (")[0]}`;
  const grid = $("work-grid");
  grid.replaceChildren();
  const entries = [
    ["Días naturales", nf0.format(result.naturalDays)],
    ["País", result.country ? countryName(result.country) : "Sin festivos"],
    ["Festivos", String(result.holidays.length)],
  ];
  for (const [label, value] of entries) {
    const cell = document.createElement("div");
    cell.className = "feature-cell";
    const l = document.createElement("span");
    l.className = "feature-cell-label";
    l.textContent = label;
    const v = document.createElement("strong");
    v.className = "feature-cell-value";
    v.textContent = value;
    cell.append(l, v);
    grid.append(cell);
  }
  renderHolidays(result.holidays);
  announce("work-announce", `${result.workdays} días laborables entre ${result.startHuman} y ${result.endHuman}.`);
  $("work-empty").hidden = true;
  reveal($("work-result"));
}

async function calcWork(record = false) {
  if (!ready) return;
  clearError("work-error");
  if (!$("work-start").value || !$("work-end").value) {
    $("work-result").hidden = true;
    $("work-empty").hidden = false;
    showError("work-error", "Introduce las dos fechas del rango.");
    return;
  }
  const button = $("work-calc");
  button.disabled = true;
  button.textContent = "Contando…";
  try {
    const result = await api().Workdays(
      $("work-start").value,
      $("work-end").value,
      zone,
      $("work-country").value,
      holidayManual(),
      $("work-inclusive").checked
    );
    renderWork(result);
    if (record) {
      chiquito("ataque");
      revealResult($("work-result"));
      pushHistory({
        kind: "work",
        label: `${result.workdays} laborables · ${result.startHuman.split(" (")[0]} → ${result.endHuman.split(" (")[0]}`,
        zone,
        data: {
          start: $("work-start").value,
          end: $("work-end").value,
          country: $("work-country").value,
          holidays: $("work-holidays").value,
          inclusive: $("work-inclusive").checked,
        },
      });
    }
  } catch (err) {
    showError("work-error", cleanError(err));
  } finally {
    button.disabled = false;
    button.textContent = "Contar laborables";
    saveState();
  }
}

function renderWorkShift(result) {
  lastWorkShift = result;
  $("work-shift-op").textContent = `${capitalize(result.operation)} ${result.workdays} ${result.workdays === 1 ? "día laborable" : "días laborables"}`;
  $("work-shift-value").textContent = result.resultHuman;
  $("work-shift-sub").textContent = `Desde ${result.startHuman} · ${result.iso}`;
  reveal($("work-shift-result"));
}

async function doWorkShift(operation, record = false) {
  if (!ready) return;
  clearError("work-shift-error");
  const base = $("work-base").value;
  const days = Number($("work-days").value);
  if (!base) {
    $("work-shift-result").hidden = true;
    showError("work-shift-error", "Introduce la fecha base.");
    return;
  }
  if (!Number.isFinite(days) || days < 1) {
    showError("work-shift-error", "Introduce un número de días laborables mayor que cero.");
    return;
  }
  const signed = operation === "sub" ? -Math.trunc(days) : Math.trunc(days);
  const button = $(operation === "sub" ? "work-sub" : "work-add");
  const other = $(operation === "sub" ? "work-add" : "work-sub");
  const label = button.textContent;
  button.disabled = true;
  other.disabled = true;
  button.textContent = operation === "sub" ? "Restando…" : "Sumando…";
  try {
    const result = await api().AddWorkdays(
      base,
      signed,
      zone,
      $("work-country").value,
      holidayManual()
    );
    renderWorkShift(result);
    if (record) {
      chiquito("ataque");
      revealResult($("work-shift-result"));
      pushHistory({
        kind: "workadd",
        label: `${capitalize(result.operation)} ${result.workdays} laborables · ${result.resultHuman.split(" (")[0]}`,
        zone,
        data: {
          base,
          days: String(days),
          operation,
          country: $("work-country").value,
          holidays: $("work-holidays").value,
        },
      });
    }
  } catch (err) {
    showError("work-shift-error", cleanError(err));
  } finally {
    button.disabled = false;
    other.disabled = false;
    button.textContent = label;
    saveState();
  }
}

function buildWorkText(result) {
  const parts = [
    `${result.workdays} ${result.workdays === 1 ? "día laborable" : "días laborables"} entre ${result.startHuman} y ${result.endHuman}.`,
    `${result.naturalDays} días naturales.`,
  ];
  if (result.holidays.length) {
    parts.push(`Festivos: ${result.holidays.map((h) => `${h.date} ${h.name}`).join("; ")}.`);
  }
  if (result.country) parts.push(`País: ${countryName(result.country)}.`);
  if (zoneText) parts.push(`Zona: ${zoneText}.`);
  return parts.join("\n");
}

// ---------------------------------------------------------------------- zonas

function renderConvert(result) {
  lastConvert = result;
  $("convert-from-label").textContent = result.fromZone;
  $("convert-from-value").textContent = result.fromHuman;
  $("convert-from-zone").textContent = result.fromISO;
  $("convert-to-label").textContent = result.toZone;
  $("convert-to-value").textContent = result.toHuman;
  $("convert-to-zone").textContent = result.toISO;
  $("convert-diff").textContent = "mismo instante, otra hora de pared";
  renderWarning("convert-warning", result.warning);
  announce("convert-announce", `${result.fromHuman} en ${result.fromZone} son ${result.toHuman} en ${result.toZone}.`);
  $("convert-empty").hidden = true;
  reveal($("convert-result"));
}

async function runConvert(record = false) {
  if (!ready) return;
  clearError("convert-error");
  const date = $("convert-date").value;
  if (!date) {
    $("convert-result").hidden = true;
    $("convert-empty").hidden = false;
    showError("convert-error", "Introduce una fecha y hora.");
    return;
  }
  const button = $("convert-run");
  button.disabled = true;
  button.textContent = "Convirtiendo…";
  try {
    const result = await api().Convert(date, $("convert-from").value, $("convert-to").value);
    renderConvert(result);
    if (record) {
      chiquito("epetekan");
      revealResult($("convert-result"));
      pushHistory({
        kind: "convert",
        label: `${result.fromHuman.split(" (")[0]} · ${$("convert-from").value} → ${$("convert-to").value}`,
        zone,
        data: { date, from: $("convert-from").value, to: $("convert-to").value },
      });
    }
  } catch (err) {
    showError("convert-error", cleanError(err));
  } finally {
    button.disabled = false;
    button.textContent = "Convertir";
    saveState();
  }
}

function buildConvertText(result) {
  return [
    `${result.fromHuman} (${result.fromZone}).`,
    `${result.toHuman} (${result.toZone}).`,
    `ISO 8601: ${result.fromISO} → ${result.toISO}.`,
    result.warning || "",
  ]
    .filter(Boolean)
    .join("\n");
}

// ---------------------------------------------------------------------- ficha

function renderFacts(result) {
  lastFacts = result;
  $("facts-long").textContent = result.long;
  $("facts-human").textContent = result.human;
  const grid = $("facts-grid");
  grid.replaceChildren();
  const entries = [
    ["Día de la semana", capitalize(result.weekday)],
    ["Semana ISO", `${result.isoWeek} de ${result.isoYear}`],
    ["Día del año", `${result.dayOfYear} de ${result.daysInYear}`],
    ["Trimestre", `T${result.quarter}`],
    ["Días del mes", String(result.daysInMonth)],
    ["Quedan del mes", String(result.daysLeftInMonth)],
    ["Quedan del año", String(result.daysLeftInYear)],
    ["Año bisiesto", result.leapYear ? "Sí" : "No"],
  ];
  for (const [label, value] of entries) {
    const cell = document.createElement("div");
    cell.className = "feature-cell";
    const l = document.createElement("span");
    l.className = "feature-cell-label";
    l.textContent = label;
    const v = document.createElement("strong");
    v.className = "feature-cell-value";
    v.textContent = value;
    cell.append(l, v);
    grid.append(cell);
  }
  announce("facts-announce", `${result.long}. Semana ISO ${result.isoWeek} de ${result.isoYear}. Día ${result.dayOfYear} de ${result.daysInYear}.`);
  $("facts-empty").hidden = true;
  reveal($("facts-result"));
}

async function runFacts(record = false) {
  if (!ready) return;
  clearError("facts-error");
  if (!$("facts-date").value) {
    $("facts-result").hidden = true;
    $("facts-empty").hidden = false;
    showError("facts-error", "Introduce una fecha.");
    return;
  }
  const button = $("facts-run");
  button.disabled = true;
  button.textContent = "Consultando…";
  try {
    const result = await api().Facts($("facts-date").value, zone);
    renderFacts(result);
    if (record) {
      chiquito("tecuen");
      revealResult($("facts-result"));
      pushHistory({
        kind: "facts",
        label: result.long,
        zone,
        data: { date: $("facts-date").value },
      });
    }
  } catch (err) {
    showError("facts-error", cleanError(err));
  } finally {
    button.disabled = false;
    button.textContent = "Ver ficha";
    saveState();
  }
}

function buildFactsText(result) {
  return [
    result.long,
    `Día de la semana: ${result.weekday}.`,
    `Semana ISO: ${result.isoWeek} de ${result.isoYear}.`,
    `Día del año: ${result.dayOfYear} de ${result.daysInYear}.`,
    `Trimestre: T${result.quarter}.`,
    `Días del mes: ${result.daysInMonth}. Quedan ${result.daysLeftInMonth}.`,
    `Quedan ${result.daysLeftInYear} días del año.`,
    `Año bisiesto: ${result.leapYear ? "sí" : "no"}.`,
    zoneText ? `Zona: ${zoneText}.` : "",
  ]
    .filter(Boolean)
    .join("\n");
}

// ------------------------------------------------------------------ sonidos

const SOUND_CLIPS = {
  grito: { src: "assets/jarl.mp3", text: "¡JARL!" },
  ataque: { src: "assets/ataque.mp3", text: "¡Al ataque!" },
  fuego: { src: "assets/fuego.mp3", text: "¡Fuegorrrlll!" },
  camarero: { src: "assets/camarero.mp3", text: "¡Camarerooorrrlll!" },
  tecuen: { src: "assets/tecuen.mp3", text: "¿Te das cuen?" },
  epetekan: { src: "assets/epetekan.mp3", text: "¡Epetekan!" },
};

const soundPlayers = new Map();

function soundPlayer(name) {
  if (soundPlayers.has(name)) return soundPlayers.get(name);
  const clip = SOUND_CLIPS[name];
  if (!clip) return null;
  try {
    const audio = new Audio(clip.src);
    audio.preload = "auto";
    audio.volume = 0.9;
    soundPlayers.set(name, audio);
    return audio;
  } catch {
    return null;
  }
}

function chiquito(name) {
  const clip = SOUND_CLIPS[name];
  if (!clip) return;

  const caption = $("catchphrase");
  if (caption) {
    caption.textContent = clip.text;
    caption.classList.remove("is-on");
    void caption.offsetWidth;
    caption.classList.add("is-on");
    clearTimeout(caption._timer);
    caption._timer = setTimeout(() => caption.classList.remove("is-on"), 1250);
  }

  if (!soundEnabled) return;
  const audio = soundPlayer(name);
  if (!audio) return;
  try {
    audio.currentTime = 0;
    audio.play().catch(() => {});
  } catch {
    // Sin audio disponible: el texto en pantalla sigue siendo el aviso.
  }
}

function renderSound() {
  const button = $("sound-toggle");
  button.setAttribute("aria-pressed", String(soundEnabled));
  button.setAttribute(
    "aria-label",
    soundEnabled ? "Sonido activado, pulsa para silenciar" : "Sonido silenciado, pulsa para activar"
  );
  button.title = soundEnabled ? "Silenciar los sonidos de Chiquito" : "Activar los sonidos de Chiquito";
  button.classList.toggle("is-muted", !soundEnabled);
}

// ----------------------------------------------------------------------- tabs

function activateTab(name, focus, auto = true) {
  if (!tabNames.includes(name)) name = "diff";
  currentTab = name;
  tabNames.forEach((item) => {
    const tab = $(`tab-${item}`);
    const panel = $(`panel-${item}`);
    const active = item === name;
    tab.classList.toggle("is-active", active);
    tab.setAttribute("aria-selected", String(active));
    tab.tabIndex = active ? 0 : -1;
    panel.hidden = !active;
    if (active) {
      panel.classList.remove("is-entering");
      void panel.offsetWidth;
      panel.classList.add("is-entering");
    }
  });
  if (focus) $(`tab-${name}`).focus();
  saveState();
  if (!auto || !ready) return;
  if (name === "age" && $("age-birth").value) calcAge(false);
  if (name === "work" && $("work-start").value && $("work-end").value) calcWork(false);
  if (name === "convert" && $("convert-date").value) runConvert(false);
  if (name === "facts" && $("facts-date").value) runFacts(false);
}

function setupTabs() {
  tabNames.forEach((name) => {
    const tab = $(`tab-${name}`);
    tab.addEventListener("click", () => activateTab(name, false, true));
    tab.addEventListener("keydown", (event) => {
      const index = tabNames.indexOf(name);
      let next = null;
      if (event.key === "ArrowRight") {
        next = tabNames[(index + 1) % tabNames.length];
      } else if (event.key === "ArrowLeft") {
        next = tabNames[(index - 1 + tabNames.length) % tabNames.length];
      } else if (event.key === "Home") {
        next = tabNames[0];
      } else if (event.key === "End") {
        next = tabNames[tabNames.length - 1];
      }
      if (next) {
        event.preventDefault();
        activateTab(next, true, true);
      }
    });
  });
}

// --------------------------------------------------------------- arranque/UI

async function setupDefaults() {
  let now = await api().Now(zone);
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/.test(now)) {
    now = toInputValue(new Date());
  }

  if (!$("date2").value) $("date2").value = now;
  if (!$("date1").value) $("date1").value = shiftInputDays($("date2").value, -30);
  if (!$("shift-date").value) $("shift-date").value = $("date2").value;
  if (!$("shift-duration").value) $("shift-duration").value = "1 mes";
  if (!$("work-end").value) $("work-end").value = $("date2").value;
  if (!$("work-start").value) $("work-start").value = shiftInputDays($("work-end").value, -7);
  if (!$("work-base").value) $("work-base").value = $("date2").value;
  if (!$("work-days").value) $("work-days").value = "10";
  if (!$("convert-date").value) $("convert-date").value = $("date2").value;
  if (!$("facts-date").value) $("facts-date").value = $("date2").value;
  if (!$("age-at").value) $("age-at").value = now.slice(0, 10);

  if (!$("convert-from").value) $("convert-from").value = zone;
  if (!$("convert-to").value) {
    $("convert-to").value = zonesById.has("America/Bogota") ? "America/Bogota" : "UTC";
  }
}

function wireEvents() {
  $("calc").addEventListener("click", () => calcDiff(true));
  $("add").addEventListener("click", () => doShift("add", true));
  $("sub").addEventListener("click", () => doShift("sub", true));
  $("copy-result").addEventListener("click", () => lastDiff && copyWithFeedback($("copy-result"), buildDiffText(lastDiff)));
  $("export-diff").addEventListener("click", exportDiff);
  $("favorite-diff").addEventListener("click", saveFavorite);
  $("copy-shift").addEventListener("click", () => lastShift && copyWithFeedback($("copy-shift"), buildShiftText(lastShift)));
  $("export-shift").addEventListener("click", exportShift);
  $("copy-age").addEventListener("click", () => lastAge && copyWithFeedback($("copy-age"), buildAgeText(lastAge)));
  $("copy-work").addEventListener("click", () => lastWork && copyWithFeedback($("copy-work"), buildWorkText(lastWork)));
  $("copy-work-shift").addEventListener("click", () => {
    if (!lastWorkShift) return;
    const text = `${capitalize(lastWorkShift.operation)} ${lastWorkShift.workdays} días laborables: ${lastWorkShift.resultHuman}. Desde ${lastWorkShift.startHuman}. ISO: ${lastWorkShift.iso}.`;
    copyWithFeedback($("copy-work-shift"), text);
  });
  $("copy-convert").addEventListener("click", () => lastConvert && copyWithFeedback($("copy-convert"), buildConvertText(lastConvert)));
  $("copy-facts").addEventListener("click", () => lastFacts && copyWithFeedback($("copy-facts"), buildFactsText(lastFacts)));

  $("totals-toggle").addEventListener("click", () => {
    totalsExpanded = !totalsExpanded;
    applyTotalsVisibility();
  });

  $("swap").addEventListener("click", () => {
    const first = $("date1").value;
    $("date1").value = $("date2").value;
    $("date2").value = first;
    chiquito("grito");
    if (hasDiff) calcDiff(false);
    saveState();
  });

  $("now1").addEventListener("click", () => setNow($("date1"), "diff-error"));
  $("now2").addEventListener("click", () => setNow($("date2"), "diff-error"));

  $("inclusive").addEventListener("change", () => {
    if (hasDiff) calcDiff(false);
    saveState();
  });

  $("zone-mode").addEventListener("change", () => {
    zoneMode = $("zone-mode").value;
    saveState();
  });

  $("sound-toggle").addEventListener("click", () => {
    soundEnabled = !soundEnabled;
    renderSound();
    saveState();
    if (soundEnabled) chiquito("grito");
  });

  $("zone-select").addEventListener("change", async () => {
    const id = resolveZoneText($("zone-select").value);
    if (!id) {
      showZoneError();
      $("zone-select").focus();
      return;
    }
    clearZoneError();
    await changeZone(id);
  });
  $("zone-select").addEventListener("input", () => {
    if (!$("zone-error").hidden) clearZoneError();
  });
  $("zone-select").addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      $("zone-select").blur();
    }
  });

  for (const id of ["date1", "date2"]) {
    $(id).addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        calcDiff(true);
      }
    });
    $(id).addEventListener("input", () => {
      clearError("diff-error");
      scheduleDiff();
    });
    $(id).addEventListener("change", () => scheduleDiff(0));
  }

  for (const id of ["shift-date", "shift-duration"]) {
    $(id).addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        doShift(lastOperation || "add", true);
      }
    });
    $(id).addEventListener("input", () => clearError("shift-error"));
  }

  for (const button of document.querySelectorAll("#shift-examples .example")) {
    button.addEventListener("click", () => {
      $("shift-duration").value = button.dataset.duration;
      clearError("shift-error");
      doShift(lastOperation || "add", true);
    });
  }

  $("age-calc").addEventListener("click", () => calcAge(true));
  $("age-now").addEventListener("click", setAgeToday);
  for (const id of ["age-birth", "age-at"]) {
    $(id).addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        calcAge(true);
      }
    });
  }

  $("work-calc").addEventListener("click", () => calcWork(true));
  $("work-add").addEventListener("click", () => doWorkShift("add", true));
  $("work-sub").addEventListener("click", () => doWorkShift("sub", true));
  $("work-days").addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      doWorkShift("add", true);
    }
  });

  $("convert-run").addEventListener("click", () => runConvert(true));
  $("convert-swap").addEventListener("click", () => {
    const from = $("convert-from").value;
    $("convert-from").value = $("convert-to").value;
    $("convert-to").value = from;
    if ($("convert-date").value) runConvert(false);
    saveState();
  });

  $("facts-run").addEventListener("click", () => runFacts(true));
  $("facts-now").addEventListener("click", () => setNow($("facts-date"), "facts-error"));

  const allIds = [
    "inclusive", "date1", "date2", "shift-date", "shift-duration",
    "age-birth", "age-at", "work-start", "work-end", "work-base",
    "work-days", "work-country", "work-holidays", "work-inclusive",
    "convert-date", "convert-from", "convert-to", "facts-date",
  ];
  for (const id of allIds) {
    const el = $(id);
    if (!el) continue;
    el.addEventListener("input", saveState);
    el.addEventListener("change", saveState);
  }
}

async function waitForBindings() {
  for (let attempt = 0; attempt < 200; attempt++) {
    if (window.go && window.go.main && window.go.main.App) {
      return true;
    }
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  return false;
}

async function updateVersion() {
  try {
    const v = await api().Version();
    $("app-version").textContent = v && v !== "dev" ? `v${v}` : "";
  } catch {
    // La versión es informativa.
  }
}

async function boot() {
  if (!(await waitForBindings())) {
    showError("diff-error", "No se pudo conectar con el backend de la aplicación.");
    return;
  }

  window.scrollTo(0, 0);
  loadLists();
  await loadZones();
  await loadCountries();
  restoreState();
  $("zone-mode").value = zoneMode;
  renderSound();
  await updateZone();
  await setupDefaults();
  ready = true;
  renderHistory();
  activateTab(currentTab, false, false);
  await calcDiff(false);
  await updateVersion();
  if (currentTab !== "diff") activateTab(currentTab, false, true);
  saveState();
  window.scrollTo(0, 0);
  setTimeout(() => window.scrollTo(0, 0), 250);
}

document.addEventListener("DOMContentLoaded", () => {
  setupTabs();
  wireEvents();
  wireHistory();
  boot().catch((err) => showError("diff-error", cleanError(err)));
});
