"use strict";

if ("scrollRestoration" in history) {
  history.scrollRestoration = "manual";
}

const $ = (id) => document.getElementById(id);
const nf = new Intl.NumberFormat("es-ES", { maximumFractionDigits: 2 });
const nf0 = new Intl.NumberFormat("es-ES", { maximumFractionDigits: 0 });

let zone = "local";
let zoneText = "";
let hasDiff = false;
let hasShift = false;
let lastOperation = "";
let ready = false;

function api() {
  return window.go.main.App;
}

function pad(n) {
  return String(n).padStart(2, "0");
}

function toInputValue(date) {
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}`
  );
}

// Suma días de calendario sin pasar por la zona local del navegador: así el
// ejemplo inicial de 30 días no cambia de hora si el sistema está en DST.
function shiftInputDays(value, days) {
  const [datePart, timePart = "00:00"] = value.split("T");
  const [year, month, day] = datePart.split("-").map(Number);
  const [hour, minute] = timePart.split(":").map(Number);
  const date = new Date(Date.UTC(year, month - 1, day + days, hour, minute));
  return (
    `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}` +
    `T${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}`
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
  el.textContent = message;
  el.hidden = false;
}

function clearError(id) {
  $(id).hidden = true;
}

function reveal(el) {
  el.hidden = false;
  el.classList.remove("reveal");
  void el.offsetWidth;
  el.classList.add("reveal");
}

function capitalize(text) {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

function hideDiffResult() {
  hasDiff = false;
  $("diff-result").hidden = true;
  $("totals-panel").hidden = true;
  $("diff-announce").textContent = "";
}

function hideShiftResult() {
  hasShift = false;
  $("shift-result").hidden = true;
  $("shift-announce").textContent = "";
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

  for (const [label, value, integer, hint] of rows) {
    const row = document.createElement("div");
    row.className = "total";

    const labelEl = document.createElement("div");
    labelEl.className = "total-label";
    labelEl.textContent = label;
    if (hint) {
      labelEl.title = hint;
    }

    const valueEl = document.createElement("div");
    valueEl.className = "total-value";
    valueEl.textContent = (integer ? nf0 : nf).format(Number(value));

    row.append(labelEl, valueEl);
    wrap.append(row);
  }
}

function renderWarning(id, message) {
  const el = $(id);
  el.textContent = message || "";
  el.hidden = !message;
}

function renderDiff(result) {
  $("natural-days").textContent = nf.format(result.naturalDays);
  $("natural-unit").textContent = result.naturalDays === 1 ? "día" : "días";
  $("natural-sub").textContent = `${result.human1} → ${result.human2}`;

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
  $("result-zone").textContent = zoneText ? `Zona: ${zoneText}` : "";
  $("diff-announce").textContent =
    `${result.naturalDays} ${result.naturalDays === 1 ? "día natural" : "días naturales"}. ` +
    `${relation}.`;

  hasDiff = true;
  reveal($("diff-result"));
  reveal($("totals-panel"));
}

function renderShift(result) {
  $("shift-op").textContent = `${capitalize(result.operation)} ${result.duration}`;
  $("shift-value").textContent = result.resultHuman;
  $("shift-sub").textContent = `Desde ${result.startHuman} · ${result.iso}`;
  renderWarning("shift-warning", result.warning);
  $("shift-announce").textContent = `${capitalize(result.operation)} ${result.duration}: ${result.resultHuman}.`;

  hasShift = true;
  reveal($("shift-result"));
}

async function calcDiff() {
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
  } catch (err) {
    hideDiffResult();
    showError("diff-error", cleanError(err));
  } finally {
    button.disabled = false;
    button.textContent = "Calcular diferencia";
  }
}

async function doShift(operation) {
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
  } catch (err) {
    hideShiftResult();
    showError("shift-error", cleanError(err));
  } finally {
    button.disabled = false;
    other.disabled = false;
    button.textContent = label;
  }
}

async function setNow(input) {
  try {
    input.value = await api().Now(zone);
    clearError("diff-error");
  } catch (err) {
    showError("diff-error", cleanError(err));
  }
}

async function loadZones() {
  const zones = await api().Zones();
  const select = $("zone-select");
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
    option.textContent = item.city;
    option.title =
      item.id === "UTC"
        ? "UTC"
        : `${item.id} · ${item.abbr} ${item.offset}${item.dst ? " · verano" : ""}`;
    group.append(option);
  }

  select.value = zone;
}

async function updateZone() {
  const state = await api().Zone(zone);
  zoneText = state.id === "UTC" ? `UTC · ${state.offset}` : `${state.city} · ${state.text}`;
  $("zone-chip").textContent = state.text;
  $("zone-chip").title =
    state.id === "local"
      ? `Hora local · ${state.text}`
      : `${state.id} · ${state.text}${state.dst ? " · horario de verano" : ""}`;
  $("zone-chip").classList.toggle("is-dst", state.dst);
  $("footer-zone").textContent = zoneText;
  $("result-zone").textContent = hasDiff ? `Zona: ${zoneText}` : "";
}

async function setupDefaults() {
  let now = await api().Now(zone);
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(now)) {
    now = toInputValue(new Date());
  }

  $("date2").value = now;
  $("date1").value = shiftInputDays(now, -30);
  $("shift-date").value = now;
}

function wireEvents() {
  $("calc").addEventListener("click", calcDiff);
  $("add").addEventListener("click", () => doShift("add"));
  $("sub").addEventListener("click", () => doShift("sub"));

  $("swap").addEventListener("click", () => {
    const first = $("date1").value;
    $("date1").value = $("date2").value;
    $("date2").value = first;
    if (hasDiff) calcDiff();
  });

  $("now1").addEventListener("click", () => setNow($("date1")));
  $("now2").addEventListener("click", () => setNow($("date2")));

  $("inclusive").addEventListener("change", () => {
    if (hasDiff) calcDiff();
  });

  $("zone-select").addEventListener("change", async (event) => {
    zone = event.target.value;
    try {
      await updateZone();
    } catch (err) {
      showError("diff-error", cleanError(err));
      return;
    }
    if (hasDiff) calcDiff();
    if (hasShift && lastOperation) doShift(lastOperation);
  });

  for (const id of ["date1", "date2"]) {
    $(id).addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        calcDiff();
      }
    });
    $(id).addEventListener("input", () => clearError("diff-error"));
  }

  for (const id of ["shift-date", "shift-duration"]) {
    $(id).addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        doShift("add");
      }
    });
    $(id).addEventListener("input", () => clearError("shift-error"));
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

async function boot() {
  if (!(await waitForBindings())) {
    showError("diff-error", "No se pudo conectar con el backend de la aplicación.");
    return;
  }

  ready = true;
  window.scrollTo(0, 0);
  await loadZones();
  await updateZone();
  await setupDefaults();
  await calcDiff();
  window.scrollTo(0, 0);
  setTimeout(() => window.scrollTo(0, 0), 250);
}

document.addEventListener("DOMContentLoaded", () => {
  wireEvents();
  boot().catch((err) => showError("diff-error", cleanError(err)));
});
