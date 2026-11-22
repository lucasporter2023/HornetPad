(() => {
  "use strict";
  const UNITS = window.HORNETPAD_UNITS;
  const deck = document.getElementById("deck");
  const statusEl = document.getElementById("status");
  const statusText = document.getElementById("statusText");
  const inApp = /HornetPadApp/.test(navigator.userAgent);
  if (inApp) document.body.classList.add("in-app");

  let order = ["LDDI", "AMPCD", "RDDI"];
  let built = false;
  let serverUp = false;
  const screens = {}; // name -> {screen, img}
  const osbs = [];    // {el, n} for layout

  // ---------- commands ----------
  let chain = Promise.resolve();
  function send(cmd) {
    chain = chain.then(() =>
      fetch("/cmd", { method: "POST", body: cmd, cache: "no-store" }).catch(() => {})
    );
  }

  function haptic() {
    try {
      if (window.HornetPadNative) window.HornetPadNative.haptic();
      else if (navigator.vibrate) navigator.vibrate(12);
    } catch (_) {}
  }

  // Generic press/release wiring with multi-touch support.
  function bindPress(el, onDown, onUp) {
    let active = null;
    el.addEventListener("pointerdown", (e) => {
      e.preventDefault();
      if (active !== null) return;
      active = e.pointerId;
      try { el.setPointerCapture(e.pointerId); } catch (_) {}
      el.classList.add("down");
      haptic();
      onDown();
    });
    const end = (e) => {
      if (active === null || e.pointerId !== active) return;
      active = null;
      el.classList.remove("down");
      onUp();
    };
    el.addEventListener("pointerup", end);
    el.addEventListener("pointercancel", end);
    el.addEventListener("lostpointercapture", end);
    el.addEventListener("contextmenu", (e) => e.preventDefault());
  }

  function makeControl(item, groupEl) {
    const el = document.createElement("div");
    el.className = "ctl";
    el.textContent = item.label;
    if (item.type === "set") {
      bindPress(el, () => {
        send(`${item.cmd} ${item.value}`);
        groupEl.querySelectorAll(".ctl").forEach((c) => c.classList.remove("selected"));
        el.classList.add("selected");
      }, () => {});
    } else if (item.type === "rocker") {
      bindPress(el, () => send(`${item.cmd} ${item.value}`), () => send(`${item.cmd} ${item.center}`));
    } else if (item.type === "pot") {
      let timer = null;
      const step = (item.step > 0 ? "+" : "") + item.step;
      bindPress(el, () => {
        send(`${item.cmd} ${step}`);
        timer = setInterval(() => send(`${item.cmd} ${step}`), 110);
      }, () => { clearInterval(timer); timer = null; });
    }
    return el;
  }

  // ---------- build ----------
  function buildUnit(name) {
    const def = UNITS[name];
    const unit = document.createElement("div");
    unit.className = "unit";

    const bezel = document.createElement("div");
    bezel.className = "bezel";
    const screen = document.createElement("div");
    screen.className = "screen";
    const img = document.createElement("img");
    img.alt = "";
    img.draggable = false;
    const nosig = document.createElement("div");
    nosig.className = "nosig";
    nosig.innerHTML = `${def.title}<br>NO SIGNAL`;
    screen.append(img, nosig);
    bezel.appendChild(screen);

    const title = document.createElement("div");
    title.className = "title";
    title.textContent = def.title;
    bezel.appendChild(title);

    for (let n = 1; n <= 20; n++) {
      const b = document.createElement("div");
      b.className = "osb";
      b.innerHTML = `<span>${n}</span>`;
      const cmd = def.osbPrefix + String(n).padStart(2, "0");
      bindPress(b, () => send(`${cmd} 1`), () => send(`${cmd} 0`));
      bezel.appendChild(b);
      osbs.push({ el: b, n });
    }

    const strip = document.createElement("div");
    strip.className = "strip";
    for (const g of def.groups) {
      const gEl = document.createElement("div");
      gEl.className = "group";
      gEl.style.setProperty("--n", g.items.length);
      const gname = document.createElement("div");
      gname.className = "gname";
      gname.textContent = g.name;
      const row = document.createElement("div");
      row.className = "row";
      for (const item of g.items) row.appendChild(makeControl(item, row));
      gEl.append(gname, row);
      strip.appendChild(gEl);
    }

    unit.append(bezel, strip);
    screens[name] = { screen, img };
    img.addEventListener("load", () => screen.classList.add("live"));
    img.addEventListener("error", () => {
      screen.classList.remove("live");
      setTimeout(() => connectStream(name), 1500);
    });
    return unit;
  }

  function build() {
    deck.innerHTML = "";
    osbs.length = 0;
    for (const name of order) if (UNITS[name]) deck.appendChild(buildUnit(name));
    built = true;
    layout();
  }

  // ---------- layout ----------
  function layout() {
    const W = window.innerWidth, H = window.innerHeight;
    const gap = Math.max(6, Math.round(W * 0.008));
    // bezel + strip (0.17 S) + spacing (0.025 S) must fit the height
    const s = Math.floor(Math.min((W - gap * 4) / 3, (H - gap * 2) / 1.2));
    document.documentElement.style.setProperty("--s", s + "px");
    document.documentElement.style.setProperty("--gap", gap + "px");
    // give spare height (16:10 screens) to the control strip for bigger buttons
    const strip = Math.max(s * 0.17, Math.min(s * 0.3, H - gap * 2 - s * 1.025));
    document.documentElement.style.setProperty("--strip", Math.floor(strip) + "px");

    const inset = s * 0.135;
    const span = s - 2 * inset;
    const slot = span / 5;
    const long = slot * 0.74;
    const short = inset * 0.62;
    const edge = (inset - short) / 2;
    for (const { el, n } of osbs) {
      const p = window.hornetpadOsbPosition(n);
      const along = inset + slot * p.slot + (slot - long) / 2;
      let x, y, w, h;
      if (p.side === "top" || p.side === "bottom") {
        w = long; h = short; x = along;
        y = p.side === "top" ? edge : s - inset + edge;
      } else {
        w = short; h = long; y = along;
        x = p.side === "left" ? edge : s - inset + edge;
      }
      Object.assign(el.style, { left: x + "px", top: y + "px", width: w + "px", height: h + "px" });
    }
  }
  window.addEventListener("resize", layout);

  // ---------- video ----------
  function connectStream(name) {
    const s = screens[name];
    if (!s || document.hidden) return;
    s.img.src = `/stream/${name}?t=${Date.now()}`;
  }
  function connectAll() { for (const n of Object.keys(screens)) connectStream(n); }
  function dropAll() {
    for (const s of Object.values(screens)) { s.img.removeAttribute("src"); s.screen.classList.remove("live"); }
  }
  document.addEventListener("visibilitychange", () => {
    // stop streams while hidden so the PC stops capturing
    if (document.hidden) dropAll(); else connectAll();
  });

  // ---------- server health ----------
  async function poll() {
    let info = null;
    try {
      const ctl = new AbortController();
      const t = setTimeout(() => ctl.abort(), 2500);
      const r = await fetch("/api/info", { cache: "no-store", signal: ctl.signal });
      clearTimeout(t);
      if (r.ok) info = await r.json();
    } catch (_) {}

    if (info) {
      if (!built || JSON.stringify(info.order) !== JSON.stringify(order)) {
        if (Array.isArray(info.order)) order = info.order;
        build();
        connectAll();
      } else if (!serverUp) {
        connectAll();
      }
      serverUp = true;
      statusEl.className = "up";
      statusText.textContent = `Connected - HornetPad ${info.version}`;
    } else {
      if (!built) build();
      if (serverUp) dropAll();
      serverUp = false;
      statusEl.className = "down";
      statusText.textContent = "Waiting for HornetPad.exe on the PC (USB cable + USB debugging)…";
    }
    setTimeout(poll, 2000);
  }

  // ---------- misc ----------
  document.getElementById("fs").addEventListener("click", () => {
    const d = document.documentElement;
    if (!document.fullscreenElement && d.requestFullscreen) d.requestFullscreen().catch(() => {});
    else if (document.exitFullscreen) document.exitFullscreen();
  });
  if (navigator.wakeLock) {
    const lock = () => navigator.wakeLock.request("screen").catch(() => {});
    lock();
    document.addEventListener("visibilitychange", () => { if (!document.hidden) lock(); });
  }
  if (new URLSearchParams(location.search).has("nonumbers")) document.body.classList.add("hide-numbers");

  build();
  poll();
})();
