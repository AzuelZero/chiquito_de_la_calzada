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
let lastDiff = null;
let lastShift = null;
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

function splitHuman(human) {
  const match = human.match(/^(.*?) \(([^)]+)\)$/);
  if (!match) {
    return [human, ""];
  }
  return [match[1], capitalize(match[2])];
}

function hideDiffResult() {
  hasDiff = false;
  lastDiff = null;
  $("diff-result").hidden = true;
  $("totals-panel").hidden = true;
  $("diff-announce").textContent = "";
}

function hideShiftResult() {
  hasShift = false;
  $("shift-result").hidden = true;
  $("shift-empty").hidden = false;
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

// Unidades que merecen verse sin desplegar, según la magnitud del resultado.
const primaryUnits = (days24) => {
  if (days24 >= 730) return ["Años", "Meses", "Semanas", "Días de 24 h"];
  if (days24 >= 61) return ["Meses", "Semanas", "Días de 24 h", "Horas"];
  if (days24 >= 14) return ["Semanas", "Días de 24 h", "Horas", "Minutos"];
  if (days24 >= 2) return ["Días de 24 h", "Horas", "Minutos", "Segundos"];
  return ["Horas", "Minutos", "Segundos", "Milisegundos"];
};

let totalsExpanded = false;
let lastPrimary = [];

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
    if (hint) {
      labelEl.title = hint;
    }

    const valueEl = document.createElement("div");
    valueEl.className = "total-value";
    const n = Number(value);
    valueEl.textContent = Number.isFinite(n) ? (integer ? nf0 : nf).format(n) : "—";

    row.append(labelEl, valueEl);
    wrap.append(row);
  }
  applyTotalsVisibility();
}

// Muestra las unidades principales o todas, según el desplegable.
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

async function copyResult() {
  if (!lastDiff) return;
  const button = $("copy-result");
  try {
    await copyText(buildDiffText(lastDiff));
    button.textContent = "Copiado";
    button.classList.add("is-copied");
    $("copy-status").textContent = "Resultado copiado al portapapeles.";
    setTimeout(() => {
      button.textContent = "Copiar resultado";
      button.classList.remove("is-copied");
      $("copy-status").textContent = "";
    }, 1800);
  } catch {
    button.textContent = "No se pudo copiar";
    $("copy-status").textContent = "No se pudo copiar el resultado.";
    setTimeout(() => {
      button.textContent = "Copiar resultado";
      $("copy-status").textContent = "";
    }, 2200);
  }
}

function buildShiftText(result) {
  const parts = [
    `${capitalize(result.operation)} ${result.duration}: ${result.resultHuman}.`,
    `Desde ${result.startHuman}.`,
    `ISO 8601: ${result.iso}.`,
  ];
  if (zoneText) {
    parts.push(`Zona: ${zoneText}.`);
  }
  return parts.join("\n");
}

async function copyShiftResult() {
  if (!lastShift) return;
  const button = $("copy-shift");
  try {
    await copyText(buildShiftText(lastShift));
    button.textContent = "Copiado";
    button.classList.add("is-copied");
    $("copy-status").textContent = "Resultado copiado al portapapeles.";
  } catch {
    button.textContent = "No se pudo copiar";
    $("copy-status").textContent = "No se pudo copiar el resultado.";
  } finally {
    setTimeout(() => {
      button.textContent = "Copiar resultado";
      button.classList.remove("is-copied");
      $("copy-status").textContent = "";
    }, 1800);
  }
}

// El ¡JARL! del intercambio: bengala visual y el grito real de Chiquito.
// El clip viaja dentro de la app; no hace falta conexión.
let jarlAudio = (() => {
  try {
    const audio = new Audio("assets/jarl.mp3");
    audio.preload = "auto";
    audio.volume = 0.9;
    return audio;
  } catch {
    return null;
  }
})();

function jarl() {
  const reduced = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  if (!reduced) {
    document.querySelector(".jarl")?.remove();
    const el = document.createElement("span");
    el.className = "jarl";
    el.textContent = "¡JARL!";
    el.setAttribute("aria-hidden", "true");
    document.querySelector(".date-row").append(el);
    setTimeout(() => el.remove(), 950);
  }
  try {
    if (!jarlAudio) {
      jarlAudio = new Audio("assets/jarl.mp3");
      jarlAudio.preload = "auto";
      jarlAudio.volume = 0.9;
    }
    jarlAudio.currentTime = 0;
    jarlAudio.play().catch(() => {});
  } catch {
    // Sin audio disponible: el JARL visual basta.
  }
}

function setupTabs() {
  const tabs = [$("tab-diff"), $("tab-shift")];
  const panels = [$("panel-diff"), $("panel-shift")];

  function activate(tab, focus) {
    tabs.forEach((item, index) => {
      const active = item === tab;
      item.classList.toggle("is-active", active);
      item.setAttribute("aria-selected", String(active));
      item.tabIndex = active ? 0 : -1;
      panels[index].hidden = !active;
      if (active) {
        panels[index].classList.remove("is-entering");
        void panels[index].offsetWidth;
        panels[index].classList.add("is-entering");
      }
    });
    if (focus) tab.focus();
  }

  tabs.forEach((tab) => {
    tab.addEventListener("click", () => activate(tab, false));
    tab.addEventListener("keydown", (event) => {
      const index = tabs.indexOf(tab);
      let next = null;
      if (event.key === "ArrowRight") {
        next = tabs[(index + 1) % tabs.length];
      } else if (event.key === "ArrowLeft") {
        next = tabs[(index - 1 + tabs.length) % tabs.length];
      } else if (event.key === "Home") {
        next = tabs[0];
      } else if (event.key === "End") {
        next = tabs[tabs.length - 1];
      }
      if (next) {
        event.preventDefault();
        activate(next, true);
      }
    });
  });
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
    // El resultado anterior se queda en pantalla: un typo no te quita
    // la respuesta que ya tenías.
    showError("diff-error", cleanError(err));
  } finally {
    button.disabled = false;
    button.textContent = "Calcular diferencia";
  }
}

// El resultado se recalcula solo al editar las fechas: nada de números
// rancios en pantalla. Con las fechas incompletas, se retira en silencio.
let diffTimer = 0;

function scheduleDiff(delay = 300) {
  clearTimeout(diffTimer);
  diffTimer = setTimeout(() => {
    if (!ready) return;
    if ($("date1").value && $("date2").value) {
      calcDiff();
    } else {
      hideDiffResult();
      clearError("diff-error");
    }
  }, delay);
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
    scheduleDiff(0);
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
  $("result-zone").textContent = hasDiff ? state.text : "";
  const note = $("panel-zone");
  if (note) note.textContent = state.text;
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
  $("copy-result").addEventListener("click", copyResult);
  $("copy-shift").addEventListener("click", copyShiftResult);
  $("totals-toggle").addEventListener("click", () => {
    totalsExpanded = !totalsExpanded;
    applyTotalsVisibility();
  });

  $("swap").addEventListener("click", () => {
    const first = $("date1").value;
    $("date1").value = $("date2").value;
    $("date2").value = first;
    jarl();
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
      // El error cae en la pestaña que está a la vista, no en la oculta.
      showError($("panel-shift").hidden ? "diff-error" : "shift-error", cleanError(err));
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
        // Enter repite la última operación usada; la primera vez, suma.
        doShift(lastOperation || "add");
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
  setupTabs();
  wireEvents();
  boot().catch((err) => showError("diff-error", cleanError(err)));
});
